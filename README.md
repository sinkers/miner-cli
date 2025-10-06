# Miner CLI

A comprehensive command-line tool for managing fleets of Bitcoin miners, PDUs, and managed switches. Supports multiple firmware types, IP ranges in CIDR notation, and provides real-time monitoring with multiple output formats.

## Overview

Miner CLI is a unified management interface designed for Bitcoin mining operations of any scale. It provides a single tool to:

- **Manage Bitcoin Miners**: Support for CGMiner API, Braiins OS, Vnish firmware, and more
- **Monitor Performance**: Real-time hashrate, temperature, power consumption, and tuning status
- **Control Infrastructure**: PDU power management and network switch configuration (coming soon)
- **Fleet Operations**: Execute commands across hundreds of devices simultaneously

## Features

### Multi-Firmware Support

- **CGMiner API**: Full support for standard CGMiner-compatible miners
- **Braiins OS/OS+**: Native integration with Braiins GraphQL API for advanced monitoring
  - Firmware version detection
  - Real-time power consumption (kW)
  - Chip temperature monitoring
  - Autotuner status tracking
- **Vnish Firmware**: Advanced tuning and performance profiles (library available, integration pending)
- **Stock Firmware**: Fallback support for standard miners

### Intelligent Scanning

The `scan` command provides comprehensive fleet visibility:

```bash
miner-cli scan 10.45.9.0/24
```

**Output includes:**
- IP address and firmware version (e.g., Braiins OS+ 25.01, Braiins OS+ 25.07)
- Switch port (when `--switch` flag is used)
- Real-time hashrate (GH/s)
- Power consumption (kW) - for Braiins OS miners
- Chip temperature (°C)
- Autotuner status (STABLE, TUNING, TESTING)
- Share statistics (Accepted, Rejected, HW Errors)
- Uptime

### Switch Port Mapping

Integrate with Cisco switches via SNMP to show which port each miner is connected to:

```bash
miner-cli scan 192.168.1.0/24 --switch 10.110.101.6 --community public
```

**Features:**
- Queries switch MAC address table via SNMP v2c
- Matches miner MAC addresses to switch ports
- Works with same-subnet deployments (ARP-based)
- For Braiins OS miners on different subnets, use gRPC authentication:
  ```bash
  miner-cli scan 10.45.6.0/24 --switch 10.110.101.6 --braiins-user root --braiins-pass root
  ```
- Displays port names (e.g., FastEthernet0/17, GigabitEthernet1/0/24)
- Supports Cisco VLAN-aware MAC lookup

### IP Range Support

- **CIDR notation**: `192.168.1.0/24`, `10.0.0.0/20`
- **IP ranges**: `10.45.1.0-10.45.20.254`
- **Single IPs**: `192.168.1.100`
- **Multiple ranges**: Combine any format in a single command

### Output Formats

- **Color**: Beautiful colored terminal output with status indicators
- **JSON**: Machine-readable output for automation and scripting
- **Table**: Structured format for easy reading

### Performance Features

- Concurrent execution with configurable worker pools (default: 255 workers)
- Automatic firmware detection via API probing
- Configurable timeouts for fast scanning
- Efficient bulk operations

## Installation

```bash
cd miner-cli
go mod download
go build -o miner-cli
```

Or install directly:

```bash
go install github.com/sinkers/miner-cli@latest
```

Or use the Makefile:

```bash
make install        # Install to $GOPATH/bin
make build          # Build binary only
make cross-compile  # Build for multiple platforms
```

## Quick Start

### Scan Your Fleet

```bash
# Scan a subnet and show all miner details
miner-cli scan 192.168.1.0/24

# Scan with verbose output
miner-cli scan 10.0.0.0/20 -v

# JSON output for automation
miner-cli scan 192.168.1.0/24 -o json

# Scan with switch port mapping (requires SNMP access to switch)
miner-cli scan 192.168.1.0/24 --switch 10.110.101.6 --community public

# With Braiins authentication for MAC address retrieval
miner-cli scan 192.168.1.0/24 --switch 10.110.101.6 --braiins-user root --braiins-pass root
```

### Get Mining Summary

```bash
# Summary for specific miners
miner-cli summary 192.168.1.100 192.168.1.101

# Summary for subnet with JSON output
miner-cli summary 192.168.1.0/24 -o json

# Multiple ranges
miner-cli summary 192.168.1.0/24 10.45.0.0/24
```

### Monitor Devices

```bash
# Get device/hashboard information
miner-cli devs 192.168.1.0/24

# Get detailed statistics
miner-cli stats 10.0.0.1-10.0.0.50 -o json

# Check pool status
miner-cli pools 192.168.1.0/24
```

## Command Reference

### Information Commands

| Command | Description |
|---------|-------------|
| `scan` | Scan network for miners with comprehensive status |
| `summary` | Get mining summary statistics |
| `devs` | Get device/hashboard information |
| `pools` | Get pool configuration and status |
| `stats` | Get detailed mining statistics |
| `version` | Get miner firmware version |
| `config` | Get miner configuration |

### Pool Management

```bash
# Add a new pool
miner-cli addpool 192.168.1.0/24 \
  --url stratum+tcp://pool.example.com:3333 \
  --user myworker \
  --pass x

# Switch to pool ID 1
miner-cli switchpool 192.168.1.0/24 --pool 1

# Enable/disable pools
miner-cli enablepool 192.168.1.0/24 --pool 2
miner-cli disablepool 192.168.1.0/24 --pool 0

# Remove a pool
miner-cli removepool 192.168.1.0/24 --pool 3
```

### Miner Control

```bash
# Restart miners
miner-cli restart 192.168.1.0/24

# Stop miners (use with caution!)
miner-cli quit 192.168.1.100

# Zero statistics
miner-cli zero 192.168.1.0/24 --which all --all
```

### Custom Commands

```bash
# Execute any CGMiner API command
miner-cli custom 192.168.1.100 --cmd "asccount"

# List all available commands
miner-cli list
```

## Global Options

- `-i, --ips`: IP ranges (can be specified multiple times)
- `-p, --port`: API port (default: 4028)
- `-t, --timeout`: Connection timeout in seconds (default: 5)
- `-w, --workers`: Number of concurrent workers (default: 255)
- `-o, --output`: Output format: color, json, table (default: color)
- `-v, --verbose`: Verbose output

## Example: Fleet Monitoring

Monitor a large mining fleet across multiple subnets:

```bash
# Scan entire facility
miner-cli scan 10.45.0.0/16 -w 500 -t 2

# Get performance summary with JSON output for analysis
miner-cli summary 10.45.0.0/16 -o json | jq '.[] | select(.Error == null)'

# Monitor specific temperature ranges
miner-cli scan 192.168.1.0/24 -o json | \
  jq '.[] | select(.response.chip_temp_c > 75)'
```

## Example: Automated Pool Switching

```bash
# Switch all miners to failover pool
miner-cli switchpool 10.45.0.0/20 --pool 1 -w 100

# Verify pool configuration
miner-cli pools 10.45.0.0/20 -o json | \
  jq '.[] | {ip: .ip, active_pool: .response.POOLS[0].URL}'
```

## Scan Output Example

```
Scanning 255 hosts for active miners...

Active Miners Found: 23 out of 255 scanned
================================================================================
IP          Firmware           Hashrate (GH/s)  Power (kW)  Temp (°C)  Tuning  Accepted  Rejected  HW Errors  Uptime
---         --------           --------------   ----------  ---------  ------  --------  --------  ---------  ------
10.45.9.1   Braiins OS+ 25.01  110776.91        3.28        62.0       STABLE  880       3         0          4h 41m
10.45.9.2   Braiins OS+ 25.01  111639.79        3.29        62.5       STABLE  1600      8         0          4h 41m
10.45.9.3   Braiins OS+ 25.07  102588.56        3.06        62.0       STABLE  1456      6         0          4h 41m
...
================================================================================
```

## Performance Tuning

### Fast Scanning

```bash
# High concurrency for large networks
miner-cli scan 10.0.0.0/16 -w 500 -t 2

# Reduce timeout for fast detection
miner-cli scan 192.168.1.0/24 -t 1
```

### Batch Operations

```bash
# Parallel pool updates
for pool in pool1.example.com pool2.example.com pool3.example.com; do
  miner-cli addpool 192.168.1.0/24 \
    --url "stratum+tcp://${pool}:3333" \
    --user myworker \
    --pass x
done
```

## Architecture

### Unified API Abstraction (In Development)

The project includes a unified API abstraction layer for seamless integration across different miner types:

- **CGMiner Adapter**: Standard CGMiner API support
- **Braiins Adapter**: gRPC-based Braiins OS integration
- **Vnish Adapter**: Vnish firmware API support

See `docs/unified-api-abstraction-*.md` for implementation details.

### Current Implementation

- **Direct API Integration**: The scan command uses direct GraphQL/HTTP APIs for optimal performance
- **Worker Pool Architecture**: Configurable concurrency for handling large fleets
- **Automatic Firmware Detection**: Queries Braiins GraphQL API to detect firmware type
- **Fallback Support**: Standard CGMiner API fallback for non-Braiins miners

## Development

### Building from Source

```bash
git clone https://github.com/sinkers/miner-cli
cd miner-cli
make build
```

### Running Tests

```bash
make test              # Run tests
make test-coverage     # Generate coverage report
make test-verbose      # Detailed test output
```

### Development Tools

```bash
make dev-setup    # Install development tools
make fmt          # Format code
make lint         # Run linter
make check        # Run all checks
```

### CI/CD

The project uses GitHub Actions for:
- Automated testing with race detection
- Cross-platform builds (Linux, macOS, Windows)
- Security scanning with Gosec and govulncheck
- Code coverage reporting

## Roadmap

### Upcoming Features

- **PDU Management**: Power distribution unit control and monitoring
- **Switch Management**: Network switch configuration and VLAN management
- **Advanced Analytics**: Historical performance tracking and alerting
- **Web Dashboard**: Real-time fleet visualization
- **Firmware Management**: Automated firmware updates and rollback

## Security Considerations

- This tool requires network access to miner API ports (default 4028)
- Braiins GraphQL API access (port 80/443) for advanced features
- Ensure API access is properly secured in production
- Some commands (like `quit`) can stop miners - use with caution
- Consider firewall rules and network segmentation for large deployments

## Troubleshooting

### Common Issues

**Connection Issues:**
- Ensure CGMiner API is enabled (`--api-listen` in cgminer config)
- Check firewall rules allow access to port 4028
- For Braiins: Verify HTTP/GraphQL access on port 80

**Timeout Errors:**
- Increase timeout: `-t 10`
- Reduce worker count if network is saturated: `-w 50`

**Missing Temperature Data:**
- Braiins OS miners: Temperature fetched from GraphQL API automatically
- Stock firmware: Use `--scan-temps` flag (deprecated, now automatic)

### Debug Mode

```bash
miner-cli scan 192.168.1.0/24 -v
```

## Contributing

Pull requests are welcome! For major changes, please open an issue first.

### Development Guidelines

1. Follow existing code style (use `make fmt`)
2. Add tests for new features
3. Update documentation
4. Ensure CI passes

## License

MIT License - See LICENSE file for details

## Support

- GitHub Issues: https://github.com/sinkers/miner-cli/issues
- Documentation: `docs/` directory in repository

## Acknowledgments

- CGMiner API by Con Kolivas
- Braiins OS by Braiins Systems
- Vnish firmware team
