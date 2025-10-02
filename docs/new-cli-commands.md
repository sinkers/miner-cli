# New CLI Command Structure

## Core Design Principles
1. **Auto-detection by default** - Automatically detect API type
2. **Backwards compatible** - Existing commands continue to work
3. **Unified operations** - Same commands work across all miner types
4. **Progressive disclosure** - Simple by default, powerful when needed

## Command Structure

### Primary Commands

```bash
# Auto-detect API and show comprehensive summary
miner-cli summary 192.168.1.0/24

# Scan and detect all miners with detailed info
miner-cli scan 192.168.1.0/24 10.0.0.1-10.0.0.50

# Get detailed status of specific miners
miner-cli status 192.168.1.100 192.168.1.101

# Control operations (auto-detected)
miner-cli start 192.168.1.100
miner-cli stop 192.168.1.100
miner-cli restart 192.168.1.100
miner-cli pause 192.168.1.100
miner-cli resume 192.168.1.100

# Pool management (unified across all APIs)
miner-cli pool list 192.168.1.100
miner-cli pool add 192.168.1.100 --url stratum+tcp://pool.example.com:3333 --user worker1 --pass x
miner-cli pool switch 192.168.1.100 --pool-id 1
miner-cli pool enable 192.168.1.100 --pool-id 2
miner-cli pool disable 192.168.1.100 --pool-id 2
miner-cli pool remove 192.168.1.100 --pool-id 3

# Hardware monitoring
miner-cli hardware 192.168.1.100
miner-cli errors 192.168.1.100
miner-cli health-check 192.168.1.0/24
```

### Global Flags

```bash
# API type override (bypass auto-detection)
--api-type cgminer|vnish|braiins

# Output formats
-o, --output json|table|csv|summary
-v, --verbose                     # Show detailed information
-q, --quiet                       # Minimal output

# Performance tuning
-w, --workers 255                 # Concurrent workers (default: 255)
-t, --timeout 2s                  # Connection timeout (default: 2s)
--detect-timeout 500ms            # API detection timeout (default: 500ms)

# Filtering
--filter-status running|stopped|paused|error
--filter-model "S19*"            # Glob pattern for model filtering
--filter-efficiency "<25"        # Filter by efficiency (J/TH)
--filter-errors                  # Only show miners with errors

# Authentication
--api-key KEY                     # API key for authenticated APIs
--username USER                   # Username for basic auth
--password PASS                   # Password for basic auth

# Port overrides
--cgminer-port 4028              # CGMiner API port
--vnish-port 80                  # Vnish REST port  
--braiins-port 50051             # Braiins gRPC port
```

## Enhanced Commands

### 1. Summary Command (Enhanced)

```bash
# Basic summary with auto-detection
miner-cli summary 192.168.1.0/24

# Summary with filtering
miner-cli summary 192.168.1.0/24 --filter-status running --filter-efficiency ">30"

# Group by subnet (default for large ranges)
miner-cli summary 192.168.1.0/16 --group-by subnet

# Export to CSV
miner-cli summary 192.168.1.0/24 -o csv > miners-report.csv

# Real-time monitoring mode
miner-cli summary 192.168.1.0/24 --watch 30s
```

**Output Example:**
```
Scan Summary
═══════════════════════════════════════════════════════════════════════════════
IPs Scanned:    254
IPs Responded:  12
API Types:      Vnish (8), Braiins (3), CGMiner (1)

Miner Status
═══════════════════════════════════════════════════════════════════════════════
IP              Status   Model         Firmware      Power   Eff(J/TH)  Hashrate     Boards  Errors  Uptime
192.168.1.100   Running  S19j Pro      Vnish 2.0.3   3247W   21.5       151.2 TH/s   3/3     None    45d 3h
192.168.1.101   Running  S19 XP        Braiins 23.03 3010W   20.1       149.8 TH/s   3/3     None    12d 7h
192.168.1.102   Tuning   S19j Pro      Vnish 2.0.3   3150W   tuning     145.0 TH/s   3/3     None    2h 15m
192.168.1.103   Paused   S19           CGMiner 4.11  0W      -          0 TH/s       3/3     None    8d 4h
192.168.1.104   Error    S19j Pro      Vnish 2.0.3   2890W   25.8       112.0 TH/s   2/3     Board#2 5h 30m

Summary by Status
═══════════════════════════════════════════════════════════════════════════════
Running:  9 miners (1,306.5 TH/s total, 21.8 J/TH avg)
Tuning:   2 miners (290.0 TH/s total)
Paused:   1 miner
Error:    0 miners

Pool Distribution
═══════════════════════════════════════════════════════════════════════════════
stratum+tcp://pool1.example.com:3333  8 miners (67%)
stratum+tcp://pool2.example.com:3333  4 miners (33%)
```

### 2. Scan Command (New Unified)

```bash
# Quick scan - just find active miners
miner-cli scan 192.168.1.0/24 --quick

# Full scan with API detection
miner-cli scan 192.168.1.0/24

# Scan and save results
miner-cli scan 192.168.1.0/24 --save-to inventory.json

# Continuous discovery mode
miner-cli scan 192.168.1.0/24 --continuous --interval 5m
```

**Output Example:**
```
Scanning for miners...
═══════════════════════════════════════════════════════════════════════════════
Progress: [████████████████████░░░░░] 80% (203/254 IPs scanned)

Discovered Miners
═══════════════════════════════════════════════════════════════════════════════
IP              API Type    Model         Firmware        Status    Hashrate
192.168.1.100   Vnish       S19j Pro      Vnish 2.0.3     Running   151.2 TH/s
192.168.1.101   Braiins     S19 XP        Braiins 23.03   Running   149.8 TH/s
192.168.1.102   Vnish       S19j Pro      Vnish 2.0.3     Tuning    145.0 TH/s
192.168.1.103   CGMiner     S19           CGMiner 4.11.1  Paused    0 TH/s

Summary: Found 12 active miners across 3 API types
```

### 3. Control Commands (Unified)

```bash
# Start multiple miners
miner-cli start 192.168.1.100-192.168.1.110

# Restart with confirmation
miner-cli restart 192.168.1.0/24 --confirm

# Pause mining during peak hours
miner-cli pause 192.168.1.0/24 --reason "Peak electricity rates"

# Resume with gradual ramp-up (if supported)
miner-cli resume 192.168.1.0/24 --ramp-up 5m

# Batch control with filters
miner-cli restart --filter-status error --filter-model "S19*" 192.168.1.0/24
```

### 4. Pool Commands (Unified)

```bash
# List all pools across multiple miners
miner-cli pool list 192.168.1.0/24

# Add backup pool to all miners
miner-cli pool add 192.168.1.0/24 \
  --url stratum+tcp://backup.pool.com:3333 \
  --user myworker \
  --pass x \
  --priority 1

# Switch all miners to specific pool
miner-cli pool switch 192.168.1.0/24 --pool-id 0

# Pool failover test
miner-cli pool test-failover 192.168.1.100

# Show pool statistics
miner-cli pool stats 192.168.1.0/24
```

### 5. Health & Monitoring Commands

```bash
# Comprehensive health check
miner-cli health-check 192.168.1.0/24

# Show only miners with issues
miner-cli health-check 192.168.1.0/24 --issues-only

# Monitor temperature
miner-cli monitor temp 192.168.1.0/24 --alert-above 85

# Watch for errors
miner-cli monitor errors 192.168.1.0/24 --watch

# Generate health report
miner-cli report 192.168.1.0/24 --format html > health-report.html
```

**Health Check Output Example:**
```
Health Check Report
═══════════════════════════════════════════════════════════════════════════════
Timestamp: 2024-01-15 14:30:00

Overall Health: WARNING (2 issues detected)

Critical Issues (0)
───────────────────────────────────────────────────────────────────────────────
None

Warnings (2)
───────────────────────────────────────────────────────────────────────────────
192.168.1.104  Board #2 not responding (2/3 boards healthy)
192.168.1.107  High rejection rate: 2.3% (threshold: 2%)

Performance Metrics
───────────────────────────────────────────────────────────────────────────────
Total Hashrate:     1,456.3 TH/s
Average Efficiency: 22.4 J/TH
Power Consumption:  32.6 kW
Best Performer:     192.168.1.101 (20.1 J/TH)
Worst Performer:    192.168.1.109 (28.3 J/TH)

Recommendations
───────────────────────────────────────────────────────────────────────────────
• Check physical connection for board #2 on 192.168.1.104
• Review pool settings for 192.168.1.107 (high rejection rate)
• Consider tuning 192.168.1.109 for better efficiency
```

### 6. Advanced Commands

```bash
# Firmware detection and reporting
miner-cli firmware list 192.168.1.0/24

# Efficiency optimization suggestions
miner-cli optimize suggest 192.168.1.0/24

# Export configuration
miner-cli config export 192.168.1.100 > miner-config.json

# Import configuration
miner-cli config import 192.168.1.101 < miner-config.json

# Benchmark performance
miner-cli benchmark 192.168.1.100 --duration 1h

# API capability report
miner-cli api-info 192.168.1.100
```

## Interactive Mode

```bash
# Enter interactive mode for a specific miner
miner-cli interactive 192.168.1.100

miner> status
Status: Running
Hashrate: 151.2 TH/s
Efficiency: 21.5 J/TH
Temperature: 65°C / 78°C (board/chip)

miner> pool list
[0] stratum+tcp://pool1.example.com:3333 (active)
[1] stratum+tcp://pool2.example.com:3333 (backup)

miner> restart
Restarting miner... Done

miner> exit
```

## Backwards Compatibility

All existing commands continue to work:
```bash
# Old style (still supported)
miner-cli devs -i 192.168.1.100
miner-cli summary -i 192.168.1.0/24 -o json

# New style (preferred)
miner-cli hardware 192.168.1.100
miner-cli summary 192.168.1.0/24 --output json
```

## Configuration File Support

```yaml
# ~/.miner-cli/config.yaml
defaults:
  workers: 100
  timeout: 3s
  output: table
  
api_keys:
  vnish: "your-vnish-api-key"
  braiins: "your-braiins-api-key"

groups:
  farm1:
    ranges:
      - 192.168.1.0/24
    api_type: vnish
  farm2:
    ranges:
      - 10.0.0.0/24
    api_type: braiins

aliases:
  all: "192.168.1.0/24 10.0.0.0/24"
  rack1: "192.168.1.100-192.168.1.120"
```

Usage with config:
```bash
# Use predefined group
miner-cli summary @farm1

# Use alias
miner-cli health-check @all

# Override config defaults
miner-cli summary @farm1 --workers 255
```

## Shell Completion

```bash
# Enable bash completion
miner-cli completion bash > /etc/bash_completion.d/miner-cli

# Enable zsh completion
miner-cli completion zsh > "${fpath[1]}/_miner-cli"

# Examples of completion
miner-cli sum<TAB>              # Completes to: summary
miner-cli summary --filter-<TAB> # Shows: --filter-status, --filter-model, --filter-efficiency
miner-cli pool <TAB>             # Shows: list, add, switch, enable, disable, remove, stats
```

## Environment Variables

```bash
# Set default API type
export MINER_CLI_API_TYPE=vnish

# Set default output format
export MINER_CLI_OUTPUT=json

# Set API keys
export MINER_CLI_VNISH_KEY="your-key"
export MINER_CLI_BRAIINS_KEY="your-key"

# Set default IP ranges
export MINER_CLI_DEFAULT_RANGE="192.168.1.0/24"
```

## Scripting Support

```bash
# JSON output for scripting
miner-cli summary 192.168.1.0/24 -o json | jq '.miners[] | select(.status == "error")'

# CSV for spreadsheets
miner-cli summary 192.168.1.0/24 -o csv > daily-report.csv

# Quiet mode for scripts
miner-cli restart 192.168.1.100 -q && echo "Restarted successfully"

# Machine-readable exit codes
miner-cli health-check 192.168.1.0/24
# Exit 0: All healthy
# Exit 1: Warnings present
# Exit 2: Critical issues
# Exit 3: Connection errors
```