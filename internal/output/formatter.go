package output

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"os"
	"regexp"
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

// ansiRegex matches ANSI escape sequences for removing them when calculating display width
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// displayWidth returns the visible width of a string, excluding ANSI escape codes
func displayWidth(s string) int {
	return len(ansiRegex.ReplaceAllString(s, ""))
}

// padRight pads a string to the specified display width (accounting for ANSI codes)
func padRight(s string, width int) string {
	dw := displayWidth(s)
	if dw >= width {
		return s
	}
	return s + strings.Repeat(" ", width-dw)
}

// tableRow represents a single row in the table with both plain and colored values
type tableRow struct {
	plainValues   []string
	coloredValues []string
}

// printAlignedTable prints a table with proper column alignment, accounting for ANSI codes
func printAlignedTable(headers []string, headerColors []func(...interface{}) string, rows []tableRow, colSpacing int) {
	if len(rows) == 0 {
		return
	}

	// Calculate maximum width for each column based on plain text
	numCols := len(headers)
	colWidths := make([]int, numCols)

	// Start with header widths
	for i, header := range headers {
		colWidths[i] = len(header)
	}

	// Check all rows
	for _, row := range rows {
		for i, val := range row.plainValues {
			if i < numCols && len(val) > colWidths[i] {
				colWidths[i] = len(val)
			}
		}
	}

	// Print header with colors
	headerParts := make([]string, numCols)
	for i, header := range headers {
		if i < len(headerColors) && headerColors[i] != nil {
			headerParts[i] = padRight(headerColors[i](header), colWidths[i])
		} else {
			headerParts[i] = padRight(header, colWidths[i])
		}
	}
	fmt.Println(strings.Join(headerParts, strings.Repeat(" ", colSpacing)))

	// Print rows with colored values
	for _, row := range rows {
		rowParts := make([]string, numCols)
		for i := 0; i < numCols && i < len(row.coloredValues); i++ {
			rowParts[i] = padRight(row.coloredValues[i], colWidths[i])
		}
		fmt.Println(strings.Join(rowParts, strings.Repeat(" ", colSpacing)))
	}
}

// scanStats holds collected statistics from scan results
type scanStats struct {
	activeCount  int
	temperatures []float64
	powerLevels  []float64
	hashrates    []float64
}

// collectScanStats gathers statistics from scan results
func (f *ScanFormatter) collectScanStats(results []client.Result) scanStats {
	var stats scanStats

	for _, result := range results {
		if result.Error == "" {
			stats.activeCount++

			// Extract data for statistics
			if result.Response != nil {
				if jsonBytes, err := json.Marshal(result.Response); err == nil {
					var respMap map[string]interface{}
					if err := json.Unmarshal(jsonBytes, &respMap); err == nil {
						// Collect chip temperature
						if val, ok := respMap["chip_temp_c"]; ok {
							if temp := toFloat64(val); temp > 0 && temp <= 200 {
								stats.temperatures = append(stats.temperatures, temp)
							}
						} else if t, ok := extractTempFromSummary(respMap); ok && t > 0 && t <= 200 {
							stats.temperatures = append(stats.temperatures, t)
						}

						// Collect power consumption (in watts)
						if val, ok := respMap["power_w"]; ok {
							if power := toFloat64(val); power > 0 {
								stats.powerLevels = append(stats.powerLevels, power)
							}
						}

						// Collect hashrate (convert to GH/s)
						if val, ok := respMap["MHS 5s"]; ok {
							if mhs := toFloat64(val); mhs > 0 {
								stats.hashrates = append(stats.hashrates, mhs/1000.0)
							}
						} else if val, ok := respMap["MHS av"]; ok {
							if mhs := toFloat64(val); mhs > 0 {
								stats.hashrates = append(stats.hashrates, mhs/1000.0)
							}
						} else if val, ok := respMap["GHS 5s"]; ok {
							if ghs := toFloat64(val); ghs > 0 {
								stats.hashrates = append(stats.hashrates, ghs)
							}
						} else if val, ok := respMap["GHS av"]; ok {
							if ghs := toFloat64(val); ghs > 0 {
								stats.hashrates = append(stats.hashrates, ghs)
							}
						}
					}
				}
			}
		}
	}

	return stats
}

// printScanHeader prints the scan results header
func (f *ScanFormatter) printScanHeader(results []client.Result, activeCount int, boldGreen, white, cyan, boldCyan, green, red, boldRed func(a ...interface{}) string) {
	fmt.Printf("\n%s %s %s %s %s %s\n",
		boldGreen("⛏️  MINER SCAN RESULTS"),
		white("│"),
		cyan("🔍 Scanned:"), boldCyan(fmt.Sprintf("%d", len(results))),
		green("✓ Active:"), boldGreen(fmt.Sprintf("%d", activeCount)))

	if activeCount == 0 {
		fmt.Printf("\n%s %s\n", red("❌"), boldRed("No active miners found"))
	}
}

// printSummaryStats prints the summary statistics section
func (f *ScanFormatter) printSummaryStats(stats scanStats, green, red, yellow, cyan, magenta, white, bold, boldCyan, boldYellow, boldMagenta func(a ...interface{}) string) {
	fmt.Println(cyan(strings.Repeat("═", 80)))
	fmt.Printf("%s %s\n", boldCyan("📊"), bold("SUMMARY STATISTICS"))
	fmt.Println(cyan(strings.Repeat("─", 80)))

	// Temperature statistics
	f.printTemperatureStats(stats, green, red, yellow, cyan, white, bold, boldCyan)
	fmt.Println()

	// Power statistics
	f.printPowerStats(stats, green, yellow, cyan, white, bold, boldCyan, boldYellow)
	fmt.Println()

	// Hashrate statistics
	f.printHashrateStats(stats, cyan, magenta, white, bold, boldCyan, boldMagenta, yellow)
}

// printTemperatureStats prints temperature statistics
func (f *ScanFormatter) printTemperatureStats(stats scanStats, green, red, yellow, cyan, white, bold, boldCyan func(a ...interface{}) string) {
	if len(stats.temperatures) > 0 {
		tempStats := calculateStats(stats.temperatures)
		tempColor := green
		tempIcon := "🌡️"
		if tempStats.avg > 75 {
			tempColor = red
			tempIcon = "🔥"
		} else if tempStats.avg > 65 {
			tempColor = yellow
			tempIcon = "⚠️"
		}

		fmt.Printf("%s %s\n", tempIcon, bold("Chip Temperature (°C):"))
		fmt.Printf("  %s %s  %s  %s %s  %s  %s %s  %s  %s %s\n",
			white("📈 Average:"), tempColor(fmt.Sprintf("%.1f", tempStats.avg)),
			white("│"),
			white("📉 Min:"), green(fmt.Sprintf("%.1f", tempStats.min)),
			white("│"),
			white("📊 Max:"), tempColor(fmt.Sprintf("%.1f", tempStats.max)),
			white("│"),
			white("σ Std Dev:"), cyan(fmt.Sprintf("%.1f", tempStats.stdDev)))
		fmt.Printf("  %s %s\n",
			white("📡 Reporting:"),
			boldCyan(fmt.Sprintf("%d/%d miners", len(stats.temperatures), stats.activeCount)))
	} else {
		fmt.Printf("%s %s %s\n", "🌡️", bold("Chip Temperature:"), yellow("No data available"))
	}
}

// printPowerStats prints power consumption statistics
func (f *ScanFormatter) printPowerStats(stats scanStats, green, yellow, cyan, white, bold, boldCyan, boldYellow func(a ...interface{}) string) {
	if len(stats.powerLevels) > 0 {
		powerStats := calculateStats(stats.powerLevels)
		fmt.Printf("%s %s\n", "⚡", bold("Power Consumption (kW):"))
		fmt.Printf("  %s %s  %s  %s %s  %s  %s %s  %s  %s %s\n",
			white("📈 Average:"), yellow(fmt.Sprintf("%.2f", powerStats.avg/1000.0)),
			white("│"),
			white("📉 Min:"), green(fmt.Sprintf("%.2f", powerStats.min/1000.0)),
			white("│"),
			white("📊 Max:"), yellow(fmt.Sprintf("%.2f", powerStats.max/1000.0)),
			white("│"),
			white("σ Std Dev:"), cyan(fmt.Sprintf("%.2f", powerStats.stdDev/1000.0)))
		fmt.Printf("  %s %s  %s  %s %s\n",
			white("📡 Reporting:"),
			boldCyan(fmt.Sprintf("%d/%d miners", len(stats.powerLevels), stats.activeCount)),
			white("│"),
			white("⚡ Total:"),
			boldYellow(fmt.Sprintf("%.2f kW", powerStats.sum/1000.0)))
	} else {
		fmt.Printf("%s %s %s\n", "⚡", bold("Power Consumption:"), yellow("No data available"))
	}
}

// printHashrateStats prints hashrate statistics
func (f *ScanFormatter) printHashrateStats(stats scanStats, cyan, magenta, white, bold, boldCyan, boldMagenta, yellow func(a ...interface{}) string) {
	if len(stats.hashrates) > 0 {
		hashStats := calculateStats(stats.hashrates)
		totalHashStr := fmt.Sprintf("%.2f GH/s", hashStats.sum)
		if hashStats.sum > 1000 {
			totalHashStr = fmt.Sprintf("%.2f TH/s", hashStats.sum/1000.0)
		}

		fmt.Printf("%s %s\n", "⛏️", bold("Hashrate (GH/s):"))
		fmt.Printf("  %s %s  %s  %s %s  %s  %s %s  %s  %s %s\n",
			white("📈 Average:"), magenta(fmt.Sprintf("%.2f", hashStats.avg)),
			white("│"),
			white("📉 Min:"), cyan(fmt.Sprintf("%.2f", hashStats.min)),
			white("│"),
			white("📊 Max:"), magenta(fmt.Sprintf("%.2f", hashStats.max)),
			white("│"),
			white("σ Std Dev:"), cyan(fmt.Sprintf("%.2f", hashStats.stdDev)))
		fmt.Printf("  %s %s  %s  %s %s\n",
			white("📡 Reporting:"),
			boldCyan(fmt.Sprintf("%d/%d miners", len(stats.hashrates), stats.activeCount)),
			white("│"),
			white("💰 Total:"),
			boldMagenta(totalHashStr))
	} else {
		fmt.Printf("%s %s %s\n", "⛏️", bold("Hashrate:"), yellow("No data available"))
	}
}

func (f *ScanFormatter) Format(results []client.Result) error {
	// Color definitions - define all colors at the start
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	magenta := color.New(color.FgMagenta).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()
	white := color.New(color.FgWhite).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	boldGreen := color.New(color.FgGreen, color.Bold).SprintFunc()
	boldYellow := color.New(color.FgYellow, color.Bold).SprintFunc()
	boldCyan := color.New(color.FgCyan, color.Bold).SprintFunc()
	boldMagenta := color.New(color.FgMagenta, color.Bold).SprintFunc()
	boldRed := color.New(color.FgRed, color.Bold).SprintFunc()

	// Sort results by IP
	sort.Slice(results, func(i, j int) bool {
		return ipToInt(results[i].IP) < ipToInt(results[j].IP)
	})

	// Collect statistics
	stats := f.collectScanStats(results)

	// Print header
	f.printScanHeader(results, stats.activeCount, boldGreen, white, cyan, boldCyan, green, red, boldRed)

	if stats.activeCount == 0 {
		return nil
	}

	// Display summary statistics
	f.printSummaryStats(stats, green, red, yellow, cyan, magenta, white, bold, boldCyan, boldYellow, boldMagenta)

	fmt.Println(cyan(strings.Repeat("═", 80)))

	// Check if any result has switch_port data to determine whether to show port column
	hasPortData := false
	for _, result := range results {
		if result.Error == "" && result.Response != nil {
			if jsonBytes, err := json.Marshal(result.Response); err == nil {
				var respMap map[string]interface{}
				if err := json.Unmarshal(jsonBytes, &respMap); err == nil {
					if _, ok := respMap["switch_port"]; ok {
						hasPortData = true
						break
					}
				}
			}
		}
	}

	// Color for table header
	headerColor := color.New(color.FgCyan, color.Bold).SprintFunc()

	// Prepare table headers
	var headers []string
	var headerColors []func(...interface{}) string
	var separatorLine string

	if hasPortData {
		headers = []string{"IP", "Port", "Firmware", "Hashrate (GH/s)", "Power (kW)", "Temp (°C)", "Tuning", "Accepted", "Rejected", "HW Errors", "Uptime"}
		headerColors = make([]func(...interface{}) string, len(headers))
		for i := range headerColors {
			headerColors[i] = headerColor
		}
		separatorLine = cyan("───  ────  ────────  ───────────────  ──────────  ─────────  ──────  ────────  ────────  ─────────  ──────")
	} else {
		headers = []string{"IP", "Firmware", "Hashrate (GH/s)", "Power (kW)", "Temp (°C)", "Tuning", "Accepted", "Rejected", "HW Errors", "Uptime"}
		headerColors = make([]func(...interface{}) string, len(headers))
		for i := range headerColors {
			headerColors[i] = headerColor
		}
		separatorLine = cyan("───  ────────  ───────────────  ──────────  ─────────  ──────  ────────  ────────  ─────────  ──────")
	}

	// Collect all rows
	var rows []tableRow

	for _, result := range results {
		// Skip failed results unless verbose
		if result.Error != "" {
			if f.Verbose {
				if hasPortData {
					rows = append(rows, tableRow{
						plainValues:   []string{result.IP, "-", "-", "offline", "-", "-", "-", "-", "-", "-", "-"},
						coloredValues: []string{white(result.IP), white("-"), white("-"), red("offline"), white("-"), white("-"), white("-"), white("-"), white("-"), white("-"), white("-")},
					})
				} else {
					rows = append(rows, tableRow{
						plainValues:   []string{result.IP, "-", "offline", "-", "-", "-", "-", "-", "-", "-"},
						coloredValues: []string{white(result.IP), white("-"), red("offline"), white("-"), white("-"), white("-"), white("-"), white("-"), white("-"), white("-")},
					})
				}
			}
			continue
		}

		// Extract summary data
		firmware := "-"
		switchPort := "-"
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
				// Get switch port info
				if val, ok := respMap["switch_port"]; ok {
					switchPort = fmt.Sprintf("%v", val)
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

		// Color the data based on values
		ipColor := white(result.IP)

		// Color firmware based on type
		firmwareColor := firmware
		if strings.Contains(firmware, "Braiins") {
			firmwareColor = blue(firmware)
		} else if firmware == "Stock" {
			firmwareColor = white(firmware)
		}

		// Color hashrate
		hashrateColor := hashrate
		if hashrate != "-" {
			hr := toFloat64(strings.TrimSuffix(hashrate, " GH/s"))
			if hr > 130000 {
				hashrateColor = boldMagenta(hashrate)
			} else if hr > 125000 {
				hashrateColor = magenta(hashrate)
			} else {
				hashrateColor = cyan(hashrate)
			}
		}

		// Color power
		powerColor := powerKW
		if powerKW != "-" {
			pw := toFloat64(strings.TrimSuffix(powerKW, " kW"))
			if pw > 4.2 {
				powerColor = red(powerKW)
			} else {
				powerColor = yellow(powerKW)
			}
		}

		// Color temperature based on value
		tempColor := chipTemp
		if chipTemp != "-" {
			temp := toFloat64(strings.TrimSuffix(chipTemp, "°C"))
			if temp > 75 {
				tempColor = boldRed(chipTemp)
			} else if temp > 65 {
				tempColor = yellow(chipTemp)
			} else {
				tempColor = green(chipTemp)
			}
		}

		// Color tuning status
		tuningColor := tuning
		if tuning == "STABLE" {
			tuningColor = green("✓ " + tuning)
		} else if tuning == "Yes" || tuning == "TUNING" {
			tuningColor = yellow("⚙ " + tuning)
		} else if tuning != "-" {
			tuningColor = cyan(tuning)
		}

		// Color HW errors
		hwErrorsColor := hwErrors
		if hwErrors != "-" {
			errors := toFloat64(hwErrors)
			if errors > 100 {
				hwErrorsColor = boldRed(hwErrors)
			} else if errors > 0 {
				hwErrorsColor = yellow(hwErrors)
			} else {
				hwErrorsColor = green(hwErrors)
			}
		}

		// Color uptime
		uptimeColor := cyan(uptime)

		// Get tuning display text (with or without icons)
		tuningDisplay := tuning
		if tuning == "STABLE" {
			tuningDisplay = "✓ " + tuning
		} else if tuning == "Yes" || tuning == "TUNING" {
			tuningDisplay = "⚙ " + tuning
		}

		if hasPortData {
			rows = append(rows, tableRow{
				plainValues:   []string{result.IP, switchPort, firmware, hashrate, powerKW, chipTemp, tuningDisplay, accepted, rejected, hwErrors, uptime},
				coloredValues: []string{ipColor, white(switchPort), firmwareColor, hashrateColor, powerColor, tempColor, tuningColor, white(accepted), white(rejected), hwErrorsColor, uptimeColor},
			})
		} else {
			rows = append(rows, tableRow{
				plainValues:   []string{result.IP, firmware, hashrate, powerKW, chipTemp, tuningDisplay, accepted, rejected, hwErrors, uptime},
				coloredValues: []string{ipColor, firmwareColor, hashrateColor, powerColor, tempColor, tuningColor, white(accepted), white(rejected), hwErrorsColor, uptimeColor},
			})
		}
	}

	// Print the table with proper alignment
	printAlignedTable(headers, headerColors, rows, 2)
	fmt.Println(separatorLine)

	// Print summary footer with colors and icons
	fmt.Println(cyan(strings.Repeat("═", 80)))
	if f.Verbose && stats.activeCount < len(results) {
		fmt.Printf("%s %s %s  %s  %s %s  %s  %s %s\n",
			white("📊"),
			white("Total:"),
			boldCyan(fmt.Sprintf("%d", len(results))),
			white("│"),
			green("✅ Active:"),
			boldGreen(fmt.Sprintf("%d", stats.activeCount)),
			white("│"),
			red("❌ Offline:"),
			boldRed(fmt.Sprintf("%d", len(results)-stats.activeCount)))
	}
	fmt.Printf("\n%s %s\n", green("✨"), bold("Scan completed successfully!"))

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

// stats holds statistical calculations
type stats struct {
	avg    float64
	min    float64
	max    float64
	stdDev float64
	sum    float64
}

// calculateStats computes statistics for a slice of float64 values
func calculateStats(values []float64) stats {
	if len(values) == 0 {
		return stats{}
	}

	// Calculate sum, min, max
	sum := 0.0
	min := values[0]
	max := values[0]
	for _, v := range values {
		sum += v
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	// Calculate average
	avg := sum / float64(len(values))

	// Calculate standard deviation
	variance := 0.0
	for _, v := range values {
		diff := v - avg
		variance += diff * diff
	}
	variance /= float64(len(values))
	stdDev := math.Sqrt(variance)

	return stats{
		avg:    avg,
		min:    min,
		max:    max,
		stdDev: stdDev,
		sum:    sum,
	}
}
