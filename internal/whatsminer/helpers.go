package whatsminer

import (
	"fmt"
	"time"
)

// MinerStats represents common miner statistics
type MinerStats struct {
	Hashrate   float64 // TH/s
	Power      float64 // Watts
	Temp       float64 // Celsius (average chip temp)
	TempBoards []float64 // Per-board temperatures
	Working    bool    // Is miner actively mining
	Model      string  // Miner model (e.g., "M30S++")
}

// GetMinerStats retrieves and parses common miner statistics
func (c *Client) GetMinerStats() (*MinerStats, error) {
	resp, err := c.GetMinerStatus("summary")
	if err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		// Try to get error message
		if msgStr, msgErr := resp.GetMsgString(); msgErr == nil {
			return nil, fmt.Errorf("API error code %d: %s", resp.Code, msgStr)
		}
		return nil, fmt.Errorf("API error code: %d", resp.Code)
	}

	msg, err := resp.GetMsg()
	if err != nil {
		return nil, fmt.Errorf("failed to parse response message: %w", err)
	}

	stats := &MinerStats{}

	// The response has a "summary" object inside msg
	var summary map[string]interface{}
	if summaryObj, ok := msg["summary"].(map[string]interface{}); ok {
		summary = summaryObj
	} else {
		// Fallback to top-level msg if no summary object
		summary = msg
	}

	// Extract hashrate (in TH/s)
	if hashrate, ok := summary["hash-realtime"].(float64); ok {
		stats.Hashrate = hashrate
	}

	// Extract power (in Watts)
	if power, ok := summary["power-realtime"].(float64); ok {
		stats.Power = power
	} else if power, ok := summary["power-realtime"].(int); ok {
		stats.Power = float64(power)
	}

	// Extract temperature (average chip temp)
	if temp, ok := summary["chip-temp-avg"].(float64); ok {
		stats.Temp = temp
	}

	// Extract per-board temperatures
	if boardTemps, ok := summary["board-temperature"].([]interface{}); ok {
		for _, temp := range boardTemps {
			if tempFloat, ok := temp.(float64); ok {
				stats.TempBoards = append(stats.TempBoards, tempFloat)
			}
		}
	}

	// Determine working status - if hashrate > 0, it's working
	stats.Working = stats.Hashrate > 0

	// Extract model from device info (need separate call)
	// For now, leave empty - will be populated from device info call
	stats.Model = ""

	return stats, nil
}

// IsWhatsMiner checks if a device at the given IP is a WhatsMiner
// Returns true if it successfully connects to the WhatsMiner API
func IsWhatsMiner(ip string, port int, timeout time.Duration) bool {
	client := NewClient(ip, port, "super", "super", timeout)
	if err := client.Connect(); err != nil {
		return false
	}
	defer client.Close()

	// Try to get device info - this is a reliable way to detect WhatsMiner
	resp, err := client.GetDeviceInfo()
	if err != nil {
		return false
	}

	// Check if response is valid WhatsMiner format
	return resp.Code == 0 || resp.Code < 0 // Valid response codes
}

// GetFirmwareVersion retrieves the firmware version string
func (c *Client) GetFirmwareVersion() (string, error) {
	resp, err := c.GetDeviceInfo()
	if err != nil {
		return "", err
	}

	if !resp.IsSuccess() {
		return "", fmt.Errorf("failed to get device info")
	}

	msg, err := resp.GetMsg()
	if err != nil {
		return "", err
	}

	// Try to get version from system info
	if system, ok := msg["system"].(map[string]interface{}); ok {
		if version, ok := system["software_version"].(string); ok {
			return version, nil
		}
	}

	return "Unknown", nil
}

// GetMinerModel retrieves the miner model string
func (c *Client) GetMinerModel() (string, error) {
	resp, err := c.GetDeviceInfo()
	if err != nil {
		return "", err
	}

	if !resp.IsSuccess() {
		return "", fmt.Errorf("failed to get device info")
	}

	msg, err := resp.GetMsg()
	if err != nil {
		return "", err
	}

	// Try to get model from miner info
	if miner, ok := msg["miner"].(map[string]interface{}); ok {
		if model, ok := miner["type"].(string); ok {
			return model, nil
		}
	}

	return "WhatsMiner", nil
}
