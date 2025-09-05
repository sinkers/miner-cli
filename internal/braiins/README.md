# Braiins OS+ gRPC Client Library

## Overview

This package provides a Go client library for interacting with Braiins OS+ miners via the gRPC API. The implementation is based on the official [Braiins OS+ API](https://github.com/braiins/bos-plus-api) and supports comprehensive mining operations.

## Features

### Implemented Services
- **Authentication**: Token-based authentication
- **Miner Information**: Hardware details, versions, uptime
- **Statistics**: Real-time hashrate, power consumption, efficiency
- **Hashboard Management**: Individual board monitoring
- **Pool Configuration**: View and modify pool settings
- **Performance Tuning**: Power and hashrate targets
- **Mining Control**: Start, stop, pause, resume operations
- **Cooling Management**: Fan speeds and temperature monitoring
- **System Operations**: Reboot, restart mining
- **Configuration**: View and modify miner settings

## Architecture

```
internal/braiins/
├── client/
│   ├── client_simple.go      # Main client implementation
│   └── integration_test.go   # Integration tests
├── models/
│   └── simple_models.go      # Simplified data models
├── proto/                    # Original protobuf definitions
│   ├── bos/
│   │   └── v1/
│   │       ├── *.proto      # Protocol buffer definitions
│   │       └── generate.sh  # Code generation script
└── bos/                      # Generated Go code
    └── v1/
        ├── *.pb.go           # Generated message types
        └── *_grpc.pb.go      # Generated gRPC services
```

## Usage

### Basic Connection

```go
import "github.com/sinkers/miner-cli/internal/braiins/client"

// Create client
client := client.NewSimpleBraiinsClient("192.168.1.100", 50051)

// Connect and authenticate
err := client.Connect("root", "root")
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

### Read Operations

```go
// Get miner information
details, err := client.GetMinerDetails()
fmt.Printf("Model: %s\n", details.Model)
fmt.Printf("Hostname: %s\n", details.Hostname)

// Get statistics
stats, err := client.GetMinerStats()
fmt.Printf("Hashrate: %.2f TH/s\n", stats.HashRate15m)
fmt.Printf("Power: %.0f W\n", stats.PowerUsage)
fmt.Printf("Efficiency: %.2f J/TH\n", stats.Efficiency)

// Get hashboards
hashboards, err := client.GetHashboards()
for _, board := range hashboards {
    fmt.Printf("Board %d: %d chips, %.1f°C\n", 
        board.Index, board.Chips, board.Temperature)
}
```

### Write Operations

```go
// Set power target
err = client.SetPowerTarget(3000) // 3000W

// Set hashrate target  
err = client.SetHashrateTarget(100.0) // 100 TH/s

// Control mining
err = client.StopMining()
err = client.StartMining()
err = client.PauseMining()
err = client.ResumeMining()
```

### System Operations

```go
// Restart mining process
err = client.RestartMining()

// Reboot system (use with caution)
err = client.Reboot()
```

## Testing

### Unit Tests

```bash
cd internal/braiins/client
go test -v
```

### Integration Tests

A comprehensive integration test suite is available:

```bash
# Build test binary
go build -o test-braiins cmd/test-braiins/main.go

# Run safe tests (read-only)
./test-braiins -host 192.168.1.100

# Run with write operations
./test-braiins -host 192.168.1.100 \
  -skip-write=false \
  -skip-pause=false \
  -wait=30s

# Run all tests (including reboot - DANGEROUS)
./test-braiins -host 192.168.1.100 \
  -skip-write=false \
  -skip-pause=false \
  -skip-restart=false \
  -skip-reboot=false \
  -wait=120s
```

### Test Categories

1. **Connection & Authentication** - Basic connectivity
2. **Read Operations** - Safe, non-disruptive queries
3. **Pool Operations** - Pool configuration (optional)
4. **Performance Operations** - Power/hashrate targets (optional)
5. **Mining Control** - Pause/resume operations (optional)
6. **System Operations** - Restart/reboot (optional, dangerous)

## Requirements

- Braiins OS+ version 23.03.1 or newer (gRPC API support)
- Network access to port 50051
- Valid credentials (default: root/root)

## Protocol Buffer Generation

To regenerate the Go code from protobuf definitions:

```bash
cd internal/braiins/proto
./generate.sh
```

This requires:
- protoc compiler
- protoc-gen-go plugin
- protoc-gen-go-grpc plugin

## Error Handling

The client includes comprehensive error handling:
- Connection timeouts (default: 10 seconds)
- Authentication failures
- Network errors
- Invalid parameter validation
- Graceful degradation for unavailable features

## Security Considerations

- Always use TLS in production (when available)
- Avoid hardcoding credentials
- Implement rate limiting for API calls
- Monitor for unusual activity
- Test disruptive operations on non-production miners first

## Compatibility

Tested with:
- Antminer S19, S19 Pro, S19j Pro
- Braiins OS+ versions 23.x, 24.x, 25.x
- Go 1.19+

## Future Enhancements

- [ ] TLS support for secure connections
- [ ] Connection pooling for multiple miners
- [ ] Batch operations for fleet management
- [ ] WebSocket support for real-time updates
- [ ] Metrics collection and export
- [ ] Integration with main miner-cli tool

## License

This implementation follows the same license as the parent miner-cli project.