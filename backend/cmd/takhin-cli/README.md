# Takhin CLI

Command-line management tool for Takhin message broker.

## Quick Start

```bash
# Build
cd backend && go build -o ../build/takhin-cli ./cmd/takhin-cli/

# Or use task
task cli:build

# Basic usage
takhin-cli -d /var/data/takhin topic list
takhin-cli -c /etc/takhin/takhin.yaml config show
```

## Features

- ✅ **Topic Management** - Create, delete, describe, configure topics
- ✅ **Consumer Groups** - List, describe, reset offsets
- ✅ **Configuration** - View, validate, export configuration
- ✅ **Data Operations** - Import/export messages, statistics

## Usage

```bash
# Specify data directory
takhin-cli -d /var/data/takhin <command>

# Or use config file
takhin-cli -c /etc/takhin/takhin.yaml <command>
```

## Commands

### Topics
```bash
takhin-cli topic list                          # List all topics
takhin-cli topic create my-topic -p 3 -r 2    # Create topic
takhin-cli topic describe my-topic             # Show details
takhin-cli topic config my-topic               # View config
takhin-cli topic delete my-topic -f            # Delete topic
```

### Consumer Groups
```bash
takhin-cli group list                          # List groups
takhin-cli group describe my-group             # Show details
takhin-cli group reset my-group -t topic -p 0  # Reset offsets
takhin-cli group export -o groups.json         # Export groups
takhin-cli group delete my-group -f            # Delete group
```

### Configuration
```bash
takhin-cli config show                         # Show config
takhin-cli config get server.port              # Get value
takhin-cli config validate --file config.yaml  # Validate
takhin-cli config export -o backup.yaml        # Export
```

### Data
```bash
takhin-cli data export -t topic -p 0           # Export data
takhin-cli data import -t topic -i data.json   # Import data
takhin-cli data stats                          # Show stats
```

## Documentation

- 📖 [Completion Summary](../TASK_8.4_CLI_COMPLETION.md) - Full implementation details
- 📘 [Quick Reference](../TASK_8.4_CLI_QUICK_REFERENCE.md) - Command examples
- 📊 [Visual Overview](../TASK_8.4_CLI_VISUAL_OVERVIEW.md) - Architecture diagrams
- 🧪 [Test Script](../scripts/test_cli.sh) - Integration tests

## Examples

See [Quick Reference](../TASK_8.4_CLI_QUICK_REFERENCE.md) for detailed examples.

## Building

```bash
# Using task
task cli:build

# Using go directly
cd backend
go build -o ../build/takhin-cli ./cmd/takhin-cli/

# Install globally (optional)
sudo cp build/takhin-cli /usr/local/bin/
```

## Testing

```bash
# Run test script
bash scripts/test_cli.sh
```

## License

Copyright 2025 Takhin Data, Inc.
