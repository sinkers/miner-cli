# Unified API Abstraction Layer Plan

## API Capabilities Analysis

### 1. CGMiner API (TCP/RPC on port 4028)
**Protocol**: TCP socket with JSON-RPC
**Available Commands**:
- `summary` - Basic mining statistics
- `devs` - Device information
- `pools` - Pool configuration and statistics
- `stats` - Detailed statistics
- `version` - Miner version information
- `switchpool`, `enablepool`, `disablepool` - Pool management
- `addpool`, `removepool` - Pool configuration
- `restart`, `quit` - Mining control

**Data Available**:
- Hash rate (MHS 5s, MHS av)
- Accepted/Rejected shares
- Hardware errors
- Temperature (limited)
- Pool status
- Basic device info

### 2. Vnish API (REST on port 80/443)
**Protocol**: HTTP REST with JSON
**Endpoints**:
- `/api/v1/summary` - Comprehensive mining status
- `/api/v1/status` - Mining operational status
- `/api/v1/pools` - Pool management
- `/api/v1/chains` - ASIC chain information
- `/api/v1/chips` - Individual chip status
- `/api/v1/settings` - Configuration management
- `/api/v1/system` - System information
- `/api/v1/restart`, `/api/v1/stop`, `/api/v1/pause` - Mining control

**Data Available**:
- Detailed hash rate and efficiency
- Power consumption
- Board/chip health status
- Comprehensive temperature data
- Fan speeds
- Uptime
- Model and firmware version
- Autotune status

### 3. Braiins OS+ API (gRPC on port 50051 + CGMiner compatibility)
**Protocol**: gRPC with protobuf + CGMiner API subset
**Services**:
- `ApiVersionService` - API version
- `ActionsService` - Mining control (pause, resume, restart, reboot)
- BOSminer API (CGMiner-compatible on 4028)
- Additional proprietary autotuning endpoints

**Data Available**:
- Hash rate and efficiency
- Power consumption (with autotuning)
- Temperature monitoring
- Board health
- Tuning status
- Extended CGMiner data

## Proposed Unified Interface

```go
// MinerClient is the main abstraction interface
type MinerClient interface {
    // Core identification
    GetInfo(ctx context.Context) (*MinerInfo, error)
    
    // Status and monitoring
    GetSummary(ctx context.Context) (*MinerSummary, error)
    GetPools(ctx context.Context) ([]PoolStatus, error)
    GetHardwareStatus(ctx context.Context) (*HardwareStatus, error)
    
    // Control operations
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Restart(ctx context.Context) error
    Pause(ctx context.Context) error
    Resume(ctx context.Context) error
    
    // Pool management
    AddPool(ctx context.Context, pool PoolConfig) error
    RemovePool(ctx context.Context, poolID int) error
    SwitchPool(ctx context.Context, poolID int) error
    EnablePool(ctx context.Context, poolID int) error
    DisablePool(ctx context.Context, poolID int) error
    
    // Diagnostics
    GetErrors(ctx context.Context) ([]MinerError, error)
    IsHealthy(ctx context.Context) (bool, error)
}

// Unified data models
type MinerInfo struct {
    Model       string
    Firmware    string
    Version     string
    APIType     string // "cgminer", "vnish", "braiins"
    Hostname    string
}

type MinerSummary struct {
    // Status
    Status      MinerStatus // running, paused, tuning, stopped, error
    Uptime      time.Duration
    
    // Performance
    HashRate    float64
    HashUnit    string
    Efficiency  float64 // J/TH
    PowerUsage  float64 // Watts
    
    // Shares
    Accepted    int64
    Rejected    int64
    
    // Hardware
    BoardsTotal   int
    BoardsHealthy int
    Temperature   TemperatureInfo
    FanSpeed      []int
    
    // Errors
    HardwareErrors int64
    ErrorMessages  []string
    
    // Pool
    ActivePool    string
    LastShareTime time.Time
}

type MinerStatus string

const (
    StatusRunning MinerStatus = "running"
    StatusPaused  MinerStatus = "paused"
    StatusTuning  MinerStatus = "tuning"
    StatusStopped MinerStatus = "stopped"
    StatusError   MinerStatus = "error"
    StatusUnknown MinerStatus = "unknown"
)

type TemperatureInfo struct {
    BoardAvg    float64
    BoardMax    float64
    ChipAvg     float64
    ChipMax     float64
    Intake      float64
    Outlet      float64
}

type HardwareStatus struct {
    Boards      []BoardStatus
    TotalChips  int
    HealthyChips int
}

type BoardStatus struct {
    Index       int
    Status      string
    HashRate    float64
    Temperature float64
    ChipCount   int
    Errors      int64
}

type PoolStatus struct {
    ID          int
    URL         string
    User        string
    Status      string
    Priority    int
    Accepted    int64
    Rejected    int64
    LastShare   time.Time
}

type PoolConfig struct {
    URL      string
    User     string
    Password string
    Priority int
}

type MinerError struct {
    Level     string // "warning", "error", "critical"
    Component string // "board", "chip", "network", "pool"
    Message   string
    Timestamp time.Time
}
```

## API Detection Mechanism

```go
type APIDetector struct {
    timeout time.Duration
}

type DetectedAPI struct {
    Type        string // "cgminer", "vnish", "braiins", "unknown"
    Version     string
    Capabilities []string
}

func (d *APIDetector) DetectAPI(ctx context.Context, host string) (*DetectedAPI, error) {
    // Parallel detection with early termination on success
    results := make(chan *DetectedAPI, 3)
    ctx, cancel := context.WithTimeout(ctx, d.timeout)
    defer cancel()
    
    // Test 1: Try Vnish REST API (most feature-rich)
    go func() {
        if api := d.tryVnishAPI(ctx, host); api != nil {
            results <- api
        }
    }()
    
    // Test 2: Try Braiins gRPC (port 50051)
    go func() {
        if api := d.tryBraiinsAPI(ctx, host); api != nil {
            results <- api
        }
    }()
    
    // Test 3: Try CGMiner API (port 4028) - fallback
    go func() {
        // Small delay to prefer newer APIs if available
        time.Sleep(100 * time.Millisecond)
        if api := d.tryCGMinerAPI(ctx, host); api != nil {
            results <- api
        }
    }()
    
    // Return first successful detection
    select {
    case api := <-results:
        return api, nil
    case <-ctx.Done():
        return &DetectedAPI{Type: "unknown"}, nil
    }
}

func (d *APIDetector) tryVnishAPI(ctx context.Context, host string) *DetectedAPI {
    // Try GET /api/v1/system
    resp, err := httpGet(ctx, fmt.Sprintf("http://%s/api/v1/system", host))
    if err == nil && resp.StatusCode == 200 {
        // Parse response for version
        return &DetectedAPI{
            Type:    "vnish",
            Version: parseVnishVersion(resp),
            Capabilities: []string{"rest", "full_monitoring", "autotune", "advanced_control"},
        }
    }
    return nil
}

func (d *APIDetector) tryBraiinsAPI(ctx context.Context, host string) *DetectedAPI {
    // Try gRPC connection on port 50051
    conn, err := grpc.DialContext(ctx, fmt.Sprintf("%s:50051", host))
    if err == nil {
        defer conn.Close()
        // Try to get API version
        return &DetectedAPI{
            Type:    "braiins",
            Version: getBraiinsVersion(conn),
            Capabilities: []string{"grpc", "cgminer_compat", "autotune", "advanced_control"},
        }
    }
    return nil
}

func (d *APIDetector) tryCGMinerAPI(ctx context.Context, host string) *DetectedAPI {
    // Try CGMiner version command
    miner := cgminer.NewCGMiner(host, 4028, d.timeout)
    version, err := miner.Version()
    if err == nil {
        // Check if it's actually Braiins BOSminer (CGMiner-compatible)
        if strings.Contains(version.CGMiner, "BOSminer") {
            return &DetectedAPI{
                Type:    "braiins",
                Version: version.CGMiner,
                Capabilities: []string{"cgminer", "basic_control"},
            }
        }
        return &DetectedAPI{
            Type:    "cgminer",
            Version: version.CGMiner,
            Capabilities: []string{"cgminer", "basic_control"},
        }
    }
    return nil
}
```

## Client Factory Implementation

```go
type MinerClientFactory struct {
    cgminerTimeout time.Duration
    httpTimeout    time.Duration
    workers        int
}

func (f *MinerClientFactory) CreateClient(api *DetectedAPI, host string) (MinerClient, error) {
    switch api.Type {
    case "vnish":
        return NewVnishAdapter(host, f.httpTimeout), nil
    case "braiins":
        // Use gRPC if available, fallback to CGMiner API
        if contains(api.Capabilities, "grpc") {
            return NewBraiinsGRPCAdapter(host), nil
        }
        return NewBraiinsCGMinerAdapter(host, f.cgminerTimeout), nil
    case "cgminer":
        return NewCGMinerAdapter(host, f.cgminerTimeout), nil
    default:
        return nil, fmt.Errorf("unsupported API type: %s", api.Type)
    }
}
```

## Implementation Plan

### Phase 1: Core Infrastructure (Week 1)
1. Define unified interfaces and data models
2. Implement API detection mechanism
3. Create client factory

### Phase 2: Adapter Implementation (Week 2-3)
1. CGMiner adapter (existing code refactoring)
2. Vnish adapter (leverage existing client)
3. Braiins adapter (CGMiner subset initially)

### Phase 3: Summary Command Enhancement (Week 4)
1. Implement unified summary across all adapters
2. Add scan statistics (IPs scanned/responded)
3. Create enhanced output formatter for summary data

### Phase 4: Testing & Optimization (Week 5)
1. Unit tests for all adapters
2. Integration tests with mock servers
3. Performance optimization for large-scale scanning

## Summary Display Implementation

```go
type ScanSummary struct {
    IPsScanned   int
    IPsResponded int
    Miners       []MinerSummary
}

func DisplaySummary(summary *ScanSummary) {
    fmt.Printf("Scan Results\n")
    fmt.Printf("============\n")
    fmt.Printf("IPs Scanned: %d\n", summary.IPsScanned)
    fmt.Printf("IPs Responded: %d\n", summary.IPsResponded)
    fmt.Printf("\n")
    
    // Table headers
    headers := []string{
        "IP", "Status", "Model", "Firmware", "Power(W)", 
        "Efficiency", "Hashrate", "Boards", "Errors", "Uptime"
    }
    
    for _, miner := range summary.Miners {
        // Format each miner's data
        row := []string{
            miner.IP,
            string(miner.Status),
            miner.Model,
            miner.Firmware,
            fmt.Sprintf("%.0f", miner.PowerUsage),
            fmt.Sprintf("%.2f J/TH", miner.Efficiency),
            fmt.Sprintf("%.2f %s", miner.HashRate, miner.HashUnit),
            fmt.Sprintf("%d/%d", miner.BoardsHealthy, miner.BoardsTotal),
            formatErrors(miner),
            formatUptime(miner.Uptime),
        }
        // Display row
    }
}

func formatErrors(m MinerSummary) string {
    if m.HardwareErrors > 0 {
        return fmt.Sprintf("HW:%d", m.HardwareErrors)
    }
    if len(m.ErrorMessages) > 0 {
        return fmt.Sprintf("%d errors", len(m.ErrorMessages))
    }
    return "None"
}
```

## Key Benefits

1. **Single Interface**: One interface for all miner types
2. **Auto-Detection**: Automatically identifies API type
3. **Graceful Degradation**: Falls back to basic CGMiner if advanced APIs unavailable
4. **Future-Proof**: Easy to add new miner APIs
5. **Comprehensive Data**: Unified data model captures all important metrics
6. **Performance**: Concurrent detection and operation
7. **Maintainability**: Clear separation of concerns with adapters