package iprange

import (
	"net"
	"testing"
)

func TestParseSingleIP(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
		hasError bool
	}{
		{
			name:     "Valid IPv4",
			input:    "192.168.1.100",
			expected: []string{"192.168.1.100"},
			hasError: false,
		},
		{
			name:     "Valid IPv6",
			input:    "::1",
			expected: []string{"::1"},
			hasError: false,
		},
		{
			name:     "Invalid IP",
			input:    "999.999.999.999",
			expected: nil,
			hasError: true,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseIPRange(tt.input)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error for input %s, but got none", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for input %s: %v", tt.input, err)
				return
			}

			ips := result.GetIPs()
			if len(ips) != len(tt.expected) {
				t.Errorf("Expected %d IPs, got %d", len(tt.expected), len(ips))
				return
			}

			for i, expectedIP := range tt.expected {
				if ips[i] != expectedIP {
					t.Errorf("Expected IP %s at index %d, got %s", expectedIP, i, ips[i])
				}
			}
		})
	}
}

func TestParseCIDR(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedMin int
		expectedMax int
		hasError    bool
	}{
		{
			name:        "Valid /24 subnet",
			input:       "192.168.1.0/24",
			expectedMin: 254,
			expectedMax: 255,
			hasError:    false,
		},
		{
			name:        "Valid /30 subnet",
			input:       "10.0.0.0/30",
			expectedMin: 2,
			expectedMax: 4,
			hasError:    false,
		},
		{
			name:        "Valid /32 subnet (single host)",
			input:       "172.16.0.1/32",
			expectedMin: 1,
			expectedMax: 1,
			hasError:    false,
		},
		{
			name:        "Invalid CIDR",
			input:       "192.168.1.0/99",
			expectedMin: 0,
			expectedMax: 0,
			hasError:    true,
		},
		{
			name:        "Invalid IP in CIDR",
			input:       "999.999.999.999/24",
			expectedMin: 0,
			expectedMax: 0,
			hasError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseIPRange(tt.input)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error for input %s, but got none", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for input %s: %v", tt.input, err)
				return
			}

			count := result.Count()
			if count < tt.expectedMin || count > tt.expectedMax {
				t.Errorf("Expected IP count between %d and %d, got %d", tt.expectedMin, tt.expectedMax, count)
			}
		})
	}
}

func TestParseRange(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
		hasError bool
	}{
		{
			name:     "Valid small range",
			input:    "192.168.1.1-192.168.1.5",
			expected: 5,
			hasError: false,
		},
		{
			name:     "Single IP range",
			input:    "10.0.0.1-10.0.0.1",
			expected: 1,
			hasError: false,
		},
		{
			name:     "Larger range",
			input:    "172.16.0.1-172.16.0.100",
			expected: 100,
			hasError: false,
		},
		{
			name:     "Invalid range format",
			input:    "192.168.1.1-192.168.1.2-192.168.1.3",
			expected: 0,
			hasError: true,
		},
		{
			name:     "Invalid start IP",
			input:    "999.999.999.999-192.168.1.2",
			expected: 0,
			hasError: true,
		},
		{
			name:     "Invalid end IP",
			input:    "192.168.1.1-999.999.999.999",
			expected: 0,
			hasError: true,
		},
		{
			name:     "Start greater than end",
			input:    "192.168.1.10-192.168.1.5",
			expected: 0,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseIPRange(tt.input)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error for input %s, but got none", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for input %s: %v", tt.input, err)
				return
			}

			if result.Count() != tt.expected {
				t.Errorf("Expected %d IPs, got %d", tt.expected, result.Count())
			}
		})
	}
}

func TestParseMultipleRanges(t *testing.T) {
	tests := []struct {
		name     string
		inputs   []string
		expected int
		hasError bool
	}{
		{
			name:     "Multiple single IPs",
			inputs:   []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"},
			expected: 3,
			hasError: false,
		},
		{
			name:     "Mixed formats",
			inputs:   []string{"192.168.1.1", "10.0.0.0/30", "172.16.0.1-172.16.0.3"},
			expected: 7, // 1 + 3 + 3 (CIDR includes network and broadcast addresses)
			hasError: false,
		},
		{
			name:     "Duplicate IPs",
			inputs:   []string{"192.168.1.1", "192.168.1.1", "192.168.1.2"},
			expected: 2,
			hasError: false,
		},
		{
			name:     "One invalid input",
			inputs:   []string{"192.168.1.1", "invalid-ip"},
			expected: 0,
			hasError: true,
		},
		{
			name:     "Empty input list",
			inputs:   []string{},
			expected: 0,
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseMultipleRanges(tt.inputs)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error for inputs %v, but got none", tt.inputs)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for inputs %v: %v", tt.inputs, err)
				return
			}

			if result.Count() != tt.expected {
				t.Errorf("Expected %d unique IPs, got %d", tt.expected, result.Count())
			}
		})
	}
}

func TestIPRangeGetIPs(t *testing.T) {
	r := &IPRange{
		IPs: []net.IP{
			net.ParseIP("192.168.1.1"),
			net.ParseIP("192.168.1.2"),
		},
	}

	ips := r.GetIPs()
	expected := []string{"192.168.1.1", "192.168.1.2"}

	if len(ips) != len(expected) {
		t.Errorf("Expected %d IPs, got %d", len(expected), len(ips))
	}

	for i, expectedIP := range expected {
		if ips[i] != expectedIP {
			t.Errorf("Expected IP %s at index %d, got %s", expectedIP, i, ips[i])
		}
	}
}

func TestParsePort(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
		hasError bool
	}{
		{
			name:     "Valid port",
			input:    "4028",
			expected: 4028,
			hasError: false,
		},
		{
			name:     "Empty string defaults to 4028",
			input:    "",
			expected: 4028,
			hasError: false,
		},
		{
			name:     "Port 1",
			input:    "1",
			expected: 1,
			hasError: false,
		},
		{
			name:     "Port 65535",
			input:    "65535",
			expected: 65535,
			hasError: false,
		},
		{
			name:     "Invalid port - zero",
			input:    "0",
			expected: 0,
			hasError: true,
		},
		{
			name:     "Invalid port - too high",
			input:    "65536",
			expected: 0,
			hasError: true,
		},
		{
			name:     "Invalid port - non-numeric",
			input:    "abc",
			expected: 0,
			hasError: true,
		},
		{
			name:     "Invalid port - negative",
			input:    "-1",
			expected: 0,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParsePort(tt.input)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error for input %s, but got none", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for input %s: %v", tt.input, err)
				return
			}

			if result != tt.expected {
				t.Errorf("Expected port %d, got %d", tt.expected, result)
			}
		})
	}
}

func BenchmarkParseCIDR(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := ParseIPRange("192.168.0.0/24")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseRange(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := ParseIPRange("192.168.1.1-192.168.1.100")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseMultipleRanges(b *testing.B) {
	inputs := []string{
		"192.168.1.0/28",
		"10.0.0.1-10.0.0.50",
		"172.16.0.100",
	}

	for i := 0; i < b.N; i++ {
		_, err := ParseMultipleRanges(inputs)
		if err != nil {
			b.Fatal(err)
		}
	}
}
