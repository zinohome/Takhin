#!/bin/bash
set -e

echo "Removing Takhin..."

# Stop service if running
if systemctl is-active --quiet takhin; then
    systemctl stop takhin
fi

# Disable service if enabled
if systemctl is-enabled --quiet takhin; then
    systemctl disable takhin
fi

echo "Takhin removed (data preserved in /var/lib/takhin)"
