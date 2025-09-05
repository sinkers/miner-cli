# Braiins OS+ Integration Test Suite

## Overview

Comprehensive integration test suite for validating Braiins OS+ gRPC API functionality against real hardware. Tests are categorized by risk level and can be selectively enabled.

## Quick Start

```bash
# Build the test binary
go build -o test-braiins cmd/test-braiins/main.go

# Run safe tests only (default)
./test-braiins -host 192.168.1.100

# Run with specific credentials
./test-braiins -host 192.168.1.100 -user admin -pass mypassword
```

## Test Categories

### 1. Safe Operations (Always Run)

#### Connection & Authentication
- Validates gRPC connectivity on port 50051
- Tests authentication with provided credentials
- Verifies token-based session management

#### Read-Only Operations
- **GetMinerDetails**: Hardware model, hostname, MAC, uptime
- **GetMinerStatistics**: Hashrate (15m/24h), power, efficiency, shares
- **GetHashboards**: Board status, chip count, temperatures
- **GetPoolGroups**: Current pool configuration
- **GetCoolingState**: Fan speeds, temperature sensors
- **GetTunerState**: Performance tuning mode
- **GetLicenseState**: License validity
- **GetConfiguration**: Full miner configuration

### 2. Pool Operations (Skip by Default)
- **UpdatePoolGroup**: Modify pool URLs and credentials
- Validates changes are applied correctly
- Tests failover behavior

### 3. Performance Operations (Skip by Default)
- **SetPowerTarget**: Configure power consumption limit
- **SetHashrateTarget**: Configure hashrate target
- Verifies settings are applied and take effect
- Monitors for stability after changes

### 4. Mining Control (Skip by Default)

#### Pause/Resume Test
- Pauses mining operation
- Waits for confirmation (configurable delay)
- Resumes mining
- Verifies mining resumes successfully

#### Restart Mining Test
- Restarts the mining process
- Waits for service to restart
- Verifies mining resumes with same configuration

### 5. System Operations (Skip by Default)

#### Reboot Test (DANGEROUS)
- Issues system reboot command
- Polls for miner to go offline
- Waits for miner to come back online
- Verifies all services restart correctly

## Command Line Options

```
-host string        Target miner IP address (default "10.45.3.1")
-port int          gRPC port (default 50051)
-user string       Username for authentication (default "root")
-pass string       Password for authentication (default "root")
-timeout duration  Connection timeout (default 10s)
-wait duration     Wait time after disruptive operations (default 30s)

Skip Flags (Safety):
-skip-write        Skip pool and performance tests (default true)
-skip-pause        Skip pause/resume mining test (default true)
-skip-restart      Skip mining restart test (default true)
-skip-reboot       Skip system reboot test (default true)
```

## Test Profiles

### Safe Mode (Default)
```bash
./test-braiins -host 192.168.1.100
```
- Only read operations
- No mining interruption
- Safe for production miners

### Performance Testing
```bash
./test-braiins -host 192.168.1.100 -skip-write=false
```
- Includes power/hashrate target changes
- May affect mining efficiency temporarily
- Monitor miner after testing

### Full Testing (Development Only)
```bash
./test-braiins -host 192.168.1.100 \
  -skip-write=false \
  -skip-pause=false \
  -skip-restart=false \
  -wait=60s
```
- All tests except reboot
- Will interrupt mining briefly
- Use on test/development miners only

### Complete Suite (DANGEROUS)
```bash
./test-braiins -host 192.168.1.100 \
  -skip-write=false \
  -skip-pause=false \
  -skip-restart=false \
  -skip-reboot=false \
  -wait=120s
```
- Includes system reboot
- Significant downtime expected
- Only use with physical access to miner

## Output Format

Tests use colored output for clarity:
- ✓ Green: Test passed
- ✗ Red: Test failed
- ⊘ Yellow: Test skipped
- ⚠ Yellow: Warning (disruptive operation)

### Example Output

```
============================================================
BRAIINS OS+ INTEGRATION TEST SUITE
============================================================

Target Miner: 192.168.1.100:50051
Username: root

=== TEST: Connection Test ===
✓ Connected to miner at 192.168.1.100:50051

=== TEST: Authentication Test ===
✓ Authentication successful

=== TEST: Read-Only Operations ===
✓ Get Miner Details completed (0.15s)
  Hostname: miner-001
  Model: Antminer S19 Pro
  MAC: 00:1A:2B:3C:4D:5E
  Uptime: 5d 14h 23m
  
✓ Get Miner Statistics completed (0.12s)
  Hashrate (15m): 110.5 TH/s
  Power Usage: 3250 W
  Efficiency: 29.4 J/TH
  
[Additional tests...]

TEST SUMMARY
============================================================
Results: 10 passed, 0 failed, 5 skipped
Total test time: 2.34s
```

## Troubleshooting

### Connection Refused
- Verify Braiins OS+ version is 23.03.1 or newer
- Check firewall settings
- Confirm gRPC API is enabled (port 50051)
- Use diagnose_braiins.sh script for connectivity check

### Authentication Failed
- Verify credentials (default: root/root)
- Check account is not locked
- Try via web interface first

### Tests Timeout
- Increase -timeout flag value
- Check network latency
- Verify miner is not overloaded

## Safety Guidelines

1. **Always start with read-only tests**
2. **Test on non-production miners first**
3. **Have physical/IPMI access for recovery**
4. **Monitor miner during and after testing**
5. **Document any issues encountered**
6. **Use appropriate wait times for your network**

## Integration with CI/CD

```yaml
# Example GitHub Actions workflow
- name: Test Braiins Integration
  run: |
    go build -o test-braiins cmd/test-braiins/main.go
    ./test-braiins -host ${{ secrets.TEST_MINER_IP }} \
      -user ${{ secrets.TEST_MINER_USER }} \
      -pass ${{ secrets.TEST_MINER_PASS }}
```

## Development

### Adding New Tests

1. Add test function following naming convention
2. Include appropriate skip flag if disruptive
3. Add clear output messages
4. Update documentation
5. Test on development miner first

### Test Implementation Pattern

```go
func testNewFeature(client *client.SimpleBraiinsClient) error {
    fmt.Println("=== TEST: New Feature ===")
    
    start := time.Now()
    result, err := client.NewFeatureCall()
    elapsed := time.Since(start)
    
    if err != nil {
        printError("New Feature", err)
        return err
    }
    
    printSuccess("New Feature", elapsed)
    fmt.Printf("  Result: %v\n", result)
    return nil
}
```

## Support

For issues specific to:
- Test suite: Create issue in this repository
- Braiins OS+: Contact Braiins support
- gRPC API: Refer to official Braiins documentation