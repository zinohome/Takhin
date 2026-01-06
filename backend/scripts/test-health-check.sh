#!/bin/bash
# Health Check Integration Test
# Tests health check endpoints for Takhin Core

set -e

TAKHIN_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
TAKHIN_BIN="${TAKHIN_DIR}/build/takhin"
CONFIG_FILE="${TAKHIN_DIR}/configs/takhin.yaml"
DATA_DIR=$(mktemp -d)
HEALTH_PORT=19091
KAFKA_PORT=19092

cleanup() {
    echo "Cleaning up..."
    if [ -n "$TAKHIN_PID" ]; then
        kill $TAKHIN_PID 2>/dev/null || true
        wait $TAKHIN_PID 2>/dev/null || true
    fi
    rm -rf "$DATA_DIR"
}

trap cleanup EXIT

echo "=== Health Check Integration Test ==="
echo ""

# Build Takhin if needed
if [ ! -f "$TAKHIN_BIN" ]; then
    echo "Building Takhin..."
    cd "$TAKHIN_DIR"
    go build -o build/takhin ./cmd/takhin
fi

# Start Takhin with health check enabled
echo "Starting Takhin..."
export TAKHIN_STORAGE_DATA_DIR="$DATA_DIR"
export TAKHIN_HEALTH_ENABLED=true
export TAKHIN_HEALTH_PORT=$HEALTH_PORT
export TAKHIN_KAFKA_ADVERTISED_PORT=$KAFKA_PORT
export TAKHIN_SERVER_PORT=$KAFKA_PORT

"$TAKHIN_BIN" -config "$CONFIG_FILE" > /tmp/takhin-test.log 2>&1 &
TAKHIN_PID=$!

echo "Takhin PID: $TAKHIN_PID"
echo "Waiting for Takhin to start..."

# Wait for health endpoint to be available
for i in {1..30}; do
    if curl -s http://localhost:$HEALTH_PORT/health/live > /dev/null 2>&1; then
        echo "Takhin started successfully!"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "ERROR: Takhin failed to start within 30 seconds"
        cat /tmp/takhin-test.log
        exit 1
    fi
    sleep 1
done

echo ""
echo "=== Test 1: Liveness Check ==="
RESPONSE=$(curl -s -w "\n%{http_code}" http://localhost:$HEALTH_PORT/health/live)
BODY=$(echo "$RESPONSE" | head -n -1)
CODE=$(echo "$RESPONSE" | tail -n 1)

echo "Status Code: $CODE"
echo "Response: $BODY"

[ "$CODE" = "200" ] && echo "✓ Liveness check passed" || { echo "ERROR: Expected 200, got $CODE"; exit 1; }

echo ""
echo "=== Test 2: Readiness Check ==="
RESPONSE=$(curl -s -w "\n%{http_code}" http://localhost:$HEALTH_PORT/health/ready)
BODY=$(echo "$RESPONSE" | head -n -1)
CODE=$(echo "$RESPONSE" | tail -n 1)

echo "Status Code: $CODE"
echo "Response: $BODY"

[ "$CODE" = "200" ] && echo "✓ Readiness check passed" || { echo "ERROR: Expected 200, got $CODE"; exit 1; }

echo ""
echo "=== Test 3: Detailed Health Check ==="
RESPONSE=$(curl -s -w "\n%{http_code}" http://localhost:$HEALTH_PORT/health)
BODY=$(echo "$RESPONSE" | head -n -1)
CODE=$(echo "$RESPONSE" | tail -n 1)

echo "Status Code: $CODE"
echo "$BODY" | jq '.' 2>/dev/null || echo "$BODY"

[ "$CODE" = "200" ] && echo "✓ Detailed health check passed" || { echo "ERROR: Expected 200, got $CODE"; exit 1; }

echo ""
echo "=== All Tests Passed! ==="
