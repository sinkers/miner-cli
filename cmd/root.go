package cmd

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "strconv"
    "strings"
    "time"

    "github.com/spf13/cobra"
    "github.com/sinkers/miner-cli/internal/client"
    "github.com/sinkers/miner-cli/internal/iprange"
    "github.com/sinkers/miner-cli/internal/output"
)

var (
	ipRanges     []string
	port         int
	timeout      int
	workers      int
	outputFormat string
	verbose      bool
	version      bool

	poolID     int
	poolURL    string
	poolUser   string
	poolPass   string
	deviceName string
	zeroWhich  string
	zeroAll    bool
	customCmd  string
	customArgs string
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
			cobraCmd.MarkFlagRequired("pool")
		case "addpool":
			cobraCmd.Flags().StringVar(&poolURL, "url", "", "Pool URL")
			cobraCmd.Flags().StringVar(&poolUser, "user", "", "Pool username")
			cobraCmd.Flags().StringVar(&poolPass, "pass", "", "Pool password")
			cobraCmd.MarkFlagRequired("url")
			cobraCmd.MarkFlagRequired("user")
			cobraCmd.MarkFlagRequired("pass")
		case "custom":
			cobraCmd.Flags().StringVar(&customCmd, "cmd", "", "Custom command to execute")
			cobraCmd.MarkFlagRequired("cmd")
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

    // Optionally enrich with chip temperature by querying device metrics
    fetchTemps, _ := cmd.Flags().GetBool("scan-temps")
    if fetchTemps {
        // Build list of active IPs
        activeIPs := make([]string, 0, len(results))
        for _, r := range results {
            if r.Error == "" {
                activeIPs = append(activeIPs, r.IP)
            }
        }
        if len(activeIPs) > 0 {
            temps := make(map[string]float64)

            // First attempt via `devs`
            devResults := cgClient.ExecuteCommand(ctx, activeIPs, port, "devs", nil)
            for _, dr := range devResults {
                if dr.Error != "" || dr.Response == nil {
                    continue
                }
                if t, ok := extractChipTempFromDevs(dr.Response); ok {
                    temps[dr.IP] = t
                }
            }

            // Fallback to `stats` for IPs with no temp yet
            missing := make([]string, 0, len(activeIPs))
            for _, ip := range activeIPs {
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

            // Merge into main results by adding chip_temp_c into the Response map
            for i := range results {
                if t, ok := temps[results[i].IP]; ok {
                    // Ensure Response is a map[string]interface{}
                    switch resp := results[i].Response.(type) {
                    case map[string]interface{}:
                        resp["chip_temp_c"] = t
                        results[i].Response = resp
                    default:
                        // Convert to a generic map if needed
                        // Best-effort: try JSON round-trip
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

func parseIntParam(param string) (int, error) {
	return strconv.Atoi(param)
}
