# Takhin CLI - Visual Overview

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                      Takhin CLI Tool                         │
│                     (takhin-cli binary)                      │
└─────────────────────────────────────────────────────────────┘
                              │
                              ├── Cobra Framework (CLI routing)
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
        ▼                     ▼                     ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│    Topic     │    │    Group     │    │   Config     │
│  Management  │    │  Management  │    │  Management  │
└──────────────┘    └──────────────┘    └──────────────┘
        │                     │                     │
        │                     │                     │
        ▼                     ▼                     ▼
┌──────────────────────────────────────────────────────────┐
│              Takhin Core Components                       │
├──────────────────────────────────────────────────────────┤
│  • pkg/storage/topic   (Topic Manager)                   │
│  • pkg/coordinator     (Consumer Groups)                 │
│  • pkg/config          (Configuration)                   │
│  • pkg/storage/log     (Log Segments)                    │
└──────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────┐
│                    File System                            │
├──────────────────────────────────────────────────────────┤
│  /var/data/takhin/                                       │
│  ├── my-topic-0/                                         │
│  │   ├── 00000000000000000000.log                       │
│  │   ├── 00000000000000000000.index                     │
│  │   └── metadata.json                                   │
│  └── my-topic-1/                                         │
│      └── ...                                              │
└──────────────────────────────────────────────────────────┘
```

## Command Structure

```
takhin-cli
│
├── topic
│   ├── list              → List all topics
│   ├── create <name>     → Create new topic
│   ├── delete <name>     → Delete topic
│   ├── describe <name>   → Show topic details
│   └── config <name>     → Get/set topic configuration
│
├── group
│   ├── list              → List consumer groups
│   ├── describe <id>     → Show group details
│   ├── delete <id>       → Delete consumer group
│   ├── reset <id>        → Reset consumer offsets
│   └── export            → Export group data
│
├── config
│   ├── show              → Display configuration
│   ├── validate          → Validate config file
│   ├── get <key>         → Get specific value
│   └── export            → Export configuration
│
├── data
│   ├── export            → Export topic data
│   ├── import            → Import data to topic
│   └── stats             → Show data statistics
│
└── version               → Show version info
```

## Data Flow: Topic Creation

```
User Command:
$ takhin-cli -d /data topic create my-topic -p 3 -r 2

         │
         ▼
┌─────────────────┐
│  CLI Parser     │  Parse args and flags
└─────────────────┘
         │
         ▼
┌─────────────────┐
│ Topic Manager   │  Create topic with 3 partitions
│ NewManager()    │
└─────────────────┘
         │
         ▼
┌─────────────────┐
│  Create Dirs    │  /data/my-topic-0/
│                 │  /data/my-topic-1/
│                 │  /data/my-topic-2/
└─────────────────┘
         │
         ▼
┌─────────────────┐
│  Set Metadata   │  ReplicationFactor = 2
└─────────────────┘
         │
         ▼
┌─────────────────┐
│  Save Metadata  │  metadata.json
└─────────────────┘
         │
         ▼
    Success! ✓
```

## Data Flow: Export Data

```
User Command:
$ takhin-cli -d /data data export -t my-topic -p 0 -o output.json

         │
         ▼
┌─────────────────┐
│  CLI Parser     │  Parse export options
└─────────────────┘
         │
         ▼
┌─────────────────┐
│ Topic Manager   │  Get topic and partition
│ GetTopic()      │
└─────────────────┘
         │
         ▼
┌─────────────────┐
│  Log Reader     │  Read records from offset
│ Read(offset)    │
└─────────────────┘
         │
         ▼
┌─────────────────┐
│  JSON Encoder   │  Convert to JSON Lines
└─────────────────┘
         │
         ▼
┌─────────────────┐
│  File Writer    │  Write to output.json
└─────────────────┘
         │
         ▼
    Exported N messages ✓
```

## User Interface Examples

### Topic List Output
```
┌────────────┬────────────┬────────────────────┐
│   TOPIC    │ PARTITIONS │ REPLICATION FACTOR │
├────────────┼────────────┼────────────────────┤
│ events     │ 10         │ 3                  │
│ logs       │ 5          │ 2                  │
│ metrics    │ 20         │ 3                  │
└────────────┴────────────┴────────────────────┘
```

### Topic Describe Output
```
Topic: events
Partitions: 10
Replication Factor: 3

Partition Details:
┌───────────┬──────────────┬─────────────┬─────────┐
│ PARTITION │ SIZE (BYTES) │   REPLICAS  │   ISR   │
├───────────┼──────────────┼─────────────┼─────────┤
│ 0         │ 1048576      │ [1 2 3]     │ [1 2 3] │
│ 1         │ 2097152      │ [2 3 4]     │ [2 3]   │
│ ...       │              │             │         │
└───────────┴──────────────┴─────────────┴─────────┘
```

### Consumer Group Describe Output
```
Group ID: my-group
Protocol Type: consumer
State: Stable
Protocol: range
Leader: consumer-1
Generation ID: 5
Members: 3

Members:
┌──────────────┬───────────┬────────────┬─────────────────┬────────────────┐
│  MEMBER ID   │ CLIENT ID │    HOST    │ SESSION TIMEOUT │ LAST HEARTBEAT │
├──────────────┼───────────┼────────────┼─────────────────┼────────────────┤
│ consumer-1   │ client-1  │ 10.0.1.10  │ 30000ms         │ 2s ago         │
│ consumer-2   │ client-2  │ 10.0.1.11  │ 30000ms         │ 1s ago         │
│ consumer-3   │ client-3  │ 10.0.1.12  │ 30000ms         │ 3s ago         │
└──────────────┴───────────┴────────────┴─────────────────┴────────────────┘
```

### Data Stats Output
```
Topic: events
Partitions: 10
  Partition 0: Size=1048576 bytes, Messages=1000
  Partition 1: Size=2097152 bytes, Messages=2000
  ...
Total Size: 52428800 bytes (50.00 MB)
Total Messages: 50000

=== Overall Statistics ===
Total Topics: 3
Total Size: 157286400 bytes (150.00 MB)
Total Messages: 150000
```

### JSON Export Format
```json
{"offset":0,"timestamp":1704564000,"key":"user-123","value":"Login event"}
{"offset":1,"timestamp":1704564001,"key":"user-456","value":"Purchase event"}
{"offset":2,"timestamp":1704564002,"key":"user-789","value":"Logout event"}
```

## Error Handling Flow

```
User Input
    │
    ▼
┌─────────────────┐
│ Validation      │  Check required flags
└─────────────────┘
    │         │
    │         └──────► Error: missing --topic flag
    ▼
┌─────────────────┐
│ Execute Command │  Run business logic
└─────────────────┘
    │         │
    │         └──────► Error: topic not found
    ▼
┌─────────────────┐
│ Format Output   │  Display results
└─────────────────┘
    │
    ▼
  Success!
```

## Integration Points

```
┌──────────────────────────────────────────────────────┐
│                   Takhin CLI                          │
└──────────────────────────────────────────────────────┘
                          │
        ┌─────────────────┼─────────────────┐
        │                 │                 │
        ▼                 ▼                 ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│   Scripts    │  │   Humans     │  │  Automation  │
└──────────────┘  └──────────────┘  └──────────────┘
        │                 │                 │
        │                 │                 │
        ▼                 ▼                 ▼
┌──────────────────────────────────────────────────────┐
│              Common Use Cases                         │
├──────────────────────────────────────────────────────┤
│  • Manual admin operations                           │
│  • Backup/restore scripts                            │
│  • CI/CD pipelines                                   │
│  • Monitoring scripts                                │
│  • Data migration                                     │
└──────────────────────────────────────────────────────┘
```

## Performance Characteristics

```
Operation          │ Time Complexity │ Notes
─────────────────────────────────────────────────────
topic list         │ O(n)            │ n = number of topics
topic create       │ O(p)            │ p = partitions
topic describe     │ O(p)            │ Reads metadata only
group list         │ O(g)            │ g = number of groups
data export        │ O(m)            │ m = messages to export
data import        │ O(m)            │ Append to log
```

## Dependencies Graph

```
takhin-cli (main)
    │
    ├── github.com/spf13/cobra
    │       └── Command routing and parsing
    │
    ├── github.com/olekukonko/tablewriter
    │       └── Formatted table output
    │
    └── Internal Packages
            ├── pkg/storage/topic
            │       └── Topic and partition management
            ├── pkg/coordinator
            │       └── Consumer group coordination
            ├── pkg/config
            │       └── Configuration loading/validation
            └── pkg/storage/log
                    └── Log segment operations
```

## File System Layout After Operations

```
/var/data/takhin/
│
├── my-topic-0/
│   ├── 00000000000000000000.log       ← Data file
│   ├── 00000000000000000000.index     ← Offset index
│   ├── 00000000000000000000.timeindex ← Time index
│   └── metadata.json                  ← Topic metadata
│
├── my-topic-1/
│   └── ...
│
└── consumer_groups/                   ← (Future: Group persistence)
    └── my-group/
        └── offsets.json
```

## Security Model

```
┌────────────┐
│    User    │
└────────────┘
      │
      ▼
┌────────────────────────────┐
│  File System Permissions   │  Standard Unix permissions
└────────────────────────────┘
      │
      ▼
┌────────────────────────────┐
│    Direct File Access      │  No network, no authentication
└────────────────────────────┘
      │
      ▼
┌────────────────────────────┐
│   Data Directory Access    │  Read/Write as needed
└────────────────────────────┘

Note: CLI operates with user's file system permissions.
      Admin access required for production data directories.
```

## Workflow Example: Topic Migration

```
Source Cluster                    Destination Cluster
     │                                   │
     ▼                                   │
┌─────────────┐                         │
│Export Data  │                         │
│takhin-cli   │                         │
│data export  │                         │
└─────────────┘                         │
     │                                   │
     ▼                                   │
┌─────────────┐                         │
│export.json  │─────────────────────────┤
└─────────────┘                         │
                                         ▼
                                 ┌──────────────┐
                                 │Create Topic  │
                                 │takhin-cli    │
                                 │topic create  │
                                 └──────────────┘
                                         │
                                         ▼
                                 ┌──────────────┐
                                 │Import Data   │
                                 │takhin-cli    │
                                 │data import   │
                                 └──────────────┘
```

## See Also

- Implementation Details: `TASK_8.4_CLI_COMPLETION.md`
- Quick Reference: `TASK_8.4_CLI_QUICK_REFERENCE.md`
- Test Script: `scripts/test_cli.sh`
