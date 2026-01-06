#!/bin/bash
set -e

echo "Installing Takhin..."

# Create data directory
mkdir -p /var/lib/takhin
chmod 755 /var/lib/takhin

# Create log directory
mkdir -p /var/log/takhin
chmod 755 /var/log/takhin

# Create user if it doesn't exist
if ! id -u takhin >/dev/null 2>&1; then
    useradd --system --no-create-home --shell /bin/false takhin
fi

# Set ownership
chown -R takhin:takhin /var/lib/takhin
chown -R takhin:takhin /var/log/takhin

echo "Takhin installation completed"
echo "Edit /etc/takhin/takhin.yaml to configure"
echo "Start with: systemctl start takhin"
