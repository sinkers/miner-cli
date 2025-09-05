// +build integration

package client

import (
	"context"
	"flag"
	"os"
	"testing"
	"time"
)

var (
	vnishHost   = flag.String("vnish.host", "", "vnish API host for integration tests")
	vnishAPIKey = flag.String("vnish.apikey", "", "vnish API key for integration tests")
)

func TestIntegration(t *testing.T) {
	if *vnishHost == "" {
		t.Skip("Integration test skipped: -vnish.host not provided")
	}

	// Allow environment variables as well
	host := *vnishHost
	if host == "" {
		host = os.Getenv("VNISH_HOST")
	}
	if host == "" {
		t.Skip("Integration test skipped: VNISH_HOST environment variable not set")
	}

	apiKey := *vnishAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("VNISH_API_KEY")
	}

	ctx := context.Background()
	client := NewClient(host, WithAPIKey(apiKey), WithTimeout(10*time.Second))

	t.Run("AuthCheck", func(t *testing.T) {
		auth, err := client.CheckAuth(ctx)
		if err != nil {
			t.Fatalf("CheckAuth failed: %v", err)
		}
		t.Logf("Auth status: %+v", auth)
	})

	t.Run("GetInfo", func(t *testing.T) {
		info, err := client.GetInfo(ctx)
		if err != nil {
			t.Fatalf("GetInfo failed: %v", err)
		}
		t.Logf("System info: %+v", info)
		
		if info.Hostname == "" {
			t.Error("expected non-empty hostname")
		}
		if info.Model == "" {
			t.Error("expected non-empty model")
		}
		if info.Version == "" {
			t.Error("expected non-empty version")
		}
	})

	t.Run("GetModel", func(t *testing.T) {
		model, err := client.GetModel(ctx)
		if err != nil {
			t.Fatalf("GetModel failed: %v", err)
		}
		t.Logf("Model info: %+v", model)
		
		if model.Manufacturer == "" {
			t.Error("expected non-empty manufacturer")
		}
		if model.Model == "" {
			t.Error("expected non-empty model")
		}
	})

	t.Run("GetStatus", func(t *testing.T) {
		status, err := client.GetStatus(ctx)
		if err != nil {
			t.Fatalf("GetStatus failed: %v", err)
		}
		t.Logf("Status: %+v", status)
		
		// Verify we got meaningful data
		if status.System.Hostname == "" {
			t.Error("expected non-empty hostname in status")
		}
	})

	t.Run("GetSummary", func(t *testing.T) {
		summary, err := client.GetSummary(ctx)
		if err != nil {
			t.Fatalf("GetSummary failed: %v", err)
		}
		t.Logf("Summary: %+v", summary)
		
		// Check for reasonable values
		if summary.Performance.HashRate < 0 {
			t.Error("expected non-negative hash rate")
		}
		if len(summary.Pools) == 0 {
			t.Error("expected at least one pool")
		}
	})

	t.Run("GetPerfSummary", func(t *testing.T) {
		perf, err := client.GetPerfSummary(ctx)
		if err != nil {
			t.Fatalf("GetPerfSummary failed: %v", err)
		}
		t.Logf("Performance: %+v", perf)
		
		if perf.HashRate < 0 {
			t.Error("expected non-negative hash rate")
		}
		if perf.PowerUsage < 0 {
			t.Error("expected non-negative power usage")
		}
	})

	t.Run("GetChains", func(t *testing.T) {
		chains, err := client.GetChains(ctx)
		if err != nil {
			t.Fatalf("GetChains failed: %v", err)
		}
		t.Logf("Found %d chains", len(chains))
		
		for i, chain := range chains {
			t.Logf("Chain %d: %+v", i, chain)
			if chain.ChipCount <= 0 {
				t.Errorf("chain %d: expected positive chip count", i)
			}
		}
	})

	t.Run("GetLayout", func(t *testing.T) {
		layout, err := client.GetLayout(ctx)
		if err != nil {
			t.Fatalf("GetLayout failed: %v", err)
		}
		t.Logf("Layout: %+v", layout)
		
		if layout.Chains <= 0 {
			t.Error("expected positive number of chains")
		}
		if layout.Chips <= 0 {
			t.Error("expected positive number of chips")
		}
	})

	t.Run("GetSettings", func(t *testing.T) {
		settings, err := client.GetSettings(ctx)
		if err != nil {
			t.Fatalf("GetSettings failed: %v", err)
		}
		t.Logf("Settings: %+v", settings)
		
		if len(settings.Pools) == 0 {
			t.Error("expected at least one pool in settings")
		}
	})

	t.Run("GetMetrics", func(t *testing.T) {
		metrics, err := client.GetMetrics(ctx)
		if err != nil {
			t.Fatalf("GetMetrics failed: %v", err)
		}
		t.Logf("Metrics timestamp: %v", metrics.Timestamp)
		t.Logf("Hash rate points: %d", len(metrics.HashRate))
		t.Logf("Temperature points: %d", len(metrics.Temperature))
	})

	t.Run("GetAutotunePresets", func(t *testing.T) {
		presets, err := client.GetAutotunePresets(ctx)
		if err != nil {
			t.Fatalf("GetAutotunePresets failed: %v", err)
		}
		t.Logf("Found %d autotune presets", len(presets.Presets))
		
		for _, preset := range presets.Presets {
			t.Logf("Preset: %s - %s (%.2f TH/s, %.2f W)", 
				preset.ID, preset.Name, preset.HashRate, preset.Power)
		}
	})

	// Test API key operations (be careful with these in production)
	t.Run("APIKeyOperations", func(t *testing.T) {
		if apiKey == "" {
			t.Skip("Skipping API key operations: no API key provided")
		}

		// List existing keys
		keys, err := client.GetAPIKeys(ctx)
		if err != nil {
			t.Logf("GetAPIKeys failed (might need permissions): %v", err)
			return
		}
		t.Logf("Found %d API keys", len(keys))
		
		// Try to add a test key
		testKeyName := "test-key-" + time.Now().Format("20060102-150405")
		result, err := client.AddAPIKey(ctx, testKeyName)
		if err != nil {
			t.Logf("AddAPIKey failed (might need permissions): %v", err)
			return
		}
		
		if result.Status.Success {
			t.Logf("Created test API key: %s", testKeyName)
			
			// Clean up - find and delete the test key
			keys, err = client.GetAPIKeys(ctx)
			if err == nil {
				for _, key := range keys {
					if key.Name == testKeyName {
						err = client.DeleteAPIKey(ctx, key.ID)
						if err != nil {
							t.Logf("Failed to delete test key: %v", err)
						} else {
							t.Logf("Deleted test key: %s", testKeyName)
						}
						break
					}
				}
			}
		}
	})

	// Test log operations
	t.Run("LogOperations", func(t *testing.T) {
		logTypes := []string{"system", "miner", "error"}
		
		for _, logType := range logTypes {
			t.Run(logType, func(t *testing.T) {
				logs, err := client.GetLogs(ctx, logType)
				if err != nil {
					t.Logf("GetLogs(%s) failed: %v", logType, err)
					return
				}
				t.Logf("Retrieved %d %s log entries", len(logs), logType)
				
				// Show a sample if available
				if len(logs) > 0 {
					t.Logf("Latest %s log: %+v", logType, logs[0])
				}
			})
		}
	})

	// Test notes operations
	t.Run("NotesOperations", func(t *testing.T) {
		// List existing notes
		notes, err := client.GetNotes(ctx)
		if err != nil {
			t.Logf("GetNotes failed: %v", err)
			return
		}
		t.Logf("Found %d notes", len(notes))
		
		// Create a test note
		testContent := "Integration test note - " + time.Now().Format(time.RFC3339)
		note, err := client.CreateNote(ctx, testContent)
		if err != nil {
			t.Logf("CreateNote failed: %v", err)
			return
		}
		t.Logf("Created note: %s", note.ID)
		
		// Get the specific note
		retrieved, err := client.GetNote(ctx, note.ID)
		if err != nil {
			t.Errorf("GetNote failed: %v", err)
		} else if retrieved.Content != testContent {
			t.Errorf("Note content mismatch: expected %s, got %s", testContent, retrieved.Content)
		}
		
		// Update the note
		updatedContent := "Updated: " + testContent
		updated, err := client.UpdateNote(ctx, note.ID, updatedContent)
		if err != nil {
			t.Logf("UpdateNote failed: %v", err)
		} else {
			t.Logf("Updated note content: %s", updated.Content)
		}
		
		// Delete the test note
		err = client.DeleteNote(ctx, note.ID)
		if err != nil {
			t.Logf("DeleteNote failed: %v", err)
		} else {
			t.Logf("Deleted test note")
		}
	})

	// Test find miner (blink LEDs)
	t.Run("FindMiner", func(t *testing.T) {
		result, err := client.FindMiner(ctx, true)
		if err != nil {
			t.Logf("FindMiner failed: %v", err)
			return
		}
		t.Logf("FindMiner result: %+v", result)
	})
}

// Run with: go test -tags=integration -vnish.host=10.45.3.1 -vnish.apikey=your-key ./internal/vnish/client