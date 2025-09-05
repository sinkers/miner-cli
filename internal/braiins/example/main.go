package main

import (
	"fmt"
	"log"
	"time"

	"github.com/sinkers/miner-cli/internal/braiins/client"
	"github.com/sinkers/miner-cli/internal/braiins/models"
	pb "github.com/sinkers/miner-cli/internal/braiins/bos/v1"
)

func main() {
	// Example: Connect to a Braiins OS+ miner
	fmt.Println("=== Braiins OS+ Client Example ===\n")

	// Create client with authentication
	opts := client.ClientOptions{
		Host:     "192.168.1.100", // Replace with your miner's IP
		Port:     50051,           // Default gRPC port
		Username: "admin",         // Replace with your username
		Password: "admin",         // Replace with your password
		Timeout:  30 * time.Second,
		UseTLS:   false,
	}

	// Connect to the miner
	fmt.Println("Connecting to Braiins OS+ miner...")
	c, err := client.NewClient(opts)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()
	fmt.Println("✓ Connected successfully\n")

	// Example 1: Get Miner Details
	fmt.Println("1. Miner Details:")
	fmt.Println("-----------------")
	details, err := c.GetMinerDetails()
	if err != nil {
		log.Printf("Failed to get miner details: %v", err)
	} else {
		minerInfo := models.ConvertMinerDetails(details)
		fmt.Printf("Hostname:    %s\n", minerInfo.Hostname)
		fmt.Printf("MAC Address: %s\n", minerInfo.MacAddress)
		fmt.Printf("Model:       %s %s\n", minerInfo.Vendor, minerInfo.Model)
		fmt.Printf("HW Version:  %s\n", minerInfo.HardwareVersion)
		fmt.Printf("FW Version:  %s\n", minerInfo.FirmwareVersion)
		fmt.Printf("BOS Version: %s (%s)\n", minerInfo.BOSVersion, minerInfo.BOSMode)
	}
	fmt.Println()

	// Example 2: Get Mining Statistics
	fmt.Println("2. Mining Statistics:")
	fmt.Println("--------------------")
	stats, err := c.GetMinerStats()
	if err != nil {
		log.Printf("Failed to get miner stats: %v", err)
	} else {
		minerStats := models.ConvertMinerStats(stats)
		fmt.Printf("Hashrate (avg):   %.2f TH/s\n", minerStats.HashRateAverage)
		fmt.Printf("Hashrate (5s):    %.2f TH/s\n", minerStats.HashRate5s)
		fmt.Printf("Hashrate (1h):    %.2f TH/s\n", minerStats.HashRate1h)
		fmt.Printf("Power:            %.0f W\n", minerStats.PowerConsumption)
		fmt.Printf("Efficiency:       %.2f J/TH\n", minerStats.Efficiency)
		fmt.Printf("Temperature:      %.1f°C\n", minerStats.Temperature)
		fmt.Printf("Accepted Shares:  %d\n", minerStats.AcceptedShares)
		fmt.Printf("Rejected Shares:  %d\n", minerStats.RejectedShares)
		
		if len(minerStats.FanSpeed) > 0 {
			fmt.Print("Fan Speeds:       ")
			for i, speed := range minerStats.FanSpeed {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Printf("%d RPM", speed)
			}
			fmt.Println()
		}
	}
	fmt.Println()

	// Example 3: Get Hashboard Information
	fmt.Println("3. Hashboard Information:")
	fmt.Println("------------------------")
	hashboards, err := c.GetHashboards()
	if err != nil {
		log.Printf("Failed to get hashboards: %v", err)
	} else {
		boards := models.ConvertHashboards(hashboards)
		for _, board := range boards {
			fmt.Printf("Board %d (%s):\n", board.Slot, board.ID)
			fmt.Printf("  Status:      %s\n", board.Status)
			fmt.Printf("  Hashrate:    %.2f TH/s\n", board.HashRate)
			fmt.Printf("  Temperature: %.1f°C\n", board.Temperature)
			fmt.Printf("  Chips:       %d/%d active\n", board.ActiveChips, board.ChipCount)
			fmt.Printf("  Voltage:     %.1fV\n", board.Voltage)
			fmt.Printf("  Frequency:   %.0f MHz\n", board.Frequency)
		}
	}
	fmt.Println()

	// Example 4: Get Pool Configuration
	fmt.Println("4. Pool Configuration:")
	fmt.Println("---------------------")
	pools, err := c.GetPools()
	if err != nil {
		log.Printf("Failed to get pools: %v", err)
	} else {
		poolList := models.ConvertPools(pools)
		for i, pool := range poolList {
			fmt.Printf("Pool %d:\n", i+1)
			fmt.Printf("  URL:       %s\n", pool.URL)
			fmt.Printf("  User:      %s\n", pool.User)
			fmt.Printf("  Status:    %s\n", pool.Status)
			fmt.Printf("  Accepted:  %d\n", pool.Accepted)
			fmt.Printf("  Rejected:  %d\n", pool.Rejected)
			if pool.LastDifficulty > 0 {
				fmt.Printf("  Last Diff: %.0f\n", pool.LastDifficulty)
			}
		}
	}
	fmt.Println()

	// Example 5: Get Performance Mode
	fmt.Println("5. Performance Mode:")
	fmt.Println("-------------------")
	perfMode, err := c.GetPerformanceMode()
	if err != nil {
		log.Printf("Failed to get performance mode: %v", err)
	} else {
		mode := models.ConvertPerformanceMode(perfMode)
		fmt.Printf("Mode: %s\n", mode.Mode)
		if mode.PowerTarget > 0 {
			fmt.Printf("Power Target: %.0f W\n", mode.PowerTarget)
		}
		if mode.HashrateTarget > 0 {
			fmt.Printf("Hashrate Target: %.2f TH/s\n", mode.HashrateTarget)
		}
		if mode.Mode == "manual" {
			fmt.Printf("Manual Voltage: %.1f V\n", mode.Voltage)
			fmt.Printf("Manual Frequency: %.0f MHz\n", mode.Frequency)
		}
	}
	fmt.Println()

	// Example 6: Pool Management Operations
	fmt.Println("6. Pool Management Examples:")
	fmt.Println("---------------------------")
	
	// Add a new pool (commented out to avoid actual changes)
	/*
	newPool := &pb.Pool{
		Url:     "stratum+tcp://new.pool.com:3333",
		User:    "wallet.worker",
		Enabled: true,
	}
	
	fmt.Println("Adding new pool...")
	_, err = c.CreatePool(newPool)
	if err != nil {
		log.Printf("Failed to create pool: %v", err)
	} else {
		fmt.Println("✓ Pool added successfully")
	}
	*/
	
	// Example of disabling a pool (commented out)
	/*
	fmt.Println("Disabling pool...")
	_, err = c.DisablePool("pool-2")
	if err != nil {
		log.Printf("Failed to disable pool: %v", err)
	} else {
		fmt.Println("✓ Pool disabled successfully")
	}
	*/
	
	fmt.Println("(Pool management operations are commented out to avoid changes)")
	fmt.Println()

	// Example 7: Performance Tuning
	fmt.Println("7. Performance Tuning Examples:")
	fmt.Println("-------------------------------")
	
	// Set power target (commented out to avoid actual changes)
	/*
	fmt.Println("Setting power target to 3000W...")
	_, err = c.SetPowerTarget(3000)
	if err != nil {
		log.Printf("Failed to set power target: %v", err)
	} else {
		fmt.Println("✓ Power target set successfully")
	}
	*/
	
	// Set hashrate target (commented out)
	/*
	fmt.Println("Setting hashrate target to 100 TH/s...")
	_, err = c.SetHashrateTarget(100)
	if err != nil {
		log.Printf("Failed to set hashrate target: %v", err)
	} else {
		fmt.Println("✓ Hashrate target set successfully")
	}
	*/
	
	fmt.Println("(Performance tuning operations are commented out to avoid changes)")
	fmt.Println()

	// Example 8: Mining Control Operations
	fmt.Println("8. Mining Control Examples:")
	fmt.Println("--------------------------")
	
	// Restart mining (commented out to avoid actual restarts)
	/*
	fmt.Println("Restarting mining...")
	_, err = c.RestartMining()
	if err != nil {
		log.Printf("Failed to restart mining: %v", err)
	} else {
		fmt.Println("✓ Mining restarted successfully")
	}
	*/
	
	// Pause mining (commented out)
	/*
	fmt.Println("Pausing mining...")
	_, err = c.PauseMining()
	if err != nil {
		log.Printf("Failed to pause mining: %v", err)
	} else {
		fmt.Println("✓ Mining paused successfully")
		
		// Wait a bit then resume
		time.Sleep(5 * time.Second)
		
		fmt.Println("Resuming mining...")
		_, err = c.ResumeMining()
		if err != nil {
			log.Printf("Failed to resume mining: %v", err)
		} else {
			fmt.Println("✓ Mining resumed successfully")
		}
	}
	*/
	
	fmt.Println("(Mining control operations are commented out to avoid disruption)")
	fmt.Println()

	// Example 9: Cooling Control
	fmt.Println("9. Cooling Control Examples:")
	fmt.Println("---------------------------")
	
	cooling, err := c.GetCooling()
	if err != nil {
		log.Printf("Failed to get cooling info: %v", err)
	} else {
		coolingInfo := models.ConvertCooling(cooling)
		if coolingInfo != nil {
			fmt.Printf("Immersion Mode: %v\n", coolingInfo.ImmersionMode)
			fmt.Printf("Fan Mode: %s\n", coolingInfo.FanMode)
			for i, fan := range coolingInfo.FanSpeed {
				fmt.Printf("Fan %d: %d RPM (%d%%)\n", fan.ID, fan.Speed, fan.SpeedPct)
				if i >= 1 { // Limit output
					break
				}
			}
		}
	}
	
	// Set immersion mode (commented out)
	/*
	fmt.Println("Enabling immersion cooling mode...")
	_, err = c.SetImmersionMode(true)
	if err != nil {
		log.Printf("Failed to set immersion mode: %v", err)
	} else {
		fmt.Println("✓ Immersion mode enabled successfully")
	}
	*/
	
	fmt.Println()

	// Example 10: License Information
	fmt.Println("10. License Information:")
	fmt.Println("-----------------------")
	license, err := c.GetLicenseState()
	if err != nil {
		log.Printf("Failed to get license state: %v", err)
	} else {
		licenseInfo := models.ConvertLicense(license)
		if licenseInfo != nil {
			fmt.Printf("License State: %s\n", licenseInfo.State)
			fmt.Printf("License Type: %s\n", licenseInfo.Type)
			if licenseInfo.ValidUntil != nil {
				fmt.Printf("Valid Until: %s\n", licenseInfo.ValidUntil.Format("2006-01-02"))
			}
			if licenseInfo.DeviceCount > 0 {
				fmt.Printf("Device Count: %d\n", licenseInfo.DeviceCount)
			}
		}
	}
	fmt.Println()

	fmt.Println("=== Example Complete ===")
}

// Helper function to demonstrate batch operations
func batchOperations(minerIPs []string) {
	fmt.Println("=== Batch Operations Example ===\n")
	
	for _, ip := range minerIPs {
		fmt.Printf("Checking miner at %s...\n", ip)
		
		opts := client.ClientOptions{
			Host:     ip,
			Port:     50051,
			Username: "admin",
			Password: "admin",
			Timeout:  5 * time.Second,
		}
		
		c, err := client.NewClient(opts)
		if err != nil {
			fmt.Printf("  ✗ Failed to connect: %v\n", err)
			continue
		}
		
		stats, err := c.GetMinerStats()
		if err != nil {
			fmt.Printf("  ✗ Failed to get stats: %v\n", err)
		} else {
			minerStats := models.ConvertMinerStats(stats)
			fmt.Printf("  ✓ Hashrate: %.2f TH/s, Power: %.0f W\n",
				minerStats.HashRateAverage, minerStats.PowerConsumption)
		}
		
		c.Close()
	}
}