package snmp

import (
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

// ARPCache provides MAC address lookups via ARP
type ARPCache struct {
	mu    sync.RWMutex
	cache map[string]string // IP -> MAC
}

// NewARPCache creates a new ARP cache
func NewARPCache() *ARPCache {
	return &ARPCache{
		cache: make(map[string]string),
	}
}

// GetMACForIP retrieves the MAC address for an IP via ARP
func (ac *ARPCache) GetMACForIP(ip string) (string, error) {
	// Check cache first
	ac.mu.RLock()
	if mac, ok := ac.cache[ip]; ok {
		ac.mu.RUnlock()
		return mac, nil
	}
	ac.mu.RUnlock()

	// Query ARP table
	mac, err := queryARP(ip)
	if err != nil {
		return "", err
	}

	// Store in cache
	ac.mu.Lock()
	ac.cache[ip] = mac
	ac.mu.Unlock()

	return mac, nil
}

// BulkGetMACs retrieves MAC addresses for multiple IPs concurrently
func (ac *ARPCache) BulkGetMACs(ips []string) map[string]string {
	results := make(map[string]string)
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}

	// Process in batches to avoid overwhelming the system
	batchSize := 50
	for i := 0; i < len(ips); i += batchSize {
		end := i + batchSize
		if end > len(ips) {
			end = len(ips)
		}

		batch := ips[i:end]
		for _, ip := range batch {
			wg.Add(1)
			go func(ip string) {
				defer wg.Done()
				if mac, err := ac.GetMACForIP(ip); err == nil && mac != "" {
					mu.Lock()
					results[ip] = mac
					mu.Unlock()
				}
			}(ip)
		}
		wg.Wait()
	}

	return results
}

// queryARP queries the system ARP table for a specific IP
func queryARP(ip string) (string, error) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin", "linux":
		cmd = exec.Command("arp", "-n", ip)
	case "windows":
		cmd = exec.Command("arp", "-a", ip)
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("arp command failed: %w", err)
	}

	return parseARPOutput(string(output), ip)
}

// parseARPOutput parses ARP command output to extract MAC address
func parseARPOutput(output, ip string) (string, error) {
	lines := strings.Split(output, "\n")

	// MAC address pattern: matches various formats
	// Examples: 00:1a:2b:3c:4d:5e, 00-1a-2b-3c-4d-5e, 001a.2b3c.4d5e
	macPattern := regexp.MustCompile(`(?i)([0-9a-f]{1,2}[:\-\.]){5}[0-9a-f]{1,2}`)

	for _, line := range lines {
		// Skip header lines and incomplete entries
		if !strings.Contains(line, ip) {
			continue
		}

		// Look for MAC address pattern
		matches := macPattern.FindString(line)
		if matches != "" {
			// Check for incomplete entries (e.g., "(incomplete)")
			if strings.Contains(strings.ToLower(line), "incomplete") {
				continue
			}
			return normalizeMACAddress(matches), nil
		}
	}

	return "", fmt.Errorf("MAC address not found for IP %s", ip)
}

// Ping sends an ICMP ping to populate the ARP cache
// This is useful before querying ARP for an IP
func Ping(ip string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin", "linux":
		cmd = exec.Command("ping", "-c", "1", "-W", "1", ip)
	case "windows":
		cmd = exec.Command("ping", "-n", "1", "-w", "1000", ip)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	// We don't care about the output, just whether the ping succeeded
	_ = cmd.Run()
	return nil
}

// PingAndGetMAC pings an IP and then retrieves its MAC address
func PingAndGetMAC(ip string) (string, error) {
	// Ping to populate ARP cache (ignore errors as the ping might fail but still populate ARP)
	_ = Ping(ip)

	// Query ARP
	return queryARP(ip)
}
