package client

import (
	"context"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name            string
		timeout         time.Duration
		workers         int
		expectedWorkers int
	}{
		{
			name:            "Valid parameters",
			timeout:         5 * time.Second,
			workers:         20,
			expectedWorkers: 20,
		},
		{
			name:            "Zero workers defaults to 10",
			timeout:         5 * time.Second,
			workers:         0,
			expectedWorkers: 10,
		},
		{
			name:            "Negative workers defaults to 10",
			timeout:         5 * time.Second,
			workers:         -5,
			expectedWorkers: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.timeout, tt.workers)

			if client.timeout != tt.timeout {
				t.Errorf("Expected timeout %v, got %v", tt.timeout, client.timeout)
			}

			if client.workers != tt.expectedWorkers {
				t.Errorf("Expected workers %d, got %d", tt.expectedWorkers, client.workers)
			}
		})
	}
}

func TestGetAvailableCommands(t *testing.T) {
	commands := GetAvailableCommands()

	expectedCommands := []string{
		"summary", "devs", "pools", "stats", "version",
		"switchpool", "enablepool", "disablepool", "addpool", "removepool",
		"restart", "quit", "custom",
	}

	if len(commands) != len(expectedCommands) {
		t.Errorf("Expected %d commands, got %d", len(expectedCommands), len(commands))
	}

	commandMap := make(map[string]bool)
	for _, cmd := range commands {
		commandMap[cmd] = true
	}

	for _, expected := range expectedCommands {
		if !commandMap[expected] {
			t.Errorf("Expected command %s not found in available commands", expected)
		}
	}
}

func TestGetCommandDescription(t *testing.T) {
	tests := []struct {
		command     string
		expectEmpty bool
	}{
		{"summary", false},
		{"devs", false},
		{"pools", false},
		{"stats", false},
		{"version", false},
		{"switchpool", false},
		{"enablepool", false},
		{"disablepool", false},
		{"addpool", false},
		{"removepool", false},
		{"restart", false},
		{"quit", false},
		{"custom", false},
		{"nonexistent", true},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			desc := GetCommandDescription(tt.command)

			if tt.expectEmpty && desc != "" {
				t.Errorf("Expected empty description for %s, got %s", tt.command, desc)
			}

			if !tt.expectEmpty && desc == "" {
				t.Errorf("Expected non-empty description for %s", tt.command)
			}
		})
	}
}

func TestExecuteCommandWithCancelledContext(t *testing.T) {
	client := NewClient(1*time.Second, 2)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	ips := []string{"192.168.1.1", "192.168.1.2"}
	results := client.ExecuteCommand(ctx, ips, 4028, "summary", nil)

	if len(results) != len(ips) {
		t.Errorf("Expected %d results, got %d", len(ips), len(results))
	}

	for _, result := range results {
		if result.Error != "context cancelled" {
			t.Errorf("Expected 'context cancelled' error, got %s", result.Error)
		}
	}
}

func TestExecuteCommandValidation(t *testing.T) {
	client := NewClient(1*time.Second, 2)
	ctx := context.Background()

	tests := []struct {
		name    string
		ips     []string
		command string
		params  map[string]interface{}
	}{
		{
			name:    "Empty IP list",
			ips:     []string{},
			command: "summary",
			params:  nil,
		},
		{
			name:    "Valid summary command",
			ips:     []string{"192.168.1.1"},
			command: "summary",
			params:  nil,
		},
		{
			name:    "Valid custom command",
			ips:     []string{"192.168.1.1"},
			command: "custom",
			params:  map[string]interface{}{"cmd": "asccount"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := client.ExecuteCommand(ctx, tt.ips, 4028, tt.command, tt.params)

			if len(results) != len(tt.ips) {
				t.Errorf("Expected %d results, got %d", len(tt.ips), len(results))
			}

			for _, result := range results {
				if result.IP == "" {
					t.Error("Result should have IP set")
				}
				if result.Port != 4028 {
					t.Errorf("Expected port 4028, got %d", result.Port)
				}
				if result.Command != tt.command {
					t.Errorf("Expected command %s, got %s", tt.command, result.Command)
				}
			}
		})
	}
}

func TestExecuteJobParameterValidation(t *testing.T) {
	client := NewClient(1*time.Second, 1)

	tests := []struct {
		name        string
		job         job
		expectError string
	}{
		{
			name: "switchpool missing pool parameter",
			job: job{
				ip:      "192.168.1.1",
				port:    4028,
				command: "switchpool",
				params:  map[string]interface{}{},
			},
			expectError: "switchpool command requires 'pool' parameter",
		},
		{
			name: "addpool missing parameters",
			job: job{
				ip:      "192.168.1.1",
				port:    4028,
				command: "addpool",
				params:  map[string]interface{}{"url": "stratum://pool.com"},
			},
			expectError: "addpool requires 'url', 'user', and 'pass' parameters",
		},
		{
			name: "custom command missing cmd parameter",
			job: job{
				ip:      "192.168.1.1",
				port:    4028,
				command: "custom",
				params:  map[string]interface{}{},
			},
			expectError: "custom command requires 'cmd' parameter",
		},
		{
			name: "unknown command",
			job: job{
				ip:      "192.168.1.1",
				port:    4028,
				command: "unknown",
				params:  nil,
			},
			expectError: "unknown command: unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.executeJob(tt.job)

			if result.Error == "" {
				t.Errorf("Expected error, but got none")
				return
			}

			if result.Error != tt.expectError {
				t.Errorf("Expected error %s, got %s", tt.expectError, result.Error)
			}

			if result.IP != tt.job.ip {
				t.Errorf("Expected IP %s, got %s", tt.job.ip, result.IP)
			}

			if result.Port != tt.job.port {
				t.Errorf("Expected port %d, got %d", tt.job.port, result.Port)
			}

			if result.Command != tt.job.command {
				t.Errorf("Expected command %s, got %s", tt.job.command, result.Command)
			}
		})
	}
}

func TestResultStructure(t *testing.T) {
	result := Result{
		IP:       "192.168.1.1",
		Port:     4028,
		Command:  "summary",
		Response: map[string]interface{}{"test": "data"},
		Error:    "",
		Duration: "100ms",
	}

	if result.IP != "192.168.1.1" {
		t.Errorf("Expected IP 192.168.1.1, got %s", result.IP)
	}

	if result.Port != 4028 {
		t.Errorf("Expected port 4028, got %d", result.Port)
	}

	if result.Command != "summary" {
		t.Errorf("Expected command summary, got %s", result.Command)
	}

	if result.Duration != "100ms" {
		t.Errorf("Expected duration 100ms, got %s", result.Duration)
	}
}

func BenchmarkExecuteCommand(b *testing.B) {
	client := NewClient(1*time.Second, 10)
	ctx := context.Background()
	ips := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.ExecuteCommand(ctx, ips, 4028, "summary", nil)
	}
}

func BenchmarkNewClient(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewClient(5*time.Second, 10)
	}
}
