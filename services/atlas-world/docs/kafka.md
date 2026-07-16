# Kafka

## Topics Consumed

| Topic | Environment Variable | Description |
|-------|---------------------|-------------|
| Channel Status Event | EVENT_TOPIC_CHANNEL_STATUS | Channel server status events |
| Tenant Configuration Status | EVENT_TOPIC_CONFIGURATION_TENANT_STATUS | Tenant configuration status events (config projection) |

## Topics Produced

| Topic | Environment Variable | Description |
|-------|---------------------|-------------|
| Channel Status Command | COMMAND_TOPIC_CHANNEL_STATUS | Channel status request commands |
| Channel Status Event | EVENT_TOPIC_CHANNEL_STATUS | Channel started events |
| World Rate Event | EVENT_TOPIC_WORLD_RATE | World rate change events |

## Message Types

### StatusCommand

Direction: Command (Produced)

| Field | Type | Description |
|-------|------|-------------|
| type | string | Command type |

Command Types:
- `STATUS_REQUEST`: Request channel services to report status

### StatusEvent

Direction: Event (Consumed/Produced)

| Field | Type | Description |
|-------|------|-------------|
| type | string | Event type |
| worldId | byte | World identifier |
| channelId | byte | Channel identifier |
| ipAddress | string | Server IP address |
| port | int | Server port |
| currentCapacity | uint32 | Current player count |
| maxCapacity | uint32 | Maximum player capacity |

Event Types:
- `STARTED`: Channel server has started
- `SHUTDOWN`: Channel server is shutting down

### WorldRateEvent

Direction: Event (Produced)

| Field | Type | Description |
|-------|------|-------------|
| type | string | Event type |
| worldId | byte | World identifier |
| rateType | string | Rate type (exp, meso, item_drop, quest_exp) |
| multiplier | float64 | New rate multiplier value |

Event Types:
- `RATE_CHANGED`: A world rate multiplier has changed

### TenantEnvelope

Direction: Event (Consumed)

| Field | Type | Description |
|-------|------|-------------|
| schema_version | int | Envelope schema version |
| id | string | Tenant identifier UUID |
| config | json.RawMessage | Tenant configuration payload |
| emitted_at | string | Event emission timestamp |

A nil message value is a tombstone keyed `tenant:{id}`.

## Transaction Semantics

- Messages are keyed by tenant ID
- Consumer group (channel status commands/events): "World Orchestrator"
- Consumer group (tenant configuration status projection): "World Orchestrator - projection - {uuid}" (per-process, replays from FirstOffset on every container start)
- Headers: SpanHeaderParser, TenantHeaderParser
