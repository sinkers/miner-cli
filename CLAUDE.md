# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

The project includes a comprehensive Makefile with common development tasks:

```bash
# Quick development cycle
make quick              # Format, test, and build
make                    # Full build pipeline (deps, fmt, vet, test, build)

# Building
make build             # Build the binary
make install           # Install to GOPATH/bin
make cross-compile     # Build for common platforms

# Testing
make test              # Run all tests
make test-coverage     # Run tests with coverage report
make test-benchmark    # Run benchmark tests
make test-verbose      # Run tests with verbose output

# Code Quality
make fmt               # Format code
make vet               # Run go vet
make lint              # Run golangci-lint (if installed)
make check             # Run all quality checks

# Dependencies
make deps              # Download dependencies
make tidy              # Clean up go.mod

# Development
make dev-setup         # Install development tools
make clean             # Clean build artifacts
make help              # Show all available commands
```

Manual commands (without Makefile):
```bash
go build -o miner-cli   # Build
go test ./...          # Test
go mod download        # Dependencies
```

## Architecture Overview

This is a Go-based CLI tool for managing CGMiner instances across multiple IP addresses concurrently. The architecture follows a modular design:

### Core Components

- **main.go** - Entry point that delegates to cmd package
- **cmd/root.go** - Cobra CLI command definitions and routing using the command pattern
- **internal/client/cgminer.go** - CGMiner API client with worker pool for concurrent execution
- **internal/iprange/parser.go** - IP range parsing supporting CIDR notation and ranges
- **internal/output/formatter.go** - Multiple output formatters (color, JSON, table)

### Key Design Patterns

- **Worker Pool Pattern**: Uses configurable number of goroutines for concurrent miner communication
- **Command Pattern**: Each CGMiner API command is handled as a discrete operation
- **Strategy Pattern**: Multiple output formatters selected at runtime
- **Builder Pattern**: Commands built dynamically with parameters

### Concurrency Model

The tool uses a worker pool architecture:
- Jobs are queued for each IP address
- Workers process jobs concurrently (default: 255 workers)
- Results are collected and formatted after all workers complete
- Context cancellation supports timeouts

### IP Range Parsing

Supports multiple input formats:
- Single IPs: `192.168.1.100`
- CIDR notation: `192.168.1.0/24`
- IP ranges: `10.0.0.1-10.0.0.50`
- Multiple ranges can be combined in a single command

### Command Extension

To add new CGMiner API commands:
1. Add command name to `GetAvailableCommands()` in `internal/client/cgminer.go:267`
2. Add description to `GetCommandDescription()` in `internal/client/cgminer.go:285`
3. Implement command logic in `executeJob()` switch statement in `internal/client/cgminer.go:105`
4. Add required flags in `cmd/root.go:88` if needed

### Error Handling Strategy

- Network errors don't stop execution for other hosts
- Each result includes success/failure status and timing information
- Context cancellation prevents hanging on slow/dead hosts
- Graceful degradation when miners are unreachable

## Testing

The project includes comprehensive test suites for all major components:

### Test Structure
- **internal/iprange/parser_test.go** - IP range parsing tests including CIDR, ranges, and edge cases
- **internal/client/cgminer_test.go** - CGMiner client functionality, parameter validation, and concurrency
- **internal/output/formatter_test.go** - Output formatter tests for JSON, color, and table formats

### Running Tests
```bash
make test              # Standard test run
make test-coverage     # Generate coverage report (creates coverage.html)
make test-benchmark    # Performance benchmarks
make test-verbose      # Detailed test output with race detection
```

### Test Coverage
Tests cover:
- IP parsing edge cases (invalid IPs, malformed CIDR, range validation)
- Client parameter validation and error handling
- Output formatting across different data structures
- Concurrency and worker pool behavior
- Benchmark tests for performance-critical paths

## Continuous Integration

The project uses GitHub Actions for automated CI/CD:

### Workflows

- **CI Pipeline** (`.github/workflows/ci.yml`) - Runs on all pushes and PRs:
  - Tests with race detection and coverage reporting
  - Cross-platform builds (Linux, macOS, Windows)
  - Code linting with golangci-lint
  - Security scanning with Gosec and govulncheck
  - Uploads coverage reports to Codecov

- **Release Pipeline** (`.github/workflows/release.yml`) - Runs on version tags:
  - Creates GitHub releases automatically
  - Builds binaries for all supported platforms
  - Packages releases as archives (tar.gz/zip)

### Local CI Commands
```bash
make check             # Run all CI checks locally
make lint              # Run linter (requires golangci-lint)
make cross-compile     # Test cross-platform builds
```

### Installation

After installing via `make install`, the binary is available system-wide:
```bash
make install           # Install to $GOPATH/bin
miner-cli --version    # Verify installation
miner-cli --help       # Show usage
```