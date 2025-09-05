# CGMiner API CLI

A comprehensive command-line tool for interacting with CGMiner API endpoints across multiple miners simultaneously. Supports IP ranges in CIDR notation and range format, with multiple output formats including colored terminal output and JSON.

## Coming Soon

### Extended Firmware Support
In addition to the standard CGMiner API, support for custom firmware APIs is being developed:

- **Vnish Firmware**: Full support for Vnish-specific APIs including advanced tuning, performance profiles, and extended monitoring capabilities. Implementation complete, integration pending.
  
- **Braiins OS+**: Comprehensive gRPC-based API support for Braiins OS+ miners including power management, performance tuning, hashboard control, and system operations. Implementation complete, integration pending.

These implementations are currently available as standalone libraries in the codebase and will be integrated into the main CLI in a future release.

## Features

- **Multiple IP Format Support**:
  - CIDR notation (e.g., `192.168.1.0/24`, `10.0.0.0/20`)
  - IP ranges (e.g., `10.45.1.0-10.45.20.254`)
  - Single IPs (e.g., `192.168.1.100`)
  - Multiple ranges in a single command

- **Comprehensive CGMiner API Coverage**:
  - All read commands (summary, devs, pools, stats, version, config, etc.)
  - Pool management (add, remove, enable, disable, switch)
  - Miner control (restart, quit)
  - Statistics management (zero stats)
  - Custom command support for any API endpoint

- **Output Formats**:
  - **Color**: Beautiful colored terminal output with status indicators
  - **JSON**: Machine-readable JSON output (with pretty-print option)
  - **Table**: Structured table format for easy reading

- **Performance Features**:
  - Concurrent execution with configurable worker count
  - Configurable timeouts
  - Efficient IP range parsing
  - Bulk operations support

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

## Usage

### Basic Command Structure

```bash
miner-cli <command> -i <ip-range> [options]
```

### Global Options

- `-i, --ips`: IP ranges (can be specified multiple times)
- `-p, --port`: CGMiner API port (default: 4028)
- `-t, --timeout`: Connection timeout in seconds (default: 5)
- `-w, --workers`: Number of concurrent workers (default: 10)
- `-o, --output`: Output format: color, json, table (default: color)
- `-v, --verbose`: Verbose output

### Commands

#### Information Commands

```bash
# Get mining summary
miner-cli summary -i 192.168.1.0/24

# Get device information
miner-cli devs -i 10.0.0.1-10.0.0.50

# Get pool information
miner-cli pools -i 192.168.1.100 -i 192.168.1.101

# Get detailed statistics
miner-cli stats -i 192.168.1.0/28 -o json

# Get miner version
miner-cli version -i 10.45.0.0/20

# Get configuration
miner-cli config -i 192.168.1.0/24 -v
```

#### Pool Management

```bash
# Add a new pool
miner-cli addpool -i 192.168.1.0/24 \
  --url stratum+tcp://pool.example.com:3333 \
  --user myworker \
  --pass x

# Switch to pool ID 1
miner-cli switchpool -i 192.168.1.0/24 --pool 1

# Enable pool ID 2
miner-cli enablepool -i 192.168.1.0/24 --pool 2

# Disable pool ID 0
miner-cli disablepool -i 192.168.1.0/24 --pool 0

# Remove pool ID 3
miner-cli removepool -i 192.168.1.0/24 --pool 3
```

#### Miner Control

```bash
# Restart miners
miner-cli restart -i 192.168.1.0/24

# Stop miners (use with caution!)
miner-cli quit -i 192.168.1.100
```

#### Statistics Management

```bash
# Zero all statistics
miner-cli zero -i 192.168.1.0/24 --which all --all

# Zero specific statistics
miner-cli zero -i 192.168.1.0/24 --which bestshare
```

#### Custom Commands

```bash
# Execute any CGMiner API command
miner-cli custom -i 192.168.1.100 --cmd "asccount"

# Custom command with arguments
miner-cli custom -i 192.168.1.100 --cmd "asc" --args '{"parameter": "0"}'
```

#### Utility Commands

```bash
# List all available commands
miner-cli list

# Scan IP ranges for active miners
miner-cli scan -i 192.168.1.0/24 -i 10.0.0.0/24
```

### Examples

#### Query Multiple IP Ranges

```bash
miner-cli summary \
  -i 192.168.1.0/24 \
  -i 10.45.1.0/24 \
  -i 172.16.0.1-172.16.0.100 \
  -o color -v
```

#### JSON Output for Automation

```bash
miner-cli stats -i 192.168.1.0/24 -o json | jq '.[] | select(.Error == null)'
```

#### Fast Scanning with High Concurrency

```bash
miner-cli scan -i 10.0.0.0/20 -w 50 -t 2
```

#### Batch Pool Configuration

```bash
# Add backup pools to all miners
for pool in pool1.example.com pool2.example.com pool3.example.com; do
  miner-cli addpool -i 192.168.1.0/24 \
    --url "stratum+tcp://${pool}:3333" \
    --user myworker \
    --pass x
done
```

## Output Format Examples

### Color Output (Default)

```
=== CGMiner API Results ===
Total: 5 | Success: 4 | Failed: 1

✓ 192.168.1.100:4028 [summary]
  Duration: 125ms
  Elapsed: 3600
  MHS av: 13500.45
  Found Blocks: 2
  Hardware Errors: 15
  Utility: 4.35

✗ 192.168.1.101:4028 [summary]
  Error: connection timeout
  Duration: 5s

=== Summary ===
Command executed on 5 hosts
Success: 4 hosts responded successfully
Failed: 1 hosts failed
```

### JSON Output

```json
[
  {
    "ip": "192.168.1.100",
    "port": 4028,
    "command": "summary",
    "response": {
      "Elapsed": 3600,
      "MHS av": 13500.45,
      "Found Blocks": 2,
      "Hardware Errors": 15,
      "Utility": 4.35
    },
    "duration": "125ms"
  },
  {
    "ip": "192.168.1.101",
    "port": 4028,
    "command": "summary",
    "error": "connection timeout",
    "duration": "5s"
  }
]
```

### Table Output

```
IP              Port    Command    Status    Duration    Details
---             ----    -------    ------    --------    -------
192.168.1.100   4028    summary    Success   125ms       {"Elapsed":3600,"MHS av":13500.45...}
192.168.1.101   4028    summary    Failed    5s          connection timeout

Summary: Total=2, Success=1, Failed=1
```

## Performance Considerations

- **Workers**: Increase `-w` for faster execution on large IP ranges
- **Timeout**: Reduce `-t` for faster scanning when expecting many offline hosts
- **IP Ranges**: Use CIDR notation for continuous ranges for better performance
- **Output Format**: JSON output is fastest for large result sets

## Error Handling

The tool handles various error conditions gracefully:

- Connection timeouts
- Invalid API responses
- Network errors
- Invalid IP ranges
- Missing required parameters

Failed connections are reported but don't stop execution for other hosts.

## Security Notes

- This tool requires network access to CGMiner API ports (default 4028)
- Ensure CGMiner API access is properly secured in production environments
- Some commands (like `quit`) can stop miners - use with caution
- Consider firewall rules when scanning large IP ranges

## Development

### Building from Source

```bash
git clone https://github.com/sinkers/miner-cli
cd miner-cli
go mod download
go build -o miner-cli
```

### Running Tests

```bash
go test ./...
```

### Adding New Commands

1. Add the command to `GetAvailableCommands()` in `internal/client/cgminer.go`
2. Add command description to `GetCommandDescription()`
3. Implement the command logic in `executeJob()`
4. Add any required flags in `cmd/root.go`

## Troubleshooting

### Common Issues

1. **Connection Refused**: Ensure CGMiner API is enabled and accessible
2. **Timeout Errors**: Increase timeout with `-t` flag
3. **Permission Denied**: Some commands require API write access
4. **Invalid IP Range**: Check CIDR notation or range format

### Debug Mode

Use verbose mode for detailed output:

```bash
miner-cli summary -i 192.168.1.100 -v
```

## License

MIT License - See LICENSE file for details

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.