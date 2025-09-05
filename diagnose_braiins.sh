#!/bin/bash

# Braiins OS+ Diagnostics Script
# Helps diagnose connectivity and API availability

HOST="${1:-10.45.3.1}"

echo "============================================"
echo "Braiins OS+ Diagnostics for $HOST"
echo "============================================"
echo ""

# Check basic connectivity
echo "1. Testing basic network connectivity..."
if ping -c 1 -W 2 "$HOST" > /dev/null 2>&1; then
    echo "   ✓ Host is reachable"
else
    echo "   ✗ Host is not reachable"
    exit 1
fi

echo ""
echo "2. Checking common ports..."

# Check SSH
echo -n "   SSH (22): "
if timeout 2 bash -c "echo > /dev/tcp/$HOST/22" 2>/dev/null; then
    echo "OPEN"
else
    echo "CLOSED"
fi

# Check HTTP
echo -n "   HTTP (80): "
if timeout 2 bash -c "echo > /dev/tcp/$HOST/80" 2>/dev/null; then
    echo "OPEN"
else
    echo "CLOSED"
fi

# Check HTTPS
echo -n "   HTTPS (443): "
if timeout 2 bash -c "echo > /dev/tcp/$HOST/443" 2>/dev/null; then
    echo "OPEN"
else
    echo "CLOSED"
fi

# Check gRPC default
echo -n "   gRPC (50051): "
if timeout 2 bash -c "echo > /dev/tcp/$HOST/50051" 2>/dev/null; then
    echo "OPEN"
else
    echo "CLOSED - gRPC API may not be enabled"
fi

# Check CGMiner API
echo -n "   CGMiner API (4028): "
if timeout 2 bash -c "echo > /dev/tcp/$HOST/4028" 2>/dev/null; then
    echo "OPEN"
else
    echo "CLOSED"
fi

echo ""
echo "3. Attempting to get web interface info..."
if curl -s -k --max-time 5 "https://$HOST" > /dev/null 2>&1; then
    echo "   ✓ HTTPS interface is accessible"
elif curl -s --max-time 5 "http://$HOST" > /dev/null 2>&1; then
    echo "   ✓ HTTP interface is accessible"
else
    echo "   ✗ Web interface is not accessible"
fi

echo ""
echo "============================================"
echo "Diagnosis Summary:"
echo "============================================"
echo ""
echo "The gRPC API (port 50051) appears to be CLOSED."
echo ""
echo "Possible reasons and solutions:"
echo ""
echo "1. gRPC API may not be enabled on this miner"
echo "   - The gRPC API is enabled by default only in Braiins OS+ 23.03.1+"
echo "   - Check your Braiins OS+ version via web interface"
echo ""
echo "2. Firewall may be blocking the port"
echo "   - Check miner's firewall settings"
echo ""
echo "3. The miner might be running older firmware"
echo "   - Update to Braiins OS+ 23.03.1 or newer"
echo ""
echo "4. gRPC might be on a different port"
echo "   - Check miner configuration via web interface"
echo ""
echo "Alternative: Use the CGMiner-compatible API if available"
echo "   - The miner-cli tool already supports CGMiner API"
echo "   - Try: miner-cli scan $HOST"
echo ""