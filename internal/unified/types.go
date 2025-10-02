//go:build ignore
// +build ignore

package unified

import (
	"context"
	"time"
)

// APIType represents the detected API type
type APIType string

const (
	APITypeCGMiner  APIType = "cgminer"
	APITypeVNish    APIType = "vnish"
	APITypeBraiins  APIType = "braiins"
	APITypeUnknown  APIType = "unknown"
)

// MinerStatus represents the current mining status
type MinerStatus string

const (
	StatusRunning  MinerStatus = "running"
	StatusPaused   MinerStatus = "paused"
	StatusStopped  MinerStatus = "stopped"
	StatusTuning   MinerStatus = "tuning"
	StatusError    MinerStatus = "error"
	StatusOffline  MinerStatus = "offline"
	StatusUnknown  MinerStatus = "unknown"
)

// MinerAPI is the unified interface for all miner types
type MinerAPI interface {
	// Connection management
	Connect(ctx context.Context) error
	Disconnect() error
	IsConnected() bool
	GetAPIType() APIType

	// Core mining operations
	GetSummary(ctx context.Context) (*UnifiedSummary, error)
	GetDevices(ctx context.Context) ([]Device, error)
	GetPools(ctx context.Context) ([]Pool, error)

	// Control operations (return ErrNotSupported if not available)
	StartMining(ctx context.Context) error
	StopMining(ctx context.Context) error
	RestartMining(ctx context.Context) error
	PauseMining(ctx context.Context) error
	ResumeMining(ctx context.Context) error

	// Pool management
	SwitchPool(ctx context.Context, poolID int) error
	AddPool(ctx context.Context, pool Pool) error
	RemovePool(ctx context.Context, poolID int) error

	// System operations
	Reboot(ctx context.Context) error
	GetLogs(ctx context.Context, lines int) ([]string, error)

	// Configuration
	GetConfig(ctx context.Context) (map[string]interface{}, error)
	SetPowerTarget(ctx context.Context, watts int) error
	SetFanSpeed(ctx context.Context, percent int) error
}

// UnifiedSummary contains all summary information across different APIs
type UnifiedSummary struct {
	// Basic Information
	IPAddress    string      `json:"ip_address"`
	APIType      APIType     `json:"api_type"`
	Model        string      `json:"model"`
	Firmware     string      `json:"firmware"`
	Version      string      `json:"version"`
	Uptime       time.Duration `json:"uptime"`

	// Mining Status
	Status       MinerStatus `json:"status"`
	StatusDetail string      `json:"status_detail"`

	// Performance Metrics
	HashRate     float64 `json:"hashrate"`      // Default unit (TH/s for modern miners)
	HashRateUnit string  `json:"hashrate_unit"`
	HashRate5s   float64 `json:"hashrate_5s"`
	HashRate1m   float64 `json:"hashrate_1m"`
	HashRate5m   float64 `json:"hashrate_5m"`
	HashRate15m  float64 `json:"hashrate_15m"`

	// Power & Efficiency
	PowerUsage float64 `json:"power_usage"` // Watts
	Efficiency float64 `json:"efficiency"`  // J/TH

	// Hardware Health
	BoardsTotal  int `json:"boards_total"`
	BoardsActive int `json:"boards_active"`
	ChipsTotal   int `json:"chips_total"`
	ChipsActive  int `json:"chips_active"`

	// Temperature
	TempAvg float64 `json:"temp_avg"`
	TempMax float64 `json:"temp_max"`
	TempMin float64 `json:"temp_min"`

	// Fans
	FanCount    int `json:"fan_count"`
	FanSpeedAvg int `json:"fan_speed_avg"` // RPM

	// Work Statistics
	Accepted int64 `json:"accepted"`
	Rejected int64 `json:"rejected"`
	HWErrors int64 `json:"hw_errors"`

	// Pool Information
	ActivePool string `json:"active_pool"`
	PoolCount  int    `json:"pool_count"`

	// Errors and Warnings
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`

	// Timing
	ResponseTime time.Duration `json:"response_time"`
	Timestamp    time.Time     `json:"timestamp"`

	// Raw response for API-specific data
	RawResponse interface{} `json:"raw_response,omitempty"`
}

// Device represents a mining device/board
type Device struct {
	Index       int     `json:"index"`
	Name        string  `json:"name"`
	Status      string  `json:"status"`
	Temperature float64 `json:"temperature"`
	HashRate    float64 `json:"hashrate"`
	Chips       int     `json:"chips"`
	ChipsActive int     `json:"chips_active"`
	HWErrors    int64   `json:"hw_errors"`
	Frequency   int     `json:"frequency"`
}

// Pool represents a mining pool
type Pool struct {
	ID        int       `json:"id"`
	URL       string    `json:"url"`
	User      string    `json:"user"`
	Status    string    `json:"status"`
	Priority  int       `json:"priority"`
	Accepted  int64     `json:"accepted"`
	Rejected  int64     `json:"rejected"`
	LastShare time.Time `json:"last_share"`
}

// APIInfo contains detected API information
type APIInfo struct {
	Type         APIType   `json:"type"`
	Port         int       `json:"port"`
	Protocol     string    `json:"protocol"`
	Version      string    `json:"version"`
	Model        string    `json:"model"`
	RequiresAuth bool      `json:"requires_auth"`
	AlsoSupports []APIType `json:"also_supports,omitempty"`
}

// DetectionResult contains the result of API detection
type DetectionResult struct {
	Host    string  `json:"host"`
	APIType APIType `json:"api_type"`
	Info    APIInfo `json:"info"`
	Error   error   `json:"error,omitempty"`
}
