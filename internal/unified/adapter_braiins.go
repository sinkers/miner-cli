package unified

import (
	"context"
	"fmt"
	"time"

	"github.com/sinkers/miner-cli/internal/braiins/client"
)

// BraiinsAdapter adapts Braiins API to the unified interface
type BraiinsAdapter struct {
	client    *client.SimpleBraiinsClient
	connected bool
	host      string
}

// NewBraiinsAdapter creates a new Braiins adapter
func NewBraiinsAdapter(braiinsClient *client.SimpleBraiinsClient) *BraiinsAdapter {
	return &BraiinsAdapter{
		client: braiinsClient,
	}
}

// Connect establishes connection to the miner
func (a *BraiinsAdapter) Connect(ctx context.Context) error {
	// Connection is established during client creation
	// Just verify it works
	details, err := a.client.GetMinerDetails()
	if err != nil {
		return fmt.Errorf("failed to connect to Braiins: %w", err)
	}
	
	a.host = details.Hostname
	a.connected = true
	return nil
}

// Disconnect closes the connection
func (a *BraiinsAdapter) Disconnect() error {
	if a.client != nil {
		a.client.Close()
	}
	a.connected = false
	return nil
}

// IsConnected returns connection status
func (a *BraiinsAdapter) IsConnected() bool {
	return a.connected
}

// GetAPIType returns the API type
func (a *BraiinsAdapter) GetAPIType() APIType {
	return APITypeBraiins
}

// GetSummary retrieves unified summary information
func (a *BraiinsAdapter) GetSummary(ctx context.Context) (*UnifiedSummary, error) {
	if !a.connected {
		if err := a.Connect(ctx); err != nil {
			return nil, err
		}
	}

	start := time.Now()

	// Get various Braiins data
	details, err := a.client.GetMinerDetails()
	if err != nil {
		return nil, fmt.Errorf("failed to get miner details: %w", err)
	}

	stats, err := a.client.GetMinerStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get miner stats: %w", err)
	}

	hashboards, err := a.client.GetHashboards()
	if err != nil {
		return nil, fmt.Errorf("failed to get hashboards: %w", err)
	}

	pools, err := a.client.GetPools()
	if err != nil {
		return nil, fmt.Errorf("failed to get pools: %w", err)
	}

	cooling, _ := a.client.GetCooling() // Optional, might fail

	// Build unified summary
	unified := &UnifiedSummary{
		IPAddress:    a.host,
		APIType:      APITypeBraiins,
		Model:        details.Model,
		Firmware:     "Braiins OS+",
		Version:      details.BosVersion,
		Timestamp:    time.Now(),
		ResponseTime: time.Since(start),
	}

	// Mining status
	switch details.Status {
	case "Mining":
		unified.Status = StatusRunning
	case "Paused":
		unified.Status = StatusPaused
	case "Stopped":
		unified.Status = StatusStopped
	case "Tuning":
		unified.Status = StatusTuning
	default:
		unified.Status = StatusUnknown
	}
	unified.StatusDetail = details.Status

	// Performance metrics
	unified.HashRate = stats.HashRate15m
	unified.HashRateUnit = "TH/s"
	unified.HashRate5s = stats.HashRate5s
	unified.HashRate1m = stats.HashRate1m
	unified.HashRate5m = stats.HashRate5m
	unified.HashRate15m = stats.HashRate15m

	// Power and efficiency
	unified.PowerUsage = stats.PowerUsage
	unified.Efficiency = stats.Efficiency

	// Hardware health
	unified.BoardsTotal = len(hashboards)
	unified.BoardsActive = 0
	unified.ChipsTotal = 0
	unified.ChipsActive = 0
	
	totalTemp := 0.0
	tempCount := 0
	unified.TempMax = 0.0
	unified.TempMin = 999.0

	for _, board := range hashboards {
		if board.Active {
			unified.BoardsActive++
		}
		unified.ChipsTotal += board.Chips
		unified.ChipsActive += board.ChipsOK
		
		// Temperature tracking
		if board.Temperature > 0 {
			totalTemp += board.Temperature
			tempCount++
			if board.Temperature > unified.TempMax {
				unified.TempMax = board.Temperature
			}
			if board.Temperature < unified.TempMin {
				unified.TempMin = board.Temperature
			}
		}
	}

	// Calculate average temperature
	if tempCount > 0 {
		unified.TempAvg = totalTemp / float64(tempCount)
	}

	// Fans
	if cooling != nil {
		unified.FanCount = len(cooling.Fans)
		if unified.FanCount > 0 {
			total := 0
			for _, fan := range cooling.Fans {
				total += fan.Speed
			}
			unified.FanSpeedAvg = total / unified.FanCount
		}
	}

	// Work statistics
	unified.Accepted = stats.SharesAccepted
	unified.Rejected = stats.SharesRejected
	unified.HWErrors = 0 // Would need to aggregate from boards

	// Pool information
	unified.PoolCount = len(pools)
	for _, pool := range pools {
		if pool.Active {
			unified.ActivePool = pool.URL
			break
		}
	}

	// Uptime
	unified.Uptime = time.Duration(details.Uptime) * time.Second

	// Errors and warnings
	for _, board := range hashboards {
		if board.Status != "OK" {
			unified.Errors = append(unified.Errors, 
				fmt.Sprintf("Board %d: %s", board.Index, board.Status))
		}
	}

	unified.RawResponse = map[string]interface{}{
		"details":    details,
		"stats":      stats,
		"hashboards": hashboards,
		"pools":      pools,
		"cooling":    cooling,
	}

	return unified, nil
}

// GetDevices retrieves device information
func (a *BraiinsAdapter) GetDevices(ctx context.Context) ([]Device, error) {
	if !a.connected {
		if err := a.Connect(ctx); err != nil {
			return nil, err
		}
	}

	hashboards, err := a.client.GetHashboards()
	if err != nil {
		return nil, fmt.Errorf("failed to get hashboards: %w", err)
	}

	devices := make([]Device, 0, len(hashboards))
	for _, board := range hashboards {
		status := "Alive"
		if !board.Active {
			status = "Dead"
		}

		devices = append(devices, Device{
			Index:       board.Index,
			Name:        fmt.Sprintf("Board %d", board.Index),
			Status:      status,
			Temperature: board.Temperature,
			HashRate:    board.HashRate,
			Chips:       board.Chips,
			ChipsActive: board.ChipsOK,
			Frequency:   board.Frequency,
		})
	}

	return devices, nil
}

// GetPools retrieves pool information
func (a *BraiinsAdapter) GetPools(ctx context.Context) ([]Pool, error) {
	if !a.connected {
		if err := a.Connect(ctx); err != nil {
			return nil, err
		}
	}

	braiinsPools, err := a.client.GetPools()
	if err != nil {
		return nil, fmt.Errorf("failed to get pools: %w", err)
	}

	pools := make([]Pool, 0, len(braiinsPools))
	for i, p := range braiinsPools {
		status := "Dead"
		if p.Active {
			status = "Alive"
		}

		pools = append(pools, Pool{
			ID:       i,
			URL:      p.URL,
			User:     p.User,
			Status:   status,
			Priority: i, // Braiins uses order as priority
			Accepted: p.Accepted,
			Rejected: p.Rejected,
		})
	}

	return pools, nil
}

// Control operations

func (a *BraiinsAdapter) StartMining(ctx context.Context) error {
	return a.client.StartMining()
}

func (a *BraiinsAdapter) StopMining(ctx context.Context) error {
	return a.client.StopMining()
}

func (a *BraiinsAdapter) RestartMining(ctx context.Context) error {
	return a.client.RestartMining()
}

func (a *BraiinsAdapter) PauseMining(ctx context.Context) error {
	return a.client.PauseMining()
}

func (a *BraiinsAdapter) ResumeMining(ctx context.Context) error {
	return a.client.ResumeMining()
}

// Pool management

func (a *BraiinsAdapter) SwitchPool(ctx context.Context, poolID int) error {
	// Braiins doesn't have direct pool switching, would need to reorder
	return ErrNotSupported
}

func (a *BraiinsAdapter) AddPool(ctx context.Context, pool Pool) error {
	// Would need to implement in Braiins client
	return ErrNotSupported
}

func (a *BraiinsAdapter) RemovePool(ctx context.Context, poolID int) error {
	// Would need to implement in Braiins client
	return ErrNotSupported
}

// System operations

func (a *BraiinsAdapter) Reboot(ctx context.Context) error {
	return a.client.RebootSystem()
}

func (a *BraiinsAdapter) GetLogs(ctx context.Context, lines int) ([]string, error) {
	// Would need to implement log fetching in Braiins client
	return nil, ErrNotSupported
}

// Configuration

func (a *BraiinsAdapter) GetConfig(ctx context.Context) (map[string]interface{}, error) {
	config, err := a.client.GetCurrentTunerConfig()
	if err != nil {
		return nil, err
	}

	// Convert to generic map
	return map[string]interface{}{
		"tuner": config,
	}, nil
}

func (a *BraiinsAdapter) SetPowerTarget(ctx context.Context, watts int) error {
	return a.client.SetPowerTarget(float64(watts))
}

func (a *BraiinsAdapter) SetFanSpeed(ctx context.Context, percent int) error {
	// Braiins uses different fan control mechanism
	return ErrNotSupported
}