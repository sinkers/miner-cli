package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sinkers/miner-cli/internal/vnish/models"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		options  []Option
		validate func(*testing.T, *Client)
	}{
		{
			name: "basic client creation",
			host: "192.168.1.100",
			validate: func(t *testing.T, c *Client) {
				if c.baseURL != "http://192.168.1.100/api/v1" {
					t.Errorf("expected baseURL http://192.168.1.100/api/v1, got %s", c.baseURL)
				}
				if c.apiKey != "" {
					t.Errorf("expected empty apiKey, got %s", c.apiKey)
				}
			},
		},
		{
			name: "client with API key",
			host: "10.0.0.1",
			options: []Option{
				WithAPIKey("test-api-key"),
			},
			validate: func(t *testing.T, c *Client) {
				if c.apiKey != "test-api-key" {
					t.Errorf("expected apiKey test-api-key, got %s", c.apiKey)
				}
			},
		},
		{
			name: "client with custom timeout",
			host: "10.0.0.1",
			options: []Option{
				WithTimeout(5 * time.Second),
			},
			validate: func(t *testing.T, c *Client) {
				if c.httpClient.Timeout != 5*time.Second {
					t.Errorf("expected timeout 5s, got %v", c.httpClient.Timeout)
				}
			},
		},
		{
			name: "client with debug mode",
			host: "10.0.0.1",
			options: []Option{
				WithDebug(true),
			},
			validate: func(t *testing.T, c *Client) {
				if !c.debug {
					t.Error("expected debug mode to be enabled")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.host, tt.options...)
			if tt.validate != nil {
				tt.validate(t, client)
			}
		})
	}
}

func TestGetInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/info" {
			t.Errorf("expected path /api/v1/info, got %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected method GET, got %s", r.Method)
		}
		
		// Check API key header
		if apiKey := r.Header.Get("X-API-Key"); apiKey != "test-key" {
			t.Errorf("expected API key test-key, got %s", apiKey)
		}

		response := models.SystemInfo{
			Hostname:    "miner-01",
			Model:       "Antminer S19",
			Version:     "1.2.3",
			Uptime:      86400,
			LoadAverage: []float64{1.5, 1.2, 1.0},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL[7:], WithAPIKey("test-key")) // Remove http://
	client.baseURL = server.URL + "/api/v1" // Override for test

	info, err := client.GetInfo(context.Background())
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}

	if info.Hostname != "miner-01" {
		t.Errorf("expected hostname miner-01, got %s", info.Hostname)
	}
	if info.Model != "Antminer S19" {
		t.Errorf("expected model Antminer S19, got %s", info.Model)
	}
	if info.Version != "1.2.3" {
		t.Errorf("expected version 1.2.3, got %s", info.Version)
	}
	if info.Uptime != 86400 {
		t.Errorf("expected uptime 86400, got %d", info.Uptime)
	}
}

func TestGetSummary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/summary" {
			t.Errorf("expected path /api/v1/summary, got %s", r.URL.Path)
		}

		response := models.Summary{
			Status: models.MiningStatus{
				Running: true,
				Paused:  false,
				Status:  "mining",
			},
			Performance: models.PerfSummary{
				HashRate:       100.5,
				HashRateUnit:   "TH/s",
				Accepted:       1000,
				Rejected:       10,
				HardwareErrors: 2,
				Efficiency:     30.5,
				PowerUsage:     3300,
			},
			Pools: []models.PoolInfo{
				{
					ID:       1,
					URL:      "stratum+tcp://pool.example.com:3333",
					User:     "worker1",
					Status:   "active",
					Priority: 0,
					Accepted: 990,
					Rejected: 10,
				},
			},
			Temperature: models.TempInfo{
				Board:  []float64{65.5, 68.0, 66.5},
				Chip:   []float64{75.0, 78.0, 76.5},
				Intake: 30.0,
				Outlet: 45.0,
			},
			FanSpeed: []int{4500, 4600},
			Uptime:   86400,
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL[7:])
	client.baseURL = server.URL + "/api/v1"

	summary, err := client.GetSummary(context.Background())
	if err != nil {
		t.Fatalf("GetSummary failed: %v", err)
	}

	if !summary.Status.Running {
		t.Error("expected status.running to be true")
	}
	if summary.Performance.HashRate != 100.5 {
		t.Errorf("expected hashrate 100.5, got %f", summary.Performance.HashRate)
	}
	if len(summary.Pools) != 1 {
		t.Errorf("expected 1 pool, got %d", len(summary.Pools))
	}
	if summary.Pools[0].URL != "stratum+tcp://pool.example.com:3333" {
		t.Errorf("expected pool URL stratum+tcp://pool.example.com:3333, got %s", summary.Pools[0].URL)
	}
}

func TestMiningOperations(t *testing.T) {
	tests := []struct {
		name     string
		method   func(*Client, context.Context) error
		path     string
		httpMethod string
	}{
		{
			name: "start mining",
			method: func(c *Client, ctx context.Context) error {
				return c.StartMining(ctx)
			},
			path: "/api/v1/mining/start",
			httpMethod: http.MethodPost,
		},
		{
			name: "stop mining",
			method: func(c *Client, ctx context.Context) error {
				return c.StopMining(ctx)
			},
			path: "/api/v1/mining/stop",
			httpMethod: http.MethodPost,
		},
		{
			name: "restart mining",
			method: func(c *Client, ctx context.Context) error {
				return c.RestartMining(ctx)
			},
			path: "/api/v1/mining/restart",
			httpMethod: http.MethodPost,
		},
		{
			name: "pause mining",
			method: func(c *Client, ctx context.Context) error {
				return c.PauseMining(ctx)
			},
			path: "/api/v1/mining/pause",
			httpMethod: http.MethodPost,
		},
		{
			name: "resume mining",
			method: func(c *Client, ctx context.Context) error {
				return c.ResumeMining(ctx)
			},
			path: "/api/v1/mining/resume",
			httpMethod: http.MethodPost,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tt.path {
					t.Errorf("expected path %s, got %s", tt.path, r.URL.Path)
				}
				if r.Method != tt.httpMethod {
					t.Errorf("expected method %s, got %s", tt.httpMethod, r.Method)
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			client := NewClient(server.URL[7:])
			client.baseURL = server.URL + "/api/v1"

			err := tt.method(client, context.Background())
			if err != nil {
				t.Errorf("%s failed: %v", tt.name, err)
			}
		})
	}
}

func TestSwitchPool(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/mining/switch-pool" {
			t.Errorf("expected path /api/v1/mining/switch-pool, got %s", r.URL.Path)
		}
		
		var req models.SwitchPoolRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		
		if req.PoolID != 2 {
			t.Errorf("expected pool ID 2, got %d", req.PoolID)
		}
		
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL[7:])
	client.baseURL = server.URL + "/api/v1"

	err := client.SwitchPool(context.Background(), 2)
	if err != nil {
		t.Fatalf("SwitchPool failed: %v", err)
	}
}

func TestGetChains(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/chains" {
			t.Errorf("expected path /api/v1/chains, got %s", r.URL.Path)
		}

		response := []models.ChainInfo{
			{
				Index:       0,
				Frequency:   650,
				Voltage:     12.5,
				Temperature: 75.5,
				Status:      "active",
				ChipCount:   120,
				HashRate:    33.5,
			},
			{
				Index:       1,
				Frequency:   650,
				Voltage:     12.5,
				Temperature: 76.0,
				Status:      "active",
				ChipCount:   120,
				HashRate:    33.8,
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL[7:])
	client.baseURL = server.URL + "/api/v1"

	chains, err := client.GetChains(context.Background())
	if err != nil {
		t.Fatalf("GetChains failed: %v", err)
	}

	if len(chains) != 2 {
		t.Errorf("expected 2 chains, got %d", len(chains))
	}
	
	if chains[0].Index != 0 {
		t.Errorf("expected chain 0 index 0, got %d", chains[0].Index)
	}
	if chains[0].Frequency != 650 {
		t.Errorf("expected chain 0 frequency 650, got %d", chains[0].Frequency)
	}
	if chains[0].ChipCount != 120 {
		t.Errorf("expected chain 0 chip count 120, got %d", chains[0].ChipCount)
	}
}

func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedError  string
	}{
		{
			name:          "401 Unauthorized",
			statusCode:    http.StatusUnauthorized,
			responseBody:  `{"error": "unauthorized"}`,
			expectedError: "HTTP 401: unauthorized",
		},
		{
			name:          "500 Internal Server Error",
			statusCode:    http.StatusInternalServerError,
			responseBody:  `{"error": "internal server error"}`,
			expectedError: "HTTP 500: internal server error",
		},
		{
			name:          "400 Bad Request without error field",
			statusCode:    http.StatusBadRequest,
			responseBody:  `{"message": "bad request"}`,
			expectedError: "HTTP 400:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := NewClient(server.URL[7:])
			client.baseURL = server.URL + "/api/v1"

			_, err := client.GetInfo(context.Background())
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			
			if !contains(err.Error(), tt.expectedError) {
				t.Errorf("expected error containing '%s', got '%s'", tt.expectedError, err.Error())
			}
		})
	}
}

func TestAPIKeyOperations(t *testing.T) {
	t.Run("GetAPIKeys", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v1/apikeys" {
				t.Errorf("expected path /api/v1/apikeys, got %s", r.URL.Path)
			}

			response := []models.ApiKeysJsonItem{
				{
					ID:        "key1",
					Name:      "Test Key 1",
					CreatedAt: time.Now().Add(-24 * time.Hour),
					LastUsed:  time.Now().Add(-1 * time.Hour),
				},
				{
					ID:        "key2",
					Name:      "Test Key 2",
					CreatedAt: time.Now().Add(-48 * time.Hour),
				},
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewClient(server.URL[7:])
		client.baseURL = server.URL + "/api/v1"

		keys, err := client.GetAPIKeys(context.Background())
		if err != nil {
			t.Fatalf("GetAPIKeys failed: %v", err)
		}

		if len(keys) != 2 {
			t.Errorf("expected 2 keys, got %d", len(keys))
		}
		if keys[0].Name != "Test Key 1" {
			t.Errorf("expected key name 'Test Key 1', got %s", keys[0].Name)
		}
	})

	t.Run("AddAPIKey", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v1/apikeys" {
				t.Errorf("expected path /api/v1/apikeys, got %s", r.URL.Path)
			}

			var req models.AddApikeyQuery
			json.NewDecoder(r.Body).Decode(&req)
			
			if req.Name != "New Test Key" {
				t.Errorf("expected key name 'New Test Key', got %s", req.Name)
			}

			response := models.AddApiKeyRes{
				Status: models.AddApiKeyStatus{
					Success: true,
					Message: "Key created successfully",
				},
				Key: "generated-api-key-123",
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewClient(server.URL[7:])
		client.baseURL = server.URL + "/api/v1"

		result, err := client.AddAPIKey(context.Background(), "New Test Key")
		if err != nil {
			t.Fatalf("AddAPIKey failed: %v", err)
		}

		if !result.Status.Success {
			t.Error("expected success to be true")
		}
		if result.Key != "generated-api-key-123" {
			t.Errorf("expected key 'generated-api-key-123', got %s", result.Key)
		}
	})

	t.Run("DeleteAPIKey", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v1/apikeys/delete" {
				t.Errorf("expected path /api/v1/apikeys/delete, got %s", r.URL.Path)
			}

			var req models.DeleteApikeyQuery
			json.NewDecoder(r.Body).Decode(&req)
			
			if req.ID != "key-to-delete" {
				t.Errorf("expected key ID 'key-to-delete', got %s", req.ID)
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient(server.URL[7:])
		client.baseURL = server.URL + "/api/v1"

		err := client.DeleteAPIKey(context.Background(), "key-to-delete")
		if err != nil {
			t.Fatalf("DeleteAPIKey failed: %v", err)
		}
	})
}

func TestSettingsOperations(t *testing.T) {
	t.Run("GetSettings", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v1/settings" {
				t.Errorf("expected path /api/v1/settings, got %s", r.URL.Path)
			}

			response := models.Settings{
				Pools: []models.PoolConfig{
					{
						ID:       1,
						URL:      "stratum+tcp://pool.example.com:3333",
						User:     "worker1",
						Password: "x",
						Priority: 0,
						Enabled:  true,
					},
				},
				Fan: models.FanSettings{
					Mode:       "auto",
					TargetTemp: 75,
					MinSpeed:   20,
					MaxSpeed:   100,
				},
				Temperature: models.TempSettings{
					TargetTemp:    75,
					HotTemp:       85,
					DangerousTemp: 95,
				},
				Advanced: models.AdvancedSettings{
					AutoTune:      true,
					AutoRestart:   true,
					LowPowerMode:  false,
					ImmersionMode: false,
				},
				Network: models.NetworkSettings{
					DHCP:      true,
					IPAddress: "192.168.1.100",
					Netmask:   "255.255.255.0",
					Gateway:   "192.168.1.1",
					DNS1:      "8.8.8.8",
					DNS2:      "8.8.4.4",
				},
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewClient(server.URL[7:])
		client.baseURL = server.URL + "/api/v1"

		settings, err := client.GetSettings(context.Background())
		if err != nil {
			t.Fatalf("GetSettings failed: %v", err)
		}

		if len(settings.Pools) != 1 {
			t.Errorf("expected 1 pool, got %d", len(settings.Pools))
		}
		if settings.Fan.Mode != "auto" {
			t.Errorf("expected fan mode 'auto', got %s", settings.Fan.Mode)
		}
		if settings.Temperature.TargetTemp != 75 {
			t.Errorf("expected target temp 75, got %d", settings.Temperature.TargetTemp)
		}
	})

	t.Run("UpdateSettings", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v1/settings" {
				t.Errorf("expected path /api/v1/settings, got %s", r.URL.Path)
			}
			if r.Method != http.MethodPost {
				t.Errorf("expected method POST, got %s", r.Method)
			}

			var settings models.Settings
			json.NewDecoder(r.Body).Decode(&settings)
			
			if settings.Fan.Mode != "manual" {
				t.Errorf("expected fan mode 'manual', got %s", settings.Fan.Mode)
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient(server.URL[7:])
		client.baseURL = server.URL + "/api/v1"

		settings := &models.Settings{
			Fan: models.FanSettings{
				Mode:     "manual",
				MinSpeed: 50,
				MaxSpeed: 100,
			},
		}

		err := client.UpdateSettings(context.Background(), settings)
		if err != nil {
			t.Fatalf("UpdateSettings failed: %v", err)
		}
	})
}

func TestContextCancellation(t *testing.T) {
	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(models.SystemInfo{})
	}))
	defer server.Close()

	client := NewClient(server.URL[7:])
	client.baseURL = server.URL + "/api/v1"

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.GetInfo(ctx)
	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}