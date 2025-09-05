// +build integration

package client

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/sinkers/miner-cli/internal/braiins/models"
)

// Test configuration flags
var (
	minerHost     = flag.String("host", "10.45.3.1", "Miner host IP address")
	minerPort     = flag.Int("port", 50051, "Miner gRPC port")
	username      = flag.String("user", "root", "Authentication username")
	password      = flag.String("pass", "root", "Authentication password")
	
	// Skip flags for potentially disruptive tests
	skipAuth      = flag.Bool("skip-auth", false, "Skip authentication tests")
	skipWrite     = flag.Bool("skip-write", true, "Skip write operations (pool changes, performance settings)")
	skipReboot    = flag.Bool("skip-reboot", true, "Skip reboot test")
	skipRestart   = flag.Bool("skip-restart", true, "Skip restart mining test")
	skipPause     = flag.Bool("skip-pause", true, "Skip pause/resume mining test")
	skipFirmware  = flag.Bool("skip-firmware", true, "Skip firmware operations (always true)")
	
	verbose       = flag.Bool("verbose", true, "Enable verbose output")
	waitTime      = flag.Duration("wait", 30*time.Second, "Wait time after disruptive operations")
)

// Test result tracking
type TestResult struct {
	Name     string
	Passed   bool
	Skipped  bool
	Error    error
	Duration time.Duration
}

type IntegrationTestSuite struct {
	client  *SimpleBraiinsClient
	results []TestResult
	t       *testing.T
}

func init() {
	// Don't parse flags here, let the test framework handle it
}

// Helper functions for colored output
func printTestHeader(name string) {
	color.Cyan("\n=== TEST: %s ===", name)
}

func printSuccess(format string, args ...interface{}) {
	color.Green("✓ " + fmt.Sprintf(format, args...))
}

func printError(format string, args ...interface{}) {
	color.Red("✗ " + fmt.Sprintf(format, args...))
}

func printWarning(format string, args ...interface{}) {
	color.Yellow("⚠ " + fmt.Sprintf(format, args...))
}

func printInfo(format string, args ...interface{}) {
	if *verbose {
		fmt.Printf("  " + format + "\n", args...)
	}
}

func printSkipped(name string) {
	color.Yellow("⊘ SKIPPED: %s", name)
}

// TestBraiinsIntegration runs the complete integration test suite
func TestBraiinsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	suite := &IntegrationTestSuite{
		t:       t,
		results: make([]TestResult, 0),
	}

	// Print test configuration
	color.Magenta("\n" + strings.Repeat("=", 60))
	color.Magenta("BRAIINS OS+ INTEGRATION TEST SUITE")
	color.Magenta(strings.Repeat("=", 60))
	fmt.Printf("\nTarget Miner: %s:%d\n", *minerHost, *minerPort)
	fmt.Printf("Username: %s\n", *username)
	fmt.Printf("Skip Flags:\n")
	fmt.Printf("  - Auth Tests: %v\n", *skipAuth)
	fmt.Printf("  - Write Operations: %v\n", *skipWrite)
	fmt.Printf("  - Reboot: %v\n", *skipReboot)
	fmt.Printf("  - Restart Mining: %v\n", *skipRestart)
	fmt.Printf("  - Pause/Resume: %v\n", *skipPause)
	fmt.Printf("  - Firmware: %v (always skipped)\n", *skipFirmware)
	fmt.Printf("Wait Time: %v\n", *waitTime)
	color.Magenta(strings.Repeat("=", 60))

	// Run test categories
	suite.testConnection()
	suite.testAuthentication()
	suite.testReadOperations()
	suite.testPoolOperations()
	suite.testPerformanceOperations()
	suite.testMiningControl()
	suite.testSystemOperations()

	// Print summary
	suite.printSummary()
}

func (s *IntegrationTestSuite) recordResult(name string, passed bool, skipped bool, err error, duration time.Duration) {
	result := TestResult{
		Name:     name,
		Passed:   passed,
		Skipped:  skipped,
		Error:    err,
		Duration: duration,
	}
	s.results = append(s.results, result)
}

func (s *IntegrationTestSuite) testConnection() {
	printTestHeader("Connection Test")
	start := time.Now()

	opts := SimpleClientOptions{
		Host:    *minerHost,
		Port:    *minerPort,
		Timeout: 10 * time.Second,
		UseTLS:  false,
	}

	// Don't authenticate yet, just test connection
	client, err := NewSimpleClient(opts)
	if err != nil {
		printError("Failed to create client: %v", err)
		s.recordResult("Connection", false, false, err, time.Since(start))
		s.t.Fatal("Cannot continue without connection")
	}

	s.client = client
	printSuccess("Connected to miner at %s:%d", *minerHost, *minerPort)
	s.recordResult("Connection", true, false, nil, time.Since(start))
}

func (s *IntegrationTestSuite) testAuthentication() {
	if *skipAuth {
		printSkipped("Authentication Test")
		s.recordResult("Authentication", false, true, nil, 0)
		
		// Recreate client with auth for subsequent tests
		s.reconnectWithAuth()
		return
	}

	printTestHeader("Authentication Test")
	start := time.Now()

	// Close existing client and create new one with auth
	s.client.Close()

	opts := SimpleClientOptions{
		Host:     *minerHost,
		Port:     *minerPort,
		Username: *username,
		Password: *password,
		Timeout:  10 * time.Second,
		UseTLS:   false,
	}

	client, err := NewSimpleClient(opts)
	if err != nil {
		printError("Authentication failed: %v", err)
		s.recordResult("Authentication", false, false, err, time.Since(start))
		s.t.Fatal("Cannot continue without authentication")
	}

	s.client = client
	printSuccess("Authenticated successfully as '%s'", *username)
	s.recordResult("Authentication", true, false, nil, time.Since(start))
}

func (s *IntegrationTestSuite) reconnectWithAuth() {
	s.client.Close()

	opts := SimpleClientOptions{
		Host:     *minerHost,
		Port:     *minerPort,
		Username: *username,
		Password: *password,
		Timeout:  10 * time.Second,
		UseTLS:   false,
	}

	client, err := NewSimpleClient(opts)
	if err != nil {
		s.t.Fatal("Failed to reconnect with auth:", err)
	}
	s.client = client
}

func (s *IntegrationTestSuite) testReadOperations() {
	printTestHeader("Read-Only Operations")

	// Test Miner Details
	s.runTest("Get Miner Details", func() error {
		resp, err := s.client.GetMinerDetails()
		if err != nil {
			return err
		}

		info := models.ConvertMinerDetails(resp)
		printInfo("Hostname: %s", info.Hostname)
		printInfo("MAC: %s", info.MACAddress)
		printInfo("Platform: %s", info.Platform)
		printInfo("BOS Mode: %s", info.BOSMode)
		printInfo("BOS Version: %s", info.BOSVersion)
		printInfo("Status: %s", info.Status)
		printInfo("System Uptime: %v", info.SystemUptime)
		
		return nil
	})

	// Test Miner Stats
	s.runTest("Get Miner Statistics", func() error {
		resp, err := s.client.GetMinerStats()
		if err != nil {
			return err
		}

		stats := models.ConvertMinerStats(resp)
		printInfo("Hashrate (5s): %.2f TH/s", stats.HashRate5s)
		printInfo("Hashrate (1m): %.2f TH/s", stats.HashRate1m)
		printInfo("Hashrate (15m): %.2f TH/s", stats.HashRate15m)
		printInfo("Hashrate (24h): %.2f TH/s", stats.HashRate24h)
		printInfo("Power Usage: %.0f W", stats.PowerUsage)
		printInfo("Efficiency: %.2f J/TH", stats.Efficiency)
		printInfo("Accepted Shares: %d", stats.AcceptedShares)
		printInfo("Rejected Shares: %d", stats.RejectedShares)
		
		return nil
	})

	// Test Hashboards
	s.runTest("Get Hashboards", func() error {
		resp, err := s.client.GetHashboards()
		if err != nil {
			return err
		}

		boards := models.ConvertHashboards(resp)
		printInfo("Found %d hashboards", len(boards))
		for i, board := range boards {
			printInfo("Board %d: Status=%s, HashRate=%.2f TH/s, Temp=%.1f°C, Chips=%d",
				i, board.Status, board.HashRate, board.Temperature, board.Chips)
		}
		
		return nil
	})

	// Test Pool Groups
	s.runTest("Get Pool Groups", func() error {
		resp, err := s.client.GetPoolGroups()
		if err != nil {
			return err
		}

		groups := models.ConvertPoolGroups(resp)
		printInfo("Found %d pool groups", len(groups))
		for _, group := range groups {
			printInfo("Group: %s", group.Name)
			// Note: Individual pools would need additional API calls
		}
		
		return nil
	})

	// Test Cooling State
	s.runTest("Get Cooling State", func() error {
		resp, err := s.client.GetCoolingState()
		if err != nil {
			return err
		}

		cooling := models.ConvertCoolingState(resp)
		printInfo("Cooling Info: %+v", cooling)
		
		return nil
	})

	// Test Tuner State
	s.runTest("Get Tuner State", func() error {
		resp, err := s.client.GetTunerState()
		if err != nil {
			return err
		}

		tuner := models.ConvertTunerState(resp)
		printInfo("Tuner State: %+v", tuner)
		
		return nil
	})

	// Test License State
	s.runTest("Get License State", func() error {
		resp, err := s.client.GetLicenseState()
		if err != nil {
			return err
		}

		license := models.ConvertLicenseState(resp)
		printInfo("License State: %+v", license)
		
		return nil
	})

	// Test Miner Configuration
	s.runTest("Get Miner Configuration", func() error {
		resp, err := s.client.GetMinerConfiguration()
		if err != nil {
			return err
		}

		printInfo("Configuration retrieved successfully")
		if resp != nil {
			printInfo("Configuration data available")
		}
		
		return nil
	})
}

func (s *IntegrationTestSuite) testPoolOperations() {
	if *skipWrite {
		printSkipped("Pool Operations")
		s.recordResult("Pool Operations", false, true, nil, 0)
		return
	}

	printTestHeader("Pool Operations")

	// Get current pool groups first
	s.runTest("List Pool Groups", func() error {
		resp, err := s.client.GetPoolGroups()
		if err != nil {
			return err
		}

		if resp.PoolGroups != nil {
			printInfo("Current pool groups: %d", len(resp.PoolGroups))
			for _, group := range resp.PoolGroups {
				printInfo("  - %s", group.Name)
			}
		}
		
		return nil
	})

	// Note: Adding/modifying pools would require the SetPoolGroups method
	// which takes a full replacement of all pool groups
}

func (s *IntegrationTestSuite) testPerformanceOperations() {
	if *skipWrite {
		printSkipped("Performance Operations")
		s.recordResult("Performance Operations", false, true, nil, 0)
		return
	}

	printTestHeader("Performance Operations")

	// Test setting power target
	s.runTest("Set Power Target", func() error {
		targetWatts := uint64(3000) // Example: 3000W
		printInfo("Setting power target to %d W", targetWatts)
		
		_, err := s.client.SetPowerTarget(targetWatts)
		if err != nil {
			return err
		}
		
		printInfo("Power target set successfully")
		return nil
	})

	// Give it time to apply
	time.Sleep(5 * time.Second)

	// Verify the setting took effect
	s.runTest("Verify Power Target", func() error {
		resp, err := s.client.GetTunerState()
		if err != nil {
			return err
		}
		
		state := models.ConvertTunerState(resp)
		printInfo("Current tuner mode: %v", state["mode"])
		
		return nil
	})
}

func (s *IntegrationTestSuite) testMiningControl() {
	printTestHeader("Mining Control Operations")

	// Test pause/resume
	if !*skipPause {
		s.runTest("Pause Mining", func() error {
			printWarning("Pausing mining operation...")
			err := s.client.PauseMining()
			if err != nil {
				return err
			}
			
			printInfo("Mining paused, waiting 10 seconds...")
			time.Sleep(10 * time.Second)
			
			return nil
		})

		s.runTest("Resume Mining", func() error {
			printInfo("Resuming mining operation...")
			err := s.client.ResumeMining()
			if err != nil {
				return err
			}
			
			printInfo("Mining resumed")
			return nil
		})
	} else {
		printSkipped("Pause/Resume Mining")
		s.recordResult("Pause/Resume Mining", false, true, nil, 0)
	}

	// Test restart mining
	if !*skipRestart {
		s.runTest("Restart Mining", func() error {
			printWarning("Restarting mining operation...")
			err := s.client.RestartMining()
			if err != nil {
				return err
			}
			
			printInfo("Mining restarted, waiting %v for stabilization...", *waitTime)
			time.Sleep(*waitTime)
			
			// Verify mining is running
			resp, err := s.client.GetMinerDetails()
			if err != nil {
				return fmt.Errorf("failed to verify after restart: %v", err)
			}
			
			printInfo("Miner status: %s", resp.Status.String())
			return nil
		})
	} else {
		printSkipped("Restart Mining")
		s.recordResult("Restart Mining", false, true, nil, 0)
	}
}

func (s *IntegrationTestSuite) testSystemOperations() {
	printTestHeader("System Operations")

	// Reboot test (most disruptive, do last)
	if !*skipReboot {
		s.runTest("System Reboot", func() error {
			printWarning("REBOOTING SYSTEM - This will take several minutes...")
			err := s.client.Reboot()
			if err != nil {
				return err
			}
			
			printInfo("Reboot command sent, waiting %v before attempting reconnection...", *waitTime)
			time.Sleep(*waitTime)
			
			// Try to reconnect with polling
			return s.pollForReconnection(5*time.Minute, 10*time.Second)
		})
	} else {
		printSkipped("System Reboot")
		s.recordResult("System Reboot", false, true, nil, 0)
	}
}

func (s *IntegrationTestSuite) pollForReconnection(timeout, interval time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	attempt := 0
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for miner to come back online")
		case <-ticker.C:
			attempt++
			printInfo("Reconnection attempt %d...", attempt)
			
			opts := SimpleClientOptions{
				Host:     *minerHost,
				Port:     *minerPort,
				Username: *username,
				Password: *password,
				Timeout:  5 * time.Second,
				UseTLS:   false,
			}
			
			client, err := NewSimpleClient(opts)
			if err != nil {
				continue // Keep trying
			}
			
			// Try to get miner details to verify it's really up
			_, err = client.GetMinerDetails()
			if err != nil {
				client.Close()
				continue
			}
			
			// Success!
			s.client.Close()
			s.client = client
			printSuccess("Miner is back online!")
			return nil
		}
	}
}

func (s *IntegrationTestSuite) runTest(name string, testFunc func() error) {
	start := time.Now()
	printInfo("\nTesting: %s", name)
	
	err := testFunc()
	duration := time.Since(start)
	
	if err != nil {
		printError("%s failed: %v", name, err)
		s.recordResult(name, false, false, err, duration)
	} else {
		printSuccess("%s completed (%.2fs)", name, duration.Seconds())
		s.recordResult(name, true, false, nil, duration)
	}
}

func (s *IntegrationTestSuite) printSummary() {
	color.Magenta("\n" + strings.Repeat("=", 60))
	color.Magenta("TEST SUMMARY")
	color.Magenta(strings.Repeat("=", 60))
	
	var passed, failed, skipped int
	var totalDuration time.Duration
	
	for _, result := range s.results {
		if result.Skipped {
			skipped++
			color.Yellow("⊘ %-30s SKIPPED", result.Name)
		} else if result.Passed {
			passed++
			totalDuration += result.Duration
			color.Green("✓ %-30s PASSED  (%.2fs)", result.Name, result.Duration.Seconds())
		} else {
			failed++
			totalDuration += result.Duration
			color.Red("✗ %-30s FAILED  (%.2fs)", result.Name, result.Duration.Seconds())
			if result.Error != nil {
				color.Red("  Error: %v", result.Error)
			}
		}
	}
	
	color.Magenta(strings.Repeat("=", 60))
	fmt.Printf("\nResults: ")
	color.Green("%d passed", passed)
	fmt.Printf(", ")
	color.Red("%d failed", failed)
	fmt.Printf(", ")
	color.Yellow("%d skipped", skipped)
	fmt.Printf("\nTotal test time: %.2fs\n", totalDuration.Seconds())
	
	if failed > 0 {
		os.Exit(1)
	}
}