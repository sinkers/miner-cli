package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sinkers/miner-cli/internal/client"
	"github.com/sinkers/miner-cli/internal/iprange"
	"github.com/sinkers/miner-cli/internal/output"
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
)

const Version = "1.0.0"

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

	if outputFormat != "json" {
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

	if outputFormat != "json" {
		fmt.Printf("Scanning %d hosts for active miners...\n", len(ips))
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second*time.Duration(len(ips)/workers+1))
	defer cancel()

	cgClient := client.NewClient(time.Duration(timeout)*time.Second, workers)
	// Use summary command to get more detailed info including hashrate
	results := cgClient.ExecuteCommand(ctx, ips, port, "summary", nil)

	// Detect firmware by querying Braiins API for active miners
	for i := range results {
		if results[i].Error == "" && results[i].Response != nil {
			// Try to detect Braiins OS
			if firmware := getBraaiinsFirmwareInfo(results[i].IP); firmware != "" {
				// Add firmware info to the response map
				switch resp := results[i].Response.(type) {
				case map[string]interface{}:
					resp["firmware"] = firmware
				default:
					// Convert to a generic map if needed
					b, err := json.Marshal(results[i].Response)
					if err == nil {
						var m map[string]interface{}
						if json.Unmarshal(b, &m) == nil {
							m["firmware"] = firmware
							results[i].Response = m
						}
					}
				}
			} else {
				// Default to "Stock" if not Braiins
				switch resp := results[i].Response.(type) {
				case map[string]interface{}:
					resp["firmware"] = "Stock"
				default:
					b, err := json.Marshal(results[i].Response)
					if err == nil {
						var m map[string]interface{}
						if json.Unmarshal(b, &m) == nil {
							m["firmware"] = "Stock"
							results[i].Response = m
						}
					}
				}
			}
		}
	}

	// Enrich with chip temperature (always enabled now)
	// Build list of active IPs with their firmware info
	activeIPs := make([]string, 0, len(results))
	firmwareMap := make(map[string]string)
	for _, r := range results {
		if r.Error == "" {
			activeIPs = append(activeIPs, r.IP)
			// Extract firmware info
			if r.Response != nil {
				if respMap, ok := r.Response.(map[string]interface{}); ok {
					if fw, ok := respMap["firmware"].(string); ok {
						firmwareMap[r.IP] = fw
					}
				}
			}
		}
	}
	if len(activeIPs) > 0 {
		temps := make(map[string]float64)

		// Query Braiins API for miners running Braiins OS
		braiiinsIPs := make([]string, 0)
		otherIPs := make([]string, 0)
		for _, ip := range activeIPs {
			if fw, ok := firmwareMap[ip]; ok && strings.Contains(strings.ToLower(fw), "braiins") {
				braiiinsIPs = append(braiiinsIPs, ip)
			} else {
				otherIPs = append(otherIPs, ip)
			}
		}

		// Fetch stats from Braiins GraphQL API
		braiinsStats := make(map[string]BraiinsStats)
		if len(braiiinsIPs) > 0 {
			for _, ip := range braiiinsIPs {
				if stats, ok := getBraiinsStats(ip); ok {
					braiinsStats[ip] = stats
					if stats.ChipTemp > 0 {
						temps[ip] = stats.ChipTemp
					}
				}
			}
		}

		// For non-Braiins miners, try devs/stats
		if len(otherIPs) > 0 {
			// First attempt via `devs`
			devResults := cgClient.ExecuteCommand(ctx, otherIPs, port, "devs", nil)
			for _, dr := range devResults {
				if dr.Error != "" || dr.Response == nil {
					continue
				}
				if t, ok := extractChipTempFromDevs(dr.Response); ok {
					temps[dr.IP] = t
				}
			}

			// Fallback to `stats` for IPs with no temp yet
			missing := make([]string, 0, len(otherIPs))
			for _, ip := range otherIPs {
				if _, ok := temps[ip]; !ok {
					missing = append(missing, ip)
				}
			}
			if len(missing) > 0 {
				statsResults := cgClient.ExecuteCommand(ctx, missing, port, "stats", nil)
				for _, sr := range statsResults {
					if sr.Error != "" || sr.Response == nil {
						continue
					}
					if t, ok := extractChipTempFromStats(sr.Response); ok {
						temps[sr.IP] = t
					}
				}
			}
		}

		// Merge into main results by adding chip_temp_c, power, and tuning data into the Response map
		for i := range results {
			// Add temperature data
			if t, ok := temps[results[i].IP]; ok {
				switch resp := results[i].Response.(type) {
				case map[string]interface{}:
					resp["chip_temp_c"] = t
					results[i].Response = resp
				default:
					b, err := json.Marshal(results[i].Response)
					if err == nil {
						var m map[string]interface{}
						if json.Unmarshal(b, &m) == nil {
							m["chip_temp_c"] = t
							results[i].Response = m
						}
					}
				}
			}

			// Add Braiins-specific stats (power and tuning)
			if stats, ok := braiinsStats[results[i].IP]; ok {
				switch resp := results[i].Response.(type) {
				case map[string]interface{}:
					if stats.PowerW > 0 {
						resp["power_w"] = stats.PowerW
					}
					resp["tuning"] = stats.Tuning
					resp["tuner_status"] = stats.TunerStatus
					results[i].Response = resp
				default:
					b, err := json.Marshal(results[i].Response)
					if err == nil {
						var m map[string]interface{}
						if json.Unmarshal(b, &m) == nil {
							if stats.PowerW > 0 {
								m["power_w"] = stats.PowerW
							}
							m["tuning"] = stats.Tuning
							m["tuner_status"] = stats.TunerStatus
							results[i].Response = m
						}
					}
				}
			}
		}
	}

	if outputFormat == "json" {
		formatter := output.GetFormatter(outputFormat, verbose)
		return formatter.Format(results)
	}

	// Use the new scan formatter for better output
	formatter := output.GetFormatter("scan", verbose)
	return formatter.Format(results)
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
