package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yourproject/miner-cli/internal/client"
	"github.com/yourproject/miner-cli/internal/iprange"
	"github.com/yourproject/miner-cli/internal/output"
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
  miner-cli summary -i 192.168.1.100
  
  # Query multiple IP ranges
  miner-cli devs -i 192.168.1.0/24 -i 10.0.0.1-10.0.0.50
  
  # Output as JSON
  miner-cli stats -i 192.168.1.0/28 -o json
  
  # Add a new pool to multiple miners
  miner-cli addpool -i 192.168.1.0/24 --url stratum+tcp://pool.example.com:3333 --user myworker --pass x`,
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
			Use:   cmdCopy,
			Short: client.GetCommandDescription(cmdCopy),
			PreRunE: func(c *cobra.Command, args []string) error {
				if len(ipRanges) == 0 {
					return fmt.Errorf("no IP ranges specified, use -i flag")
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

	rootCmd.AddCommand(&cobra.Command{
		Use:   "scan",
		Short: "Scan IP ranges to find active miners",
		PreRunE: func(c *cobra.Command, args []string) error {
			if len(ipRanges) == 0 {
				return fmt.Errorf("no IP ranges specified, use -i flag")
			}
			return nil
		},
		RunE: scanMiners,
	})
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

	formatter := output.GetFormatter(outputFormat, verbose)
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
	results := cgClient.ExecuteCommand(ctx, ips, port, "version", nil)

	activeMiners := []string{}
	for _, result := range results {
		if result.Error == "" {
			activeMiners = append(activeMiners, fmt.Sprintf("%s:%d", result.IP, result.Port))
		}
	}

	if outputFormat == "json" {
		formatter := output.GetFormatter(outputFormat, verbose)
		return formatter.Format(results)
	}

	fmt.Printf("\nActive Miners Found: %d\n", len(activeMiners))
	fmt.Println(strings.Repeat("=", 40))
	for _, miner := range activeMiners {
		fmt.Println(miner)
	}

	return nil
}

func parseIntParam(param string) (int, error) {
	return strconv.Atoi(param)
}
