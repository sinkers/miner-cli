package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/yourproject/miner-cli/internal/client"
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
	fmt.Printf("Total: %d | %s: %d | %s: %d\n\n",
		len(results),
		green("Success"),
		successCount,
		red("Failed"),
		errorCount,
	)

	for _, result := range results {
		header := fmt.Sprintf("%s:%d [%s]", result.IP, result.Port, result.Command)

		if result.Error != "" {
			fmt.Printf("%s %s\n", red("✗"), bold(header))
			fmt.Printf("  %s: %s\n", red("Error"), result.Error)
			if f.Verbose {
				fmt.Printf("  %s: %s\n", cyan("Duration"), result.Duration)
			}
		} else {
			fmt.Printf("%s %s\n", green("✓"), bold(header))
			if f.Verbose {
				fmt.Printf("  %s: %s\n", cyan("Duration"), result.Duration)
			}

			f.formatResponse(result.Response, "  ")
		}
		fmt.Println()
	}

	fmt.Printf("%s\n", bold("=== Summary ==="))
	fmt.Printf("Command executed on %d hosts\n", len(results))
	if successCount > 0 {
		fmt.Printf("%s: %d hosts responded successfully\n", green("Success"), successCount)
	}
	if errorCount > 0 {
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

	fmt.Printf("\nSummary: Total=%d, Success=%d, Failed=%d\n",
		len(results), successCount, errorCount)

	return nil
}
