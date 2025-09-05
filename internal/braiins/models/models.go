package models

import (
	"time"
	
	pb "github.com/sinkers/miner-cli/internal/braiins/bos/v1"
)

// MinerInfo represents simplified miner information
type MinerInfo struct {
	Hostname       string
	MacAddress     string
	Model          string
	Vendor         string
	HardwareVersion string
	FirmwareVersion string
	BOSVersion     string
	BOSMode        string
	Uptime         time.Duration
	Status         string
}

// MinerStats represents simplified mining statistics
type MinerStats struct {
	HashRateAverage   float64 // TH/s
	HashRate5s        float64 // TH/s
	HashRate1m        float64 // TH/s
	HashRate5m        float64 // TH/s
	HashRate15m       float64 // TH/s
	HashRate30m       float64 // TH/s
	HashRate1h        float64 // TH/s
	HashRate1d        float64 // TH/s
	PowerConsumption  float64 // Watts
	Efficiency        float64 // J/TH
	Temperature       float64 // Celsius
	FanSpeed          []int   // RPM for each fan
	AcceptedShares    uint64
	RejectedShares    uint64
	StaleShares       uint64
}

// HashboardInfo represents hashboard information
type HashboardInfo struct {
	ID            string
	Slot          int
	Status        string
	HashRate      float64 // TH/s
	Temperature   float64 // Celsius
	ChipCount     int
	ActiveChips   int
	Voltage       float64
	Frequency     float64
	HardwareErrors uint64
}

// PoolInfo represents mining pool information
type PoolInfo struct {
	ID            string
	URL           string
	User          string
	Status        string
	Enabled       bool
	Priority      int
	Accepted      uint64
	Rejected      uint64
	Stale         uint64
	LastDifficulty float64
	HashRate      float64 // TH/s
}

// PerformanceMode represents performance settings
type PerformanceMode struct {
	Mode          string  // "power_target", "hashrate_target", "manual"
	PowerTarget   float64 // Watts
	HashrateTarget float64 // TH/s
	Voltage       float64
	Frequency     float64
}

// CoolingInfo represents cooling configuration and status
type CoolingInfo struct {
	ImmersionMode bool
	FanMode       string // "auto", "manual", "immersion"
	FanSpeed      []FanInfo
	Temperature   TemperatureInfo
}

// FanInfo represents individual fan information
type FanInfo struct {
	ID       int
	Speed    int    // RPM
	SpeedPct int    // Percentage
	Status   string // "ok", "error", "missing"
}

// TemperatureInfo represents temperature readings
type TemperatureInfo struct {
	Board        []float64 // Temperature per hashboard
	Chip         []float64 // Average chip temperature per hashboard
	Intake       float64
	Exhaust      float64
}

// LicenseInfo represents license state
type LicenseInfo struct {
	State        string // "valid", "expired", "missing"
	Type         string // "per_device", "bulk"
	ValidUntil   *time.Time
	DeviceCount  int
	Fingerprint  string
}

// NetworkInfo represents network configuration
type NetworkInfo struct {
	Hostname     string
	IPAddress    string
	MACAddress   string
	Gateway      string
	DNS          []string
	DHCP         bool
}

// Conversion functions to convert from protobuf to models

// ConvertMinerDetails converts protobuf MinerDetailsResponse to MinerInfo
func ConvertMinerDetails(resp *pb.MinerDetailsResponse) *MinerInfo {
	if resp == nil {
		return nil
	}
	
	info := &MinerInfo{}
	
	if resp.Uid != nil {
		info.Hostname = resp.Uid.Hostname
		info.MacAddress = resp.Uid.MacAddress
	}
	
	if resp.MinerModel != nil {
		info.Model = resp.MinerModel.Model
		info.Vendor = resp.MinerModel.Vendor
		info.HardwareVersion = resp.MinerModel.HardwareVersion
	}
	
	if resp.FirmwareVersion != nil {
		info.FirmwareVersion = resp.FirmwareVersion.Firmware
		info.BOSVersion = resp.FirmwareVersion.Bos
		info.BOSMode = resp.FirmwareVersion.BosMode
	}
	
	return info
}

// ConvertMinerStats converts protobuf MinerStatsResponse to MinerStats
func ConvertMinerStats(resp *pb.MinerStatsResponse) *MinerStats {
	if resp == nil {
		return nil
	}
	
	stats := &MinerStats{}
	
	// Convert hashrates
	if resp.Hashrate != nil {
		if resp.Hashrate.Average != nil {
			stats.HashRateAverage = float64(resp.Hashrate.Average.Value)
		}
		if resp.Hashrate.Instant != nil {
			stats.HashRate5s = float64(resp.Hashrate.Instant.Value)
		}
		if resp.Hashrate.Average1M != nil {
			stats.HashRate1m = float64(resp.Hashrate.Average1M.Value)
		}
		if resp.Hashrate.Average5M != nil {
			stats.HashRate5m = float64(resp.Hashrate.Average5M.Value)
		}
		if resp.Hashrate.Average15M != nil {
			stats.HashRate15m = float64(resp.Hashrate.Average15M.Value)
		}
		if resp.Hashrate.Average30M != nil {
			stats.HashRate30m = float64(resp.Hashrate.Average30M.Value)
		}
		if resp.Hashrate.Average1H != nil {
			stats.HashRate1h = float64(resp.Hashrate.Average1H.Value)
		}
		if resp.Hashrate.Average24H != nil {
			stats.HashRate1d = float64(resp.Hashrate.Average24H.Value)
		}
	}
	
	// Convert power consumption
	if resp.Power != nil && resp.Power.Value != 0 {
		// Convert based on unit
		switch resp.Power.Unit {
		case pb.Power_WATT:
			stats.PowerConsumption = float64(resp.Power.Value)
		case pb.Power_KILOWATT:
			stats.PowerConsumption = float64(resp.Power.Value) * 1000
		}
	}
	
	// Calculate efficiency if both hashrate and power are available
	if stats.HashRateAverage > 0 && stats.PowerConsumption > 0 {
		stats.Efficiency = stats.PowerConsumption / stats.HashRateAverage
	}
	
	// Convert shares
	if resp.Shares != nil {
		stats.AcceptedShares = resp.Shares.Accepted
		stats.RejectedShares = resp.Shares.Rejected
		// Note: Stale shares might be in a different field
	}
	
	// Convert fans
	if resp.Fans != nil {
		for _, fan := range resp.Fans.Fans {
			if fan.Rpm != nil {
				stats.FanSpeed = append(stats.FanSpeed, int(fan.Rpm.Value))
			}
		}
	}
	
	// Convert temperature
	if resp.Temperature != nil && len(resp.Temperature.Boards) > 0 {
		// Take average of board temperatures
		var sum float64
		for _, board := range resp.Temperature.Boards {
			if board.Chip != nil {
				sum += float64(board.Chip.Value)
			}
		}
		if len(resp.Temperature.Boards) > 0 {
			stats.Temperature = sum / float64(len(resp.Temperature.Boards))
		}
	}
	
	return stats
}

// ConvertHashboards converts protobuf HashboardsResponse to HashboardInfo slice
func ConvertHashboards(resp *pb.HashboardsResponse) []HashboardInfo {
	if resp == nil || resp.Hashboards == nil {
		return nil
	}
	
	var boards []HashboardInfo
	
	for _, hb := range resp.Hashboards {
		board := HashboardInfo{
			ID:   hb.Uid,
			Slot: int(hb.Slot),
		}
		
		// Convert status
		if hb.Status != nil {
			board.Status = hb.Status.Status.String()
		}
		
		// Convert hashrate
		if hb.Hashrate != nil && hb.Hashrate.Average != nil {
			board.HashRate = float64(hb.Hashrate.Average.Value)
		}
		
		// Convert temperature
		if hb.Temperature != nil && hb.Temperature.Chip != nil {
			board.Temperature = float64(hb.Temperature.Chip.Value)
		}
		
		// Convert chips
		if hb.Chips != nil {
			board.ChipCount = int(hb.Chips.Total)
			board.ActiveChips = int(hb.Chips.Operational)
		}
		
		// Convert voltage
		if hb.Voltage != nil {
			board.Voltage = float64(hb.Voltage.Value)
		}
		
		// Convert frequency
		if hb.Frequency != nil {
			board.Frequency = float64(hb.Frequency.Value)
		}
		
		boards = append(boards, board)
	}
	
	return boards
}

// ConvertPools converts protobuf PoolsResponse to PoolInfo slice
func ConvertPools(resp *pb.PoolsResponse) []PoolInfo {
	if resp == nil || resp.Groups == nil || len(resp.Groups) == 0 {
		return nil
	}
	
	var pools []PoolInfo
	
	// Usually there's one group, iterate through all
	for _, group := range resp.Groups {
		if group.Pools == nil {
			continue
		}
		
		for i, pool := range group.Pools {
			poolInfo := PoolInfo{
				ID:       pool.Uid,
				URL:      pool.Url,
				User:     pool.User,
				Enabled:  pool.Enabled,
				Priority: i + 1, // Priority based on order
			}
			
			// Set status based on enabled and connection
			if pool.Enabled {
				poolInfo.Status = "enabled"
			} else {
				poolInfo.Status = "disabled"
			}
			
			// Convert statistics if available
			if pool.Statistics != nil {
				poolInfo.Accepted = pool.Statistics.Accepted
				poolInfo.Rejected = pool.Statistics.Rejected
				poolInfo.Stale = pool.Statistics.Stale
				
				if pool.Statistics.LastDifficulty != nil {
					poolInfo.LastDifficulty = pool.Statistics.LastDifficulty.Value
				}
			}
			
			pools = append(pools, poolInfo)
		}
	}
	
	return pools
}

// ConvertPerformanceMode converts protobuf GetPerformanceModeResponse to PerformanceMode
func ConvertPerformanceMode(resp *pb.GetPerformanceModeResponse) *PerformanceMode {
	if resp == nil || resp.TunerMode == nil {
		return nil
	}
	
	mode := &PerformanceMode{}
	
	// Determine mode type
	switch resp.TunerMode.Mode.(type) {
	case *pb.TunerMode_PowerTarget:
		mode.Mode = "power_target"
		if pt := resp.TunerMode.GetPowerTarget(); pt != nil && pt.PowerTarget != nil {
			mode.PowerTarget = float64(pt.PowerTarget.Value)
		}
	case *pb.TunerMode_HashrateTarget:
		mode.Mode = "hashrate_target"
		if ht := resp.TunerMode.GetHashrateTarget(); ht != nil && ht.HashrateTarget != nil {
			mode.HashrateTarget = float64(ht.HashrateTarget.Value)
		}
	case *pb.TunerMode_Manual:
		mode.Mode = "manual"
		if m := resp.TunerMode.GetManual(); m != nil {
			if m.GlobalVoltage != nil {
				mode.Voltage = float64(m.GlobalVoltage.Value)
			}
			if m.GlobalFrequency != nil {
				mode.Frequency = float64(m.GlobalFrequency.Value)
			}
		}
	}
	
	return mode
}

// ConvertCooling converts protobuf CoolingResponse to CoolingInfo
func ConvertCooling(resp *pb.CoolingResponse) *CoolingInfo {
	if resp == nil || resp.Cooling == nil {
		return nil
	}
	
	info := &CoolingInfo{
		ImmersionMode: resp.Cooling.ImmersionMode,
	}
	
	// Convert fan control mode
	if resp.Cooling.FanControl != nil {
		switch resp.Cooling.FanControl.Mode.(type) {
		case *pb.FanControl_Auto:
			info.FanMode = "auto"
		case *pb.FanControl_Manual:
			info.FanMode = "manual"
			// Could extract manual fan speed settings here
		case *pb.FanControl_Immersion:
			info.FanMode = "immersion"
		}
	}
	
	// Convert fans if available
	if resp.Fans != nil && resp.Fans.Fans != nil {
		for i, fan := range resp.Fans.Fans {
			fanInfo := FanInfo{
				ID: i,
			}
			
			if fan.Rpm != nil {
				fanInfo.Speed = int(fan.Rpm.Value)
			}
			
			if fan.SpeedPercent != nil {
				fanInfo.SpeedPct = int(fan.SpeedPercent.Value)
			}
			
			info.FanSpeed = append(info.FanSpeed, fanInfo)
		}
	}
	
	// Convert temperature if available
	if resp.Temperature != nil {
		info.Temperature = TemperatureInfo{}
		
		// Board temperatures
		if resp.Temperature.Boards != nil {
			for _, board := range resp.Temperature.Boards {
				if board.Board != nil {
					info.Temperature.Board = append(info.Temperature.Board, float64(board.Board.Value))
				}
				if board.Chip != nil {
					info.Temperature.Chip = append(info.Temperature.Chip, float64(board.Chip.Value))
				}
			}
		}
	}
	
	return info
}

// ConvertLicense converts protobuf LicenseStateResponse to LicenseInfo
func ConvertLicense(resp *pb.LicenseStateResponse) *LicenseInfo {
	if resp == nil || resp.State == nil {
		return nil
	}
	
	info := &LicenseInfo{}
	
	// Convert state
	switch resp.State.State {
	case pb.LicenseState_VALID:
		info.State = "valid"
	case pb.LicenseState_EXPIRED:
		info.State = "expired"
	case pb.LicenseState_MISSING:
		info.State = "missing"
	default:
		info.State = "unknown"
	}
	
	// Convert license details if available
	if resp.License != nil {
		// License type
		switch resp.License.Type {
		case pb.License_PER_DEVICE:
			info.Type = "per_device"
		case pb.License_BULK:
			info.Type = "bulk"
		}
		
		// Valid until date
		if resp.License.ValidUntil != nil {
			t := time.Unix(resp.License.ValidUntil.Seconds, int64(resp.License.ValidUntil.Nanos))
			info.ValidUntil = &t
		}
		
		// Device count for bulk licenses
		info.DeviceCount = int(resp.License.DeviceCount)
		
		// Fingerprint
		info.Fingerprint = resp.License.Ntp
	}
	
	return info
}