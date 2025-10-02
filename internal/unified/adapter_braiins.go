//go:build ignore
// +build ignore

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

	a.host = details.GetHostname()
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

	poolGroups, err := a.client.GetPoolGroups()
	if err != nil {
		return nil, fmt.Errorf("failed to get pools: %w", err)
	}

	cooling, _ := a.client.GetCoolingState() // Optional, might fail

	// Build unified summary
	unified := &UnifiedSummary{
		IPAddress:    a.host,
		APIType:      APITypeBraiins,
		Model:        details.GetHardwareIdentifier(),
		Firmware:     "Braiins OS+",
		Version:      details.GetBosVersion().GetCurrent(),
		Timestamp:    time.Now(),
		ResponseTime: time.Since(start),
	}

	// Mining status - details.Status is a MinerStatus enum, not a string
	switch details.GetMinerStatus() {
	case 1: // MINER_STATUS_MINING
		unified.Status = StatusRunning
	case 2: // MINER_STATUS_PAUSED
		unified.Status = StatusPaused
	case 3: // MINER_STATUS_STOPPED
		unified.Status = StatusStopped
	case 4: // MINER_STATUS_TUNING
		unified.Status = StatusTuning
	default:
		unified.Status = StatusUnknown
	}
	unified.StatusDetail = details.GetMinerStatus().String()

	// Performance metrics
	unified.HashRate = stats.GetHashrate_15M().GetTerahashPerSecond()
	unified.HashRateUnit = "TH/s"
	unified.HashRate5s = stats.GetHashrate_5S().GetTerahashPerSecond()
	unified.HashRate1m = stats.GetHashrate_1M().GetTerahashPerSecond()
	unified.HashRate5m = stats.GetHashrate_5M().GetTerahashPerSecond()
	unified.HashRate15m = stats.GetHashrate_15M().GetTerahashPerSecond()

	// Power and efficiency
	unified.PowerUsage = float64(stats.GetPowerConsumption().GetWatt())
	unified.Efficiency = stats.GetEfficiency().GetJoulePerTerahash()

	// Hardware health
	hashboardsList := hashboards.GetHashboards()
	unified.BoardsTotal = len(hashboardsList)
	unified.BoardsActive = 0
	unified.ChipsTotal = 0
	unified.ChipsActive = 0

	totalTemp := 0.0
	tempCount := 0
	unified.TempMax = 0.0
	unified.TempMin = 999.0

	for _, board := range hashboardsList {
		// Check if board is active (no critical errors)
		if board.GetStatus() == 0 { // Assuming 0 is OK/Active
			unified.BoardsActive++
		}
		unified.ChipsTotal += int(board.GetChipsTotal())
		unified.ChipsActive += int(board.GetChipsOk())

		// Temperature tracking
		temp := board.GetTemperature().GetCelsius()
		if temp > 0 {
			totalTemp += temp
			tempCount++
			if temp > unified.TempMax {
				unified.TempMax = temp
			}
			if temp < unified.TempMin {
				unified.TempMin = temp
			}
		}
	}

	// Calculate average temperature
	if tempCount > 0 {
		unified.TempAvg = totalTemp / float64(tempCount)
	}

	// Fans
	if cooling != nil {
		fans := cooling.GetFans()
		unified.FanCount = len(fans)
		if unified.FanCount > 0 {
			total := 0
			for _, fan := range fans {
				total += int(fan.GetSpeed())
			}
			unified.FanSpeedAvg = total / unified.FanCount
		}
	}

	// Work statistics
	unified.Accepted = int(stats.GetSharesAccepted())
	unified.Rejected = int(stats.GetSharesRejected())
	unified.HWErrors = 0 // Would need to aggregate from boards

	// Pool information - poolGroups contains multiple pool groups
	unified.PoolCount = 0
	if poolGroups != nil && len(poolGroups.GetPoolGroups()) > 0 {
		for _, group := range poolGroups.GetPoolGroups() {
			unified.PoolCount += len(group.GetPools())
			// Get first active pool
			if unified.ActivePool == "" {
				for _, pool := range group.GetPools() {
					if pool.GetEnabled() {
						unified.ActivePool = pool.GetUrl()
						break
					}
				}
			}
		}
	}

	// Uptime
	unified.Uptime = time.Duration(details.GetUptimeS()) * time.Second

	// Errors and warnings
	for i, board := range hashboardsList {
		if board.GetStatus() != 0 { // Assuming 0 is OK
			unified.Errors = append(unified.Errors,
				fmt.Sprintf("Board %d: status %d", i, board.GetStatus()))
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

	hashboardsResp, err := a.client.GetHashboards()
	if err != nil {
		return nil, fmt.Errorf("failed to get hashboards: %w", err)
	}

	hashboards := hashboardsResp.GetHashboards()
	devices := make([]Device, 0, len(hashboards))
	for i, board := range hashboards {
		status := "Alive"
		if board.GetStatus() != 0 { // Assuming 0 is OK/Active
			status = "Dead"
		}

		devices = append(devices, Device{
			Index:       i,
			Name:        fmt.Sprintf("Board %d", i),
			Status:      status,
			Temperature: board.GetTemperature().GetCelsius(),
			HashRate:    board.GetHashrate().GetTerahashPerSecond(),
			Chips:       int(board.GetChipsTotal()),
			ChipsActive: int(board.GetChipsOk()),
			Frequency:   int(board.GetFrequency()),
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

	poolGroups, err := a.client.GetPoolGroups()
	if err != nil {
		return nil, fmt.Errorf("failed to get pools: %w", err)
	}

	pools := make([]Pool, 0)
	poolID := 0
	for _, group := range poolGroups.GetPoolGroups() {
		for _, p := range group.GetPools() {
			status := "Dead"
			if p.GetEnabled() {
				status = "Alive"
			}

			pools = append(pools, Pool{
				ID:       poolID,
				URL:      p.GetUrl(),
				User:     p.GetUser(),
				Status:   status,
				Priority: poolID,
				Accepted: 0, // Not available in this response
				Rejected: 0, // Not available in this response
			})
			poolID++
		}
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
	return a.client.Restart()
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
	return a.client.Reboot()
}

func (a *BraiinsAdapter) GetLogs(ctx context.Context, lines int) ([]string, error) {
	// Would need to implement log fetching in Braiins client
	return nil, ErrNotSupported
}

// Configuration

func (a *BraiinsAdapter) GetConfig(ctx context.Context) (map[string]interface{}, error) {
	tunerState, err := a.client.GetTunerState()
	if err != nil {
		return nil, err
	}

	minerConfig, err := a.client.GetMinerConfiguration()
	if err != nil {
		return nil, err
	}

	// Convert to generic map
	return map[string]interface{}{
		"tuner": tunerState,
		"miner": minerConfig,
	}, nil
}

func (a *BraiinsAdapter) SetPowerTarget(ctx context.Context, watts int) error {
	_, err := a.client.SetPowerTarget(uint64(watts))
	return err
}

func (a *BraiinsAdapter) SetFanSpeed(ctx context.Context, percent int) error {
	// Braiins uses different fan control mechanism
	return ErrNotSupported
}