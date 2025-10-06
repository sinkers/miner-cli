package output

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/sinkers/miner-cli/internal/client"
)

type Formatter interface {
	Format(results []client.Result) error
}

func GetFormatter(format string, verbose bool) Formatter {
	switch strings.ToLower(format) {
	case "json":
		return &JSONFormatter{Pretty: verbose}
	case "table":
		return &TableFormatter{Verbose: verbose}
	case "summary":
		return &SummaryTableFormatter{}
	case "scan":
		return &ScanFormatter{Verbose: verbose}
	default:
		return &ColorFormatter{Verbose: verbose}
	}
}

type JSONFormatter struct {
	Pretty bool
}

func (f *JSONFormatter) Format(results []client.Result) error {
	var data []byte
	var err error

	if f.Pretty {
		data, err = json.MarshalIndent(results, "", "  ")
	} else {
		data, err = json.Marshal(results)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(data))
	return nil
}

type ColorFormatter struct {
	Verbose bool
}

func (f *ColorFormatter) Format(results []client.Result) error {
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()

	successCount := 0
	errorCount := 0

	for _, result := range results {
		if result.Error == "" {
			successCount++
		} else {
			errorCount++
		}
	}

	fmt.Printf("\n%s\n", bold("=== CGMiner API Results ==="))
	if f.Verbose || errorCount == 0 {
		fmt.Printf("Total: %d | %s: %d | %s: %d\n\n",
			len(results),
			green("Success"),
			successCount,
			red("Failed"),
			errorCount,
		)
	} else {
		fmt.Printf("Total: %d | %s: %d\n\n",
			len(results),
			green("Success"),
			successCount,
		)
	}

	for i, result := range results {
		header := fmt.Sprintf("%s:%d [%s]", result.IP, result.Port, result.Command)

		if result.Error != "" {
			if f.Verbose {
				fmt.Printf("%s %s\n", red("✗"), bold(header))
				fmt.Printf("  %s: %s\n", red("Error"), result.Error)
				fmt.Printf("  %s: %s\n", cyan("Duration"), result.Duration)
			}
		} else {
			fmt.Printf("%s %s\n", green("✓"), bold(header))
			if f.Verbose {
				fmt.Printf("  %s: %s\n", cyan("Duration"), result.Duration)
			}

			f.formatResponse(result.Response, "  ")
		}
		
		// Only add blank line between entries, not after the last one
		if i < len(results)-1 {
			fmt.Println()
		}
	}

	fmt.Printf("\n%s\n", bold("=== Summary ==="))
	fmt.Printf("Command executed on %d hosts\n", len(results))
	if successCount > 0 {
		fmt.Printf("%s: %d hosts responded successfully\n", green("Success"), successCount)
	}
	if errorCount > 0 && f.Verbose {
		fmt.Printf("%s: %d hosts failed\n", red("Failed"), errorCount)
	}

	return nil
}

func (f *ColorFormatter) formatResponse(response interface{}, indent string) {
	yellow := color.New(color.FgYellow).SprintFunc()
	white := color.New(color.FgWhite).SprintFunc()

	switch v := response.(type) {
	case string:
		fmt.Printf("%s%s: %s\n", indent, yellow("Response"), white(v))
	case map[string]interface{}:
		for key, value := range v {
			f.formatKeyValue(key, value, indent)
		}
	case []interface{}:
		for i, item := range v {
			fmt.Printf("%s%s[%d]:\n", indent, yellow("Item"), i)
			f.formatResponse(item, indent+"  ")
		}
	default:
		jsonData, _ := json.MarshalIndent(v, indent, "  ")
		fmt.Printf("%s%s\n", indent, string(jsonData))
	}
}

func (f *ColorFormatter) formatKeyValue(key string, value interface{}, indent string) {
	yellow := color.New(color.FgYellow).SprintFunc()
	white := color.New(color.FgWhite).SprintFunc()

	switch v := value.(type) {
	case string, int, int64, float32, float64, bool:
		fmt.Printf("%s%s: %v\n", indent, yellow(key), white(v))
	case map[string]interface{}:
		fmt.Printf("%s%s:\n", indent, yellow(key))
		for k, val := range v {
			f.formatKeyValue(k, val, indent+"  ")
		}
	case []interface{}:
		fmt.Printf("%s%s: [%d items]\n", indent, yellow(key), len(v))
		if f.Verbose {
			for i, item := range v {
				fmt.Printf("%s  [%d]:\n", indent, i)
				f.formatResponse(item, indent+"    ")
			}
		}
	default:
		fmt.Printf("%s%s: %v\n", indent, yellow(key), white(v))
	}
}

type TableFormatter struct {
	Verbose bool
}

func (f *TableFormatter) Format(results []client.Result) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Fprintln(w, "IP\tPort\tCommand\tStatus\tDuration\tDetails")
	fmt.Fprintln(w, "---\t----\t-------\t------\t--------\t-------")

	for _, result := range results {
		// Skip error results if not in verbose mode
		if result.Error != "" && !f.Verbose {
			continue
		}

		status := "Success"
		details := ""

		if result.Error != "" {
			status = "Failed"
			details = result.Error
		} else if f.Verbose {
			if jsonData, err := json.Marshal(result.Response); err == nil {
				details = string(jsonData)
				if len(details) > 50 {
					details = details[:47] + "..."
				}
			}
		}

		fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\t%s\n",
			result.IP,
			result.Port,
			result.Command,
			status,
			result.Duration,
			details,
		)
	}

	w.Flush()

	successCount := 0
	errorCount := 0
	for _, result := range results {
		if result.Error == "" {
			successCount++
		} else {
			errorCount++
		}
	}

	if f.Verbose {
		fmt.Printf("\nSummary: Total=%d, Success=%d, Failed=%d\n",
			len(results), successCount, errorCount)
	} else {
		fmt.Printf("\nSummary: Total=%d, Success=%d\n",
			len(results), successCount)
	}

	return nil
}

// SummaryTableFormatter formats summary results grouped by subnet
type SummaryTableFormatter struct{}

func (f *SummaryTableFormatter) Format(results []client.Result) error {
	// Group results by subnet
	subnetMap := make(map[string][]client.Result)
	
	for _, result := range results {
		// Skip failed results
		if result.Error != "" {
			continue
		}
		
		// Determine subnet (using /24 for simplicity)
		ip := net.ParseIP(result.IP)
		if ip == nil {
			continue
		}
		
		// Get the /24 subnet
		subnet := getSubnet24(result.IP)
		subnetMap[subnet] = append(subnetMap[subnet], result)
	}
	
	// Sort subnet keys
	var subnets []string
	for subnet := range subnetMap {
		subnets = append(subnets, subnet)
	}
	sort.Strings(subnets)
	
	// Create tabwriter
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	
	// Process each subnet
	for _, subnet := range subnets {
		results := subnetMap[subnet]
		
		// Print subnet header
		fmt.Fprintf(w, "\n=== Subnet: %s ===\n", subnet)
		fmt.Fprintln(w, "IP\tAccepted\tMHS 5s\tMHS av\tHardware Errors")
		fmt.Fprintln(w, "---\t--------\t------\t------\t---------------")
		
		// Sort IPs within subnet
		sort.Slice(results, func(i, j int) bool {
			return ipToInt(results[i].IP) < ipToInt(results[j].IP)
		})
		
		// Print each result
		for _, result := range results {
			accepted := "-"
			mhs5s := "-"
			mhsAv := "-"
			hwErrors := "-"
			
			// Extract summary data from response
			// The response might be a struct or a map depending on how it was processed
			// Convert to JSON and back to map to handle both cases uniformly
			if jsonBytes, err := json.Marshal(result.Response); err == nil {
				var respMap map[string]interface{}
				if err := json.Unmarshal(jsonBytes, &respMap); err == nil {
					// Direct access to fields (for standard summary response)
					if val, ok := respMap["Accepted"]; ok {
						accepted = fmt.Sprintf("%v", val)
					}
					if val, ok := respMap["MHS 5s"]; ok {
						mhs5s = fmt.Sprintf("%.2f", toFloat64(val))
					}
					if val, ok := respMap["MHS av"]; ok {
						mhsAv = fmt.Sprintf("%.2f", toFloat64(val))
					}
					if val, ok := respMap["Hardware Errors"]; ok {
						hwErrors = fmt.Sprintf("%v", val)
					}
				}
			}
			
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				result.IP,
				accepted,
				mhs5s,
				mhsAv,
				hwErrors,
			)
		}
	}
	
	w.Flush()
	
	// Print summary
	totalSuccess := 0
	for _, results := range subnetMap {
		totalSuccess += len(results)
	}
	
	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Total subnets: %d\n", len(subnets))
	fmt.Printf("Total hosts responding: %d\n", totalSuccess)
	
	return nil
}

// getSubnet24 returns the /24 subnet for an IP address
func getSubnet24(ipStr string) string {
	parts := strings.Split(ipStr, ".")
	if len(parts) != 4 {
		return ipStr
	}
	return fmt.Sprintf("%s.%s.%s.0/24", parts[0], parts[1], parts[2])
}

// ipToInt converts an IP string to an integer for sorting
func ipToInt(ipStr string) uint32 {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return 0
	}
	ip = ip.To4()
	if ip == nil {
		return 0
	}
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

// toFloat64 safely converts an interface{} to float64
func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		var f float64
		_, _ = fmt.Sscanf(val, "%f", &f)
		return f
	default:
		return 0
	}
}

// ScanFormatter displays scan results with miner details including hashrate
type ScanFormatter struct {
    Verbose bool
}

func (f *ScanFormatter) Format(results []client.Result) error {
	// Sort results by IP
	sort.Slice(results, func(i, j int) bool {
		return ipToInt(results[i].IP) < ipToInt(results[j].IP)
	})

	// Count active miners
	activeCount := 0
	for _, result := range results {
		if result.Error == "" {
			activeCount++
		}
	}

	fmt.Printf("\nActive Miners Found: %d out of %d scanned\n", activeCount, len(results))
	if activeCount == 0 {
		return nil
	}

	fmt.Println(strings.Repeat("=", 80))

	// Create tabwriter
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

    // Print header
    fmt.Fprintln(w, "IP\tFirmware\tHashrate (GH/s)\tPower (kW)\tTemp (°C)\tTuning\tAccepted\tRejected\tHW Errors\tUptime")
    fmt.Fprintln(w, "---\t--------\t--------------\t----------\t---------\t------\t--------\t--------\t---------\t------")

	for _, result := range results {
		// Skip failed results unless verbose
		if result.Error != "" {
			if f.Verbose {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					result.IP,
					"-",
					"offline",
					"-",
					"-",
					"-",
					"-",
					"-",
					"-",
					"-",
				)
			}
			continue
		}

		// Extract summary data
        firmware := "-"
        hashrate := "-"
        powerKW := "-"
        chipTemp := "-"
        tuning := "-"
        accepted := "-"
        rejected := "-"
        hwErrors := "-"
        uptime := "-"

		// Convert response to map for easy access
		if jsonBytes, err := json.Marshal(result.Response); err == nil {
			var respMap map[string]interface{}
			if err := json.Unmarshal(jsonBytes, &respMap); err == nil {
                // Get firmware info
                if val, ok := respMap["firmware"]; ok {
                    firmware = fmt.Sprintf("%v", val)
                }
                // Get hashrate (prefer MHS 5s over MHS av, convert MH/s to GH/s)
                if val, ok := respMap["MHS 5s"]; ok {
                    mhs := toFloat64(val)
                    hashrate = fmt.Sprintf("%.2f", mhs/1000.0) // Convert MH/s to GH/s
                } else if val, ok := respMap["MHS av"]; ok {
                    mhs := toFloat64(val)
                    hashrate = fmt.Sprintf("%.2f", mhs/1000.0) // Convert MH/s to GH/s
                } else if val, ok := respMap["GHS 5s"]; ok {
                    // Some miners report in GH/s directly
                    hashrate = fmt.Sprintf("%.2f", toFloat64(val))
                } else if val, ok := respMap["GHS av"]; ok {
                    hashrate = fmt.Sprintf("%.2f", toFloat64(val))
                }

                // Chip temperature (prefer explicitly attached value)
                if val, ok := respMap["chip_temp_c"]; ok {
                    chipTemp = fmt.Sprintf("%.1f", toFloat64(val))
                } else {
                    if t, ok := extractTempFromSummary(respMap); ok {
                        chipTemp = fmt.Sprintf("%.1f", t)
                    }
                }

                // Power consumption (Braiins miners only)
                if val, ok := respMap["power_w"]; ok {
                    powerW := toFloat64(val)
                    powerKW = fmt.Sprintf("%.2f", powerW/1000.0)
                }

                // Tuning status (Braiins miners only)
                if val, ok := respMap["tuning"]; ok {
                    if isTuning, ok := val.(bool); ok && isTuning {
                        tuning = "Yes"
                    } else {
                        // Show tuner status if available
                        if status, ok := respMap["tuner_status"].(string); ok && status != "" {
                            tuning = status
                        }
                    }
                }

                if val, ok := respMap["Accepted"]; ok {
                    accepted = fmt.Sprintf("%v", val)
                }
				if val, ok := respMap["Rejected"]; ok {
					rejected = fmt.Sprintf("%v", val)
				}
				if val, ok := respMap["Hardware Errors"]; ok {
					hwErrors = fmt.Sprintf("%v", val)
				}
				if val, ok := respMap["Elapsed"]; ok {
					// Convert seconds to human-readable format
					seconds := int(toFloat64(val))
					days := seconds / 86400
					hours := (seconds % 86400) / 3600
					minutes := (seconds % 3600) / 60
					if days > 0 {
						uptime = fmt.Sprintf("%dd %dh", days, hours)
					} else if hours > 0 {
						uptime = fmt.Sprintf("%dh %dm", hours, minutes)
					} else {
						uptime = fmt.Sprintf("%dm", minutes)
					}
				}
			}
		}

        fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
            result.IP,
            firmware,
            hashrate,
            powerKW,
            chipTemp,
            tuning,
            accepted,
            rejected,
            hwErrors,
            uptime,
        )
	}

	w.Flush()

	// Print summary
	fmt.Println(strings.Repeat("=", 80))
	if f.Verbose && activeCount < len(results) {
		fmt.Printf("Total: %d hosts scanned, %d active, %d offline\n",
			len(results), activeCount, len(results)-activeCount)
	}

	return nil
}

// extractTempFromSummary scans a summary response map for plausible
// temperature fields and returns a representative value.
func extractTempFromSummary(resp map[string]interface{}) (float64, bool) {
    // Prefer explicit fields
    preferredKeys := []string{"Temperature", "Chip Temp", "ChipTemp", "Temp"}
    for _, k := range preferredKeys {
        if v, ok := resp[k]; ok {
            t := toFloat64(v)
            if t > 0 && t <= 200 {
                return t, true
            }
        }
    }
    // Fallback: scan any temp-like keys
    maxT := 0.0
    found := false
    for k, v := range resp {
        lk := strings.ToLower(k)
        if strings.Contains(lk, "temp") {
            t := toFloat64(v)
            if t > 0 && t <= 200 {
                if !found || t > maxT {
                    maxT = t
                    found = true
                }
            }
        }
    }
    return maxT, found
}
