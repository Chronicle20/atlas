# Kafka

## Topics Consumed

| Topic | Environment Variable | Description |
|-------|---------------------|-------------|
| Channel Status Event | EVENT_TOPIC_CHANNEL_STATUS | Channel server status events |
| Tenant Configuration Status | EVENT_TOPIC_CONFIGURATION_TENANT_STATUS | Tenant configuration status events (config projection) |
| World Broadcast Command | COMMAND_TOPIC_WORLD_BROADCAST | Megaphone/Maple TV broadcast enqueue commands |

## Topics Produced

| Topic | Environment Variable | Description |
|-------|---------------------|-------------|
| Channel Status Command | COMMAND_TOPIC_CHANNEL_STATUS | Channel status request commands |
| Channel Status Event | EVENT_TOPIC_CHANNEL_STATUS | Channel started events |
| World Rate Event | EVENT_TOPIC_WORLD_RATE | World rate change events |
| World Broadcast Status Event | EVENT_TOPIC_WORLD_BROADCAST_STATUS | Broadcast queue status events (QUEUED/STARTED/ENDED) |

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

### EnqueueCommand

Direction: Command (Consumed)

| Field | Type | Description |
|-------|------|-------------|
| family | string | Broadcast family (TV or AVATAR) |
| worldId | byte | World identifier |
| channelId | byte | Channel identifier |
| characterId | uint32 | Character identifier of the sender |
| senderName | string | Sender display name |
| senderMedal | string | Sender medal name |
| messages | []string | Broadcast message lines |
| whispersOn | bool | Whether whispers are enabled during the broadcast |
| itemId | uint32 | Item identifier associated with the broadcast |
| tvMessageType | string | Semantic message type key (NORMAL, STAR, HEART); resolved to a client wire byte at the packet layer, not carried here |
| durationSeconds | uint32 | Duration, in seconds, the broadcast is active once started |
| senderLook | sharedsaga.AvatarSnapshot | Sender avatar appearance snapshot |
| receiverName | string | Receiver display name |
| receiverLook | sharedsaga.AvatarSnapshot | Receiver avatar appearance snapshot (nullable) |

### StatusEvent (broadcast)

Direction: Event (Produced)

| Field | Type | Description |
|-------|------|-------------|
| type | string | Event type |
| family | string | Broadcast family (TV or AVATAR) |
| worldId | byte | World identifier |
| characterId | uint32 | Character identifier of the entry |
| waitSeconds | uint32 | Estimated wait, in seconds, before the entry activates (QUEUED only) |
| totalWaitSeconds | uint32 | Total active duration in seconds, equal to the entry's durationSeconds (STARTED only) |
| channelId | byte | Channel identifier (STARTED only) |
| senderName | string | Sender display name (STARTED only) |
| senderMedal | string | Sender medal name (STARTED only) |
| messages | []string | Broadcast message lines (STARTED only) |
| whispersOn | bool | Whether whispers are enabled during the broadcast (STARTED only) |
| itemId | uint32 | Item identifier associated with the broadcast (STARTED only) |
| tvMessageType | string | Semantic message type key (NORMAL, STAR, HEART) (STARTED only) |
| senderLook | sharedsaga.AvatarSnapshot | Sender avatar appearance snapshot (STARTED only) |
| receiverName | string | Receiver display name (STARTED only) |
| receiverLook | sharedsaga.AvatarSnapshot | Receiver avatar appearance snapshot (STARTED only, nullable) |

Event Types:
- `QUEUED`: The entry was appended to the (worldId, family) queue
- `STARTED`: The entry was activated and is now the active broadcast for the (worldId, family) queue
- `ENDED`: The active entry's slot expired and was cleared

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
- Consumer group (world broadcast commands): "World Orchestrator"
- World broadcast command consumer does not use LastOffset: EnqueueCommand is a one-shot command with no re-emission, so starting from LastOffset would silently drop enqueue commands produced while the consumer group was down
- World Broadcast Status Event messages are keyed by worldId
- Headers: SpanHeaderParser, TenantHeaderParser
