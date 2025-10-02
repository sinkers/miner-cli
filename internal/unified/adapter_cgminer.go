package unified

import (
	"context"
	"fmt"
	"time"

	cgminer "github.com/x1unix/go-cgminer-api"
)

// CGMinerAdapter adapts CGMiner API to the unified interface
type CGMinerAdapter struct {
	host      string
	port      int
	timeout   time.Duration
	client    *cgminer.CGMiner
	connected bool
}

// NewCGMinerAdapter creates a new CGMiner adapter
func NewCGMinerAdapter(host string, port int, timeout time.Duration) *CGMinerAdapter {
	return &CGMinerAdapter{
		host:    host,
		port:    port,
		timeout: timeout,
	}
}

// Connect establishes connection to the miner
func (a *CGMinerAdapter) Connect(ctx context.Context) error {
	a.client = cgminer.NewCGMiner(a.host, a.port, a.timeout)
	
	// Test connection with version command
	_, err := a.client.Version()
	if err != nil {
		return fmt.Errorf("failed to connect to CGMiner: %w", err)
	}
	
	a.connected = true
	return nil
}

// Disconnect closes the connection
func (a *CGMinerAdapter) Disconnect() error {
	a.connected = false
	a.client = nil
	return nil
}

// IsConnected returns connection status
func (a *CGMinerAdapter) IsConnected() bool {
	return a.connected
}

// GetAPIType returns the API type
func (a *CGMinerAdapter) GetAPIType() APIType {
	return APITypeCGMiner
}

// GetSummary retrieves unified summary information
func (a *CGMinerAdapter) GetSummary(ctx context.Context) (*UnifiedSummary, error) {
	if !a.connected {
		if err := a.Connect(ctx); err != nil {
			return nil, err
		}
	}

	start := time.Now()
	
	// Get summary data
	summary, err := a.client.Summary()
	if err != nil {
		return nil, fmt.Errorf("failed to get summary: %w", err)
	}

	// Get version for model/firmware info
	version, _ := a.client.Version()
	
	// Get device information for board count
	devs, _ := a.client.Devs()
	
	// Get pool information
	pools, _ := a.client.Pools()

	// Build unified summary
	unified := &UnifiedSummary{
		IPAddress:    a.host,
		APIType:      APITypeCGMiner,
		Timestamp:    time.Now(),
		ResponseTime: time.Since(start),
		RawResponse:  summary,
	}

	// Basic info from version
	if version != nil && len(*version) > 0 {
		v := (*version)[0]
		unified.Version = v.CGMiner
		if v.CGMiner == "" {
			unified.Version = v.BMMiner
		}
		// Try to infer model from Type field
		unified.Model = v.Type
	}

	// Mining status (CGMiner is always running if responding)
	unified.Status = StatusRunning
	unified.StatusDetail = "Mining"

	// Performance metrics from summary
	if summary != nil {
		// Convert MHS to THS for modern miners
		mhs5s := summary.MHS5s
		mhsAv := summary.MHSav
		
		// Check if it's actually GHS (newer miners)
		if summary.GHS5s > 0 {
			unified.HashRate5s = summary.GHS5s / 1000.0  // Convert GHS to THS
			unified.HashRate = summary.GHSav / 1000.0
			unified.HashRateUnit = "TH/s"
		} else if mhs5s > 1000000 { // Likely THS reported as MHS
			unified.HashRate5s = mhs5s / 1000000.0
			unified.HashRate = mhsAv / 1000000.0
			unified.HashRateUnit = "TH/s"
		} else if mhs5s > 1000 { // Likely GHS reported as MHS
			unified.HashRate5s = mhs5s / 1000.0
			unified.HashRate = mhsAv / 1000.0
			unified.HashRateUnit = "GH/s"
		} else {
			unified.HashRate5s = mhs5s
			unified.HashRate = mhsAv
			unified.HashRateUnit = "MH/s"
		}

		// Set other hashrate averages (CGMiner only provides 5s and average)
		unified.HashRate15m = unified.HashRate // Use average as 15m

		// Work statistics
		unified.Accepted = summary.Accepted
		unified.Rejected = summary.Rejected
		unified.HWErrors = summary.HardwareErrors
		
		// Uptime
		unified.Uptime = time.Duration(summary.Elapsed) * time.Second
	}

	// Device/Board information
	if devs != nil {
		unified.BoardsTotal = len(*devs)
		unified.BoardsActive = 0
		
		totalTemp := 0.0
		tempCount := 0
		unified.TempMax = 0.0
		unified.TempMin = 999.0
		
		for _, dev := range *devs {
			if dev.Status == "Alive" {
				unified.BoardsActive++
			}
			
			// Temperature tracking
			if dev.Temperature > 0 {
				totalTemp += dev.Temperature
				tempCount++
				if dev.Temperature > unified.TempMax {
					unified.TempMax = dev.Temperature
				}
				if dev.Temperature < unified.TempMin {
					unified.TempMin = dev.Temperature
				}
			}
			
			// Accumulate hardware errors
			unified.HWErrors += dev.HardwareErrors
		}
		
		// Calculate average temperature
		if tempCount > 0 {
			unified.TempAvg = totalTemp / float64(tempCount)
		}
	}

	// Pool information
	if pools != nil {
		unified.PoolCount = len(*pools)
		for _, pool := range *pools {
			if pool.Status == "Alive" {
				unified.ActivePool = pool.URL
				break
			}
		}
	}

	// Power and efficiency (not available in standard CGMiner)
	// These would need custom implementations or specific miner support
	unified.PowerUsage = 0
	unified.Efficiency = 0

	return unified, nil
}

// GetDevices retrieves device information
func (a *CGMinerAdapter) GetDevices(ctx context.Context) ([]Device, error) {
	if !a.connected {
		if err := a.Connect(ctx); err != nil {
			return nil, err
		}
	}

	devs, err := a.client.Devs()
	if err != nil {
		return nil, fmt.Errorf("failed to get devices: %w", err)
	}

	if devs == nil {
		return []Device{}, nil
	}

	devices := make([]Device, 0, len(*devs))
	for i, dev := range *devs {
		devices = append(devices, Device{
			Index:       i,
			Name:        dev.Name,
			Status:      dev.Status,
			Temperature: dev.Temperature,
			HashRate:    dev.MHSav,
			HWErrors:    dev.HardwareErrors,
		})
	}

	return devices, nil
}

// GetPools retrieves pool information
func (a *CGMinerAdapter) GetPools(ctx context.Context) ([]Pool, error) {
	if !a.connected {
		if err := a.Connect(ctx); err != nil {
			return nil, err
		}
	}

	cgPools, err := a.client.Pools()
	if err != nil {
		return nil, fmt.Errorf("failed to get pools: %w", err)
	}

	if cgPools == nil {
		return []Pool{}, nil
	}

	pools := make([]Pool, 0, len(cgPools))
	for _, p := range cgPools {
		pool := Pool{
			ID:       p.POOL,
			URL:      p.URL,
			User:     p.User,
			Status:   p.Status,
			Priority: p.Priority,
			Accepted: p.Accepted,
			Rejected: p.Rejected,
		}
		
		// Convert last share time
		if p.LastShareTime > 0 {
			pool.LastShare = time.Unix(p.LastShareTime, 0)
		}
		
		pools = append(pools, pool)
	}

	return pools, nil
}

// Control operations - CGMiner has limited control

// StartMining - CGMiner doesn't support start (always running)
func (a *CGMinerAdapter) StartMining(ctx context.Context) error {
	return ErrNotSupported
}

// StopMining - CGMiner supports quit
func (a *CGMinerAdapter) StopMining(ctx context.Context) error {
	if !a.connected {
		if err := a.Connect(ctx); err != nil {
			return err
		}
	}
	return a.client.Quit()
}

// RestartMining - CGMiner supports restart
func (a *CGMinerAdapter) RestartMining(ctx context.Context) error {
	if !a.connected {
		if err := a.Connect(ctx); err != nil {
			return err
		}
	}
	return a.client.Restart()
}

// PauseMining - Not supported in standard CGMiner
func (a *CGMinerAdapter) PauseMining(ctx context.Context) error {
	return ErrNotSupported
}

// ResumeMining - Not supported in standard CGMiner
func (a *CGMinerAdapter) ResumeMining(ctx context.Context) error {
	return ErrNotSupported
}

// Pool management

// SwitchPool switches to a different pool
func (a *CGMinerAdapter) SwitchPool(ctx context.Context, poolID int) error {
	if !a.connected {
		if err := a.Connect(ctx); err != nil {
			return err
		}
	}

	pools, err := a.client.Pools()
	if err != nil {
		return err
	}

	if poolID >= len(pools) {
		return fmt.Errorf("pool ID %d not found", poolID)
	}

	return a.client.SwitchPool(&pools[poolID])
}

// AddPool adds a new pool
func (a *CGMinerAdapter) AddPool(ctx context.Context, pool Pool) error {
	if !a.connected {
		if err := a.Connect(ctx); err != nil {
			return err
		}
	}

	return a.client.AddPool(pool.URL, pool.User, "")
}

// RemovePool removes a pool
func (a *CGMinerAdapter) RemovePool(ctx context.Context, poolID int) error {
	if !a.connected {
		if err := a.Connect(ctx); err != nil {
			return err
		}
	}

	pools, err := a.client.Pools()
	if err != nil {
		return err
	}

	if poolID >= len(pools) {
		return fmt.Errorf("pool ID %d not found", poolID)
	}

	return a.client.RemovePool(&pools[poolID])
}

// System operations - mostly not supported in CGMiner

// Reboot - Not supported in standard CGMiner
func (a *CGMinerAdapter) Reboot(ctx context.Context) error {
	return ErrNotSupported
}

// GetLogs - Not supported in standard CGMiner
func (a *CGMinerAdapter) GetLogs(ctx context.Context, lines int) ([]string, error) {
	return nil, ErrNotSupported
}

// GetConfig - Limited support in CGMiner
func (a *CGMinerAdapter) GetConfig(ctx context.Context) (map[string]interface{}, error) {
	if !a.connected {
		if err := a.Connect(ctx); err != nil {
			return nil, err
		}
	}

	// CGMiner has a config command but it's limited
	// Return basic configuration from pools and version
	config := make(map[string]interface{})
	
	if pools, err := a.client.Pools(); err == nil {
		config["pools"] = pools
	}
	
	if version, err := a.client.Version(); err == nil && version != nil && len(*version) > 0 {
		config["version"] = (*version)[0]
	}

	return config, nil
}

// SetPowerTarget - Not supported in standard CGMiner
func (a *CGMinerAdapter) SetPowerTarget(ctx context.Context, watts int) error {
	return ErrNotSupported
}

// SetFanSpeed - Not supported in standard CGMiner
func (a *CGMinerAdapter) SetFanSpeed(ctx context.Context, percent int) error {
	return ErrNotSupported
}