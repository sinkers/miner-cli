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
	"github.com/sinkers/miner-cli/internal/vnish/models"
)

// Test configuration flags for comprehensive testing
var (
	// Connection settings
	testHost     = flag.String("test.host", "10.45.3.1", "Vnish miner host IP address")
	testAPIKey   = flag.String("test.apikey", "", "Vnish API key for authentication")
	testTimeout  = flag.Duration("http.timeout", 10*time.Second, "HTTP request timeout")
	
	// Skip flags for potentially disruptive tests
	skipAuthTests     = flag.Bool("skip-auth", false, "Skip authentication tests")
	skipWriteOps      = flag.Bool("skip-write", true, "Skip write operations (pool changes, settings)")
	skipReboot        = flag.Bool("skip-reboot", true, "Skip reboot test")
	skipRestart       = flag.Bool("skip-restart", true, "Skip restart mining test")
	skipAutotuneOps   = flag.Bool("skip-autotune", true, "Skip autotune operations")
	skipAPIKeyOps     = flag.Bool("skip-apikey", true, "Skip API key management tests")
	skipNotesOps      = flag.Bool("skip-notes", false, "Skip notes operations")
	skipFindMiner     = flag.Bool("skip-find", true, "Skip find miner (LED blink) test")
	
	// Test behavior settings
	verboseOutput     = flag.Bool("verbose", true, "Enable verbose output")
	waitAfterReboot   = flag.Duration("wait-reboot", 2*time.Minute, "Wait time after reboot")
	waitAfterRestart  = flag.Duration("wait-restart", 30*time.Second, "Wait time after restart")
)

// TestResult tracks individual test results
type TestResult struct {
	Name     string
	Passed   bool
	Skipped  bool
	Error    error
	Duration time.Duration
}

// VnishIntegrationTestSuite manages the full test suite
type VnishIntegrationTestSuite struct {
	client  *Client
	results []TestResult
	t       *testing.T
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
	if *verboseOutput {
		fmt.Printf("  " + format + "\n", args...)
	}
}

func printSkipped(name string) {
	color.Yellow("⊘ SKIPPED: %s", name)
}

// TestVnishComprehensiveIntegration runs the complete integration test suite
func TestVnishComprehensiveIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	// Check for API key from environment if not provided
	apiKey := *testAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("VNISH_API_KEY")
	}

	suite := &VnishIntegrationTestSuite{
		t:       t,
		results: make([]TestResult, 0),
	}

	// Print test configuration
	color.Magenta("\n" + strings.Repeat("=", 60))
	color.Magenta("VNISH COMPREHENSIVE INTEGRATION TEST SUITE")
	color.Magenta(strings.Repeat("=", 60))
	fmt.Printf("\nTarget Miner: %s\n", *testHost)
	fmt.Printf("API Key: %s\n", maskAPIKey(apiKey))
	fmt.Printf("Timeout: %v\n", *testTimeout)
	fmt.Printf("\nSkip Flags:\n")
	fmt.Printf("  - Auth Tests: %v\n", *skipAuthTests)
	fmt.Printf("  - Write Operations: %v\n", *skipWriteOps)
	fmt.Printf("  - Reboot: %v\n", *skipReboot)
	fmt.Printf("  - Restart Mining: %v\n", *skipRestart)
	fmt.Printf("  - Autotune Operations: %v\n", *skipAutotuneOps)
	fmt.Printf("  - API Key Operations: %v\n", *skipAPIKeyOps)
	fmt.Printf("  - Notes Operations: %v\n", *skipNotesOps)
	fmt.Printf("  - Find Miner (LED): %v\n", *skipFindMiner)
	fmt.Printf("\nWait Times:\n")
	fmt.Printf("  - After Reboot: %v\n", *waitAfterReboot)
	fmt.Printf("  - After Restart: %v\n", *waitAfterRestart)
	color.Magenta(strings.Repeat("=", 60))

	// Create client
	suite.createClient(apiKey)

	// Run test categories
	suite.testConnectionAndAuth()
	suite.testSystemInformation()
	suite.testPerformanceMetrics()
	suite.testHardwareStatus()
	suite.testPoolOperations()
	suite.testAutotuneOperations()
	suite.testSettingsOperations()
	suite.testLogOperations()
	suite.testNotesOperations()
	suite.testAPIKeyOperations()
	suite.testMiningControl()
	suite.testSystemOperations()

	// Print summary
	suite.printSummary()
}

func maskAPIKey(key string) string {
	if key == "" {
		return "(not provided)"
	}
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

func (s *VnishIntegrationTestSuite) createClient(apiKey string) {
	printTestHeader("Client Creation")
	start := time.Now()

	var opts []Option
	if apiKey != "" {
		opts = append(opts, WithAPIKey(apiKey))
	}
	opts = append(opts, WithTimeout(*testTimeout))

	s.client = NewClient(*testHost, opts...)
	printSuccess("Client created for %s", *testHost)
	s.recordResult("Client Creation", true, false, nil, time.Since(start))
}

func (s *VnishIntegrationTestSuite) recordResult(name string, passed bool, skipped bool, err error, duration time.Duration) {
	result := TestResult{
		Name:     name,
		Passed:   passed,
		Skipped:  skipped,
		Error:    err,
		Duration: duration,
	}
	s.results = append(s.results, result)
}

func (s *VnishIntegrationTestSuite) testConnectionAndAuth() {
	if *skipAuthTests {
		printSkipped("Authentication Tests")
		s.recordResult("Authentication", false, true, nil, 0)
		return
	}

	printTestHeader("Connection and Authentication")

	s.runTest("Check Authentication", func() error {
		ctx := context.Background()
		auth, err := s.client.CheckAuth(ctx)
		if err != nil {
			return fmt.Errorf("authentication check failed: %v", err)
		}
		
		printInfo("Authenticated: %v", auth.Authenticated)
		printInfo("Method: %s", auth.Method)
		
		if !auth.Authenticated {
			printWarning("Not authenticated - some operations may be restricted")
		}
		
		return nil
	})
}

func (s *VnishIntegrationTestSuite) testSystemInformation() {
	printTestHeader("System Information")

	// Test GetInfo
	s.runTest("Get System Info", func() error {
		ctx := context.Background()
		info, err := s.client.GetInfo(ctx)
		if err != nil {
			return err
		}

		printInfo("Hostname: %s", info.Hostname)
		printInfo("Model: %s", info.Model)
		printInfo("Version: %s", info.Version)
		printInfo("Uptime: %v seconds", info.Uptime)
		printInfo("Load Average: %v", info.LoadAverage)
		
		if info.Hostname == "" {
			return fmt.Errorf("expected non-empty hostname")
		}
		if info.Model == "" {
			return fmt.Errorf("expected non-empty model")
		}
		
		return nil
	})

	// Test GetModel
	s.runTest("Get Model Details", func() error {
		ctx := context.Background()
		model, err := s.client.GetModel(ctx)
		if err != nil {
			return err
		}

		printInfo("Manufacturer: %s", model.Manufacturer)
		printInfo("Model: %s", model.Model)
		printInfo("Description: %s", model.Description)
		
		if model.Manufacturer == "" {
			return fmt.Errorf("expected non-empty manufacturer")
		}
		
		return nil
	})

	// Test GetLayout
	s.runTest("Get Hardware Layout", func() error {
		ctx := context.Background()
		layout, err := s.client.GetLayout(ctx)
		if err != nil {
			return err
		}

		printInfo("Chains: %d", layout.Chains)
		printInfo("Chips per Chain: %d", layout.Chips)
		printInfo("Total Chips: %d", layout.Chains*layout.Chips)
		
		if layout.Chains <= 0 {
			return fmt.Errorf("expected positive number of chains")
		}
		
		return nil
	})
}

func (s *VnishIntegrationTestSuite) testPerformanceMetrics() {
	printTestHeader("Performance Metrics")

	// Test GetStatus
	s.runTest("Get Full Status", func() error {
		ctx := context.Background()
		status, err := s.client.GetStatus(ctx)
		if err != nil {
			return err
		}

		printInfo("System:")
		printInfo("  Hostname: %s", status.System.Hostname)
		printInfo("  Version: %s", status.System.Version)
		printInfo("  Uptime: %v seconds", status.System.Uptime)
		
		printInfo("Performance:")
		printInfo("  HashRate: %.2f %s", status.Performance.HashRate, status.Performance.HashRateUnit)
		printInfo("  Power: %.2f W", status.Performance.PowerUsage)
		printInfo("  Efficiency: %.2f J/TH", status.Performance.Efficiency)
		printInfo("  Accepted: %d", status.Performance.Accepted)
		printInfo("  Rejected: %d", status.Performance.Rejected)
		
		return nil
	})

	// Test GetSummary
	s.runTest("Get Summary", func() error {
		ctx := context.Background()
		summary, err := s.client.GetSummary(ctx)
		if err != nil {
			return err
		}

		printInfo("Hash Rate: %.2f TH/s", summary.Performance.HashRate)
		printInfo("Power Usage: %.2f W", summary.Performance.PowerUsage)
		printInfo("Efficiency: %.2f J/TH", summary.Performance.Efficiency)
		printInfo("Uptime: %v seconds", summary.Uptime)
		printInfo("Shares Accepted: %d", summary.Performance.Accepted)
		printInfo("Shares Rejected: %d", summary.Performance.Rejected)
		printInfo("Active Pools: %d", len(summary.Pools))
		
		for i, pool := range summary.Pools {
			printInfo("Pool %d: %s (%s)", i+1, pool.URL, pool.Status)
		}
		
		return nil
	})

	// Test GetPerfSummary
	s.runTest("Get Performance Summary", func() error {
		ctx := context.Background()
		perf, err := s.client.GetPerfSummary(ctx)
		if err != nil {
			return err
		}

		printInfo("Current Performance:")
		printInfo("  Hash Rate: %.2f %s", perf.HashRate, perf.HashRateUnit)
		printInfo("  Power Usage: %.2f W", perf.PowerUsage)
		printInfo("  Efficiency: %.2f J/TH", perf.Efficiency)
		printInfo("  Hardware Errors: %d", perf.HardwareErrors)
		
		if perf.HashRate < 0 {
			return fmt.Errorf("expected non-negative hash rate")
		}
		
		return nil
	})

	// Test GetMetrics
	s.runTest("Get Historical Metrics", func() error {
		ctx := context.Background()
		metrics, err := s.client.GetMetrics(ctx)
		if err != nil {
			return err
		}

		printInfo("Metrics collected at: %v", metrics.Timestamp)
		printInfo("Hash rate data points: %d", len(metrics.HashRate))
		printInfo("Temperature data points: %d", len(metrics.Temperature))
		printInfo("Power data points: %d", len(metrics.Power))
		printInfo("Fan speed data points: %d", len(metrics.FanSpeed))
		
		if len(metrics.HashRate) > 0 {
			latest := metrics.HashRate[len(metrics.HashRate)-1]
			printInfo("Latest hash rate: %.2f at %v", latest.Value, latest.Time)
		}
		
		return nil
	})
}

func (s *VnishIntegrationTestSuite) testHardwareStatus() {
	printTestHeader("Hardware Status")

	// Test GetChains
	s.runTest("Get Chain Status", func() error {
		ctx := context.Background()
		chains, err := s.client.GetChains(ctx)
		if err != nil {
			return err
		}

		printInfo("Found %d chains", len(chains))
		
		for i, chain := range chains {
			printInfo("Chain %d:", i)
			printInfo("  Index: %d", chain.Index)
			printInfo("  Status: %s", chain.Status)
			printInfo("  HashRate: %.2f TH/s", chain.HashRate)
			printInfo("  Temperature: %.1f°C", chain.Temperature)
			printInfo("  Chip Count: %d", chain.ChipCount)
			printInfo("  Frequency: %d MHz", chain.Frequency)
			printInfo("  Voltage: %.2f V", chain.Voltage)
			
			if chain.ChipCount <= 0 {
				return fmt.Errorf("chain %d: expected positive chip count", i)
			}
		}
		
		return nil
	})

	// Note: Fan status is typically part of the status/summary endpoints for vnish
}

func (s *VnishIntegrationTestSuite) testPoolOperations() {
	printTestHeader("Pool Operations")

	// Always test reading pools
	s.runTest("Get Pool Settings", func() error {
		ctx := context.Background()
		settings, err := s.client.GetSettings(ctx)
		if err != nil {
			return err
		}

		printInfo("Found %d pools configured", len(settings.Pools))
		
		for i, pool := range settings.Pools {
			printInfo("Pool %d:", i+1)
			printInfo("  URL: %s", pool.URL)
			printInfo("  User: %s", pool.User)
			printInfo("  Pass: %s", maskPassword(pool.Password))
			
			if pool.URL == "" {
				printWarning("  Pool %d has empty URL", i+1)
			}
		}
		
		if len(settings.Pools) == 0 {
			return fmt.Errorf("expected at least one pool")
		}
		
		return nil
	})

	// Test pool updates only if write operations are allowed
	if !*skipWriteOps {
		s.runTest("Update Pool Settings", func() error {
			ctx := context.Background()
			
			// First get current settings
			current, err := s.client.GetSettings(ctx)
			if err != nil {
				return err
			}
			
			if len(current.Pools) == 0 {
				return fmt.Errorf("no pools to update")
			}
			
			// Create a test modification (just change the password slightly)
			testPool := current.Pools[0]
			originalPass := testPool.Password
			testPool.Password = originalPass + "-test"
			
			printInfo("Updating pool 1 password for testing...")
			
			// Update the pool
			settings := &models.Settings{
				Pools: []models.PoolConfig{testPool},
			}
			
			err = s.client.UpdateSettings(ctx, settings)
			if err != nil {
				return err
			}
			
			printInfo("Pool updated successfully")
			
			// Restore original settings
			time.Sleep(2 * time.Second)
			testPool.Password = originalPass
			settings.Pools[0] = testPool
			
			err = s.client.UpdateSettings(ctx, settings)
			if err != nil {
				printWarning("Failed to restore original pool: %v", err)
			} else {
				printInfo("Original pool settings restored")
			}
			
			return nil
		})
	} else {
		printSkipped("Update Pool Settings")
		s.recordResult("Update Pool Settings", false, true, nil, 0)
	}
}

func (s *VnishIntegrationTestSuite) testAutotuneOperations() {
	printTestHeader("Autotune Operations")

	// Always test reading autotune presets
	s.runTest("Get Autotune Presets", func() error {
		ctx := context.Background()
		presets, err := s.client.GetAutotunePresets(ctx)
		if err != nil {
			return err
		}

		printInfo("Found %d autotune presets", len(presets.Presets))
		
		for _, preset := range presets.Presets {
			printInfo("Preset: %s", preset.ID)
			printInfo("  Name: %s", preset.Name)
			printInfo("  Description: %s", preset.Description)
			printInfo("  HashRate: %.2f TH/s", preset.HashRate)
			printInfo("  Power: %.2f W", preset.Power)
			printInfo("  Efficiency: %.2f J/TH", preset.Efficiency)
		}
		
		return nil
	})

	// Test autotune mode changes only if allowed
	if !*skipAutotuneOps {
		s.runTest("Set Autotune Mode", func() error {
			ctx := context.Background()
			
			// Get current settings
			settings, err := s.client.GetSettings(ctx)
			if err != nil {
				return err
			}
			
			originalAutotune := settings.Advanced.AutoTune
			printInfo("Current autotune enabled: %v", originalAutotune)
			
			// Toggle autotune
			testAutotune := !originalAutotune
			
			printInfo("Setting autotune to: %v", testAutotune)
			
			newSettings := &models.Settings{
				Advanced: models.AdvancedSettings{
					AutoTune: testAutotune,
				},
			}
			
			err = s.client.UpdateSettings(ctx, newSettings)
			if err != nil {
				return err
			}
			
			printInfo("Autotune setting changed successfully")
			
			// Restore original mode
			time.Sleep(2 * time.Second)
			newSettings.Advanced.AutoTune = originalAutotune
			err = s.client.UpdateSettings(ctx, newSettings)
			if err != nil {
				printWarning("Failed to restore original autotune setting: %v", err)
			} else {
				printInfo("Original autotune setting restored")
			}
			
			return nil
		})
	} else {
		printSkipped("Set Autotune Mode")
		s.recordResult("Set Autotune Mode", false, true, nil, 0)
	}
}

func (s *VnishIntegrationTestSuite) testSettingsOperations() {
	printTestHeader("Settings Operations")

	s.runTest("Get All Settings", func() error {
		ctx := context.Background()
		settings, err := s.client.GetSettings(ctx)
		if err != nil {
			return err
		}

		printInfo("Settings retrieved:")
		printInfo("  Autotune Enabled: %v", settings.Advanced.AutoTune)
		printInfo("  Auto Restart: %v", settings.Advanced.AutoRestart)
		printInfo("  Low Power Mode: %v", settings.Advanced.LowPowerMode)
		printInfo("  Immersion Mode: %v", settings.Advanced.ImmersionMode)
		printInfo("  Pools: %d configured", len(settings.Pools))
		
		printInfo("  Network DHCP: %v", settings.Network.DHCP)
		if !settings.Network.DHCP {
			printInfo("  IP: %s", settings.Network.IPAddress)
			printInfo("  Netmask: %s", settings.Network.Netmask)
			printInfo("  Gateway: %s", settings.Network.Gateway)
		}
		
		return nil
	})
}

func (s *VnishIntegrationTestSuite) testLogOperations() {
	printTestHeader("Log Operations")

	logTypes := []string{"system", "miner", "error"}
	
	for _, logType := range logTypes {
		testName := fmt.Sprintf("Get %s Logs", strings.Title(logType))
		s.runTest(testName, func() error {
			ctx := context.Background()
			logs, err := s.client.GetLogs(ctx, logType)
			if err != nil {
				// Some log types might not be available
				printWarning("%s logs not available: %v", logType, err)
				return nil
			}
			
			printInfo("Retrieved %d %s log entries", len(logs), logType)
			
			// Show a few recent entries if available
			showCount := 3
			if len(logs) < showCount {
				showCount = len(logs)
			}
			
			for i := 0; i < showCount; i++ {
				log := logs[i]
				printInfo("  [%v] %s: %s", log.Timestamp, log.Level, log.Message)
			}
			
			return nil
		})
	}
}

func (s *VnishIntegrationTestSuite) testNotesOperations() {
	if *skipNotesOps {
		printSkipped("Notes Operations")
		s.recordResult("Notes Operations", false, true, nil, 0)
		return
	}

	printTestHeader("Notes Operations")

	var testNoteID string

	// List existing notes
	s.runTest("List Notes", func() error {
		ctx := context.Background()
		notes, err := s.client.GetNotes(ctx)
		if err != nil {
			return err
		}
		
		printInfo("Found %d existing notes", len(notes))
		for i, note := range notes {
			if i < 3 { // Show first 3 notes
				printInfo("  Note %s: %s... (created: %v)", 
					note.ID, truncateString(note.Content, 50), note.CreatedAt)
			}
		}
		
		return nil
	})

	// Create a test note
	s.runTest("Create Note", func() error {
		ctx := context.Background()
		content := fmt.Sprintf("Integration test note - %s", time.Now().Format(time.RFC3339))
		
		note, err := s.client.CreateNote(ctx, content)
		if err != nil {
			return err
		}
		
		testNoteID = note.ID
		printInfo("Created note with ID: %s", note.ID)
		printInfo("Content: %s", note.Content)
		
		return nil
	})

	// Get specific note
	if testNoteID != "" {
		s.runTest("Get Specific Note", func() error {
			ctx := context.Background()
			note, err := s.client.GetNote(ctx, testNoteID)
			if err != nil {
				return err
			}
			
			printInfo("Retrieved note %s", note.ID)
			printInfo("Content: %s", note.Content)
			printInfo("Created: %v", note.CreatedAt)
			
			return nil
		})

		// Update note
		s.runTest("Update Note", func() error {
			ctx := context.Background()
			newContent := fmt.Sprintf("Updated: Integration test - %s", time.Now().Format(time.RFC3339))
			
			note, err := s.client.UpdateNote(ctx, testNoteID, newContent)
			if err != nil {
				return err
			}
			
			printInfo("Updated note %s", note.ID)
			printInfo("New content: %s", note.Content)
			
			return nil
		})

		// Delete note
		s.runTest("Delete Note", func() error {
			ctx := context.Background()
			err := s.client.DeleteNote(ctx, testNoteID)
			if err != nil {
				return err
			}
			
			printInfo("Deleted test note %s", testNoteID)
			return nil
		})
	}
}

func (s *VnishIntegrationTestSuite) testAPIKeyOperations() {
	if *skipAPIKeyOps {
		printSkipped("API Key Operations")
		s.recordResult("API Key Operations", false, true, nil, 0)
		return
	}

	printTestHeader("API Key Operations")

	var testKeyID string

	// List existing keys
	s.runTest("List API Keys", func() error {
		ctx := context.Background()
		keys, err := s.client.GetAPIKeys(ctx)
		if err != nil {
			printWarning("GetAPIKeys failed (might need permissions): %v", err)
			return nil // Don't fail test if no permission
		}
		
		printInfo("Found %d API keys", len(keys))
		for _, key := range keys {
			printInfo("  Key: %s (ID: %s, Created: %v)", key.Name, key.ID, key.CreatedAt)
		}
		
		return nil
	})

	// Create a test key
	s.runTest("Create API Key", func() error {
		ctx := context.Background()
		keyName := fmt.Sprintf("test-key-%d", time.Now().Unix())
		
		result, err := s.client.AddAPIKey(ctx, keyName)
		if err != nil {
			printWarning("AddAPIKey failed (might need permissions): %v", err)
			return nil // Don't fail test if no permission
		}
		
		if !result.Status.Success {
			printWarning("Failed to create API key: %s", result.Status.Message)
			return nil
		}
		
		// Find the created key
		keys, err := s.client.GetAPIKeys(ctx)
		if err == nil {
			for _, key := range keys {
				if key.Name == keyName {
					testKeyID = key.ID
					printInfo("Created API key: %s (ID: %s)", keyName, key.ID)
					if result.Key != "" {
						printInfo("Key value: %s", result.Key) // Only available on creation
					}
					break
				}
			}
		}
		
		return nil
	})

	// Delete test key if created
	if testKeyID != "" {
		s.runTest("Delete API Key", func() error {
			ctx := context.Background()
			err := s.client.DeleteAPIKey(ctx, testKeyID)
			if err != nil {
				printWarning("Failed to delete test key: %v", err)
				return nil
			}
			
			printInfo("Deleted test API key %s", testKeyID)
			return nil
		})
	}
}

func (s *VnishIntegrationTestSuite) testMiningControl() {
	printTestHeader("Mining Control Operations")

	// Test restart mining
	if !*skipRestart {
		s.runTest("Restart Mining", func() error {
			ctx := context.Background()
			
			printWarning("Restarting mining operation...")
			err := s.client.RestartMining(ctx)
			if err != nil {
				return err
			}
			
			printInfo("Mining restart initiated, waiting %v for stabilization...", *waitAfterRestart)
			time.Sleep(*waitAfterRestart)
			
			// Verify mining is running
			status, err := s.client.GetStatus(ctx)
			if err != nil {
				return fmt.Errorf("failed to verify after restart: %v", err)
			}
			
			printInfo("Mining status after restart: HashRate=%.2f %s", status.Performance.HashRate, status.Performance.HashRateUnit)
			return nil
		})
	} else {
		printSkipped("Restart Mining")
		s.recordResult("Restart Mining", false, true, nil, 0)
	}

	// Test find miner (LED blink)
	if !*skipFindMiner {
		s.runTest("Find Miner (LED Blink)", func() error {
			ctx := context.Background()
			
			printInfo("Activating miner LEDs...")
			result, err := s.client.FindMiner(ctx, true)
			if err != nil {
				return err
			}
			
			if !result.Success {
				return fmt.Errorf("find miner failed: %s", result.Message)
			}
			
			printInfo("Miner LEDs activated for identification")
			printInfo("Waiting 5 seconds...")
			time.Sleep(5 * time.Second)
			
			// Turn off LEDs
			printInfo("Deactivating miner LEDs...")
			result, err = s.client.FindMiner(ctx, false)
			if err != nil {
				printWarning("Failed to turn off LEDs: %v", err)
			} else if result.Success {
				printInfo("Miner LEDs deactivated")
			} else {
				printWarning("Failed to turn off LEDs: %s", result.Message)
			}
			
			return nil
		})
	} else {
		printSkipped("Find Miner (LED Blink)")
		s.recordResult("Find Miner", false, true, nil, 0)
	}
}

func (s *VnishIntegrationTestSuite) testSystemOperations() {
	printTestHeader("System Operations")

	// Reboot test (most disruptive, do last)
	if !*skipReboot {
		s.runTest("System Reboot", func() error {
			ctx := context.Background()
			
			printWarning("REBOOTING SYSTEM - This will take several minutes...")
			err := s.client.Reboot(ctx)
			if err != nil {
				return err
			}
			
			printInfo("Reboot command sent, waiting %v before attempting reconnection...", *waitAfterReboot)
			time.Sleep(*waitAfterReboot)
			
			// Try to reconnect with polling
			return s.pollForReconnection(5*time.Minute, 10*time.Second)
		})
	} else {
		printSkipped("System Reboot")
		s.recordResult("System Reboot", false, true, nil, 0)
	}
}

func (s *VnishIntegrationTestSuite) pollForReconnection(timeout, interval time.Duration) error {
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
			
			// Try to get miner info
			info, err := s.client.GetInfo(context.Background())
			if err != nil {
				continue // Keep trying
			}
			
			// Success!
			printSuccess("Miner is back online!")
			printInfo("Hostname: %s, Uptime: %v seconds", info.Hostname, info.Uptime)
			return nil
		}
	}
}

func (s *VnishIntegrationTestSuite) runTest(name string, testFunc func() error) {
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

func (s *VnishIntegrationTestSuite) printSummary() {
	color.Magenta("\n" + strings.Repeat("=", 60))
	color.Magenta("TEST SUMMARY")
	color.Magenta(strings.Repeat("=", 60))
	
	var passed, failed, skipped int
	var totalDuration time.Duration
	
	for _, result := range s.results {
		if result.Skipped {
			skipped++
			color.Yellow("⊘ %-35s SKIPPED", result.Name)
		} else if result.Passed {
			passed++
			totalDuration += result.Duration
			color.Green("✓ %-35s PASSED  (%.2fs)", result.Name, result.Duration.Seconds())
		} else {
			failed++
			totalDuration += result.Duration
			color.Red("✗ %-35s FAILED  (%.2fs)", result.Name, result.Duration.Seconds())
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

// Helper functions
func maskPassword(pass string) string {
	if pass == "" {
		return "(empty)"
	}
	if len(pass) <= 4 {
		return "***"
	}
	return pass[:2] + "***"
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}