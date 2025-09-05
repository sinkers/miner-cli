# Braiins OS+ Integration Test Suite

## Overview

This directory contains a comprehensive integration test suite for Braiins OS+ miners using the gRPC API.

## Prerequisites

1. **Braiins OS+ Version**: The miner must be running Braiins OS+ version 23.03.1 or newer (gRPC API enabled by default)
2. **Network Access**: The test machine must have network access to the miner on port 50051
3. **Credentials**: Default credentials are root/root, but can be configured

## Files

- `internal/braiins/client/` - Braiins gRPC client implementation
- `internal/braiins/models/` - Data models for API responses  
- `cmd/test-braiins/main.go` - Standalone test program
- `test-braiins` - Compiled test binary
- `test_braiins.sh` - Convenience wrapper script
- `diagnose_braiins.sh` - Diagnostic script to check connectivity

## Running Tests

### Quick Start (Safe Mode - Read Only)

```bash
# Run read-only tests against default host (10.45.3.1)
./test-braiins

# Run against a specific host
./test-braiins -host 192.168.1.100

# With custom credentials
./test-braiins -host 192.168.1.100 -user admin -pass mypassword
```

### Advanced Usage

```bash
# Enable specific test categories
./test-braiins -host 192.168.1.100 \
  -skip-write=false \     # Enable pool and performance tests
  -skip-pause=false \      # Enable pause/resume mining test
  -skip-restart=false \    # Enable mining restart test
  -wait=60s               # Wait 60s after disruptive operations

# Run ALL tests including reboot (DANGEROUS!)
./test-braiins -host 192.168.1.100 \
  -skip-write=false \
  -skip-pause=false \
  -skip-restart=false \
  -skip-reboot=false \
  -wait=120s
```

### Using the Shell Script Wrapper

```bash
# Safe mode (default)
./test_braiins.sh --host 192.168.1.100

# Enable write operations
./test_braiins.sh --host 192.168.1.100 --enable-write

# Unsafe mode (all non-reboot tests)
./test_braiins.sh --host 192.168.1.100 --unsafe

# With custom wait time
./test_braiins.sh --host 192.168.1.100 --wait 60s
```

## Test Categories

### 1. Connection & Authentication
- Tests basic connectivity to gRPC port
- Tests authentication with provided credentials

### 2. Read-Only Operations (Safe)
- **Get Miner Details**: Hardware info, versions, uptime
- **Get Miner Statistics**: Hashrate, power, efficiency, shares
- **Get Hashboards**: Individual board status and performance
- **Get Pool Groups**: Current pool configuration
- **Get Cooling State**: Fan speeds, temperatures
- **Get Tuner State**: Performance tuning mode
- **Get License State**: License status
- **Get Configuration**: Miner configuration

### 3. Pool Operations (Skip by default)
- List current pool groups
- Modify pool configurations (if implemented)

### 4. Performance Operations (Skip by default)
- Set power target
- Set hashrate target
- Verify settings applied

### 5. Mining Control (Skip by default)
- **Pause/Resume**: Temporarily pause and resume mining
- **Restart Mining**: Restart the mining process
- Includes wait periods and verification

### 6. System Operations (Skip by default)
- **Reboot**: Full system reboot with reconnection polling
- Waits for miner to come back online
- Verifies successful restart

## Diagnostics

If tests fail to connect, run the diagnostic script:

```bash
./diagnose_braiins.sh 192.168.1.100
```

This will check:
- Network connectivity
- Port availability (SSH, HTTP, HTTPS, gRPC, CGMiner API)
- Web interface accessibility

## Common Issues

### Connection Refused on Port 50051

**Possible Causes:**
1. **Old Firmware**: gRPC API requires Braiins OS+ 23.03.1+
2. **Firewall**: Port may be blocked by firewall rules
3. **API Disabled**: Check miner configuration
4. **Network Isolation**: Ensure test machine can reach miner network

**Solutions:**
1. Update to latest Braiins OS+ firmware
2. Check firewall settings on miner
3. Verify gRPC API is enabled in miner settings
4. Test from a machine on the same network as the miner

### Authentication Failed

**Possible Causes:**
1. Incorrect credentials
2. Account locked
3. API authentication disabled

**Solutions:**
1. Verify username/password (default: root/root)
2. Check via web interface
3. Reset credentials if needed

## Test Output

The test suite provides colored output with clear status indicators:

- ✓ Green: Test passed
- ✗ Red: Test failed  
- ⊘ Yellow: Test skipped
- ⚠ Yellow: Warning (for disruptive operations)

Example output:
```
============================================================
BRAIINS OS+ INTEGRATION TEST SUITE
============================================================

Target Miner: 192.168.1.100:50051
Username: root

=== TEST: Connection Test ===
✓ Connected to miner at 192.168.1.100:50051

=== TEST: Read-Only Operations ===
✓ Get Miner Details completed (0.15s)
  Hostname: miner-001
  Platform: AM3+
  BOS Version: 24.10
  
✓ Get Miner Statistics completed (0.12s)
  Hashrate (15m): 110.5 TH/s
  Power Usage: 3250 W
  Efficiency: 29.4 J/TH

TEST SUMMARY
============================================================
Results: 8 passed, 0 failed, 4 skipped
Total test time: 2.34s
```

## Safety Considerations

⚠️ **WARNING**: Some tests can disrupt mining operations!

- **Always start with read-only tests** (default behavior)
- **Test on non-production miners first**
- **Have physical access** in case of issues
- **Understand the impact** of each test category
- **Use appropriate wait times** after disruptive operations

## Building from Source

```bash
# Install dependencies
go mod download

# Build the test program
go build -o test-braiins cmd/test-braiins/main.go

# Run tests
./test-braiins -host <miner-ip>
```

## Contributing

When adding new tests:
1. Add appropriate skip flags for disruptive operations
2. Include proper error handling
3. Add informative output for debugging
4. Update this documentation

## Support

For issues with:
- **Test Suite**: Create an issue in this repository
- **Braiins OS+**: Contact Braiins support or Telegram group
- **gRPC API**: Refer to Braiins API documentation