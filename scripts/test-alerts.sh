#!/bin/bash
# Alert Testing Script for Takhin AlertManager Integration
# This script tests alert rules by triggering conditions and verifying notifications

set -e

PROMETHEUS_URL="${PROMETHEUS_URL:-http://localhost:9091}"
ALERTMANAGER_URL="${ALERTMANAGER_URL:-http://localhost:9093}"
TAKHIN_URL="${TAKHIN_URL:-http://localhost:9090}"

COLOR_GREEN='\033[0;32m'
COLOR_RED='\033[0;31m'
COLOR_YELLOW='\033[1;33m'
COLOR_BLUE='\033[0;34m'
COLOR_NC='\033[0m' # No Color

# Print colored output
print_status() {
    echo -e "${COLOR_BLUE}[INFO]${COLOR_NC} $1"
}

print_success() {
    echo -e "${COLOR_GREEN}[SUCCESS]${COLOR_NC} $1"
}

print_error() {
    echo -e "${COLOR_RED}[ERROR]${COLOR_NC} $1"
}

print_warning() {
    echo -e "${COLOR_YELLOW}[WARNING]${COLOR_NC} $1"
}

# Check if services are running
check_services() {
    print_status "Checking service availability..."
    
    if curl -sf "$PROMETHEUS_URL/-/healthy" > /dev/null 2>&1; then
        print_success "Prometheus is healthy"
    else
        print_error "Prometheus is not accessible at $PROMETHEUS_URL"
        exit 1
    fi
    
    if curl -sf "$ALERTMANAGER_URL/-/healthy" > /dev/null 2>&1; then
        print_success "AlertManager is healthy"
    else
        print_error "AlertManager is not accessible at $ALERTMANAGER_URL"
        exit 1
    fi
    
    if curl -sf "$TAKHIN_URL/metrics" > /dev/null 2>&1; then
        print_success "Takhin metrics endpoint is accessible"
    else
        print_warning "Takhin metrics endpoint is not accessible at $TAKHIN_URL"
    fi
}

# Check if alert rules are loaded
check_alert_rules() {
    print_status "Checking alert rules configuration..."
    
    RULES_COUNT=$(curl -s "$PROMETHEUS_URL/api/v1/rules" | jq '.data.groups | length')
    
    if [ "$RULES_COUNT" -gt 0 ]; then
        print_success "Found $RULES_COUNT alert rule groups"
        
        # List all alert rules
        print_status "Alert rules loaded:"
        curl -s "$PROMETHEUS_URL/api/v1/rules" | jq -r '.data.groups[].rules[].name' | while read -r alert_name; do
            echo "  - $alert_name"
        done
    else
        print_error "No alert rules found. Check prometheus-alerts.yml configuration."
        exit 1
    fi
}

# Check AlertManager configuration
check_alertmanager_config() {
    print_status "Checking AlertManager configuration..."
    
    CONFIG_STATUS=$(curl -s "$ALERTMANAGER_URL/api/v2/status" | jq -r '.config.original')
    
    if [ -n "$CONFIG_STATUS" ]; then
        print_success "AlertManager configuration is loaded"
        
        # Check receivers
        RECEIVERS=$(curl -s "$ALERTMANAGER_URL/api/v2/status" | jq -r '.config.receivers[].name')
        print_status "Configured receivers:"
        echo "$RECEIVERS" | while read -r receiver; do
            echo "  - $receiver"
        done
    else
        print_error "Failed to retrieve AlertManager configuration"
        exit 1
    fi
}

# Get current alerts
get_current_alerts() {
    print_status "Checking currently firing alerts..."
    
    ALERTS=$(curl -s "$PROMETHEUS_URL/api/v1/alerts" | jq -r '.data.alerts[] | select(.state=="firing") | .labels.alertname')
    
    if [ -n "$ALERTS" ]; then
        print_warning "Currently firing alerts:"
        echo "$ALERTS" | while read -r alert; do
            echo "  🔥 $alert"
        done
    else
        print_success "No alerts currently firing"
    fi
}

# Test alert by simulating condition (for testing purposes)
test_alert_simulation() {
    print_status "Simulating alert conditions (for testing)..."
    
    # This section would contain actual load generation or metric manipulation
    # For demonstration, we'll just check if alerts can be triggered
    
    print_status "To manually test alerts, you can:"
    echo "  1. Stop Takhin service to trigger 'TakhinDown' alert"
    echo "  2. Generate high load to trigger latency alerts"
    echo "  3. Fill disk space to trigger storage alerts"
    echo "  4. Create consumer lag to trigger consumer alerts"
}

# Send test alert to AlertManager
send_test_alert() {
    print_status "Sending test alert to AlertManager..."
    
    TEST_ALERT='{
        "labels": {
            "alertname": "TestAlert",
            "severity": "warning",
            "instance": "test-instance",
            "category": "test"
        },
        "annotations": {
            "summary": "This is a test alert",
            "description": "Alert testing for Takhin monitoring system"
        }
    }'
    
    RESPONSE=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "[${TEST_ALERT}]" \
        "$ALERTMANAGER_URL/api/v2/alerts")
    
    if [ $? -eq 0 ]; then
        print_success "Test alert sent successfully"
        print_status "Check your configured notification channels (Email, Slack)"
    else
        print_error "Failed to send test alert: $RESPONSE"
    fi
}

# Check alert routing
test_alert_routing() {
    print_status "Testing alert routing configuration..."
    
    # Test different severity levels
    for severity in critical high warning; do
        print_status "Testing $severity severity routing..."
        
        TEST_ALERT="{
            \"labels\": {
                \"alertname\": \"RoutingTest${severity}\",
                \"severity\": \"${severity}\",
                \"instance\": \"test-instance\",
                \"category\": \"test\"
            },
            \"annotations\": {
                \"summary\": \"Routing test for ${severity} severity\",
                \"description\": \"Testing alert routing for ${severity} level\"
            }
        }"
        
        curl -s -X POST \
            -H "Content-Type: application/json" \
            -d "[${TEST_ALERT}]" \
            "$ALERTMANAGER_URL/api/v2/alerts" > /dev/null
        
        print_success "Sent test alert with ${severity} severity"
    done
    
    print_status "Check AlertManager UI at $ALERTMANAGER_URL for alert routing"
}

# Verify alert notification channels
verify_notification_channels() {
    print_status "Verifying notification channel configuration..."
    
    # Check environment variables for notification settings
    if [ -z "$SLACK_WEBHOOK_URL" ]; then
        print_warning "SLACK_WEBHOOK_URL not set - Slack notifications will not work"
    else
        print_success "Slack webhook URL configured"
    fi
    
    if [ -z "$SMTP_PASSWORD" ]; then
        print_warning "SMTP_PASSWORD not set - Email notifications may not work"
    else
        print_success "SMTP password configured"
    fi
    
    print_status "To configure notification channels:"
    echo "  export SLACK_WEBHOOK_URL='https://hooks.slack.com/services/YOUR/WEBHOOK/URL'"
    echo "  export SMTP_PASSWORD='your-smtp-password'"
}

# Query specific alert metrics
query_alert_metrics() {
    print_status "Querying alert-related metrics..."
    
    # Check error rate
    ERROR_RATE=$(curl -s "$PROMETHEUS_URL/api/v1/query?query=rate(takhin_kafka_request_errors_total[5m])" | \
        jq -r '.data.result[0].value[1] // "0"')
    print_status "Current error rate: ${ERROR_RATE} errors/sec"
    
    # Check replication lag
    MAX_LAG=$(curl -s "$PROMETHEUS_URL/api/v1/query?query=max(takhin_replication_lag_offsets)" | \
        jq -r '.data.result[0].value[1] // "0"')
    print_status "Maximum replication lag: ${MAX_LAG} offsets"
    
    # Check consumer lag
    MAX_CONSUMER_LAG=$(curl -s "$PROMETHEUS_URL/api/v1/query?query=max(takhin_consumer_group_lag_offsets)" | \
        jq -r '.data.result[0].value[1] // "0"')
    print_status "Maximum consumer lag: ${MAX_CONSUMER_LAG} offsets"
    
    # Check memory usage
    MEMORY_USAGE=$(curl -s "$PROMETHEUS_URL/api/v1/query?query=takhin_go_memory_heap_alloc_bytes" | \
        jq -r '.data.result[0].value[1] // "0"')
    MEMORY_MB=$(echo "scale=2; $MEMORY_USAGE / 1048576" | bc)
    print_status "Current memory usage: ${MEMORY_MB} MB"
}

# Main test execution
main() {
    echo "=================================="
    echo "Takhin AlertManager Test Suite"
    echo "=================================="
    echo ""
    
    check_services
    echo ""
    
    check_alert_rules
    echo ""
    
    check_alertmanager_config
    echo ""
    
    get_current_alerts
    echo ""
    
    verify_notification_channels
    echo ""
    
    query_alert_metrics
    echo ""
    
    print_status "Running test alerts..."
    send_test_alert
    echo ""
    
    test_alert_routing
    echo ""
    
    echo "=================================="
    print_success "Alert testing completed!"
    echo "=================================="
    echo ""
    echo "Next steps:"
    echo "  1. Check AlertManager UI at: $ALERTMANAGER_URL"
    echo "  2. Check Prometheus UI at: $PROMETHEUS_URL"
    echo "  3. Verify notifications in Email/Slack"
    echo "  4. Review firing alerts and adjust thresholds if needed"
}

# Run main function
main
