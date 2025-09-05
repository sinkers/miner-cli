# Vnish API Client

A comprehensive Go client library for interacting with the Vnish firmware REST API.

## Features

This client provides full coverage of the Vnish API including:

### System & Status
- System information and model details
- Mining status and performance summaries
- Hardware layout information

### Mining Operations
- Start/Stop/Restart mining
- Pause/Resume mining
- Switch between mining pools

### Hardware Monitoring
- Chain information and statistics
- Chip-level details
- Temperature monitoring
- Fan speed control

### Configuration Management
- Pool configuration
- Fan settings (auto/manual/immersion modes)
- Temperature thresholds
- Network settings
- Advanced settings (autotune, auto-restart, power modes)

### API Management
- API key creation and deletion
- Authentication verification

### Maintenance Operations
- Firmware updates
- Settings backup and restore
- Factory reset
- System reboot
- Log retrieval and management
- Metrics collection

### Additional Features
- Warranty activation/cancellation
- Notes management
- UI settings customization
- Miner lock/unlock
- LED blink for physical identification

## Installation

```go
import "github.com/sinkers/miner-cli/internal/vnish/client"
```

## Usage

### Creating a Client

```go
// Basic client
vnishClient := client.NewClient("10.45.3.1")

// Client with options
vnishClient := client.NewClient("10.45.3.1",
    client.WithAPIKey("your-api-key"),
    client.WithTimeout(30*time.Second),
    client.WithDebug(true),
)
```

### Example Operations

```go
ctx := context.Background()

// Get system information
info, err := vnishClient.GetInfo(ctx)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Miner: %s %s\n", info.Model, info.Version)

// Get mining summary
summary, err := vnishClient.GetSummary(ctx)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Hash Rate: %.2f %s\n", 
    summary.Performance.HashRate, 
    summary.Performance.HashRateUnit)

// Control mining
err = vnishClient.RestartMining(ctx)
if err != nil {
    log.Fatal(err)
}

// Switch pool
err = vnishClient.SwitchPool(ctx, 2) // Switch to pool ID 2
if err != nil {
    log.Fatal(err)
}

// Get chain information
chains, err := vnishClient.GetChains(ctx)
for _, chain := range chains {
    fmt.Printf("Chain %d: %s\n", chain.Index, chain.Status)
}
```

## Testing

### Unit Tests

Run the unit tests with mocked HTTP responses:

```bash
go test -v ./internal/vnish/client
```

### Integration Tests

Run integration tests against a real Vnish API:

```bash
# Using flags
go test -tags=integration -vnish.host=10.45.3.1 -vnish.apikey=your-key ./internal/vnish/client

# Using environment variables
export VNISH_HOST=10.45.3.1
export VNISH_API_KEY=your-api-key
go test -tags=integration ./internal/vnish/client
```

### Coverage

Generate a coverage report:

```bash
go test -coverprofile=coverage.out ./internal/vnish/client
go tool cover -html=coverage.out
```

## API Methods

### System Information
- `GetInfo()` - System information
- `GetModel()` - Model details
- `GetStatus()` - Current status
- `GetSummary()` - Mining summary
- `GetPerfSummary()` - Performance summary
- `GetLayout()` - Hardware layout
- `GetFactoryInfo()` - Factory information

### Mining Control
- `StartMining()` - Start mining
- `StopMining()` - Stop mining
- `RestartMining()` - Restart mining
- `PauseMining()` - Pause mining
- `ResumeMining()` - Resume mining
- `SwitchPool(poolID)` - Switch active pool

### Hardware Monitoring
- `GetChains()` - Chain information
- `GetChips()` - Chip details

### Configuration
- `GetSettings()` - Current settings
- `UpdateSettings(settings)` - Update settings
- `BackupSettings()` - Backup configuration
- `RestoreSettings(backup)` - Restore configuration
- `FactoryReset()` - Factory reset

### Autotune
- `GetAutotunePresets()` - Available presets
- `ResetAutotune(chainIndex)` - Reset specific chain
- `ResetAllAutotune()` - Reset all chains

### API Keys
- `GetAPIKeys()` - List API keys
- `AddAPIKey(name)` - Create new API key
- `DeleteAPIKey(id)` - Delete API key
- `CheckAuth()` - Verify authentication

### Logs & Metrics
- `GetLogs(logType)` - Retrieve logs
- `ClearLogs(logType)` - Clear logs
- `GetMetrics()` - Get metrics data

### Notes
- `GetNotes()` - List all notes
- `GetNote(id)` - Get specific note
- `CreateNote(content)` - Create note
- `UpdateNote(id, content)` - Update note
- `DeleteNote(id)` - Delete note

### System Operations
- `Reboot()` - Reboot system
- `FindMiner(blink)` - LED identification
- `Lock(password)` - Lock miner
- `Unlock(password)` - Unlock miner

### Firmware
- `UpdateFirmware(request)` - Update firmware
- `RemoveFirmware()` - Remove custom firmware

### Warranty
- `ActivateWarranty()` - Activate warranty
- `CancelWarranty()` - Cancel warranty

## Error Handling

The client provides detailed error messages for:
- Network connectivity issues
- Authentication failures
- Invalid requests
- Server errors
- Context cancellation/timeout

All methods return an error that should be checked:

```go
info, err := vnishClient.GetInfo(ctx)
if err != nil {
    // Handle error
    log.Printf("Failed to get info: %v", err)
    return
}
```

## Context Support

All methods accept a context for cancellation and timeout:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

info, err := vnishClient.GetInfo(ctx)
```

## Debug Mode

Enable debug output to see HTTP requests and responses:

```go
vnishClient := client.NewClient("10.45.3.1", 
    client.WithDebug(true))
```

## Thread Safety

The client is thread-safe and can be used concurrently from multiple goroutines.