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
			if respMap, ok := result.Response.(map[string]interface{}); ok {
				if summaryList, ok := respMap["SUMMARY"].([]interface{}); ok && len(summaryList) > 0 {
					if summary, ok := summaryList[0].(map[string]interface{}); ok {
						if val, ok := summary["Accepted"]; ok {
							accepted = fmt.Sprintf("%v", val)
						}
						if val, ok := summary["MHS 5s"]; ok {
							mhs5s = fmt.Sprintf("%.2f", toFloat64(val))
						}
						if val, ok := summary["MHS av"]; ok {
							mhsAv = fmt.Sprintf("%.2f", toFloat64(val))
						}
						if val, ok := summary["Hardware Errors"]; ok {
							hwErrors = fmt.Sprintf("%v", val)
						}
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
		fmt.Sscanf(val, "%f", &f)
		return f
	default:
		return 0
	}
}
