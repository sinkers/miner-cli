package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/yourproject/miner-cli/internal/vnish/client"
)

func main() {
	var (
		host   = flag.String("host", "10.45.3.1", "Vnish API host")
		apiKey = flag.String("apikey", "", "API key for authentication")
		debug  = flag.Bool("debug", false, "Enable debug output")
	)
	flag.Parse()

	// Create client with options
	opts := []client.Option{
		client.WithTimeout(10 * time.Second),
		client.WithDebug(*debug),
	}
	
	if *apiKey != "" {
		opts = append(opts, client.WithAPIKey(*apiKey))
	}

	vnishClient := client.NewClient(*host, opts...)
	ctx := context.Background()

	// Example: Get system information
	fmt.Println("=== System Information ===")
	info, err := vnishClient.GetInfo(ctx)
	if err != nil {
		log.Printf("Failed to get info: %v", err)
	} else {
		fmt.Printf("Hostname: %s\n", info.Hostname)
		fmt.Printf("Model: %s\n", info.Model)
		fmt.Printf("Version: %s\n", info.Version)
		fmt.Printf("Uptime: %d seconds\n", info.Uptime)
	}

	// Example: Get mining summary
	fmt.Println("\n=== Mining Summary ===")
	summary, err := vnishClient.GetSummary(ctx)
	if err != nil {
		log.Printf("Failed to get summary: %v", err)
	} else {
		fmt.Printf("Mining Status: %s\n", summary.Status.Status)
		fmt.Printf("Hash Rate: %.2f %s\n", summary.Performance.HashRate, summary.Performance.HashRateUnit)
		fmt.Printf("Power Usage: %.0f W\n", summary.Performance.PowerUsage)
		fmt.Printf("Efficiency: %.2f J/TH\n", summary.Performance.Efficiency)
		fmt.Printf("Accepted Shares: %d\n", summary.Performance.Accepted)
		fmt.Printf("Rejected Shares: %d\n", summary.Performance.Rejected)
		
		fmt.Println("\nPools:")
		for _, pool := range summary.Pools {
			fmt.Printf("  [%d] %s (%s) - Status: %s\n", 
				pool.ID, pool.URL, pool.User, pool.Status)
		}
		
		fmt.Println("\nTemperatures:")
		fmt.Printf("  Intake: %.1f°C\n", summary.Temperature.Intake)
		fmt.Printf("  Outlet: %.1f°C\n", summary.Temperature.Outlet)
		if len(summary.Temperature.Board) > 0 {
			fmt.Printf("  Boards: ")
			for i, temp := range summary.Temperature.Board {
				if i > 0 {
					fmt.Printf(", ")
				}
				fmt.Printf("%.1f°C", temp)
			}
			fmt.Println()
		}
	}

	// Example: Get chain information
	fmt.Println("\n=== Chain Information ===")
	chains, err := vnishClient.GetChains(ctx)
	if err != nil {
		log.Printf("Failed to get chains: %v", err)
	} else {
		for _, chain := range chains {
			fmt.Printf("Chain %d: Status=%s, Freq=%dMHz, Voltage=%.2fV, Temp=%.1f°C, Chips=%d, HashRate=%.2fTH/s\n",
				chain.Index, chain.Status, chain.Frequency, chain.Voltage, 
				chain.Temperature, chain.ChipCount, chain.HashRate)
		}
	}

	// Example: Get performance summary
	fmt.Println("\n=== Performance Summary ===")
	perf, err := vnishClient.GetPerfSummary(ctx)
	if err != nil {
		log.Printf("Failed to get performance: %v", err)
	} else {
		fmt.Printf("Hash Rate: %.2f %s\n", perf.HashRate, perf.HashRateUnit)
		fmt.Printf("Power Usage: %.0f W\n", perf.PowerUsage)
		fmt.Printf("Efficiency: %.2f J/TH\n", perf.Efficiency)
		fmt.Printf("Hardware Errors: %d\n", perf.HardwareErrors)
	}

	// Example: Check authentication
	fmt.Println("\n=== Authentication Check ===")
	auth, err := vnishClient.CheckAuth(ctx)
	if err != nil {
		log.Printf("Failed to check auth: %v", err)
	} else {
		fmt.Printf("Authenticated: %v\n", auth.Authenticated)
		if auth.Method != "" {
			fmt.Printf("Method: %s\n", auth.Method)
		}
	}
}