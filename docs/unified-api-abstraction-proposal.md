# Unified Miner API Abstraction Layer Proposal

## Executive Summary

This proposal outlines a unified abstraction layer that can interface with CGMiner, VNish, and Braiins APIs, providing a consistent interface for common mining operations while supporting automatic API detection.

## API Comparison Matrix

| Feature | CGMiner API | VNish API | Braiins gRPC |
|---------|------------|-----------|--------------|
| Protocol | TCP Socket | REST/HTTP | gRPC |
| Port | 4028 | 80/443 | 50051 |
| Auth | None/IP-based | API Key | Username/Password |
| Format | JSON-RPC | JSON | Protobuf |
| Real-time data | Yes | Yes | Yes |
| Configuration | Limited | Full | Full |
| Firmware updates | No | Yes | Yes |

## Proposed Abstraction Layer

### Core Interface Definition

```go
package miner

import (
    "context"
    "time"
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
    
    // Control operations
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

// APIType represents the detected API type
type APIType string

const (
    APITypeCGMiner  APIType = "cgminer"
    APITypeVNish    APIType = "vnish"
    APITypeBraiins  APIType = "braiins"
    APITypeUnknown  APIType = "unknown"
)
```

### Unified Data Structures

```go
// UnifiedSummary contains all summary information across different APIs
type UnifiedSummary struct {
    // Basic Information
    IPAddress    string
    APIType      APIType
    Model        string
    Firmware     string
    Version      string
    Uptime       time.Duration
    
    // Mining Status
    Status       MinerStatus
    StatusDetail string
    
    // Performance Metrics
    HashRate     float64
    HashRateUnit string
    HashRate5s   float64
    HashRate1m   float64
    HashRate5m   float64
    HashRate15m  float64
    
    // Power & Efficiency
    PowerUsage   float64  // Watts
    Efficiency   float64  // J/TH
    
    // Hardware Health
    BoardsTotal  int
    BoardsActive int
    ChipsTotal   int
    ChipsActive  int
    
    // Temperature
    TempAvg      float64
    TempMax      float64
    TempMin      float64
    
    // Fans
    FanCount     int
    FanSpeedAvg  int // RPM
    
    // Work Statistics
    Accepted     int64
    Rejected     int64
    HWErrors     int64
    
    // Pool Information
    ActivePool   string
    PoolCount    int
    
    // Errors and Warnings
    Errors       []string
    Warnings     []string
    
    // Raw response for API-specific data
    RawResponse  interface{}
}

// MinerStatus represents the current mining status
type MinerStatus string

const (
    StatusRunning  MinerStatus = "running"
    StatusPaused   MinerStatus = "paused"
    StatusStopped  MinerStatus = "stopped"
    StatusTuning   MinerStatus = "tuning"
    StatusError    MinerStatus = "error"
    StatusOffline  MinerStatus = "offline"
)

// Device represents a mining device/board
type Device struct {
    Index        int
    Name         string
    Status       string
    Temperature  float64
    HashRate     float64
    Chips        int
    ChipsActive  int
    HWErrors     int64
    Frequency    int
}

// Pool represents a mining pool
type Pool struct {
    ID           int
    URL          string
    User         string
    Status       string
    Priority     int
    Accepted     int64
    Rejected     int64
    LastShare    time.Time
}
```

## API Detection Mechanism

### Proposed Detection Flow

```go
package detector

import (
    "context"
    "fmt"
    "net"
    "net/http"
    "time"
)

// DetectMinerAPI automatically detects the API type of a miner
func DetectMinerAPI(ctx context.Context, host string) (APIType, APIInfo, error) {
    // Parallel detection with timeout
    detectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    
    type result struct {
        apiType APIType
        info    APIInfo
        err     error
    }
    
    results := make(chan result, 3)
    
    // Try Braiins gRPC (port 50051)
    go func() {
        if info, err := detectBraiins(detectCtx, host); err == nil {
            results <- result{APITypeBraiins, info, nil}
        } else {
            results <- result{APITypeUnknown, APIInfo{}, err}
        }
    }()
    
    // Try VNish REST API (port 80/443)
    go func() {
        if info, err := detectVNish(detectCtx, host); err == nil {
            results <- result{APITypeVNish, info, nil}
        } else {
            results <- result{APITypeUnknown, APIInfo{}, err}
        }
    }()
    
    // Try CGMiner API (port 4028)
    go func() {
        if info, err := detectCGMiner(detectCtx, host); err == nil {
            results <- result{APITypeCGMiner, info, nil}
        } else {
            results <- result{APITypeUnknown, APIInfo{}, err}
        }
    }()
    
    // Collect results with priority
    var lastError error
    for i := 0; i < 3; i++ {
        select {
        case r := <-results:
            if r.apiType != APITypeUnknown {
                // Additional verification for dual-API support
                if r.apiType == APITypeBraiins || r.apiType == APITypeVNish {
                    // These might also expose CGMiner API
                    r.info.AlsoSupports = checkCGMinerCompat(ctx, host)
                }
                return r.apiType, r.info, nil
            }
            lastError = r.err
        case <-detectCtx.Done():
            return APITypeUnknown, APIInfo{}, fmt.Errorf("detection timeout")
        }
    }
    
    return APITypeUnknown, APIInfo{}, lastError
}

// Detection functions for each API type
func detectBraiins(ctx context.Context, host string) (APIInfo, error) {
    // Try gRPC connection on port 50051
    // Check for specific Braiins services
    conn, err := grpc.DialContext(ctx, fmt.Sprintf("%s:50051", host),
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithBlock())
    if err != nil {
        return APIInfo{}, err
    }
    defer conn.Close()
    
    // Try to get version without auth (some endpoints work)
    client := pb.NewMinerServiceClient(conn)
    resp, err := client.GetMinerDetails(ctx, &pb.GetMinerDetailsRequest{})
    if err == nil || strings.Contains(err.Error(), "unauthenticated") {
        return APIInfo{
            Type:     APITypeBraiins,
            Port:     50051,
            Protocol: "gRPC",
            Version:  resp.GetBosVersion(),
            Model:    resp.GetModel(),
        }, nil
    }
    
    return APIInfo{}, err
}

func detectVNish(ctx context.Context, host string) (APIInfo, error) {
    // Try HTTP endpoints that don't require auth
    client := &http.Client{Timeout: 3 * time.Second}
    
    // VNish specific endpoint
    resp, err := client.Get(fmt.Sprintf("http://%s/api/v1/info", host))
    if err != nil {
        return APIInfo{}, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode == 200 || resp.StatusCode == 401 {
        var info models.SystemInfo
        json.NewDecoder(resp.Body).Decode(&info)
        
        return APIInfo{
            Type:     APITypeVNish,
            Port:     80,
            Protocol: "HTTP/REST",
            Version:  info.Version,
            Model:    info.Model,
            RequiresAuth: resp.StatusCode == 401,
        }, nil
    }
    
    return APIInfo{}, fmt.Errorf("not VNish API")
}

func detectCGMiner(ctx context.Context, host string) (APIInfo, error) {
    // Try CGMiner API on port 4028
    conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:4028", host), 3*time.Second)
    if err != nil {
        return APIInfo{}, err
    }
    defer conn.Close()
    
    // Send version command
    _, err = conn.Write([]byte(`{"command":"version"}`))
    if err != nil {
        return APIInfo{}, err
    }
    
    // Read response
    buffer := make([]byte, 4096)
    conn.SetReadDeadline(time.Now().Add(2 * time.Second))
    n, err := conn.Read(buffer)
    if err != nil {
        return APIInfo{}, err
    }
    
    // Parse response
    var response map[string]interface{}
    if err := json.Unmarshal(buffer[:n], &response); err == nil {
        // Check for CGMiner signature fields
        if _, ok := response["VERSION"]; ok {
            return APIInfo{
                Type:     APITypeCGMiner,
                Port:     4028,
                Protocol: "TCP/JSON-RPC",
                Version:  extractVersion(response),
            }, nil
        }
    }
    
    return APIInfo{}, fmt.Errorf("not CGMiner API")
}

// APIInfo contains detected API information
type APIInfo struct {
    Type         APIType
    Port         int
    Protocol     string
    Version      string
    Model        string
    RequiresAuth bool
    AlsoSupports []APIType  // For miners that support multiple APIs
}
```

## Implementation Strategy

### Phase 1: Core Abstraction (Week 1-2)
1. Define unified interfaces and data structures
2. Implement API detection mechanism
3. Create factory pattern for API client creation

### Phase 2: API Adapters (Week 2-3)
1. Implement CGMiner adapter
2. Implement VNish adapter
3. Implement Braiins adapter
4. Ensure consistent error handling

### Phase 3: Unified Summary (Week 3-4)
1. Implement unified summary collection
2. Add status determination logic
3. Create formatted output for CLI

### Phase 4: Testing & Optimization (Week 4-5)
1. Integration tests with real miners
2. Performance optimization
3. Error handling improvements
4. Documentation

## Summary Display Implementation

```go
// SummaryDisplay formats the unified summary for CLI output
func DisplaySummary(results []UnifiedSummary) {
    // Statistics
    totalScanned := len(results)
    totalResponded := 0
    byStatus := make(map[MinerStatus]int)
    byFirmware := make(map[string]int)
    totalHashRate := 0.0
    totalPower := 0.0
    
    for _, r := range results {
        if r.Status != StatusOffline {
            totalResponded++
            byStatus[r.Status]++
            byFirmware[r.Firmware]++
            totalHashRate += r.HashRate15m
            totalPower += r.PowerUsage
        }
    }
    
    // Print summary header
    fmt.Printf("=== Mining Farm Summary ===\n")
    fmt.Printf("Scanned: %d | Responded: %d | Success Rate: %.1f%%\n\n",
        totalScanned, totalResponded, 
        float64(totalResponded)/float64(totalScanned)*100)
    
    // Status breakdown
    fmt.Printf("Status Breakdown:\n")
    fmt.Printf("  Running: %d\n", byStatus[StatusRunning])
    fmt.Printf("  Paused:  %d\n", byStatus[StatusPaused])
    fmt.Printf("  Stopped: %d\n", byStatus[StatusStopped])
    fmt.Printf("  Tuning:  %d\n", byStatus[StatusTuning])
    fmt.Printf("  Error:   %d\n\n", byStatus[StatusError])
    
    // Performance summary
    fmt.Printf("Total Hash Rate: %.2f TH/s\n", totalHashRate)
    fmt.Printf("Total Power:     %.0f kW\n", totalPower/1000)
    fmt.Printf("Avg Efficiency:  %.2f J/TH\n\n", totalPower/totalHashRate)
    
    // Detailed table
    w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
    fmt.Fprintln(w, "IP\tStatus\tModel\tFirmware\tHashrate\tPower\tEff.\tBoards\tErrors\tUptime")
    fmt.Fprintln(w, "--\t------\t-----\t--------\t--------\t-----\t----\t------\t------\t------")
    
    for _, r := range results {
        if r.Status == StatusOffline {
            fmt.Fprintf(w, "%s\tOFFLINE\t-\t-\t-\t-\t-\t-\t-\t-\n", r.IPAddress)
            continue
        }
        
        errors := "-"
        if len(r.Errors) > 0 {
            errors = fmt.Sprintf("%d", len(r.Errors))
        }
        
        fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%.1f TH/s\t%.0f W\t%.1f\t%d/%d\t%s\t%s\n",
            r.IPAddress,
            r.Status,
            r.Model,
            r.Firmware,
            r.HashRate15m,
            r.PowerUsage,
            r.Efficiency,
            r.BoardsActive,
            r.BoardsTotal,
            errors,
            formatUptime(r.Uptime),
        )
    }
    w.Flush()
}
```

## Benefits

1. **Unified Interface**: Single API for all miner types
2. **Automatic Detection**: No manual configuration needed
3. **Graceful Degradation**: Falls back to CGMiner API when available
4. **Future Proof**: Easy to add new miner APIs
5. **Type Safety**: Strongly typed Go interfaces
6. **Performance**: Parallel detection and operations
7. **Extensibility**: Plugin architecture for new miners

## Conclusion

This abstraction layer provides a robust foundation for managing heterogeneous mining farms with different firmware types, while maintaining backward compatibility with existing CGMiner-only implementations.