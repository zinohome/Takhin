# Consumer Group Monitoring - Quick Start Guide

## Features Overview

### 1. Consumer Groups List Page
**Route:** `/consumer-groups`

**Features:**
- View all consumer groups at a glance
- Real-time state monitoring (auto-refresh every 5s)
- State color indicators:
  - 🟢 **Green (Stable)**: Group is actively consuming
  - 🟠 **Orange (Rebalancing)**: Group is rebalancing partitions
  - 🔵 **Blue (Empty)**: No active members
  - 🔴 **Red (Dead)**: Group is no longer active
- Click any group ID to view details

### 2. Consumer Group Detail Page
**Route:** `/consumer-groups/{groupId}`

**Sections:**

#### Overview Card
- **State**: Current group state with color tag
- **Total Lag**: Aggregate lag across all partitions
- **Members**: Count of active consumers
- **Progress**: Overall consumption percentage with progress bar
- **Metadata**: Protocol type and protocol name

#### Members Table
Shows all active members in the consumer group:
- Member ID (unique identifier)
- Client ID (application identifier)
- Client Host (consumer location)
- Assigned Partitions (count)

#### Offset Commits & Lag Table
Comprehensive lag monitoring per topic-partition:
- **Topic**: Topic name
- **Partition**: Partition number
- **Current Offset**: Last committed offset
- **High Water Mark**: Latest available offset
- **Lag**: Uncommitted messages (color-coded)
  - 🟢 Green: 0 lag (caught up)
  - 🟠 Orange: 1-1000 messages behind
  - 🔴 Red: >1000 messages behind
- **Progress**: Visual progress bar per partition

### 3. Reset Offsets Functionality

**Access:** Click "Reset Offsets" button on detail page

**Requirements:**
- Group must be in **Empty** or **Dead** state
- Button is disabled for active groups

**Strategies:**

1. **Reset to Earliest**
   - Sets offset to 0
   - Replays all messages from beginning
   - Use case: Reprocess all data

2. **Reset to Latest**
   - Sets offset to high water mark
   - Skips all existing messages
   - Use case: Start fresh, ignore backlog

3. **Reset to Specific** (API only)
   - Provide exact offsets per topic-partition
   - Use case: Resume from specific checkpoint

**Workflow:**
1. Stop all consumers (group becomes Empty)
2. Click "Reset Offsets"
3. Select strategy (earliest or latest)
4. Review warning message
5. Confirm reset
6. Restart consumers with new offsets

## API Endpoints

### List Consumer Groups
```bash
GET /api/consumer-groups
```

**Response:**
```json
[
  {
    "groupId": "my-consumer-group",
    "state": "Stable",
    "members": 3
  }
]
```

### Get Consumer Group Details
```bash
GET /api/consumer-groups/{groupId}
```

**Response:**
```json
{
  "groupId": "my-consumer-group",
  "state": "Stable",
  "protocolType": "consumer",
  "protocol": "range",
  "members": [
    {
      "memberId": "consumer-1-abc123",
      "clientId": "consumer-1",
      "clientHost": "/192.168.1.100",
      "partitions": [0, 1]
    }
  ],
  "offsetCommits": [
    {
      "topic": "orders",
      "partition": 0,
      "offset": 1500,
      "highWaterMark": 2000,
      "lag": 500,
      "metadata": ""
    }
  ]
}
```

### Reset Offsets
```bash
POST /api/consumer-groups/{groupId}/reset-offsets
Content-Type: application/json

{
  "strategy": "earliest"  // or "latest" or "specific"
}
```

**For specific offsets:**
```json
{
  "strategy": "specific",
  "offsets": {
    "orders": {
      "0": 1000,
      "1": 2000
    },
    "payments": {
      "0": 500
    }
  }
}
```

## Usage Examples

### Monitor Consumer Lag

1. Navigate to Consumer Groups page
2. Find your consumer group in the list
3. Click on the group ID
4. Check the "Total Lag" statistic in the Overview
5. Review per-partition lag in the Offset Commits table
6. Red-colored lags indicate high backlog requiring attention

### Replay Messages (Reset to Earliest)

**Scenario:** Need to reprocess all messages due to bug fix

1. Stop all consumers in the group
2. Wait for group state to become "Empty"
3. Navigate to the group detail page
4. Click "Reset Offsets" button
5. Select "Reset to Earliest"
6. Confirm the action
7. Restart consumers - they will start from offset 0

### Skip Backlog (Reset to Latest)

**Scenario:** Large backlog, only want new messages

1. Stop all consumers in the group
2. Wait for group state to become "Empty"
3. Navigate to the group detail page
4. Click "Reset Offsets" button
5. Select "Reset to Latest"
6. Confirm the action
7. Restart consumers - they will start from current HWM

### Monitor Rebalancing

1. Open consumer group detail page
2. Watch state change to "PreparingRebalance"
3. Observe members list updating
4. State changes to "CompletingRebalance"
5. Finally returns to "Stable"
6. Check partition reassignments in members table

## Lag Interpretation

### Healthy Consumer Group
- **State**: Stable
- **Lag**: 0-100 messages per partition
- **Progress**: >95%
- **Members**: Active and stable

### Struggling Consumer Group
- **State**: Stable
- **Lag**: >1000 messages per partition (red)
- **Progress**: <80%
- **Action**: Scale consumers or optimize processing

### Inactive Consumer Group
- **State**: Empty or Dead
- **Lag**: Increasing
- **Progress**: Decreasing
- **Action**: Restart consumers or reset offsets

## Best Practices

### Lag Monitoring
1. Set up monitoring alerts for lag >1000
2. Check lag trends over time
3. Investigate sudden lag increases
4. Balance partitions across consumers

### Offset Resets
1. Always stop consumers first
2. Wait for Empty state before resetting
3. Test reset strategy in staging first
4. Document why reset was needed
5. Monitor closely after reset

### Consumer Groups
1. Use descriptive group IDs
2. Keep groups small (3-10 consumers)
3. Match partition count to expected consumers
4. Monitor member stability
5. Handle rebalances gracefully in code

## Troubleshooting

### "Reset Offsets" Button Disabled
- **Cause**: Group is in Stable or Rebalancing state
- **Solution**: Stop all consumers, wait for Empty state

### High Lag Not Decreasing
- **Cause**: Consumers too slow or not enough consumers
- **Solution**: Scale horizontally (add consumers) or optimize processing

### Frequent Rebalancing
- **Cause**: Consumers timing out or joining/leaving
- **Solution**: Increase session timeout, fix consumer stability

### Group Not Visible
- **Cause**: No offset commits yet
- **Solution**: Consumers need to commit at least once

## Real-time Updates

All pages auto-refresh every 5 seconds to show:
- Latest group states
- Current lag values
- Member changes
- Offset progress

Click the "Refresh" button for immediate updates.

## Related Tasks
- Task 2.2: Topic monitoring integration
- Task 2.3: Message browsing integration
- Task 2.4: Producer monitoring (future)

## Support
For issues or questions, check the main documentation or API Swagger UI at `/swagger/index.html`
