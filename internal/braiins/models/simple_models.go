package models

import (
	"time"

	pb "github.com/sinkers/miner-cli/internal/braiins/bos/v1"
)

// SimpleMinerInfo represents basic miner information
type SimpleMinerInfo struct {
	UID            string
	Hostname       string
	MACAddress     string
	Platform       string
	BOSMode        string
	BOSVersion     string
	SystemUptime   time.Duration
	BOSMinerUptime time.Duration
	Status         string
}

// SimpleMinerStats represents mining statistics
type SimpleMinerStats struct {
	HashRate5s      float64 // TH/s
	HashRate1m      float64 // TH/s
	HashRate15m     float64 // TH/s
	HashRate24h     float64 // TH/s
	PowerUsage      float64 // Watts
	Efficiency      float64 // J/TH
	AcceptedShares  uint64
	RejectedShares  uint64
	Temperature     float64 // Celsius average
	FanSpeeds       []int   // RPM
}

// SimpleHashboard represents hashboard information
type SimpleHashboard struct {
	Index           int
	Status          string
	HashRate        float64 // TH/s
	Temperature     float64 // Celsius
	Chips           int
	Voltage         float64
	Frequency       float64
}

// SimplePool represents pool information
type SimplePool struct {
	URL            string
	User           string
	Password       string
	Enabled        bool
	Active         bool
	AcceptedShares uint64
	RejectedShares uint64
}

// SimplePoolGroup represents a group of pools
type SimplePoolGroup struct {
	Name  string
	Pools []SimplePool
}

// Conversion functions

// ConvertMinerDetails converts GetMinerDetailsResponse to SimpleMinerInfo
func ConvertMinerDetails(resp *pb.GetMinerDetailsResponse) *SimpleMinerInfo {
	if resp == nil {
		return nil
	}

	info := &SimpleMinerInfo{
		UID:        resp.Uid,
		Hostname:   resp.Hostname,
		MACAddress: resp.MacAddress,
	}

	// Convert platform
	if resp.Platform != pb.Platform_PLATFORM_UNSPECIFIED {
		info.Platform = resp.Platform.String()
	}

	// Convert BOS mode
	if resp.BosMode != pb.BosMode_BOS_MODE_UNSPECIFIED {
		info.BOSMode = resp.BosMode.String()
	}

	// Convert BOS version
	if resp.BosVersion != nil {
		info.BOSVersion = resp.BosVersion.Current
	}

	// Convert uptimes
	info.SystemUptime = time.Duration(resp.SystemUptimeS) * time.Second
	info.BOSMinerUptime = time.Duration(resp.BosminerUptimeS) * time.Second

	// Convert status
	if resp.Status != pb.MinerStatus_MINER_STATUS_UNSPECIFIED {
		info.Status = resp.Status.String()
	}

	return info
}

// ConvertMinerStats converts GetMinerStatsResponse to SimpleMinerStats
func ConvertMinerStats(resp *pb.GetMinerStatsResponse) *SimpleMinerStats {
	if resp == nil {
		return nil
	}

	stats := &SimpleMinerStats{}

	// Convert hashrates from MinerStats
	if resp.MinerStats != nil && resp.MinerStats.RealHashrate != nil {
		// Convert from GH/s to TH/s (divide by 1000)
		if resp.MinerStats.RealHashrate.Last_5S != nil {
			stats.HashRate5s = resp.MinerStats.RealHashrate.Last_5S.GigahashPerSecond / 1000.0
		}
		if resp.MinerStats.RealHashrate.Last_1M != nil {
			stats.HashRate1m = resp.MinerStats.RealHashrate.Last_1M.GigahashPerSecond / 1000.0
		}
		if resp.MinerStats.RealHashrate.Last_15M != nil {
			stats.HashRate15m = resp.MinerStats.RealHashrate.Last_15M.GigahashPerSecond / 1000.0
		}
		if resp.MinerStats.RealHashrate.Last_24H != nil {
			stats.HashRate24h = resp.MinerStats.RealHashrate.Last_24H.GigahashPerSecond / 1000.0
		}
	}

	// Convert power from PowerStats
	if resp.PowerStats != nil && resp.PowerStats.ApproximatedConsumption != nil {
		stats.PowerUsage = float64(resp.PowerStats.ApproximatedConsumption.Watt)
	}

	// Calculate efficiency
	if stats.HashRate15m > 0 && stats.PowerUsage > 0 {
		stats.Efficiency = stats.PowerUsage / stats.HashRate15m
	}

	// Note: Share statistics would need to be retrieved from pool-specific calls
	// PoolStats structure doesn't directly contain shares in this version

	return stats
}

// ConvertHashboards converts GetHashboardsResponse to SimpleHashboard slice
func ConvertHashboards(resp *pb.GetHashboardsResponse) []SimpleHashboard {
	if resp == nil || resp.Hashboards == nil {
		return nil
	}

	var boards []SimpleHashboard

	for i, board := range resp.Hashboards {
		if board == nil {
			continue
		}

		hb := SimpleHashboard{
			Index: i, // Use the array index
		}

		// Set status based on enabled flag
		if board.Enabled {
			hb.Status = "enabled"
		} else {
			hb.Status = "disabled"
		}

		// Convert chip count
		if board.ChipsCount != nil {
			hb.Chips = int(board.ChipsCount.Value)
		}

		// Convert voltage
		if board.CurrentVoltage != nil {
			hb.Voltage = float64(board.CurrentVoltage.Volt)
		}

		// Convert frequency
		if board.CurrentFrequency != nil {
			hb.Frequency = float64(board.CurrentFrequency.Hertz) / 1e6 // Convert Hz to MHz
		}

		boards = append(boards, hb)
	}

	return boards
}

// ConvertPoolGroups converts GetPoolGroupsResponse to SimplePoolGroup slice
func ConvertPoolGroups(resp *pb.GetPoolGroupsResponse) []SimplePoolGroup {
	if resp == nil || resp.PoolGroups == nil {
		return nil
	}

	var groups []SimplePoolGroup

	for _, group := range resp.PoolGroups {
		if group == nil {
			continue
		}

		sg := SimplePoolGroup{
			Name: group.Name,
		}

		// Note: PoolGroup doesn't have a Pools field in the protobuf
		// Pools are retrieved separately or through different methods
		// For now, we'll leave the pools array empty
		// In a real implementation, you'd need to make additional calls to get pool details

		groups = append(groups, sg)
	}

	return groups
}

// ConvertCoolingState converts GetCoolingStateResponse to cooling information
func ConvertCoolingState(resp *pb.GetCoolingStateResponse) map[string]interface{} {
	if resp == nil {
		return nil
	}

	result := make(map[string]interface{})

	// Convert fan information
	if resp.Fans != nil {
		var fans []map[string]interface{}
		for i, fan := range resp.Fans {
			if fan == nil {
				continue
			}
			// FanState structure varies, using index as identifier
			fanInfo := map[string]interface{}{
				"index": i,
			}
			fans = append(fans, fanInfo)
		}
		result["fans"] = fans
	}

	// Convert highest temperature
	if resp.HighestTemperature != nil {
		result["highest_temperature"] = map[string]interface{}{
			"location": resp.HighestTemperature.Location,
		}
	}

	return result
}

// ConvertTunerState converts GetTunerStateResponse to performance information
func ConvertTunerState(resp *pb.GetTunerStateResponse) map[string]interface{} {
	if resp == nil {
		return nil
	}

	result := make(map[string]interface{})

	// Get overall tuner state
	result["tuner_state"] = resp.OverallTunerState.String()

	// Get mode-specific state
	switch mode := resp.ModeState.(type) {
	case *pb.GetTunerStateResponse_PowerTargetModeState:
		if mode.PowerTargetModeState != nil {
			result["mode"] = "power_target"
		}
	case *pb.GetTunerStateResponse_HashrateTargetModeState:
		if mode.HashrateTargetModeState != nil {
			result["mode"] = "hashrate_target"
		}
	}

	return result
}

// ConvertLicenseState converts GetLicenseStateResponse to license information
func ConvertLicenseState(resp *pb.GetLicenseStateResponse) map[string]interface{} {
	if resp == nil {
		return nil
	}

	result := make(map[string]interface{})

	// Get license state based on the type
	switch resp.State.(type) {
	case *pb.GetLicenseStateResponse_None:
		result["state"] = "none"
	case *pb.GetLicenseStateResponse_Limited:
		result["state"] = "limited"
		// Limited license details would be in the structure if available
	case *pb.GetLicenseStateResponse_Valid:
		result["state"] = "valid"
		// Valid license details would be in the structure if available
	case *pb.GetLicenseStateResponse_Expired:
		result["state"] = "expired"
		// Expired license details would be in the structure if available
	}

	return result
}