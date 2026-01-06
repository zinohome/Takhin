#!/bin/bash
# Test script for Takhin CLI
# Demonstrates basic functionality

set -e

CLI="./build/takhin-cli"
TEST_DIR="/tmp/takhin-cli-test-$$"
TOPIC="test-topic"

echo "🧪 Takhin CLI Integration Test"
echo "================================"
echo ""

# Setup
echo "📁 Setting up test environment..."
mkdir -p "$TEST_DIR"
trap "rm -rf $TEST_DIR" EXIT

# Test 1: Version
echo ""
echo "✅ Test 1: Version command"
$CLI version

# Test 2: Create topic
echo ""
echo "✅ Test 2: Create topic"
$CLI -d "$TEST_DIR" topic create "$TOPIC" --partitions 2 --replication-factor 1

# Test 3: List topics
echo ""
echo "✅ Test 3: List topics"
$CLI -d "$TEST_DIR" topic list

# Test 4: Describe topic
echo ""
echo "✅ Test 4: Describe topic"
$CLI -d "$TEST_DIR" topic describe "$TOPIC"

# Test 5: Topic configuration
echo ""
echo "✅ Test 5: Get topic configuration"
$CLI -d "$TEST_DIR" topic config "$TOPIC"

# Test 6: Set topic configuration
echo ""
echo "✅ Test 6: Set topic configuration"
$CLI -d "$TEST_DIR" topic config "$TOPIC" --set replication.factor=2

# Test 7: Data statistics
echo ""
echo "✅ Test 7: Data statistics"
$CLI -d "$TEST_DIR" data stats

# Test 8: Export data (empty topic)
echo ""
echo "✅ Test 8: Export data (should be empty)"
$CLI -d "$TEST_DIR" data export --topic "$TOPIC" --partition 0 > /tmp/export-$$.json
echo "Exported $(wc -l < /tmp/export-$$.json) messages"
rm /tmp/export-$$.json

# Test 9: Group list (should be empty)
echo ""
echo "✅ Test 9: List consumer groups"
$CLI -d "$TEST_DIR" group list

# Test 10: Delete topic
echo ""
echo "✅ Test 10: Delete topic"
$CLI -d "$TEST_DIR" topic delete "$TOPIC" --force

# Test 11: Verify deletion
echo ""
echo "✅ Test 11: Verify topic deletion"
if $CLI -d "$TEST_DIR" topic list | grep -q "$TOPIC"; then
  echo "❌ Topic still exists!"
  exit 1
else
  echo "✓ Topic successfully deleted"
fi

echo ""
echo "================================"
echo "✅ All tests passed!"
echo "================================"
