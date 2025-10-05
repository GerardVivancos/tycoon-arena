#!/bin/bash
# Launch server and multiple clients with labeled output

set -e

# Configuration
NUM_CLIENTS=${1:-2}  # Default to 2 clients, or use first argument
SERVER_DIR="server"
CLIENT_DIR="client"
GODOT_PATH="/Applications/Godot_mono.app/Contents/MacOS/Godot"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Track PIDs for cleanup
PIDS=()

# Cleanup function
cleanup() {
    echo -e "\n${RED}[CLEANUP]${NC} Stopping all processes..."
    for pid in "${PIDS[@]}"; do
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid" 2>/dev/null || true
        fi
    done
    wait 2>/dev/null
    echo -e "${RED}[CLEANUP]${NC} Done"
    exit 0
}

# Set up trap for Ctrl+C
trap cleanup SIGINT SIGTERM

echo -e "${GREEN}=== Starting Game Server and Clients ===${NC}"
echo -e "${GREEN}Clients: ${NUM_CLIENTS}${NC}"
echo ""

# Start server
echo -e "${YELLOW}[STARTUP]${NC} Starting server..."
(
    cd "$SERVER_DIR"
    go run main.go 2>&1 | while IFS= read -r line; do
        echo -e "${YELLOW}[SERVER]${NC} $line"
    done
) &
SERVER_PID=$!
PIDS+=($SERVER_PID)

# Wait for server to start
sleep 1

# Start clients
COLORS=("$GREEN" "$BLUE" "$CYAN" "$MAGENTA" "$RED" "$YELLOW")
for i in $(seq 1 "$NUM_CLIENTS"); do
    COLOR=${COLORS[$((i-1))]}
    echo -e "${COLOR}[STARTUP]${NC} Starting CLIENT${i}..."

    (
        cd "$CLIENT_DIR"
        # Launch Godot with GUI and capture output
        "$GODOT_PATH" 2>&1 | while IFS= read -r line; do
            echo -e "${COLOR}[CLIENT${i}]${NC} $line"
        done
    ) &

    CLIENT_PID=$!
    PIDS+=($CLIENT_PID)

    # Stagger client starts so windows don't overlap completely
    sleep 1
done

echo ""
echo -e "${GREEN}=== All processes started ===${NC}"
echo -e "${GREEN}Press Ctrl+C to stop all processes${NC}"
echo ""

# Wait for all processes
wait
