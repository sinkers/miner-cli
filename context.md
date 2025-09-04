# miner-cli Codebase Context

## MODULE ARCHITECTURE

### Core Components

- **main.go** - Application entry point that initializes and delegates to cmd package
- **cmd/root.go** - Cobra CLI command definitions and routing using command pattern
  - Handles command-line parsing and flag management
  - Dispatches to appropriate client functions

### Client Modules

#### CGMiner Client (`internal/client/cgminer.go`)
- Original CGMiner API client for managing ASIC miners
- Implements worker pool pattern for concurrent operations
- Supports multiple IP addresses/ranges simultaneously
- Commands: devs, pools, summary, stats, version, restart, addpool, switchpool, enable, disable, delete

#### Vnish Client (`internal/vnish/`)
- New comprehensive vnish firmware API client (separate branch)
- Full-featured REST API implementation
- **client/client.go** - Main client implementation
  - HTTP client with authentication support
  - Context-aware operations with cancellation
  - Comprehensive error handling
- **models/models.go** - Data models matching vnish OpenAPI spec
  - Complete type definitions for all API endpoints
  - JSON serialization/deserialization
- **client/client_test.go** - Unit tests with mocked HTTP server
- **client/integration_test.go** - Integration tests for real vnish APIs

### Utilities

- **internal/iprange/parser.go** - IP range parsing and expansion
  - Supports CIDR notation (192.168.1.0/24)
  - Supports IP ranges (10.0.0.1-10.0.0.50)
  - Validates and expands to individual IPs

- **internal/output/formatter.go** - Output formatting strategies
  - Color formatter for terminal display
  - JSON formatter for machine-readable output
  - Table formatter for structured data

## DATA FLOW

1. **Command Execution Flow**:
   - User command → cmd/root.go → Parse flags/args
   - Determine operation type (CGMiner or vnish)
   - Parse IP ranges → Create client → Execute command
   - Format results → Display output

2. **CGMiner Client Flow**:
   - Create worker pool (default 255 workers)
   - Queue jobs for each IP address
   - Workers execute CGMiner API calls concurrently
   - Collect results → Format → Display

3. **Vnish Client Flow**:
   - Initialize HTTP client with authentication
   - Make REST API calls with proper headers
   - Parse JSON responses into Go structs
   - Return typed results for processing

## DEPENDENCIES

### External Libraries
- `github.com/spf13/cobra` - CLI framework
- `github.com/fatih/color` - Terminal color output
- Standard library: net, http, encoding/json, context

### Internal Dependencies
- cmd → internal/client (CGMiner operations)
- cmd → internal/vnish (vnish operations)
- cmd → internal/iprange (IP parsing)
- cmd → internal/output (formatting)
- vnish/client → vnish/models (data types)

## ENTRY POINTS

### Main Execution
- `miner-cli` - Primary command-line interface
- Subcommands defined in cmd/root.go

### Test Execution
- `go test ./...` - Run all unit tests
- `go test -tags=integration ./internal/vnish/client` - Run vnish integration tests

## CONFIGURATION

### Command-line Flags
- `--ips` - Target IP addresses/ranges
- `--port` - CGMiner API port (default 4028)
- `--timeout` - Operation timeout
- `--workers` - Concurrent worker count
- `--format` - Output format (color, json, table)
- `--command` - CGMiner command to execute

### Environment Variables
- `VNISH_HOST` - Default vnish API host for testing
- `VNISH_API_KEY` - API key for vnish authentication

## CORE LOGIC

### CGMiner Client
- Manages pool of goroutines for concurrent miner communication
- Implements CGMiner RPC protocol over TCP
- Handles connection timeouts and error recovery
- Supports all standard CGMiner API commands

### Vnish Client
- Full REST API implementation for vnish firmware
- Supports all vnish operations:
  - Mining control (start, stop, restart, pause, resume)
  - Pool management
  - Hardware monitoring (chains, chips, temperature)
  - Settings management
  - Firmware updates
  - API key management
  - Autotune operations
  - Logging and metrics
- Context-aware with proper cancellation
- Comprehensive error handling and retry logic

### IP Range Parser
- Expands CIDR blocks to individual IPs
- Validates IP addresses and ranges
- Handles edge cases and malformed input

### Output Formatters
- Adapts output based on data type and user preference
- Provides consistent formatting across different commands
- Supports both human-readable and machine-parseable formats

## BUILD & DEPLOYMENT

### Building
```bash
make build        # Build binary
make install      # Install to $GOPATH/bin
make cross-compile # Build for multiple platforms
```

### Testing
```bash
make test         # Run unit tests
make test-coverage # Generate coverage report
make test-benchmark # Run benchmarks
```

### CI/CD
- GitHub Actions workflow for automated testing
- Cross-platform builds (Linux, macOS, Windows)
- Security scanning with gosec and govulncheck