package output

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/sinkers/miner-cli/internal/client"
)

func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestGetFormatter(t *testing.T) {
	tests := []struct {
		format   string
		verbose  bool
		expected string
	}{
		{"json", false, "JSONFormatter"},
		{"JSON", true, "JSONFormatter"},
		{"table", false, "TableFormatter"},
		{"TABLE", true, "TableFormatter"},
		{"color", false, "ColorFormatter"},
		{"COLOR", true, "ColorFormatter"},
		{"unknown", false, "ColorFormatter"},
		{"", false, "ColorFormatter"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			formatter := GetFormatter(tt.format, tt.verbose)

			switch tt.expected {
			case "JSONFormatter":
				if _, ok := formatter.(*JSONFormatter); !ok {
					t.Errorf("Expected JSONFormatter, got %T", formatter)
				}
			case "TableFormatter":
				if _, ok := formatter.(*TableFormatter); !ok {
					t.Errorf("Expected TableFormatter, got %T", formatter)
				}
			case "ColorFormatter":
				if _, ok := formatter.(*ColorFormatter); !ok {
					t.Errorf("Expected ColorFormatter, got %T", formatter)
				}
			}
		})
	}
}

func TestJSONFormatter(t *testing.T) {
	tests := []struct {
		name    string
		results []client.Result
		pretty  bool
	}{
		{
			name: "Single successful result",
			results: []client.Result{
				{
					IP:       "192.168.1.1",
					Port:     4028,
					Command:  "summary",
					Response: map[string]interface{}{"hashrate": 1000.0},
					Duration: "100ms",
				},
			},
			pretty: false,
		},
		{
			name: "Multiple mixed results",
			results: []client.Result{
				{
					IP:       "192.168.1.1",
					Port:     4028,
					Command:  "summary",
					Response: map[string]interface{}{"hashrate": 1000.0},
					Duration: "100ms",
				},
				{
					IP:       "192.168.1.2",
					Port:     4028,
					Command:  "summary",
					Error:    "connection timeout",
					Duration: "5s",
				},
			},
			pretty: true,
		},
		{
			name:    "Empty results",
			results: []client.Result{},
			pretty:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := &JSONFormatter{Pretty: tt.pretty}

			output := captureOutput(func() {
				err := formatter.Format(tt.results)
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			})

			// Verify it's valid JSON
			var parsed []client.Result
			err := json.Unmarshal([]byte(output), &parsed)
			if err != nil {
				t.Errorf("Output is not valid JSON: %v\nOutput: %s", err, output)
			}

			// Verify content matches
			if len(parsed) != len(tt.results) {
				t.Errorf("Expected %d results, got %d", len(tt.results), len(parsed))
			}

			// Check if pretty formatting was applied when requested
			if tt.pretty && !strings.Contains(output, "\n") {
				t.Error("Expected pretty-printed JSON with newlines")
			}
		})
	}
}

func TestColorFormatter(t *testing.T) {
	tests := []struct {
		name    string
		results []client.Result
		verbose bool
	}{
		{
			name: "Single successful result",
			results: []client.Result{
				{
					IP:       "192.168.1.1",
					Port:     4028,
					Command:  "summary",
					Response: "success response",
					Duration: "100ms",
				},
			},
			verbose: false,
		},
		{
			name: "Single failed result",
			results: []client.Result{
				{
					IP:       "192.168.1.1",
					Port:     4028,
					Command:  "summary",
					Error:    "connection timeout",
					Duration: "5s",
				},
			},
			verbose: true,
		},
		{
			name: "Mixed results",
			results: []client.Result{
				{
					IP:       "192.168.1.1",
					Port:     4028,
					Command:  "summary",
					Response: map[string]interface{}{"status": "ok"},
					Duration: "100ms",
				},
				{
					IP:       "192.168.1.2",
					Port:     4028,
					Command:  "summary",
					Error:    "connection refused",
					Duration: "2s",
				},
			},
			verbose: true,
		},
		{
			name:    "Empty results",
			results: []client.Result{},
			verbose: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := &ColorFormatter{Verbose: tt.verbose}

			output := captureOutput(func() {
				err := formatter.Format(tt.results)
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			})

			// Basic output validation
			if len(tt.results) > 0 {
				if !strings.Contains(output, "CGMiner API Results") {
					t.Error("Expected header not found in output")
				}

				if !strings.Contains(output, "Summary") {
					t.Error("Expected summary section not found in output")
				}

				// Check for IP addresses in output
				for _, result := range tt.results {
					if !strings.Contains(output, result.IP) {
						t.Errorf("Expected IP %s not found in output", result.IP)
					}
				}
			}
		})
	}
}

func TestTableFormatter(t *testing.T) {
	tests := []struct {
		name    string
		results []client.Result
		verbose bool
	}{
		{
			name: "Single result",
			results: []client.Result{
				{
					IP:       "192.168.1.1",
					Port:     4028,
					Command:  "summary",
					Response: map[string]interface{}{"test": "data"},
					Duration: "100ms",
				},
			},
			verbose: false,
		},
		{
			name: "Multiple results with verbose",
			results: []client.Result{
				{
					IP:       "192.168.1.1",
					Port:     4028,
					Command:  "summary",
					Response: map[string]interface{}{"hashrate": 1000.0},
					Duration: "100ms",
				},
				{
					IP:       "192.168.1.2",
					Port:     4028,
					Command:  "summary",
					Error:    "timeout",
					Duration: "5s",
				},
			},
			verbose: true,
		},
		{
			name:    "Empty results",
			results: []client.Result{},
			verbose: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := &TableFormatter{Verbose: tt.verbose}

			output := captureOutput(func() {
				err := formatter.Format(tt.results)
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			})

			if len(tt.results) > 0 {
				// Check for table headers
				expectedHeaders := []string{"IP", "Port", "Command", "Status", "Duration", "Details"}
				for _, header := range expectedHeaders {
					if !strings.Contains(output, header) {
						t.Errorf("Expected header '%s' not found in output", header)
					}
				}

				// Check for summary
				if !strings.Contains(output, "Summary:") {
					t.Error("Expected summary not found in output")
				}

				// Check for IP addresses
				for _, result := range tt.results {
					if !strings.Contains(output, result.IP) {
						t.Errorf("Expected IP %s not found in output", result.IP)
					}
				}
			}
		})
	}
}

func TestColorFormatterFormatResponse(t *testing.T) {
	formatter := &ColorFormatter{Verbose: true}

	tests := []struct {
		name     string
		response interface{}
	}{
		{
			name:     "String response",
			response: "simple string",
		},
		{
			name: "Map response",
			response: map[string]interface{}{
				"key1": "value1",
				"key2": 42,
				"key3": true,
			},
		},
		{
			name: "Array response",
			response: []interface{}{
				"item1",
				"item2",
				map[string]interface{}{"nested": "value"},
			},
		},
		{
			name: "Complex nested response",
			response: map[string]interface{}{
				"nested_map": map[string]interface{}{
					"inner_key": "inner_value",
				},
				"nested_array": []interface{}{1, 2, 3},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				formatter.formatResponse(tt.response, "  ")
			})

			// Basic validation - should not be empty and should not panic
			if output == "" {
				t.Error("Expected non-empty output")
			}
		})
	}
}

func BenchmarkJSONFormatter(b *testing.B) {
	results := []client.Result{
		{
			IP:       "192.168.1.1",
			Port:     4028,
			Command:  "summary",
			Response: map[string]interface{}{"hashrate": 1000.0, "temp": 65.5},
			Duration: "100ms",
		},
		{
			IP:       "192.168.1.2",
			Port:     4028,
			Command:  "summary",
			Error:    "connection timeout",
			Duration: "5s",
		},
	}

	formatter := &JSONFormatter{Pretty: false}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		captureOutput(func() {
			formatter.Format(results)
		})
	}
}

func BenchmarkColorFormatter(b *testing.B) {
	results := []client.Result{
		{
			IP:       "192.168.1.1",
			Port:     4028,
			Command:  "summary",
			Response: map[string]interface{}{"hashrate": 1000.0, "temp": 65.5},
			Duration: "100ms",
		},
		{
			IP:       "192.168.1.2",
			Port:     4028,
			Command:  "summary",
			Error:    "connection timeout",
			Duration: "5s",
		},
	}

	formatter := &ColorFormatter{Verbose: false}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		captureOutput(func() {
			formatter.Format(results)
		})
	}
}

func BenchmarkTableFormatter(b *testing.B) {
	results := []client.Result{
		{
			IP:       "192.168.1.1",
			Port:     4028,
			Command:  "summary",
			Response: map[string]interface{}{"hashrate": 1000.0, "temp": 65.5},
			Duration: "100ms",
		},
		{
			IP:       "192.168.1.2",
			Port:     4028,
			Command:  "summary",
			Error:    "connection timeout",
			Duration: "5s",
		},
	}

	formatter := &TableFormatter{Verbose: false}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		captureOutput(func() {
			formatter.Format(results)
		})
	}
}
