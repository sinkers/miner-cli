package models

import "time"

// Common error response
type ErrDescr struct {
	Error string `json:"error"`
}

// API Key related models
type ApiKeysJsonItem struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	LastUsed  time.Time `json:"last_used,omitempty"`
}

type AddApikeyQuery struct {
	Name string `json:"name"`
}

type AddApiKeyRes struct {
	Status AddApiKeyStatus `json:"status"`
	Key    string          `json:"key,omitempty"`
}

type AddApiKeyStatus struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

type DeleteApikeyQuery struct {
	ID string `json:"id"`
}

// Authentication
type AuthCheck struct {
	Authenticated bool   `json:"authenticated"`
	Method        string `json:"method,omitempty"`
}

// Warranty
type WarrantyStatus struct {
	Active    bool      `json:"active"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
	Message   string    `json:"message,omitempty"`
}

// Mining operations
type MiningStatus struct {
	Running bool   `json:"running"`
	Paused  bool   `json:"paused"`
	Status  string `json:"status"`
}

type SwitchPoolRequest struct {
	PoolID int `json:"pool_id"`
}

// System info
type SystemInfo struct {
	Hostname    string `json:"hostname"`
	Model       string `json:"model"`
	Version     string `json:"version"`
	Uptime      int64  `json:"uptime"`
	LoadAverage []float64 `json:"load_average"`
}

type ModelInfo struct {
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	Description  string `json:"description"`
}

// Chain and chip information
type ChainInfo struct {
	Index       int     `json:"index"`
	Frequency   int     `json:"frequency"`
	Voltage     float64 `json:"voltage"`
	Temperature float64 `json:"temperature"`
	Status      string  `json:"status"`
	ChipCount   int     `json:"chip_count"`
	HashRate    float64 `json:"hash_rate"`
}

type ChipInfo struct {
	Index       int     `json:"index"`
	ChainIndex  int     `json:"chain_index"`
	Frequency   int     `json:"frequency"`
	Temperature float64 `json:"temperature"`
	Status      string  `json:"status"`
}

// Performance
type PerfSummary struct {
	HashRate       float64 `json:"hash_rate"`
	HashRateUnit   string  `json:"hash_rate_unit"`
	Accepted       int64   `json:"accepted"`
	Rejected       int64   `json:"rejected"`
	HardwareErrors int64   `json:"hardware_errors"`
	Efficiency     float64 `json:"efficiency"`
	PowerUsage     float64 `json:"power_usage"`
}

type Summary struct {
	Status         MiningStatus `json:"status"`
	Performance    PerfSummary  `json:"performance"`
	Pools          []PoolInfo   `json:"pools"`
	Temperature    TempInfo     `json:"temperature"`
	FanSpeed       []int        `json:"fan_speed"`
	Uptime         int64        `json:"uptime"`
	LastShareTime  time.Time    `json:"last_share_time"`
}

type PoolInfo struct {
	ID         int    `json:"id"`
	URL        string `json:"url"`
	User       string `json:"user"`
	Status     string `json:"status"`
	Priority   int    `json:"priority"`
	Accepted   int64  `json:"accepted"`
	Rejected   int64  `json:"rejected"`
	Stale      int64  `json:"stale"`
	LastShare  time.Time `json:"last_share,omitempty"`
}

type TempInfo struct {
	Board  []float64 `json:"board"`
	Chip   []float64 `json:"chip"`
	Intake float64   `json:"intake"`
	Outlet float64   `json:"outlet"`
}

// Settings
type Settings struct {
	Pools         []PoolConfig     `json:"pools"`
	Fan           FanSettings      `json:"fan"`
	Temperature   TempSettings     `json:"temperature"`
	Advanced      AdvancedSettings `json:"advanced"`
	Network       NetworkSettings  `json:"network"`
}

type PoolConfig struct {
	ID       int    `json:"id"`
	URL      string `json:"url"`
	User     string `json:"user"`
	Password string `json:"password"`
	Priority int    `json:"priority"`
	Enabled  bool   `json:"enabled"`
}

type FanSettings struct {
	Mode       string `json:"mode"` // auto, manual, immersion
	TargetTemp int    `json:"target_temp,omitempty"`
	MinSpeed   int    `json:"min_speed,omitempty"`
	MaxSpeed   int    `json:"max_speed,omitempty"`
}

type TempSettings struct {
	TargetTemp    int `json:"target_temp"`
	HotTemp       int `json:"hot_temp"`
	DangerousTemp int `json:"dangerous_temp"`
}

type AdvancedSettings struct {
	AutoTune      bool   `json:"autotune"`
	AutoRestart   bool   `json:"auto_restart"`
	LowPowerMode  bool   `json:"low_power_mode"`
	ImmersionMode bool   `json:"immersion_mode"`
	CustomFirmware string `json:"custom_firmware,omitempty"`
}

type NetworkSettings struct {
	DHCP       bool   `json:"dhcp"`
	IPAddress  string `json:"ip_address,omitempty"`
	Netmask    string `json:"netmask,omitempty"`
	Gateway    string `json:"gateway,omitempty"`
	DNS1       string `json:"dns1,omitempty"`
	DNS2       string `json:"dns2,omitempty"`
}

// Autotune
type AutotunePresets struct {
	Presets []AutotunePreset `json:"presets"`
}

type AutotunePreset struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	HashRate    float64 `json:"hash_rate"`
	Power       float64 `json:"power"`
	Efficiency  float64 `json:"efficiency"`
}

// Firmware
type FirmwareUpdateRequest struct {
	URL      string `json:"url,omitempty"`
	FilePath string `json:"file_path,omitempty"`
	Version  string `json:"version,omitempty"`
}

type FirmwareUpdateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Version string `json:"version,omitempty"`
}

// Logs
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Source    string    `json:"source,omitempty"`
}

// Metrics
type Metrics struct {
	Timestamp   time.Time          `json:"timestamp"`
	HashRate    []MetricPoint      `json:"hash_rate"`
	Temperature []MetricPoint      `json:"temperature"`
	FanSpeed    []MetricPoint      `json:"fan_speed"`
	Power       []MetricPoint      `json:"power"`
	Shares      ShareMetrics       `json:"shares"`
	Errors      ErrorMetrics       `json:"errors"`
}

type MetricPoint struct {
	Time  time.Time   `json:"time"`
	Value float64     `json:"value"`
	Label string      `json:"label,omitempty"`
}

type ShareMetrics struct {
	Accepted   []MetricPoint `json:"accepted"`
	Rejected   []MetricPoint `json:"rejected"`
	Stale      []MetricPoint `json:"stale"`
}

type ErrorMetrics struct {
	Hardware []MetricPoint `json:"hardware"`
	Network  []MetricPoint `json:"network"`
	Other    []MetricPoint `json:"other"`
}

// Notes
type Note struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateNoteRequest struct {
	Content string `json:"content"`
}

// UI Settings
type UISettings struct {
	Theme           string `json:"theme"`
	Language        string `json:"language"`
	RefreshInterval int    `json:"refresh_interval"`
	Notifications   bool   `json:"notifications"`
}

// Lock/Unlock
type LockRequest struct {
	Password string `json:"password"`
}

type UnlockRequest struct {
	Password string `json:"password"`
}

type LockStatus struct {
	Locked   bool      `json:"locked"`
	LockedAt time.Time `json:"locked_at,omitempty"`
	LockedBy string    `json:"locked_by,omitempty"`
}

// Layout
type LayoutInfo struct {
	Chains  int    `json:"chains"`
	Chips   int    `json:"chips"`
	Fans    int    `json:"fans"`
	PSUs    int    `json:"psus"`
	Boards  int    `json:"boards"`
	Layout  string `json:"layout"`
}

// Status response
type Status struct {
	Mining      MiningStatus `json:"mining"`
	System      SystemInfo   `json:"system"`
	Performance PerfSummary  `json:"performance"`
	Temperature TempInfo     `json:"temperature"`
	Fans        []int        `json:"fans"`
	Errors      []string     `json:"errors,omitempty"`
	Warnings    []string     `json:"warnings,omitempty"`
}

// Backup/Restore
type BackupResponse struct {
	Success  bool   `json:"success"`
	Filename string `json:"filename,omitempty"`
	Size     int64  `json:"size,omitempty"`
	Message  string `json:"message,omitempty"`
}

type RestoreRequest struct {
	Filename string `json:"filename,omitempty"`
	Data     []byte `json:"data,omitempty"`
}

// Factory info
type FactoryInfo struct {
	SerialNumber    string    `json:"serial_number"`
	ManufactureDate time.Time `json:"manufacture_date"`
	FirmwareVersion string    `json:"firmware_version"`
	HardwareVersion string    `json:"hardware_version"`
	BatchNumber     string    `json:"batch_number"`
}

// Find miner
type FindMinerRequest struct {
	Blink bool `json:"blink"`
}

type FindMinerResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}