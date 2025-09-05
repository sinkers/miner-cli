#!/bin/bash

# Braiins OS+ Integration Test Runner
# This script runs integration tests against a real Braiins OS+ miner

set -e

# Default values
HOST="${BRAIINS_HOST:-10.45.3.1}"
PORT="${BRAIINS_PORT:-50051}"
USER="${BRAIINS_USER:-root}"
PASS="${BRAIINS_PASS:-root}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${CYAN}================================${NC}"
echo -e "${CYAN}Braiins OS+ Integration Tests${NC}"
echo -e "${CYAN}================================${NC}"

# Parse command line arguments
SKIP_FLAGS=""
VERBOSE="-verbose=true"
SAFE_MODE=true

while [[ $# -gt 0 ]]; do
    case $1 in
        --host)
            HOST="$2"
            shift 2
            ;;
        --port)
            PORT="$2"
            shift 2
            ;;
        --user)
            USER="$2"
            shift 2
            ;;
        --pass)
            PASS="$2"
            shift 2
            ;;
        --unsafe)
            SAFE_MODE=false
            shift
            ;;
        --quiet)
            VERBOSE="-verbose=false"
            shift
            ;;
        --skip-auth)
            SKIP_FLAGS="$SKIP_FLAGS -skip-auth=true"
            shift
            ;;
        --enable-write)
            SKIP_FLAGS="$SKIP_FLAGS -skip-write=false"
            shift
            ;;
        --enable-restart)
            SKIP_FLAGS="$SKIP_FLAGS -skip-restart=false"
            shift
            ;;
        --enable-pause)
            SKIP_FLAGS="$SKIP_FLAGS -skip-pause=false"
            shift
            ;;
        --enable-reboot)
            SKIP_FLAGS="$SKIP_FLAGS -skip-reboot=false"
            shift
            ;;
        --wait)
            SKIP_FLAGS="$SKIP_FLAGS -wait=$2"
            shift 2
            ;;
        --help)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --host HOST         Miner host IP (default: 10.45.3.1)"
            echo "  --port PORT         gRPC port (default: 50051)"
            echo "  --user USER         Username (default: root)"
            echo "  --pass PASS         Password (default: root)"
            echo "  --unsafe            Run all tests including disruptive ones"
            echo "  --quiet             Disable verbose output"
            echo "  --skip-auth         Skip authentication tests"
            echo "  --enable-write      Enable write operations (pools, performance)"
            echo "  --enable-restart    Enable mining restart test"
            echo "  --enable-pause      Enable pause/resume test"
            echo "  --enable-reboot     Enable system reboot test"
            echo "  --wait DURATION     Wait time after disruptive ops (default: 30s)"
            echo "  --help              Show this help message"
            echo ""
            echo "Environment variables:"
            echo "  BRAIINS_HOST        Default miner host"
            echo "  BRAIINS_PORT        Default gRPC port"
            echo "  BRAIINS_USER        Default username"
            echo "  BRAIINS_PASS        Default password"
            echo ""
            echo "Examples:"
            echo "  # Run safe read-only tests"
            echo "  $0"
            echo ""
            echo "  # Run all tests including disruptive ones"
            echo "  $0 --unsafe"
            echo ""
            echo "  # Test specific miner with write operations"
            echo "  $0 --host 192.168.1.100 --enable-write"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Set default skip flags for safe mode
if [ "$SAFE_MODE" = true ]; then
    # In safe mode, skip all potentially disruptive operations by default
    DEFAULT_SKIPS="-skip-write=true -skip-reboot=true -skip-restart=true -skip-pause=true"
else
    # In unsafe mode, enable most operations but still skip reboot by default
    DEFAULT_SKIPS="-skip-write=false -skip-reboot=true -skip-restart=false -skip-pause=false"
    echo -e "${YELLOW}⚠ WARNING: Running in UNSAFE mode - This may disrupt mining operations!${NC}"
    echo -e "${YELLOW}Press Ctrl+C within 5 seconds to abort...${NC}"
    sleep 5
fi

# Display test configuration
echo -e "${BLUE}Test Configuration:${NC}"
echo "  Host: $HOST:$PORT"
echo "  User: $USER"
echo "  Mode: $([ "$SAFE_MODE" = true ] && echo "SAFE (read-only)" || echo "UNSAFE (includes writes)")"
echo ""

# Check if Go is installed
if ! command -v go &> /dev/null; then
    export PATH=/usr/local/go/bin:$PATH
    if ! command -v go &> /dev/null; then
        echo -e "${RED}Error: Go is not installed or not in PATH${NC}"
        exit 1
    fi
fi

# Change to the client directory
cd /workspace/internal/braiins/client

# Run the integration tests
echo -e "${CYAN}Starting integration tests...${NC}"
echo ""

go test -tags=integration \
    -timeout=10m \
    -run TestBraiinsIntegration \
    -host="$HOST" \
    -port="$PORT" \
    -user="$USER" \
    -pass="$PASS" \
    $DEFAULT_SKIPS \
    $SKIP_FLAGS \
    $VERBOSE

TEST_RESULT=$?

echo ""
if [ $TEST_RESULT -eq 0 ]; then
    echo -e "${GREEN}✓ All tests completed successfully!${NC}"
else
    echo -e "${RED}✗ Some tests failed. Check the output above for details.${NC}"
    exit $TEST_RESULT
fi