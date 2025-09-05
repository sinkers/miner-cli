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

	// Convert hashrates
	if resp.HashrateAverage != nil {
		if resp.HashrateAverage.Last_5S != nil {
			stats.HashRate5s = float64(resp.HashrateAverage.Last_5S.Value)
		}
		if resp.HashrateAverage.Last_1M != nil {
			stats.HashRate1m = float64(resp.HashrateAverage.Last_1M.Value)
		}
		if resp.HashrateAverage.Last_15M != nil {
			stats.HashRate15m = float64(resp.HashrateAverage.Last_15M.Value)
		}
		if resp.HashrateAverage.Last_24H != nil {
			stats.HashRate24h = float64(resp.HashrateAverage.Last_24H.Value)
		}
	}

	// Convert power
	if resp.PowerConsumption != nil {
		switch resp.PowerConsumption.Unit {
		case pb.Power_UNIT_WATT:
			stats.PowerUsage = float64(resp.PowerConsumption.Value)
		case pb.Power_UNIT_KILOWATT:
			stats.PowerUsage = float64(resp.PowerConsumption.Value) * 1000
		}
	}

	// Calculate efficiency
	if stats.HashRate15m > 0 && stats.PowerUsage > 0 {
		stats.Efficiency = stats.PowerUsage / stats.HashRate15m
	}

	// Convert temperature (average board temps)
	if resp.Temperature != nil {
		var totalTemp float64
		var count int
		for _, board := range resp.Temperature.Boards {
			if board != nil {
				totalTemp += float64(board.Value)
				count++
			}
		}
		if count > 0 {
			stats.Temperature = totalTemp / float64(count)
		}
	}

	// Convert fans
	if resp.Cooling != nil && resp.Cooling.Fans != nil {
		for _, fan := range resp.Cooling.Fans {
			if fan.Rpm != nil {
				stats.FanSpeeds = append(stats.FanSpeeds, int(fan.Rpm.Value))
			}
		}
	}

	// Convert shares
	if resp.Work != nil && resp.Work.PoolWork != nil {
		// Sum up shares from all pools
		for _, pool := range resp.Work.PoolWork {
			if pool.Shares != nil {
				stats.AcceptedShares += pool.Shares.Accepted
				stats.RejectedShares += pool.Shares.Rejected
			}
		}
	}

	return stats
}

// ConvertHashboards converts GetHashboardsResponse to SimpleHashboard slice
func ConvertHashboards(resp *pb.GetHashboardsResponse) []SimpleHashboard {
	if resp == nil || resp.Boards == nil {
		return nil
	}

	var boards []SimpleHashboard

	for _, board := range resp.Boards {
		if board == nil {
			continue
		}

		hb := SimpleHashboard{
			Index: int(board.Index),
		}

		// Convert status
		if board.Status != nil {
			hb.Status = board.Status.State.String()
		}

		// Convert hashrate
		if board.HashrateAverage != nil && board.HashrateAverage.Last_15M != nil {
			hb.HashRate = float64(board.HashrateAverage.Last_15M.Value)
		}

		// Convert temperature
		if board.Temperature != nil {
			hb.Temperature = float64(board.Temperature.Value)
		}

		// Convert chip count
		if board.Chips != nil {
			hb.Chips = int(board.Chips.Detected)
		}

		// Convert voltage
		if board.Voltage != nil {
			hb.Voltage = float64(board.Voltage.Value)
		}

		// Convert frequency
		if board.Frequency != nil {
			hb.Frequency = float64(board.Frequency.Value)
		}

		boards = append(boards, hb)
	}

	return boards
}

// ConvertPoolGroups converts GetPoolGroupsResponse to SimplePoolGroup slice
func ConvertPoolGroups(resp *pb.GetPoolGroupsResponse) []SimplePoolGroup {
	if resp == nil || resp.Groups == nil {
		return nil
	}

	var groups []SimplePoolGroup

	for _, group := range resp.Groups {
		if group == nil {
			continue
		}

		sg := SimplePoolGroup{
			Name: group.Name,
		}

		// Convert pools
		if group.Pools != nil {
			for _, pool := range group.Pools {
				if pool == nil {
					continue
				}

				sp := SimplePool{
					URL:      pool.Url,
					User:     pool.User,
					Password: pool.Password,
					Enabled:  pool.Enabled,
				}

				// Get pool work statistics if available
				// Note: This would need to come from a different call or be matched by URL

				sg.Pools = append(sg.Pools, sp)
			}
		}

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

	// Convert cooling configuration
	if resp.Cooling != nil {
		cooling := make(map[string]interface{})

		// Check immersion mode
		cooling["immersion_mode"] = resp.Cooling.ImmersionMode

		// Get fan control mode
		if resp.Cooling.FanControl != nil {
			switch fc := resp.Cooling.FanControl.Control.(type) {
			case *pb.FanControl_FixedSpeed:
				cooling["fan_mode"] = "fixed"
				cooling["fan_speed_percent"] = fc.FixedSpeed.Value
			case *pb.FanControl_TargetTemperature:
				cooling["fan_mode"] = "auto"
				cooling["target_temperature"] = fc.TargetTemperature.Value
			}
		}

		// Get fan information
		if resp.Cooling.Fans != nil {
			var fans []map[string]interface{}
			for i, fan := range resp.Cooling.Fans {
				fanInfo := map[string]interface{}{
					"index": i,
				}
				if fan.Rpm != nil {
					fanInfo["rpm"] = fan.Rpm.Value
				}
				if fan.SpeedPercent != nil {
					fanInfo["speed_percent"] = fan.SpeedPercent.Value
				}
				fans = append(fans, fanInfo)
			}
			cooling["fans"] = fans
		}

		result["cooling"] = cooling
	}

	// Convert temperatures
	if resp.Temperature != nil {
		temps := make(map[string]interface{})

		// Board temperatures
		if resp.Temperature.Boards != nil {
			var boardTemps []float64
			for _, board := range resp.Temperature.Boards {
				if board != nil {
					boardTemps = append(boardTemps, float64(board.Value))
				}
			}
			temps["boards"] = boardTemps
		}

		result["temperature"] = temps
	}

	return result
}

// ConvertTunerState converts GetTunerStateResponse to performance information
func ConvertTunerState(resp *pb.GetTunerStateResponse) map[string]interface{} {
	if resp == nil {
		return nil
	}

	result := make(map[string]interface{})

	// Get tuner status
	if resp.Status != nil {
		result["status"] = resp.Status.State.String()
		result["running"] = resp.Status.Running
	}

	// Get current tuning mode
	if resp.Config != nil {
		config := make(map[string]interface{})

		if resp.Config.Mode != nil {
			switch mode := resp.Config.Mode.Mode.(type) {
			case *pb.TuningMode_PowerTarget:
				config["mode"] = "power_target"
				if mode.PowerTarget != nil {
					config["power_target_watts"] = mode.PowerTarget.Value
				}
			case *pb.TuningMode_HashrateTarget:
				config["mode"] = "hashrate_target"
				if mode.HashrateTarget != nil {
					config["hashrate_target_thps"] = mode.HashrateTarget.Value
				}
			case *pb.TuningMode_Manual:
				config["mode"] = "manual"
			}
		}

		result["config"] = config
	}

	// Get profiles
	if resp.TunedProfileUids != nil {
		result["tuned_profiles"] = resp.TunedProfileUids
	}

	return result
}

// ConvertLicenseState converts GetLicenseStateResponse to license information
func ConvertLicenseState(resp *pb.GetLicenseStateResponse) map[string]interface{} {
	if resp == nil {
		return nil
	}

	result := make(map[string]interface{})

	// Get license state
	if resp.State != pb.LicenseState_LICENSE_STATE_UNSPECIFIED {
		result["state"] = resp.State.String()
	}

	// Get last update time
	if resp.LastUpdateTimestamp != nil {
		result["last_update"] = time.Unix(resp.LastUpdateTimestamp.Seconds, int64(resp.LastUpdateTimestamp.Nanos))
	}

	// Get license info if available
	if resp.License != nil {
		license := make(map[string]interface{})
		license["id"] = resp.License.Id

		// Parse validity
		if resp.License.ValidUntil != nil {
			license["valid_until"] = time.Unix(resp.License.ValidUntil.Seconds, int64(resp.License.ValidUntil.Nanos))
		}

		result["license"] = license
	}

	return result
}