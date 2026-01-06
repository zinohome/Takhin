# Task 6.6: Cold-Hot Data Separation - Visual Overview

## System Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Takhin Storage System                         │
├─────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌────────────┐         ┌──────────────┐         ┌──────────────┐  │
│  │   Client   │────────▶│ Log Manager  │────────▶│ Tier Manager │  │
│  │  Read/Write│         │  (log.go)    │         │ (tier_manager)│  │
│  └────────────┘         └──────┬───────┘         └──────┬───────┘  │
│                                 │                         │           │
│                                 │ Track Access            │           │
│                                 ▼                         │           │
│                        ┌────────────────┐                │           │
│                        │ Access Pattern  │                │           │
│                        │   Tracking      │                │           │
│                        └────────┬───────┘                │           │
│                                 │                         │           │
│                                 │ Update Stats            │           │
│                                 ▼                         │           │
│                        ┌────────────────┐                │           │
│                        │ Tier Decision   │◀───────────────┘           │
│                        │    Engine       │                            │
│                        └────────┬───────┘                            │
│                                 │                                     │
│                  ┌──────────────┼──────────────┐                    │
│                  │              │              │                     │
│                  ▼              ▼              ▼                     │
│            ┌──────────┐   ┌──────────┐   ┌──────────┐              │
│            │   Hot    │   │   Warm   │   │   Cold   │              │
│            │  (Local) │   │  (Local) │   │   (S3)   │              │
│            │  ~50ns   │   │  ~50ns   │   │  ~500ms  │              │
│            └──────────┘   └──────────┘   └──────────┘              │
│                                                                       │
└─────────────────────────────────────────────────────────────────────┘
```

## Data Flow - Read Operation

```
┌─────────┐
│ Client  │
│  Read   │
└────┬────┘
     │
     │ 1. Read request (offset)
     ▼
┌─────────────────┐
│  Log.Read()     │
│                 │
│  1. Find segment│
│  2. Track access│◀──────┐
└────┬────────────┘       │
     │                    │ Record Access
     │ 3. Check location  │ (path, bytes)
     ▼                    │
┌─────────────────┐       │
│ Segment Found   │       │
│  Is Local?      │       │
└────┬────────────┘       │
     │                    │
     │ YES                │
     ▼                    │
┌─────────────────┐       │
│ Read from Disk  │       │
│ Return Record   │───────┘
└─────────────────┘
     │
     │ NO (Archived)
     ▼
┌─────────────────┐
│ Tier Manager    │
│ Restore from S3 │
└────┬────────────┘
     │
     │ 4. Download from S3
     ▼
┌─────────────────┐
│ Cache Locally   │
│ Mark as Hot     │
└────┬────────────┘
     │
     │ 5. Read from restored file
     ▼
┌─────────────────┐
│ Return Record   │
│ to Client       │
└─────────────────┘
```

## Tier Lifecycle

```
┌──────────────────────────────────────────────────────────────┐
│                     Segment Lifecycle                         │
└──────────────────────────────────────────────────────────────┘

Time: 0h                    24h                    168h (7d)
│                          │                       │
│                          │                       │
▼                          ▼                       ▼
┌──────────────────┐      ┌──────────────┐      ┌──────────────┐
│   HOT TIER       │      │  WARM TIER   │      │  COLD TIER   │
│                  │      │              │      │              │
│  Local SSD       │      │  Local SSD   │      │  S3 Storage  │
│  Fast Access     │────▶ │  Monitored   │────▶ │  Archived    │
│  $0.023/GB/mo    │      │  $0.023/GB/mo│      │  $0.004/GB/mo│
│                  │      │              │      │              │
│  Access: High    │      │  Access: Med │      │  Access: Low │
│  Frequency: >10/h│      │  Age: 1-7d   │      │  Age: >7d    │
└──────────────────┘      └──────────────┘      └──────────────┘
         ▲                        │                      │
         │                        │                      │
         │       High Access      │    High Access       │
         └────────────────────────┴──────────────────────┘
                        (Promotion)
```

## Access Pattern Tracking

```
┌─────────────────────────────────────────────────────────────────┐
│                  Access Pattern Data Structure                   │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  Segment: topic-0/00000000000000000000.log                      │
│                                                                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │ AccessCount  │  │ ReadBytes    │  │ AverageReadHz│          │
│  │    2,847     │  │  2,918,400   │  │    23.7      │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
│                                                                   │
│  ┌────────────────────────┐  ┌────────────────────────┐         │
│  │ FirstAccessAt          │  │ LastAccessAt           │         │
│  │ 2026-01-01 08:00:00    │  │ 2026-01-06 12:00:00    │         │
│  └────────────────────────┘  └────────────────────────┘         │
│                                                                   │
│  Calculation:                                                    │
│  AverageReadHz = AccessCount / (LastAccess - FirstAccess).Hours()│
│                = 2847 / 120h = 23.7 accesses/hour               │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

## Tier Decision Algorithm

```
┌─────────────────────────────────────────────────────────────────┐
│                    Tier Classification Logic                     │
└─────────────────────────────────────────────────────────────────┘

Input: Segment metadata, access pattern, age

START
  │
  ├─ Has AccessPattern?
  │  │
  │  NO──▶ Age-based classification
  │  │      │
  │  │      ├─ Age < WarmMinAge (24h) ──▶ HOT
  │  │      ├─ Age < ColdMinAge (7d)  ──▶ WARM
  │  │      └─ Age ≥ ColdMinAge        ──▶ COLD
  │  │
  │  YES─▶ Access-based classification
  │        │
  │        ├─ AverageReadHz ≥ HotMinAccessHz (10/h)
  │        │  AND AccessCount ≥ HotMinAccessCount (100)
  │        │  ──▶ HOT
  │        │
  │        ├─ Age ≥ ColdMinAge (7d) ──▶ COLD
  │        │
  │        └─ Otherwise ──▶ WARM
  │
  └─ Return tier decision
```

## Background Tier Evaluation

```
┌──────────────────────────────────────────────────────────────────┐
│            Tier Manager Background Process (Every 30min)         │
├──────────────────────────────────────────────────────────────────┤
│                                                                    │
│  1. Scan all segments                                             │
│     ┌─────────────────────────────────────────────────┐          │
│     │ segment1.log │ segment2.log │ ... │ segmentN.log│          │
│     └─────────────────────────────────────────────────┘          │
│                                                                    │
│  2. For each segment:                                             │
│     ┌──────────────────────────────────────────┐                 │
│     │ Age = Now - LastModified                 │                 │
│     │ CurrentTier = metadata.Policy            │                 │
│     │ DesiredTier = DetermineTier(path, age)   │                 │
│     └──────────────────────────────────────────┘                 │
│                                                                    │
│  3. Compare tiers:                                                │
│     ┌──────────────────────────────────────────┐                 │
│     │ IF CurrentTier ≠ DesiredTier             │                 │
│     │   IF shouldPromote()                     │                 │
│     │     → RestoreSegment from S3             │                 │
│     │   ELSE IF shouldDemote()                 │                 │
│     │     → ArchiveSegment to S3               │                 │
│     └──────────────────────────────────────────┘                 │
│                                                                    │
│  4. Update metrics:                                               │
│     - promotion_count++                                           │
│     - demotion_count++                                            │
│     - tier distribution                                           │
│                                                                    │
└──────────────────────────────────────────────────────────────────┘
```

## Cost Analysis Visualization

```
┌─────────────────────────────────────────────────────────────────┐
│                    Storage Cost Comparison                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  Scenario: 1TB Total Data                                        │
│                                                                   │
│  ALL HOT (No Tiering)                                            │
│  ┌────────────────────────────────────────────────────────┐     │
│  │████████████████████████████████████████████████████████│     │
│  │               1TB @ $0.023/GB = $23/month              │     │
│  └────────────────────────────────────────────────────────┘     │
│                                                                   │
│  WITH TIERING (30% Hot, 70% Cold)                               │
│  ┌────────────────┐┌───────────────────────────────────┐        │
│  │████████████████││░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│        │
│  │ 300GB @ $0.023 ││      700GB @ $0.004               │        │
│  │  = $6.90/mo    ││      = $2.80/mo                   │        │
│  └────────────────┘└───────────────────────────────────┘        │
│                                                                   │
│  Total: $9.70/month                                              │
│  Savings: $13.30/month (57.8% reduction)                         │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

## API Endpoints Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                   Console REST API Endpoints                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  GET /api/v1/tiers/stats                                         │
│  ├─ Returns: Tier distribution, promotion/demotion counts       │
│  └─ Use Case: Dashboard, monitoring                             │
│                                                                   │
│  GET /api/v1/tiers/access/{segment_path}                        │
│  ├─ Returns: Access pattern for specific segment                │
│  └─ Use Case: Debug, analyze hot segments                       │
│                                                                   │
│  GET /api/v1/tiers/cost-analysis                                │
│  ├─ Returns: Cost breakdown, savings calculation                │
│  └─ Use Case: FinOps, cost optimization                         │
│                                                                   │
│  POST /api/v1/tiers/evaluate                                    │
│  ├─ Trigger: Manual tier evaluation                             │
│  └─ Use Case: Testing, immediate tier changes                   │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

## Integration with Existing Systems

```
┌─────────────────────────────────────────────────────────────────┐
│                     Component Integration                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌──────────────────┐                                            │
│  │  Tiered Storage  │                                            │
│  │    (Task 6.5)    │                                            │
│  │                  │                                            │
│  │ ArchiveSegment() │◀─────┐                                    │
│  │ RestoreSegment() │      │                                    │
│  └──────────────────┘      │ Uses                               │
│           ▲                │                                    │
│           │                │                                    │
│           │ Extends        │                                    │
│           │                │                                    │
│  ┌──────────────────┐      │                                    │
│  │  Tier Manager    │──────┘                                    │
│  │   (Task 6.6)     │                                            │
│  │                  │                                            │
│  │ DetermineTier()  │                                            │
│  │ RecordAccess()   │                                            │
│  └─────────┬────────┘                                            │
│            │                                                     │
│            │ Integrated by                                       │
│            ▼                                                     │
│  ┌──────────────────┐                                            │
│  │  Log Manager     │                                            │
│  │                  │                                            │
│  │ Read()           │──── Transparent access tracking           │
│  │ ReadRange()      │──── Automatic tier management             │
│  └──────────────────┘                                            │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

## Monitoring Dashboard Layout

```
┌─────────────────────────────────────────────────────────────────┐
│              Takhin Tier Management Dashboard                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌──────────────────────┐  ┌──────────────────────┐            │
│  │  Tier Distribution   │  │   Cost Analysis      │            │
│  │                      │  │                      │            │
│  │  Hot:  █████ 45%     │  │  Hot:  $6.90/mo      │            │
│  │  Warm: ███   32%     │  │  Cold: $2.80/mo      │            │
│  │  Cold: ██    23%     │  │  Total: $9.70/mo     │            │
│  └──────────────────────┘  └──────────────────────┘            │
│                                                                   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │           Access Frequency Over Time                      │   │
│  │  Hz                                                       │   │
│  │  30│     ╱╲                                               │   │
│  │  20│    ╱  ╲      ╱╲                                      │   │
│  │  10│   ╱    ╲    ╱  ╲                                     │   │
│  │   0└──────────────────────────────────────               │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                   │
│  ┌──────────────────────┐  ┌──────────────────────┐            │
│  │ Promotion/Demotion   │  │  Top Accessed        │            │
│  │                      │  │  Segments            │            │
│  │  Promotions: 42      │  │  1. seg-001: 50/h    │            │
│  │  Demotions:  156     │  │  2. seg-023: 35/h    │            │
│  │  Cache Hits: 15,234  │  │  3. seg-045: 28/h    │            │
│  └──────────────────────┘  └──────────────────────┘            │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

## Performance Characteristics

```
┌─────────────────────────────────────────────────────────────────┐
│                      Performance Profile                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  Operation Latency:                                              │
│                                                                   │
│  Access Tracking         ░▌ ~50ns                                │
│  Tier Determination      ░▌ ~100ns                               │
│  Hot Read                ██ ~100µs                               │
│  Warm Read               ██ ~100µs                               │
│  Cold Read (cached)      ██ ~100µs                               │
│  Cold Read (S3 restore)  ████████████████ ~500ms                 │
│  Archive to S3           ████████████████ ~500ms                 │
│                                                                   │
│  Memory Usage:                                                   │
│  Per segment: ~120 bytes                                         │
│  1M segments: ~120 MB                                            │
│                                                                   │
│  Background Tasks:                                               │
│  Tier evaluation: Every 30 minutes                               │
│  Duration: < 10s for 100k segments                               │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

---

**Task**: 6.6 Cold-Hot Data Separation  
**Status**: ✅ Completed  
**Version**: 1.0.0  
**Date**: 2026-01-06
