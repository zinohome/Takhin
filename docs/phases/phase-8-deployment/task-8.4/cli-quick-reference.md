# Takhin CLI - Quick Reference

## Installation

```bash
# Build the CLI
task cli:build

# Or build manually
cd backend && go build -o ../build/takhin-cli ./cmd/takhin-cli/

# Add to PATH (optional)
sudo cp build/takhin-cli /usr/local/bin/
```

## Global Options

```bash
-c, --config string     Config file path
-d, --data-dir string   Data directory path
```

**Note**: Either `--config` or `--data-dir` must be specified.

## Topic Management

### List Topics
```bash
takhin-cli -d /var/data/takhin topic list
```

### Create Topic
```bash
# Basic
takhin-cli -d /var/data/takhin topic create my-topic

# With options
takhin-cli -d /var/data/takhin topic create my-topic \
  --partitions 3 \
  --replication-factor 2
```

### Describe Topic
```bash
takhin-cli -d /var/data/takhin topic describe my-topic
```

### Configure Topic
```bash
# View configuration
takhin-cli -d /var/data/takhin topic config my-topic

# Set configuration
takhin-cli -d /var/data/takhin topic config my-topic \
  --set replication.factor=3 \
  --set replica.lag.max.ms=10000
```

### Delete Topic
```bash
# With confirmation
takhin-cli -d /var/data/takhin topic delete my-topic

# Force delete
takhin-cli -d /var/data/takhin topic delete my-topic --force
```

## Consumer Group Management

### List Groups
```bash
takhin-cli -d /var/data/takhin group list
```

### Describe Group
```bash
takhin-cli -d /var/data/takhin group describe my-group
```

### Reset Offsets
```bash
# Reset to earliest
takhin-cli -d /var/data/takhin group reset my-group \
  --topic my-topic \
  --partition 0 \
  --to-earliest

# Reset to latest
takhin-cli -d /var/data/takhin group reset my-group \
  --topic my-topic \
  --partition 0 \
  --to-latest

# Reset to specific offset
takhin-cli -d /var/data/takhin group reset my-group \
  --topic my-topic \
  --partition 0 \
  --offset 1000
```

### Export Groups
```bash
# Export all groups
takhin-cli -d /var/data/takhin group export -o groups.json

# Export specific group
takhin-cli -d /var/data/takhin group export \
  --group my-group \
  --output my-group.json
```

### Delete Group
```bash
# With confirmation
takhin-cli -d /var/data/takhin group delete my-group

# Force delete
takhin-cli -d /var/data/takhin group delete my-group --force
```

## Configuration Management

### Show Configuration
```bash
# JSON format (default)
takhin-cli -c /etc/takhin/takhin.yaml config show

# YAML format
takhin-cli -c /etc/takhin/takhin.yaml config show --format yaml
```

### Get Specific Value
```bash
takhin-cli -c /etc/takhin/takhin.yaml config get server.port
takhin-cli -c /etc/takhin/takhin.yaml config get kafka.broker.id
```

### Validate Configuration
```bash
takhin-cli config validate --file /etc/takhin/takhin.yaml
```

### Export Configuration
```bash
# Export as YAML
takhin-cli -c /etc/takhin/takhin.yaml config export \
  --output backup.yaml \
  --format yaml

# Export as JSON
takhin-cli -c /etc/takhin/takhin.yaml config export \
  --output backup.json \
  --format json
```

## Data Import/Export

### Export Data
```bash
# Export all messages
takhin-cli -d /var/data/takhin data export \
  --topic my-topic \
  --partition 0 \
  --output messages.json

# Export limited messages
takhin-cli -d /var/data/takhin data export \
  --topic my-topic \
  --partition 0 \
  --max-messages 1000 \
  --output sample.json

# Export from specific offset
takhin-cli -d /var/data/takhin data export \
  --topic my-topic \
  --partition 0 \
  --from-offset 1000 \
  --output recent.json

# Export to stdout
takhin-cli -d /var/data/takhin data export \
  --topic my-topic \
  --partition 0
```

### Import Data
```bash
takhin-cli -d /var/data/takhin data import \
  --topic my-topic \
  --partition 0 \
  --input messages.json
```

### Data Statistics
```bash
# Show all topics
takhin-cli -d /var/data/takhin data stats

# Show specific topic
takhin-cli -d /var/data/takhin data stats --topic my-topic
```

## Common Workflows

### Backup Topic Data
```bash
#!/bin/bash
TOPIC="my-topic"
PARTITIONS=3
OUTPUT_DIR="backup"

mkdir -p $OUTPUT_DIR

for i in $(seq 0 $((PARTITIONS-1))); do
  takhin-cli -d /var/data/takhin data export \
    --topic $TOPIC \
    --partition $i \
    --output $OUTPUT_DIR/${TOPIC}-${i}.json
done
```

### Migrate Topic to New Cluster
```bash
#!/bin/bash
# On source cluster
takhin-cli -d /var/data/takhin-source data export \
  --topic my-topic --partition 0 --output topic-data.json

# On destination cluster
takhin-cli -d /var/data/takhin-dest topic create my-topic --partitions 1
takhin-cli -d /var/data/takhin-dest data import \
  --topic my-topic --partition 0 --input topic-data.json
```

### Monitor Consumer Lag
```bash
#!/bin/bash
# Get consumer group info
takhin-cli -d /var/data/takhin group describe my-group

# Get topic high watermark
takhin-cli -d /var/data/takhin topic describe my-topic
```

### Bulk Delete Old Topics
```bash
#!/bin/bash
# List all topics and delete those starting with "test-"
takhin-cli -d /var/data/takhin topic list | grep "^test-" | while read topic; do
  takhin-cli -d /var/data/takhin topic delete $topic --force
done
```

## Environment Variables

Can be used instead of flags:
```bash
export TAKHIN_CLI_DATA_DIR=/var/data/takhin
export TAKHIN_CLI_CONFIG=/etc/takhin/takhin.yaml

# Then use without flags
takhin-cli topic list
takhin-cli group list
```

## Scripting Tips

### Error Handling
```bash
#!/bin/bash
set -e  # Exit on error

if ! takhin-cli -d /var/data/takhin topic describe my-topic &> /dev/null; then
  echo "Topic does not exist, creating..."
  takhin-cli -d /var/data/takhin topic create my-topic --partitions 3
fi
```

### JSON Processing with jq
```bash
# Get consumer group offset for specific topic/partition
takhin-cli -d /var/data/takhin group export --group my-group | \
  jq -r '.my-group.offsets."my-topic"[0].Offset'

# List topics with more than 5 partitions
takhin-cli -d /var/data/takhin topic list | awk '$2 > 5 {print $1}'
```

### Parallel Operations
```bash
#!/bin/bash
# Export multiple partitions in parallel
for i in {0..9}; do
  (
    takhin-cli -d /var/data/takhin data export \
      --topic my-topic --partition $i \
      --output partition-${i}.json
  ) &
done
wait
```

## Troubleshooting

### Permission Issues
```bash
# Check data directory permissions
ls -la /var/data/takhin

# Fix permissions if needed
sudo chown -R $USER:$USER /var/data/takhin
```

### Topic Not Found
```bash
# Verify data directory
takhin-cli -d /var/data/takhin topic list

# Check if topic exists on disk
ls /var/data/takhin/
```

### Configuration Issues
```bash
# Validate config before use
takhin-cli config validate --file /etc/takhin/takhin.yaml

# Check specific config values
takhin-cli -c /etc/takhin/takhin.yaml config get storage.data.dir
```

## Performance Tips

1. **Use `--force` for automation** - Skip confirmations in scripts
2. **Limit export size** - Use `--max-messages` for large topics
3. **Parallel operations** - Export/import partitions in parallel
4. **Local operations** - CLI works directly with files, no network overhead

## See Also

- Full documentation: `TASK_8.4_CLI_COMPLETION.md`
- Takhin documentation: `docs/`
- Task commands: `task --list`
