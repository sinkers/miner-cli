package unified

import (
	"context"
	"fmt"
	"time"

	"github.com/sinkers/miner-cli/internal/vnish/client"
)

// VNishAdapter adapts VNish API to the unified interface
type VNishAdapter struct {
	client    *client.Client
	connected bool
	host      string
}

// NewVNishAdapter creates a new VNish adapter
func NewVNishAdapter(vnishClient *client.Client) *VNishAdapter {
	return &VNishAdapter{
		client: vnishClient,
	}
}

// Connect establishes connection to the miner
func (a *VNishAdapter) Connect(ctx context.Context) error {
	// VNish uses HTTP, so we just verify connectivity
	info, err := a.client.GetInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to VNish: %w", err)
	}
	
	a.host = info.Hostname
	a.connected = true
	return nil
}

// Disconnect closes the connection
func (a *VNishAdapter) Disconnect() error {
	a.connected = false
	return nil
}

// IsConnected returns connection status
func (a *VNishAdapter) IsConnected() bool {
	return a.connected
}

// GetAPIType returns the API type
func (a *VNishAdapter) GetAPIType() APIType {
	return APITypeVNish
}

// GetSummary retrieves unified summary information
func (a *VNishAdapter) GetSummary(ctx context.Context) (*UnifiedSummary, error) {
	if !a.connected {
		if err := a.Connect(ctx); err != nil {
			return nil, err
		}
	}

	start := time.Now()

	// Get various VNish data
	info, err := a.client.GetInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get info: %w", err)
	}

	summary, err := a.client.GetSummary(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get summary: %w", err)
	}

	status, err := a.client.GetStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	// Build unified summary
	unified := &UnifiedSummary{
		IPAddress:    a.host,
		APIType:      APITypeVNish,
		Model:        info.Model,
		Firmware:     "VNish",
		Version:      info.Version,
		Timestamp:    time.Now(),
		ResponseTime: time.Since(start),
	}

	// Mining status
	if status.Mining {
		unified.Status = StatusRunning
	} else if status.Paused {
		unified.Status = StatusPaused
	} else {
		unified.Status = StatusStopped
	}
	unified.StatusDetail = status.Status

	// Performance metrics
	if summary.Performance != nil {
		unified.HashRate = summary.Performance.HashRate
		unified.HashRateUnit = summary.Performance.HashRateUnit
		unified.HashRate5s = summary.Performance.HashRate5s
		unified.HashRate1m = summary.Performance.HashRate1m
		unified.HashRate5m = summary.Performance.HashRate5m
		unified.HashRate15m = summary.Performance.HashRate15m
		
		// Power and efficiency
		unified.PowerUsage = summary.Performance.PowerUsage
		unified.Efficiency = summary.Performance.Efficiency
	}

	// Hardware health
	if summary.Hardware != nil {
		unified.BoardsTotal = summary.Hardware.BoardsTotal
		unified.BoardsActive = summary.Hardware.BoardsActive
		unified.ChipsTotal = summary.Hardware.ChipsTotal
		unified.ChipsActive = summary.Hardware.ChipsActive
	}

	// Temperature
	if summary.Temperature != nil {
		unified.TempAvg = summary.Temperature.Average
		unified.TempMax = summary.Temperature.Max
		unified.TempMin = summary.Temperature.Min
	}

	// Fans
	if summary.Fans != nil {
		unified.FanCount = len(summary.Fans.Speeds)
		if unified.FanCount > 0 {
			total := 0
			for _, speed := range summary.Fans.Speeds {
				total += speed
			}
			unified.FanSpeedAvg = total / unified.FanCount
		}
	}

	// Work statistics
	if summary.Work != nil {
		unified.Accepted = summary.Work.Accepted
		unified.Rejected = summary.Work.Rejected
		unified.HWErrors = summary.Work.HardwareErrors
	}

	// Pool information
	pools, _ := a.client.GetPools(ctx)
	if pools != nil {
		unified.PoolCount = len(pools.Pools)
		for _, pool := range pools.Pools {
			if pool.Active {
				unified.ActivePool = pool.URL
				break
			}
		}
	}

	// Uptime
	unified.Uptime = time.Duration(info.Uptime) * time.Second

	// Errors and warnings
	if status.Errors != nil && len(status.Errors) > 0 {
		unified.Errors = status.Errors
	}
	if status.Warnings != nil && len(status.Warnings) > 0 {
		unified.Warnings = status.Warnings
	}

	unified.RawResponse = map[string]interface{}{
		"info":    info,
		"summary": summary,
		"status":  status,
	}

	return unified, nil
}

// GetDevices retrieves device information
func (a *VNishAdapter) GetDevices(ctx context.Context) ([]Device, error) {
	// Implementation would fetch chain/board information from VNish
	return nil, ErrNotSupported // Placeholder
}

// GetPools retrieves pool information
func (a *VNishAdapter) GetPools(ctx context.Context) ([]Pool, error) {
	// Implementation would fetch pool information from VNish
	return nil, ErrNotSupported // Placeholder
}

// Control operations

func (a *VNishAdapter) StartMining(ctx context.Context) error {
	return a.client.StartMining(ctx)
}

func (a *VNishAdapter) StopMining(ctx context.Context) error {
	return a.client.StopMining(ctx)
}

func (a *VNishAdapter) RestartMining(ctx context.Context) error {
	return a.client.RestartMining(ctx)
}

func (a *VNishAdapter) PauseMining(ctx context.Context) error {
	return a.client.PauseMining(ctx)
}

func (a *VNishAdapter) ResumeMining(ctx context.Context) error {
	return a.client.ResumeMining(ctx)
}

// Pool management

func (a *VNishAdapter) SwitchPool(ctx context.Context, poolID int) error {
	return a.client.SwitchPool(ctx, poolID)
}

func (a *VNishAdapter) AddPool(ctx context.Context, pool Pool) error {
	// Would need to implement pool addition in VNish client
	return ErrNotSupported
}

func (a *VNishAdapter) RemovePool(ctx context.Context, poolID int) error {
	// Would need to implement pool removal in VNish client
	return ErrNotSupported
}

// System operations

func (a *VNishAdapter) Reboot(ctx context.Context) error {
	return a.client.Reboot(ctx)
}

func (a *VNishAdapter) GetLogs(ctx context.Context, lines int) ([]string, error) {
	logs, err := a.client.GetLogs(ctx, lines)
	if err != nil {
		return nil, err
	}
	
	// Convert log entries to strings
	logLines := make([]string, 0, len(logs.Entries))
	for _, entry := range logs.Entries {
		logLines = append(logLines, fmt.Sprintf("%s: %s", entry.Timestamp, entry.Message))
	}
	
	return logLines, nil
}

// Configuration

func (a *VNishAdapter) GetConfig(ctx context.Context) (map[string]interface{}, error) {
	// Would fetch various configuration from VNish
	return nil, ErrNotSupported // Placeholder
}

func (a *VNishAdapter) SetPowerTarget(ctx context.Context, watts int) error {
	// VNish supports power modes
	return a.client.SetPowerMode(ctx, fmt.Sprintf("%dW", watts))
}

func (a *VNishAdapter) SetFanSpeed(ctx context.Context, percent int) error {
	// VNish supports fan control
	return a.client.SetFanMode(ctx, "manual", percent)
}