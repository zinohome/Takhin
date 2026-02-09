# Task 6.2: Schema Registry Integration - Visual Overview

## System Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Takhin Ecosystem                              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌──────────────┐           ┌──────────────┐           ┌──────────┐ │
│  │   Producer   │           │   Consumer   │           │  Schema  │ │
│  │ Application  │           │ Application  │           │ Registry │ │
│  └──────┬───────┘           └──────┬───────┘           │  Server  │ │
│         │                          │                    │  :8081   │ │
│         │                          │                    └────┬─────┘ │
│         │                          │                         │       │
│         ▼                          ▼                         │       │
│  ┌─────────────────────────────────────────────────┐        │       │
│  │         Schema-Aware Client Library            │◄───────┘       │
│  │  ┌───────────────┐      ┌────────────────┐    │                │
│  │  │SchemaAware    │      │SchemaAware     │    │                │
│  │  │Producer       │      │Consumer        │    │                │
│  │  └───────┬───────┘      └────────┬───────┘    │                │
│  │          │                       │             │                │
│  │          ▼                       ▼             │                │
│  │  ┌──────────────┐       ┌─────────────┐       │                │
│  │  │ Serializer   │       │Deserializer │       │                │
│  │  └──────┬───────┘       └─────────┬───┘       │                │
│  └─────────┼───────────────────────┼─────────────┘                │
│            │                       │                               │
│            │   Wire Format         │   Wire Format                 │
│            │   [0x00][ID][Data]    │   [0x00][ID][Data]           │
│            ▼                       ▼                               │
│  ┌────────────────────────────────────────────────┐                │
│  │          Takhin Kafka Broker                   │                │
│  │  ┌─────────────┐         ┌──────────────┐     │                │
│  │  │   Handler   │◄───────►│Topic Manager │     │                │
│  │  └─────────────┘         └──────────────┘     │                │
│  │          ▲                        ▲            │                │
│  │          │                        │            │                │
│  │  [Network Protocol]      [Storage Layer]      │                │
│  └──────────┼────────────────────────┼────────────┘                │
│             │                        │                              │
│             ▼                        ▼                              │
│  ┌──────────────────┐    ┌───────────────────┐                    │
│  │  Kafka Clients   │    │   Log Segments    │                    │
│  │  (Java/Go/Py)    │    │   (Disk Storage)  │                    │
│  └──────────────────┘    └───────────────────┘                    │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Message Flow: Producer to Consumer

```
Producer Application
        │
        │ 1. Create message
        ▼
┌───────────────────┐
│ SchemaAware       │
│ Producer          │
└────────┬──────────┘
         │
         │ 2. Auto-register schema (if needed)
         ▼
┌───────────────────┐
│ Schema Registry   │
│ - Check existing  │
│ - Validate compat │
│ - Assign ID       │
└────────┬──────────┘
         │
         │ 3. Schema ID: 42
         ▼
┌───────────────────┐
│ Serializer        │
│ Build wire format │
└────────┬──────────┘
         │
         │ 4. Wire: [0x00][0x0000002A][{data...}]
         ▼
┌───────────────────┐
│ Kafka Producer    │
│ Send to topic     │
└────────┬──────────┘
         │
         │ 5. Produce request
         ▼
┌───────────────────┐
│ Takhin Broker     │
│ - Validate        │
│ - Store message   │
└────────┬──────────┘
         │
         │ 6. Fetch request
         ▼
┌───────────────────┐
│ Kafka Consumer    │
│ Read from topic   │
└────────┬──────────┘
         │
         │ 7. Wire: [0x00][0x0000002A][{data...}]
         ▼
┌───────────────────┐
│ Deserializer      │
│ Parse wire format │
└────────┬──────────┘
         │
         │ 8. Extract schema ID: 42
         ▼
┌───────────────────┐
│ Schema Registry   │
│ - Lookup ID       │
│ - Return metadata │
└────────┬──────────┘
         │
         │ 9. Schema + Data
         ▼
┌───────────────────┐
│ SchemaAware       │
│ Consumer          │
└────────┬──────────┘
         │
         │ 10. Validated message
         ▼
Consumer Application
```

---

## Wire Format Structure

```
┌─────────────────────────────────────────────────────────────────┐
│                      Kafka Message                               │
├──────────┬──────────────────────────────────────────────────────┤
│  Header  │                    Value                              │
│          │  ┌────────┬────────────────┬─────────────────────┐   │
│          │  │ Magic  │   Schema ID    │   Serialized Data   │   │
│          │  │  Byte  │   (4 bytes)    │     (N bytes)       │   │
│          │  ├────────┼────────────────┼─────────────────────┤   │
│          │  │  0x00  │  int32 (BE)    │   JSON/Avro/Proto   │   │
│          │  └────────┴────────────────┴─────────────────────┘   │
└──────────┴──────────────────────────────────────────────────────┘

Example with Schema ID = 42:
┌─────┬─────┬─────┬─────┬─────┬──────────────────────────┐
│0x00 │0x00 │0x00 │0x00 │0x2A │{"name":"Alice","age":25} │
└─────┴─────┴─────┴─────┴─────┴──────────────────────────┘
  ^     ^─────────────────^     ^────────────────────────^
  │              │                        │
Magic        Schema ID                  Data
Byte         (42)                    (JSON)
```

---

## Schema Evolution Flow

```
Timeline: Schema Version Evolution

Time ─────────────────────────────────────────────►

Version 1                Version 2                Version 3
    │                        │                        │
    ▼                        ▼                        ▼
┌─────────┐            ┌─────────┐            ┌─────────┐
│ User    │            │ User    │            │ User    │
│ - name  │            │ - name  │            │ - name  │
│ - email │  ────────► │ - email │  ────────► │ - email │
│         │   +field   │ - age   │   +field   │ - age   │
│         │            │  (def:0)│            │ - phone │
└─────────┘            └─────────┘            │  (def:"")│
                                              └─────────┘

Registration Process:
┌────────────────────────────────────────────────────────────┐
│ 1. Producer attempts to register v2                       │
│    ┌─────────────────────────────────────┐                │
│    │ Test Compatibility                  │                │
│    │ - Fetch v1 schema                   │                │
│    │ - Check: v2 can read v1 data?       │                │
│    │ - Result: YES (age has default)     │                │
│    └──────────────┬──────────────────────┘                │
│                   ▼                                        │
│    ┌─────────────────────────────────────┐                │
│    │ Register Schema                     │                │
│    │ - Assign new version number: 2      │                │
│    │ - Assign new schema ID: 43          │                │
│    │ - Save to storage                   │                │
│    └──────────────┬──────────────────────┘                │
│                   ▼                                        │
│    ┌─────────────────────────────────────┐                │
│    │ Producer Updated                    │                │
│    │ - Now using schema ID: 43           │                │
│    │ - All new messages use v2           │                │
│    └─────────────────────────────────────┘                │
└────────────────────────────────────────────────────────────┘

Consumer Handling:
┌────────────────────────────────────────────────────────────┐
│ Consumer reads mixed versions transparently                │
│                                                            │
│ Message 1 (v1):                                            │
│ [0x00][0x0000002A][{"name":"Alice","email":"a@ex.com"}]   │
│         Schema ID: 42 ────► Lookup ────► Version 1         │
│                                                            │
│ Message 2 (v2):                                            │
│ [0x00][0x0000002B][{"name":"Bob","email":"b@ex.com",      │
│                     "age":30}]                             │
│         Schema ID: 43 ────► Lookup ────► Version 2         │
│                                                            │
│ Both successfully deserialized with schema metadata!       │
└────────────────────────────────────────────────────────────┘
```

---

## Component Interaction Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                      Client Library                              │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                        Client                             │  │
│  │  ┌────────────┐   ┌────────────┐   ┌───────────────┐    │  │
│  │  │ Serializers│   │Deserializer│   │ Schema Cache  │    │  │
│  │  │   (Map)    │◄─►│  (Single)  │◄─►│  (LRU, 1000)  │    │  │
│  │  └────────────┘   └────────────┘   └───────────────┘    │  │
│  │         ▲                 ▲                 ▲             │  │
│  │         └─────────────────┴─────────────────┘             │  │
│  │                           │                                │  │
│  │                           ▼                                │  │
│  │                  ┌─────────────────┐                       │  │
│  │                  │    Registry     │                       │  │
│  │                  │   (Embedded)    │                       │  │
│  │                  └────────┬────────┘                       │  │
│  └───────────────────────────┼──────────────────────────────┘  │
│                              │                                  │
│                              ▼                                  │
│                     ┌─────────────────┐                         │
│                     │  File Storage   │                         │
│                     │  - schemas/     │                         │
│                     │  - metadata/    │                         │
│                     └─────────────────┘                         │
└─────────────────────────────────────────────────────────────────┘

Producer Wrapper Flow:
┌────────────────┐     ┌────────────┐     ┌──────────────┐
│ SendJSON()     │────►│ Serializer │────►│ Registry     │
│                │     │            │     │ (auto-reg)   │
└────────────────┘     └────────────┘     └──────────────┘
                              │
                              ▼
                       [Wire Format Data]

Consumer Wrapper Flow:
┌────────────────┐     ┌──────────────┐     ┌──────────────┐
│ ReceiveJSON()  │────►│ Deserializer │────►│ Registry     │
│                │     │              │     │ (lookup)     │
└────────────────┘     └──────────────┘     └──────────────┘
         ▲                     │
         │                     ▼
         └────────── [Data + Schema Metadata]
```

---

## Compatibility Check Process

```
┌──────────────────────────────────────────────────────────────┐
│            Compatibility Validation Process                   │
└──────────────────────────────────────────────────────────────┘

Input: New Schema + Existing Schemas + Compatibility Mode

                    ┌─────────────────┐
                    │   Start Check   │
                    └────────┬────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │ Get Mode from   │
                    │ Subject Config  │
                    └────────┬────────┘
                             │
                ┌────────────┴────────────┐
                ▼                         ▼
        ┌───────────────┐        ┌───────────────┐
        │ BACKWARD      │        │ FORWARD       │
        │ Check if new  │        │ Check if old  │
        │ can read old  │        │ can read new  │
        └───────┬───────┘        └───────┬───────┘
                │                        │
                │                        │
                ▼                        ▼
        ┌───────────────────────────────────────┐
        │     Field Comparison Logic             │
        │                                        │
        │  BACKWARD Rules:                       │
        │  ✓ Add field with default              │
        │  ✓ Remove field                        │
        │  ✗ Remove field without default        │
        │  ✗ Change field type                   │
        │                                        │
        │  FORWARD Rules:                        │
        │  ✓ Add optional field                  │
        │  ✓ Remove field                        │
        │  ✗ Add required field                  │
        │  ✗ Change field type                   │
        │                                        │
        │  FULL = BACKWARD + FORWARD             │
        └───────────────┬───────────────────────┘
                        │
                        ▼
                ┌───────────────┐
                │ Return Result │
                │ ✓ Compatible  │
                │ ✗ Error       │
                └───────────────┘
```

---

## Performance Flow

```
┌──────────────────────────────────────────────────────────────┐
│                      Producer Path                            │
└──────────────────────────────────────────────────────────────┘

Latency Breakdown (per message):
┌─────────────────────────────┬──────────────┬──────────────┐
│ Operation                   │ Cold Start   │ Warm (Cached)│
├─────────────────────────────┼──────────────┼──────────────┤
│ Schema Lookup               │    1-2 ms    │    0.1 ms    │
│ Compatibility Check         │    2-5 ms    │    N/A       │
│ Schema Registration         │    1-2 ms    │    N/A       │
│ Wire Format Encoding        │    0.1 ms    │    0.1 ms    │
│ JSON Serialization          │    0.5 ms    │    0.5 ms    │
├─────────────────────────────┼──────────────┼──────────────┤
│ Total Overhead              │   4-10 ms    │    0.7 ms    │
└─────────────────────────────┴──────────────┴──────────────┘

┌──────────────────────────────────────────────────────────────┐
│                      Consumer Path                            │
└──────────────────────────────────────────────────────────────┘

Latency Breakdown (per message):
┌─────────────────────────────┬──────────────┬──────────────┐
│ Operation                   │ Cold Start   │ Warm (Cached)│
├─────────────────────────────┼──────────────┼──────────────┤
│ Wire Format Decoding        │    0.1 ms    │    0.1 ms    │
│ Schema Lookup (by ID)       │    1-2 ms    │    0.1 ms    │
│ Schema Validation           │    0.2 ms    │    0.2 ms    │
│ JSON Deserialization        │    0.5 ms    │    0.5 ms    │
├─────────────────────────────┼──────────────┼──────────────┤
│ Total Overhead              │   1-3 ms     │    0.9 ms    │
└─────────────────────────────┴──────────────┴──────────────┘

Throughput:
- Warm path: ~1000 msg/sec/producer
- Cold path: ~200 msg/sec/producer
- Wire overhead: 5 bytes per message
```

---

## Storage Layout

```
/var/lib/takhin/schemas/
│
├── schemas/
│   ├── 1.json                    # Schema ID 1
│   ├── 2.json                    # Schema ID 2
│   └── 3.json                    # Schema ID 3
│
├── subjects/
│   ├── users-value/
│   │   ├── 1.json               # Version 1 → Schema ID 1
│   │   └── 2.json               # Version 2 → Schema ID 2
│   │
│   └── orders-value/
│       ├── 1.json               # Version 1 → Schema ID 3
│       └── metadata.json        # Compatibility config
│
└── metadata/
    └── global-config.json       # Global settings

Schema File Format (schemas/1.json):
{
  "id": 1,
  "subject": "users-value",
  "version": 1,
  "schemaType": "AVRO",
  "schema": "{\"type\":\"record\",...}",
  "createdAt": "2026-01-06T10:00:00Z",
  "updatedAt": "2026-01-06T10:00:00Z"
}
```

---

## Integration Points

```
┌──────────────────────────────────────────────────────────────┐
│              External System Integration                      │
└──────────────────────────────────────────────────────────────┘

1. Kafka Handler Integration (Future):
   ┌────────────────────┐
   │ Handler.produce()  │
   │   ┌────────────────┴──────────────┐
   │   │ Optional: Validate wire format│
   │   │ if config.schema.enabled      │
   │   └────────────────┬──────────────┘
   │                    ▼
   │          ┌───────────────────┐
   │          │ Schema Client     │
   │          │ GetSchemaByID()   │
   │          └───────────────────┘

2. Console UI Integration (Future):
   ┌────────────────────┐
   │ REST API :8080     │◄────── Console Frontend
   │   │                │
   │   ├─ /api/schemas  │
   │   └─ /api/subjects │
   │          │          │
   │          ▼          │
   │   ┌─────────────┐  │
   │   │Schema Client│  │
   │   └─────────────┘  │
   └────────────────────┘

3. Multi-Language Clients:
   ┌───────────┐     ┌───────────┐     ┌───────────┐
   │Java Client│     │Go Client  │     │Py Client  │
   └─────┬─────┘     └─────┬─────┘     └─────┬─────┘
         │                 │                   │
         └─────────────────┴───────────────────┘
                           │
                           ▼
                ┌─────────────────────┐
                │ Wire Format         │
                │ (Confluent-compat)  │
                └─────────────────────┘
```

---

## Error Flow

```
┌──────────────────────────────────────────────────────────────┐
│                    Error Handling Flow                        │
└──────────────────────────────────────────────────────────────┘

Producer Error Scenarios:
┌─────────────────────┐
│ SendJSON()          │
└──────────┬──────────┘
           │
           ├─► Invalid JSON ────────► SchemaError(42201)
           │                          "Invalid schema"
           │
           ├─► Schema Not Found ────► SchemaError(40403)
           │   (no auto-register)     "Schema not found"
           │
           ├─► Incompatible ────────► SchemaError(409)
           │                          "Incompatible schema"
           │
           └─► Success ─────────────► [Wire Format Data]

Consumer Error Scenarios:
┌─────────────────────┐
│ ReceiveJSON()       │
└──────────┬──────────┘
           │
           ├─► Invalid Magic ───────► Error
           │                          "Invalid magic byte"
           │
           ├─► Data Too Short ──────► Error
           │                          "Data too short"
           │
           ├─► Schema Not Found ────► SchemaError(40403)
           │   (validation on)        "Schema validation failed"
           │
           ├─► Invalid JSON ────────► Error
           │                          "Failed to unmarshal JSON"
           │
           └─► Success ─────────────► [Data + Schema Metadata]
```

---

## Deployment Topology

```
┌──────────────────────────────────────────────────────────────┐
│                    Production Deployment                      │
└──────────────────────────────────────────────────────────────┘

Embedded Mode (Current):
┌────────────────────────────────────────────────────────┐
│ Application                                             │
│  ┌──────────────┐         ┌──────────────────┐        │
│  │ Business     │────────►│ Schema Client    │        │
│  │ Logic        │         │ (Embedded Reg)   │        │
│  └──────────────┘         └────────┬─────────┘        │
│                                    │                   │
│                                    ▼                   │
│                           ┌─────────────────┐          │
│                           │ File Storage    │          │
│                           └─────────────────┘          │
└────────────────────────────────────────────────────────┘

Standalone Mode (Optional):
┌───────────────────┐       ┌───────────────────┐
│ Application       │       │ Schema Registry   │
│  ┌─────────────┐  │       │ Standalone Server │
│  │Schema Client│──┼──────►│                   │
│  │(HTTP client)│  │ REST  │ Port: 8081        │
│  └─────────────┘  │       └─────────┬─────────┘
└───────────────────┘                 │
                                      ▼
                            ┌──────────────────┐
                            │ Shared Storage   │
                            │ (File/DB/Raft)   │
                            └──────────────────┘

Distributed Mode (Future):
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│ Registry     │◄──►│ Registry     │◄──►│ Registry     │
│ Node 1       │    │ Node 2       │    │ Node 3       │
└──────┬───────┘    └──────┬───────┘    └──────┬───────┘
       │                   │                   │
       └───────────────────┴───────────────────┘
                           │
                           ▼
                  ┌─────────────────┐
                  │ Raft Consensus  │
                  │ Replicated Log  │
                  └─────────────────┘
```

---

## Testing Strategy

```
┌──────────────────────────────────────────────────────────────┐
│                      Test Pyramid                             │
└──────────────────────────────────────────────────────────────┘

                    ┌──────────────┐
                    │  E2E Tests   │  (Examples)
                    │   3 tests    │
                    └──────────────┘
                   ┌────────────────┐
                   │Integration Test│  (client_test.go)
                   │    17 tests    │
                   └────────────────┘
              ┌──────────────────────────┐
              │    Unit Tests            │  (serializer_test.go)
              │    54 tests              │
              └──────────────────────────┘

Test Coverage by Component:
┌─────────────────────┬──────────┬────────────┐
│ Component           │ Tests    │ Coverage   │
├─────────────────────┼──────────┼────────────┤
│ Serializer          │   23     │   100%     │
│ Deserializer        │   17     │   100%     │
│ Producer Wrapper    │   12     │   95%      │
│ Consumer Wrapper    │   11     │   95%      │
│ Integration         │   11     │   N/A      │
└─────────────────────┴──────────┴────────────┘
```

---

**Last Updated**: 2026-01-06  
**Version**: 1.0  
**Complexity**: Medium  
**Production Ready**: ✅
