package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	braiinsClient "github.com/sinkers/miner-cli/internal/braiins/client"
	"github.com/sinkers/miner-cli/internal/client"
	"github.com/sinkers/miner-cli/internal/iprange"
	"github.com/sinkers/miner-cli/internal/output"
	"github.com/sinkers/miner-cli/internal/snmp"
	"github.com/sinkers/miner-cli/internal/whatsminer"
	"github.com/spf13/cobra"
)

var (
	ipRanges     []string
	port         int
	timeout      int
	workers      int
	outputFormat string
	verbose      bool

	poolID    int
	poolURL   string
	poolUser  string
	poolPass  string
	customCmd string

	// SNMP switch integration
	switchIP        string
	switchCommunity string

	// Braiins authentication for MAC address retrieval
	braiinsUsername string
	braiinsPassword string
)

const (
	Version          = "1.0.0"
	outputFormatJSON = "json"

	// Miner status constants
	statusRunning = "Running"
	statusPaused  = "Paused"
	statusStopped = "Stopped"
	statusOffline = "Offline"
	statusUnknown = "Unknown"
)

var rootCmd = &cobra.Command{
	Use:     "miner-cli",
	Short:   "CGMiner API CLI - Execute commands across multiple miners",
	Version: Version,
	Long: `A comprehensive CLI tool for interacting with CGMiner API endpoints.
Supports multiple IP formats including CIDR notation and ranges.

Examples:
  # Query a single miner
  miner-cli summary 192.168.1.100
  
  # Query multiple IP ranges
  miner-cli devs 192.168.1.0/24 10.0.0.1-10.0.0.50
  
  # Output as JSON
  miner-cli stats 192.168.1.0/28 -o json
  
  # Add a new pool to multiple miners
  miner-cli addpool 192.168.1.0/24 --url stratum+tcp://pool.example.com:3333 --user myworker --pass x
  
  # Legacy format with -i flag (still supported)
  miner-cli summary -i 192.168.1.100`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringSliceVarP(&ipRanges, "ips", "i", []string{}, "IP ranges (CIDR or range format, can be specified multiple times)")
	rootCmd.PersistentFlags().IntVarP(&port, "port", "p", 4028, "CGMiner API port")
	rootCmd.PersistentFlags().IntVarP(&timeout, "timeout", "t", 2, "Connection timeout in seconds")
	rootCmd.PersistentFlags().IntVarP(&workers, "workers", "w", 255, "Number of concurrent workers")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "color", "Output format (color, json, table)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	commands := client.GetAvailableCommands()
	for _, cmd := range commands {
		cmdCopy := cmd
		cobraCmd := &cobra.Command{
			Use:   cmdCopy + " [IP_RANGES...]",
			Short: client.GetCommandDescription(cmdCopy),
			Args:  cobra.ArbitraryArgs,
			PreRunE: func(c *cobra.Command, args []string) error {
				// Combine positional args with -i flag values
				if len(args) > 0 {
					ipRanges = append(ipRanges, args...)
				}
				if len(ipRanges) == 0 {
					return fmt.Errorf("no IP ranges specified")
				}
				return nil
			},
			RunE: func(c *cobra.Command, args []string) error {
				return executeCommand(cmdCopy)
			},
		}

		switch cmdCopy {
		case "switchpool", "enablepool", "disablepool", "removepool":
			cobraCmd.Flags().IntVar(&poolID, "pool", 0, "Pool ID")
			_ = cobraCmd.MarkFlagRequired("pool")
		case "addpool":
			cobraCmd.Flags().StringVar(&poolURL, "url", "", "Pool URL")
			cobraCmd.Flags().StringVar(&poolUser, "user", "", "Pool username")
			cobraCmd.Flags().StringVar(&poolPass, "pass", "", "Pool password")
			_ = cobraCmd.MarkFlagRequired("url")
			_ = cobraCmd.MarkFlagRequired("user")
			_ = cobraCmd.MarkFlagRequired("pass")
		case "custom":
			cobraCmd.Flags().StringVar(&customCmd, "cmd", "", "Custom command to execute")
			_ = cobraCmd.MarkFlagRequired("cmd")
		}

		rootCmd.AddCommand(cobraCmd)
	}

	rootCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all available commands",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Available CGMiner API Commands:")
			fmt.Println("================================")
			for _, c := range commands {
				fmt.Printf("%-15s - %s\n", c, client.GetCommandDescription(c))
			}
		},
	})

	// Define scan-specific flags
	scanCmd := &cobra.Command{
		Use:   "scan [IP_RANGES...]",
		Short: "Scan IP ranges to find active miners",
		Args:  cobra.ArbitraryArgs,
		PreRunE: func(c *cobra.Command, args []string) error {
			// Combine positional args with -i flag values
			if len(args) > 0 {
				ipRanges = append(ipRanges, args...)
			}
			if len(ipRanges) == 0 {
				return fmt.Errorf("no IP ranges specified")
			}
			return nil
		},
		RunE: scanMiners,
	}

	// Flag to optionally fetch chip temperatures via a second pass
	scanCmd.Flags().Bool("scan-temps", false, "Fetch chip temperature via device metrics (slower)")
	scanCmd.Flags().Bool("check-errors", false, "Check WhatsMiner devices for error codes")
	scanCmd.Flags().StringVar(&switchIP, "switch", "", "Switch IP address for SNMP port lookup (e.g., 10.110.101.6)")
	scanCmd.Flags().StringVar(&switchCommunity, "community", "public", "SNMP community string for switch")
	scanCmd.Flags().StringVar(&braiinsUsername, "braiins-user", "root", "Braiins OS username for MAC address retrieval")
	scanCmd.Flags().StringVar(&braiinsPassword, "braiins-pass", "root", "Braiins OS password for MAC address retrieval")
	rootCmd.AddCommand(scanCmd)
}

func executeCommand(command string) error {
	if len(ipRanges) == 0 {
		return fmt.Errorf("no IP ranges specified")
	}

	ipRange, err := iprange.ParseMultipleRanges(ipRanges)
	if err != nil {
		return fmt.Errorf("failed to parse IP ranges: %w", err)
	}

	ips := ipRange.GetIPs()
	if len(ips) == 0 {
		return fmt.Errorf("no valid IPs in specified ranges")
	}

	if outputFormat != outputFormatJSON {
		fmt.Printf("Executing '%s' on %d hosts...\n", command, len(ips))
	}

	params := make(map[string]interface{})

	switch command {
	case "switchpool", "enablepool", "disablepool", "removepool":
		params["pool"] = poolID
	case "addpool":
		params["url"] = poolURL
		params["user"] = poolUser
		params["pass"] = poolPass
	case "custom":
		params["cmd"] = customCmd
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second*time.Duration(len(ips)/workers+1))
	defer cancel()

	cgClient := client.NewClient(time.Duration(timeout)*time.Second, workers)
	results := cgClient.ExecuteCommand(ctx, ips, port, command, params)

	// Use summary formatter for summary command unless explicitly overridden
	formatToUse := outputFormat
	if command == "summary" && outputFormat == "color" {
		formatToUse = "summary"
	}

	formatter := output.GetFormatter(formatToUse, verbose)
	return formatter.Format(results)
}

func scanMiners(cmd *cobra.Command, args []string) error {
	if len(ipRanges) == 0 {
		return fmt.Errorf("no IP ranges specified")
	}

	ipRange, err := iprange.ParseMultipleRanges(ipRanges)
	if err != nil {
		return fmt.Errorf("failed to parse IP ranges: %w", err)
	}

	ips := ipRange.GetIPs()
	if len(ips) == 0 {
		return fmt.Errorf("no valid IPs in specified ranges")
	}

	if outputFormat != outputFormatJSON {
		fmt.Printf("Scanning %d hosts for active miners...\n", len(ips))
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second*time.Duration(len(ips)/workers+1))
	defer cancel()

	cgClient := client.NewClient(time.Duration(timeout)*time.Second, workers)
	results := cgClient.ExecuteCommand(ctx, ips, port, "summary", nil)

	// Detect and enrich firmware information
	enrichFirmwareInfo(results)

	// Try to get info from Braiins GraphQL for miners that didn't respond to CGMiner
	enrichBraiinsMiners(results)

	// Try to get info from WhatsMiner API for devices that didn't respond to CGMiner
	checkErrors, _ := cmd.Flags().GetBool("check-errors")
	enrichWhatsMiners(results, checkErrors)

	// Setup switch port mapping if requested
	portMap, arpCache := setupSwitchMapping()

	// Build list of active IPs and firmware map (include miners with recoverable errors)
	activeIPs, firmwareMap := extractActiveMiners(results)

	// Resolve MAC addresses if doing port lookups
	macAddresses := resolveMACAddresses(arpCache, activeIPs)

	// Enrich results with temperature, power, and port data (includes tuner status)
	enrichMinerData(ctx, cgClient, results, activeIPs, firmwareMap, macAddresses, portMap, arpCache)

	// Detect miner status (running, paused, stopped) - must be after enrichMinerData
	enrichMinerStatus(results, activeIPs)

	// Filter results to only show actual devices (not offline/non-existent IPs)
	// This removes IPs where no device exists at all
	filteredResults := make([]client.Result, 0, len(results))
	for _, r := range results {
		// Include devices that:
		// 1. Have no error (successful response), OR
		// 2. Have a "Disconnected" or "Code: 302" error (online but stopped/paused)
		// Exclude truly offline devices that never responded
		isOnline := r.Error == "" ||
			strings.Contains(r.Error, "Disconnected") ||
			strings.Contains(r.Error, "Code: 302")

		if isOnline {
			filteredResults = append(filteredResults, r)
		}
	}

	if outputFormat == outputFormatJSON {
		formatter := output.GetFormatter(outputFormat, verbose)
		return formatter.Format(filteredResults)
	}

	// Use the new scan formatter for better output
	formatter := output.GetFormatter("scan", verbose)
	return formatter.Format(filteredResults)
}

// enrichFirmwareInfo detects and adds firmware information to results
func enrichFirmwareInfo(results []client.Result) {
	for i := range results {
		if results[i].Error == "" && results[i].Response != nil {
			firmware := getBraaiinsFirmwareInfo(results[i].IP)
			if firmware == "" {
				firmware = "Stock"
			}
			addFieldToResponse(&results[i], "firmware", firmware)
		}
	}
}

// enrichBraiinsMiners tries to get info from Braiins GraphQL for miners that didn't respond to CGMiner
func enrichBraiinsMiners(results []client.Result) {
	for i := range results {
		// Only try Braiins API for miners that had errors
		if results[i].Error != "" {
			// Check if it's a "Disconnected" or similar error (miner is online but not mining)
			if strings.Contains(results[i].Error, "Disconnected") ||
			   strings.Contains(results[i].Error, "Code: 302") {
				// Try to get info from Braiins GraphQL
				if firmware := getBraaiinsFirmwareInfo(results[i].IP); firmware != "" {
					// Miner is a Braiins OS miner, create a minimal response
					results[i].Response = map[string]interface{}{
						"firmware": firmware,
						"MHS 5s":   0,
						"MHS av":   0,
						"Accepted": 0,
						"Rejected": 0,
						"Hardware Errors": 0,
						"Elapsed": 0,
					}
					// Keep the error but it will be treated as "disconnected"
				}
			}
		}
	}
}

// enrichWhatsMiners tries to get info from WhatsMiner API for miners that didn't respond to CGMiner
func enrichWhatsMiners(results []client.Result, checkErrors bool) {
	// Find IPs that need WhatsMiner detection
	var ipsToCheck []int
	for i := range results {
		// Skip if already has a successful response or is a known Braiins miner
		if results[i].Error == "" {
			continue
		}
		if results[i].Response != nil {
			if fw, ok := results[i].Response.(map[string]interface{})["firmware"]; ok {
				if fwStr, ok := fw.(string); ok && strings.Contains(fwStr, "Braiins") {
					continue
				}
			}
		}
		ipsToCheck = append(ipsToCheck, i)
	}

	if len(ipsToCheck) == 0 {
		return
	}

	// Use concurrency for WhatsMiner detection
	type wmResult struct {
		index    int
		response map[string]interface{}
		error    string
	}

	resultsChan := make(chan wmResult, len(ipsToCheck))
	maxWorkers := 50 // Limit concurrent connections
	sem := make(chan struct{}, maxWorkers)

	for _, idx := range ipsToCheck {
		go func(i int) {
			sem <- struct{}{} // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			ip := results[i].IP

			// Try to detect WhatsMiner on port 4433
			wmClient := whatsminer.NewClient(ip, 4433, "super", "super", 2*time.Second)
			if err := wmClient.Connect(); err != nil {
				resultsChan <- wmResult{index: i}
				return
			}

			// Get device info to confirm it's a WhatsMiner and extract model
			deviceInfo, err := wmClient.GetDeviceInfo()
			if err != nil || !deviceInfo.IsSuccess() {
				wmClient.Close()
				resultsChan <- wmResult{index: i}
				return
			}

			// Extract model from device info
			model := "WhatsMiner"
			if msg, err := deviceInfo.GetMsg(); err == nil {
				if miner, ok := msg["miner"].(map[string]interface{}); ok {
					if minerType, ok := miner["type"].(string); ok {
						model = fmt.Sprintf("WhatsMiner %s", minerType)
					}
				}
			}

			// Extract error codes if requested
			var errorCodes []whatsminer.ErrorCodeInfo
			if checkErrors {
				errorCodes = whatsminer.GetErrorCodesFromDeviceInfo(deviceInfo)
			}

			// Get miner stats
			stats, err := wmClient.GetMinerStats()
			wmClient.Close()

			if err != nil {
				// Create minimal response even if stats fail
				resultsChan <- wmResult{
					index: i,
					response: map[string]interface{}{
						"firmware": model,
						"MHS 5s":   0,
						"MHS av":   0,
					},
					error: "",
				}
				return
			}

			// Convert TH/s to MH/s for consistency with CGMiner API
			hashrateMHS := stats.Hashrate * 1000000.0

			// Create response in CGMiner format for compatibility
			resp := map[string]interface{}{
				"firmware":        model,
				"MHS 5s":          hashrateMHS,
				"MHS av":          hashrateMHS,
				"power_w":         stats.Power,
				"chip_temp_c":     stats.Temp,
				"Accepted":        0, // Not available from WhatsMiner summary
				"Rejected":        0,
				"Hardware Errors": 0,
				"Elapsed":         0,
				"working":         stats.Working,
			}

			// Add error codes if they exist
			if len(errorCodes) > 0 {
				resp["error_codes"] = errorCodes
			}

			errMsg := ""
			if !stats.Working {
				errMsg = "Disconnected" // Indicate miner is not mining
			}

			resultsChan <- wmResult{
				index:    i,
				response: resp,
				error:    errMsg,
			}
		}(idx)
	}

	// Collect results
	for range ipsToCheck {
		result := <-resultsChan
		if result.response != nil {
			results[result.index].Response = result.response
			results[result.index].Error = result.error
		}
	}
}

// enrichMinerStatus detects and adds miner status (running, paused, stopped) to results
func enrichMinerStatus(results []client.Result, activeIPs []string) {
	if len(activeIPs) == 0 {
		return
	}

	// Add status to results based on hashrate, tuner status, error messages, and pool status
	for i := range results {
		if results[i].Response != nil {
			// Determine status from response data
			status := determineMinerStatusFromResponse(results[i].Response)

			// Override status if we have a "Disconnected" error
			if results[i].Error != "" &&
			   (strings.Contains(results[i].Error, "Disconnected") ||
			    strings.Contains(results[i].Error, "Code: 302")) {
				status = statusStopped
			}

			addFieldToResponse(&results[i], "miner_status", status)
		} else if results[i].Error != "" {
			// Miner is truly offline - initialize response if nil
			if results[i].Response == nil {
				results[i].Response = make(map[string]interface{})
			}
			addFieldToResponse(&results[i], "miner_status", statusOffline)
		}
	}
}

// determineMinerStatusFromResponse determines status from response data
func determineMinerStatusFromResponse(response interface{}) string {
	respMap, ok := response.(map[string]interface{})
	if !ok {
		// Try to convert to map
		b, err := json.Marshal(response)
		if err != nil {
			return statusUnknown
		}
		if err := json.Unmarshal(b, &respMap); err != nil {
			return statusUnknown
		}
	}

	// Check hashrate first
	hashrate := 0.0
	if val, ok := respMap["MHS 5s"]; ok {
		if hr, ok := toFloat64Loose(val); ok {
			hashrate = hr
		}
	} else if val, ok := respMap["MHS av"]; ok {
		if hr, ok := toFloat64Loose(val); ok {
			hashrate = hr
		}
	} else if val, ok := respMap["GHS 5s"]; ok {
		if hr, ok := toFloat64Loose(val); ok {
			hashrate = hr * 1000.0
		}
	} else if val, ok := respMap["GHS av"]; ok {
		if hr, ok := toFloat64Loose(val); ok {
			hashrate = hr * 1000.0
		}
	}

	// Check tuner status for Braiins miners
	tunerStatus := ""
	if val, ok := respMap["tuner_status"]; ok {
		tunerStatus = strings.ToUpper(fmt.Sprintf("%v", val))
	}

	// Determine status based on hashrate and tuner status
	if hashrate > 100 {
		// Miner is actively hashing
		return statusRunning
	} else if hashrate == 0 && (tunerStatus == "DISABLED" || tunerStatus == "PAUSED") {
		// Braiins miner with disabled/paused tuner (manually paused)
		return statusPaused
	} else if hashrate < 100 && hashrate > 0 {
		// Very low hashrate, likely starting up or having issues
		return statusStopped
	} else if hashrate == 0 {
		// No hashrate but responding (pools might be dead or disabled)
		return statusStopped
	}

	return statusRunning
}

// setupSwitchMapping initializes switch port mapping if requested
func setupSwitchMapping() (*snmp.PortMACMap, *snmp.ARPCache) {
	if switchIP == "" {
		return nil, nil
	}

	if outputFormat != outputFormatJSON {
		fmt.Printf("Querying switch %s for port mappings...\n", switchIP)
	}

	switchClient := snmp.NewSwitchClient(switchIP, switchCommunity)
	portMap, err := switchClient.GetMACAddressTable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to query switch: %v\n", err)
		return nil, nil
	}

	if outputFormat != outputFormatJSON {
		fmt.Printf("Retrieved %d MAC-to-port mappings from switch\n", portMap.Size())
	}

	return portMap, snmp.NewARPCache()
}

// extractActiveMiners builds list of active IPs and their firmware info
func extractActiveMiners(results []client.Result) ([]string, map[string]string) {
	activeIPs := make([]string, 0, len(results))
	firmwareMap := make(map[string]string)

	for _, r := range results {
		// Include miners with no error OR miners with "Disconnected" error (they're online but not mining)
		isActive := r.Error == "" ||
			strings.Contains(r.Error, "Disconnected") ||
			strings.Contains(r.Error, "Code: 302")

		if isActive && r.Response != nil {
			activeIPs = append(activeIPs, r.IP)
			if respMap, ok := r.Response.(map[string]interface{}); ok {
				if fw, ok := respMap["firmware"].(string); ok {
					firmwareMap[r.IP] = fw
				}
			}
		}
	}

	return activeIPs, firmwareMap
}

// resolveMACAddresses retrieves MAC addresses for active miners
func resolveMACAddresses(arpCache *snmp.ARPCache, activeIPs []string) map[string]string {
	macAddresses := make(map[string]string)
	if arpCache == nil || len(activeIPs) == 0 {
		return macAddresses
	}

	if outputFormat != outputFormatJSON {
		fmt.Printf("Resolving MAC addresses for %d active miners...\n", len(activeIPs))
	}

	for _, ip := range activeIPs {
		go func(addr string) {
			_ = snmp.Ping(addr)
		}(ip)
	}

	time.Sleep(500 * time.Millisecond)

	macAddresses = arpCache.BulkGetMACs(activeIPs)
	if outputFormat != outputFormatJSON {
		fmt.Printf("Resolved %d MAC addresses via ARP\n", len(macAddresses))
	}

	return macAddresses
}

// enrichMinerData adds temperature, power, and port information to results
func enrichMinerData(ctx context.Context, cgClient *client.Client, results []client.Result,
	activeIPs []string, firmwareMap map[string]string, macAddresses map[string]string,
	portMap *snmp.PortMACMap, arpCache *snmp.ARPCache) {
	if len(activeIPs) == 0 {
		return
	}

	temps := make(map[string]float64)
	braiinsStats := fetchBraiinsData(activeIPs, firmwareMap, temps, macAddresses, arpCache)
	fetchNonBraiinsTemps(ctx, cgClient, activeIPs, firmwareMap, temps)
	mergeDataIntoResults(results, temps, braiinsStats, macAddresses, portMap)
}

// fetchBraiinsData gets temperature and stats from Braiins miners
func fetchBraiinsData(activeIPs []string, firmwareMap map[string]string, temps map[string]float64,
	macAddresses map[string]string, arpCache *snmp.ARPCache) map[string]BraiinsStats {
	braiinsStats := make(map[string]BraiinsStats)
	for _, ip := range activeIPs {
		fw, ok := firmwareMap[ip]
		if !ok || !strings.Contains(strings.ToLower(fw), "braiins") {
			continue
		}

		if stats, ok := getBraiinsStats(ip); ok {
			braiinsStats[ip] = stats
			if stats.ChipTemp > 0 {
				temps[ip] = stats.ChipTemp
			}
		}

		if arpCache != nil {
			if _, hasMac := macAddresses[ip]; !hasMac {
				if mac, ok := getBraaiinsMACAddress(ip, braiinsUsername, braiinsPassword); ok && mac != "" {
					macAddresses[ip] = mac
				}
			}
		}
	}

	return braiinsStats
}

// fetchNonBraiinsTemps gets temperatures from non-Braiins miners
func fetchNonBraiinsTemps(ctx context.Context, cgClient *client.Client, activeIPs []string,
	firmwareMap map[string]string, temps map[string]float64) {
	otherIPs := make([]string, 0)
	for _, ip := range activeIPs {
		if fw, ok := firmwareMap[ip]; !ok || !strings.Contains(strings.ToLower(fw), "braiins") {
			otherIPs = append(otherIPs, ip)
		}
	}

	if len(otherIPs) == 0 {
		return
	}

	devResults := cgClient.ExecuteCommand(ctx, otherIPs, port, "devs", nil)
	for _, dr := range devResults {
		if dr.Error == "" && dr.Response != nil {
			if t, ok := extractChipTempFromDevs(dr.Response); ok {
				temps[dr.IP] = t
			}
		}
	}

	missing := make([]string, 0, len(otherIPs))
	for _, ip := range otherIPs {
		if _, ok := temps[ip]; !ok {
			missing = append(missing, ip)
		}
	}

	if len(missing) > 0 {
		statsResults := cgClient.ExecuteCommand(ctx, missing, port, "stats", nil)
		for _, sr := range statsResults {
			if sr.Error == "" && sr.Response != nil {
				if t, ok := extractChipTempFromStats(sr.Response); ok {
					temps[sr.IP] = t
				}
			}
		}
	}
}

// mergeDataIntoResults adds all enrichment data to results
func mergeDataIntoResults(results []client.Result, temps map[string]float64,
	braiinsStats map[string]BraiinsStats, macAddresses map[string]string, portMap *snmp.PortMACMap) {
	for i := range results {
		if t, ok := temps[results[i].IP]; ok {
			addFieldToResponse(&results[i], "chip_temp_c", t)
		}

		if stats, ok := braiinsStats[results[i].IP]; ok {
			if stats.PowerW > 0 {
				addFieldToResponse(&results[i], "power_w", stats.PowerW)
			}
			addFieldToResponse(&results[i], "tuning", stats.Tuning)
			addFieldToResponse(&results[i], "tuner_status", stats.TunerStatus)
		}

		if portMap != nil && results[i].Error == "" {
			if mac, ok := macAddresses[results[i].IP]; ok && mac != "" {
				if port, found := portMap.Get(mac); found {
					addFieldToResponse(&results[i], "switch_port", port)
				}
			}
		}
	}
}

// addFieldToResponse adds a field to a result's response, handling type conversion
func addFieldToResponse(result *client.Result, key string, value interface{}) {
	// Handle nil response by creating a new map
	if result.Response == nil {
		result.Response = make(map[string]interface{})
	}

	switch resp := result.Response.(type) {
	case map[string]interface{}:
		resp[key] = value
		result.Response = resp
	default:
		b, err := json.Marshal(result.Response)
		if err == nil {
			var m map[string]interface{}
			if json.Unmarshal(b, &m) == nil {
				m[key] = value
				result.Response = m
			}
		}
	}
}

// extractChipTempFromDevs attempts to extract a representative chip temperature
// from a CGMiner "devs" response. It returns the maximum temperature observed
// across devices when available.
func extractChipTempFromDevs(resp interface{}) (float64, bool) {
	// Normalize via JSON into a slice of generic maps
	b, err := json.Marshal(resp)
	if err != nil {
		return 0, false
	}
	var arr []map[string]interface{}
	if err := json.Unmarshal(b, &arr); err != nil {
		// Some implementations may return a single object
		var m map[string]interface{}
		if err := json.Unmarshal(b, &m); err != nil {
			return 0, false
		}
		arr = []map[string]interface{}{m}
	}

	maxT := -1.0
	for _, dev := range arr {
		for k, v := range dev {
			lk := strings.ToLower(k)
			if strings.Contains(lk, "temp") {
				// Parse number-like value
				if t, ok := toFloat64Loose(v); ok {
					// Filter out invalid temps
					if t > 0 && t <= 200 && t > maxT {
						maxT = t
					}
				}
			}
		}
	}
	if maxT >= 0 {
		return maxT, true
	}
	return 0, false
}

// toFloat64Loose converts common numeric encodings to float64
func toFloat64Loose(v interface{}) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case json.Number:
		f, err := t.Float64()
		return f, err == nil
	case string:
		var f float64
		if _, err := fmt.Sscanf(t, "%f", &f); err == nil {
			return f, true
		}
		return 0, false
	default:
		return 0, false
	}
}

// extractChipTempFromStats attempts to extract a representative chip temperature
// from a CGMiner "stats" response. It scans all temp-like keys and returns
// the maximum plausible value.
func extractChipTempFromStats(resp interface{}) (float64, bool) {
	b, err := json.Marshal(resp)
	if err != nil {
		return 0, false
	}
	var arr []map[string]interface{}
	if err := json.Unmarshal(b, &arr); err != nil {
		// Some implementations may return a single object
		var m map[string]interface{}
		if err := json.Unmarshal(b, &m); err != nil {
			return 0, false
		}
		arr = []map[string]interface{}{m}
	}
	maxT := -1.0
	var visit func(map[string]interface{})
	visit = func(m map[string]interface{}) {
		for k, v := range m {
			lk := strings.ToLower(k)
			if strings.Contains(lk, "temp") {
				if t, ok := toFloat64Loose(v); ok {
					if t > 0 && t <= 200 && t > maxT {
						maxT = t
					}
				}
			}
			// If nested map, scan recursively
			if nested, ok := v.(map[string]interface{}); ok {
				visit(nested)
			}
		}
	}
	for _, m := range arr {
		visit(m)
	}
	if maxT >= 0 {
		return maxT, true
	}
	return 0, false
}

// getBraaiinsFirmwareInfo queries the Braiins OS GraphQL API to detect firmware version
func getBraaiinsFirmwareInfo(ip string) string {
	client := &http.Client{Timeout: 2 * time.Second}

	query := `{"query":"{ bos { info { mode version { full isPlus } } } }"}`

	resp, err := client.Post(
		fmt.Sprintf("http://%s/graphql", ip),
		"application/json",
		strings.NewReader(query),
	)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Bos struct {
				Info struct {
					Mode    string `json:"mode"`
					Version struct {
						Full   string `json:"full"`
						IsPlus bool   `json:"isPlus"`
					} `json:"version"`
				} `json:"info"`
			} `json:"bos"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ""
	}

	if result.Data.Bos.Info.Version.Full != "" {
		version := result.Data.Bos.Info.Version.Full
		// Simplify version string - extract just the version number
		// Format: "2025-02-21-0-e05df053-25.01-plus" -> "Braiins OS+ 25.01"
		parts := strings.Split(version, "-")
		if len(parts) > 0 {
			// Find the version number (e.g., "25.01")
			for _, part := range parts {
				if strings.Contains(part, ".") && len(part) <= 6 {
					suffix := ""
					if result.Data.Bos.Info.Version.IsPlus {
						suffix = "+"
					}
					return fmt.Sprintf("Braiins OS%s %s", suffix, part)
				}
			}
		}
		// Fallback to showing if it's Plus or not
		if result.Data.Bos.Info.Version.IsPlus {
			return "Braiins OS+"
		}
		return "Braiins OS"
	}

	return ""
}

// BraiinsStats holds mining statistics from Braiins OS
type BraiinsStats struct {
	ChipTemp    float64
	PowerW      int
	Tuning      bool
	TunerStatus string
}

// getBraiinsStats queries the Braiins OS GraphQL API for chip temperature, power, and tuning status
func getBraiinsStats(ip string) (BraiinsStats, bool) {
	client := &http.Client{Timeout: 2 * time.Second}

	query := `{"query":"{ bosminer { info { workSolver { temperatures { name degreesC } power { approxConsumptionW } tuner { status } } } } }"}`

	resp, err := client.Post(
		fmt.Sprintf("http://%s/graphql", ip),
		"application/json",
		strings.NewReader(query),
	)
	if err != nil {
		return BraiinsStats{}, false
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Bosminer struct {
				Info struct {
					WorkSolver struct {
						Temperatures []struct {
							Name     string  `json:"name"`
							DegreesC float64 `json:"degreesC"`
						} `json:"temperatures"`
						Power struct {
							ApproxConsumptionW int `json:"approxConsumptionW"`
						} `json:"power"`
						Tuner struct {
							Status string `json:"status"`
						} `json:"tuner"`
					} `json:"workSolver"`
				} `json:"info"`
			} `json:"bosminer"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return BraiinsStats{}, false
	}

	stats := BraiinsStats{}

	// Find the Chip temperature (handle both "chip" and "MAX_CHIP")
	maxTemp := 0.0
	for _, temp := range result.Data.Bosminer.Info.WorkSolver.Temperatures {
		lowerName := strings.ToLower(temp.Name)
		if (lowerName == "chip" || lowerName == "max_chip") && temp.DegreesC > maxTemp {
			maxTemp = temp.DegreesC
		}
	}
	stats.ChipTemp = maxTemp

	// Get power consumption
	stats.PowerW = result.Data.Bosminer.Info.WorkSolver.Power.ApproxConsumptionW

	// Check tuning status
	status := strings.ToUpper(result.Data.Bosminer.Info.WorkSolver.Tuner.Status)
	stats.TunerStatus = status
	stats.Tuning = (status == "TUNING" || status == "TESTING")

	return stats, true
}

// getBraaiinsMACAddress retrieves MAC address from Braiins miner via gRPC
func getBraaiinsMACAddress(ip, username, password string) (string, bool) {
	client, err := braiinsClient.NewSimpleClient(braiinsClient.SimpleClientOptions{
		Host:     ip,
		Port:     50051,
		Username: username,
		Password: password,
		Timeout:  2 * time.Second,
		UseTLS:   false,
	})
	if err != nil {
		return "", false
	}
	defer client.Close()

	networkInfo, err := client.GetNetworkInfo()
	if err != nil {
		return "", false
	}

	return networkInfo.GetMacAddress(), networkInfo.GetMacAddress() != ""
}
