# Kafka Integration

## Topics Consumed

### EVENT_TOPIC_CHARACTER_STATUS

Character status events for handling logout scenarios.

| Field | Type | Description |
|-------|------|-------------|
| worldId | byte | World identifier |
| characterId | uint32 | Character identifier |
| type | string | Event type |
| body | object | Event body |

**Event Types:**

- `LOGOUT`: Character logout event

**LOGOUT Body:**

| Field | Type | Description |
|-------|------|-------------|
| channelId | byte | Channel identifier |
| mapId | uint32 | Map identifier |

**Consumer Group:** atlas-transports

**Handler:** On logout, if character is in a transport staging or en-route map, warps character to route start map.

### EVENT_TOPIC_CHANNEL_STATUS

Channel status events for tracking active channels.

| Field | Type | Description |
|-------|------|-------------|
| type | string | Event type |
| worldId | byte | World identifier |
| channelId | byte | Channel identifier |
| ipAddress | string | Channel IP address |
| port | int | Channel port |

**Event Types:**

- `STARTED`: Channel started event
- `SHUTDOWN`: Channel shutdown event

**Consumer Group:** atlas-transports

**Handler:** Registers or unregisters channels in the channel registry based on event type.

## Topics Produced

### EVENT_TOPIC_TRANSPORT_STATUS

Transport route status events.

| Field | Type | Description |
|-------|------|-------------|
| routeId | uuid.UUID | Route identifier |
| type | string | Event type |
| body | object | Event body |

**Event Types:**

- `ARRIVED`: Route has arrived (transition to open_entry state)
- `DEPARTED`: Route has departed (transition to in_transit state)

**ARRIVED Body:**

| Field | Type | Description |
|-------|------|-------------|
| mapId | map.Id | Observation map ID |

**DEPARTED Body:**

| Field | Type | Description |
|-------|------|-------------|
| mapId | map.Id | Observation map ID |

**Partition Key:** Route ID string

### COMMAND_TOPIC_CHARACTER

Character commands for map changes.

| Field | Type | Description |
|-------|------|-------------|
| worldId | world.Id | World identifier |
| characterId | uint32 | Character identifier |
| type | string | Command type |
| body | object | Command body |

**Command Types:**

- `CHANGE_MAP`: Change character map

**CHANGE_MAP Body:**

| Field | Type | Description |
|-------|------|-------------|
| channelId | channel.Id | Channel identifier |
| mapId | map.Id | Target map ID |
| portalId | uint32 | Target portal ID |

**Partition Key:** Character ID integer

## Message Types

### StatusEvent[E] (kafka/message/transport/kafka.go)

Generic transport status event.

### Command[E] (kafka/message/character/kafka.go)

Generic character command.

### StatusEvent (kafka/message/channel/kafka.go)

Channel status event.

### StatusEvent[E] (kafka/message/character/kafka.go)

Generic character status event.

## Transaction Semantics

Messages are buffered and emitted together via the message.Buffer pattern. The producer emits all buffered messages in a single operation.

Header parsing includes:
- SpanHeaderParser: For distributed tracing
- TenantHeaderParser: For multi-tenant context
