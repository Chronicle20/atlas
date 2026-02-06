# Kafka Integration

## Topics Consumed

### EVENT_TOPIC_CHARACTER_STATUS

Character status events for handling logout and login scenarios.

| Field | Type | Description |
|-------|------|-------------|
| transactionId | uuid.UUID | Transaction identifier |
| worldId | world.Id | World identifier |
| characterId | uint32 | Character identifier |
| type | string | Event type |
| body | object | Event body |

**Event Types:**

- `LOGOUT`: Character logout event
- `LOGIN`: Character login event

**LOGOUT Body:**

| Field | Type | Description |
|-------|------|-------------|
| channelId | channel.Id | Channel identifier |
| mapId | map.Id | Map identifier |
| instance | uuid.UUID | Instance identifier |

**LOGIN Body:**

| Field | Type | Description |
|-------|------|-------------|
| channelId | channel.Id | Channel identifier |
| mapId | map.Id | Map identifier |
| instance | uuid.UUID | Instance identifier |

**Consumer Group:** Transport Service

**Handler:** On logout, removes character from any active instance transport and warps character to route start map if in a scheduled transport staging or en-route map. On login, warps character to route start map if logged in at an instance transit map (crash recovery).

### EVENT_TOPIC_CHANNEL_STATUS

Channel status events for tracking active channels.

| Field | Type | Description |
|-------|------|-------------|
| type | string | Event type |
| worldId | world.Id | World identifier |
| channelId | channel.Id | Channel identifier |
| ipAddress | string | Channel IP address |
| port | int | Channel port |

**Event Types:**

- `STARTED`: Channel started event
- `SHUTDOWN`: Channel shutdown event

**Consumer Group:** Transport Service

**Handler:** Registers or unregisters channels in the channel registry based on event type.

### COMMAND_TOPIC_INSTANCE_TRANSPORT

Instance transport commands for starting instance-based transports.

| Field | Type | Description |
|-------|------|-------------|
| transactionId | uuid.UUID | Transaction identifier |
| worldId | world.Id | World identifier |
| characterId | uint32 | Character identifier |
| type | string | Command type |
| body | object | Command body |

**Command Types:**

- `START`: Start instance transport

**START Body:**

| Field | Type | Description |
|-------|------|-------------|
| routeId | uuid.UUID | Route identifier |
| channelId | channel.Id | Channel identifier |

**Consumer Group:** Transport Service

**Handler:** Creates or joins an instance transport, warps character to transit map, and emits STARTED event.

### EVENT_TOPIC_MAP_STATUS

Map status events for handling character exits from instance transit maps.

| Field | Type | Description |
|-------|------|-------------|
| transactionId | uuid.UUID | Transaction identifier |
| worldId | world.Id | World identifier |
| channelId | channel.Id | Channel identifier |
| mapId | map.Id | Map identifier |
| instance | uuid.UUID | Instance identifier |
| type | string | Event type |
| body | object | Event body |

**Event Types:**

- `CHARACTER_EXIT`: Character exited a map

**CHARACTER_EXIT Body:**

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character identifier |

**Consumer Group:** Transport Service

**Consumer Start Offset:** Latest (LastOffset)

**Handler:** If character is in an instance transport and exits their transit map instance, removes character from the instance and emits CANCELLED event.

## Topics Produced

### EVENT_TOPIC_TRANSPORT_STATUS

Transport route status events for scheduled routes.

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

### EVENT_TOPIC_INSTANCE_TRANSPORT

Instance transport lifecycle events.

| Field | Type | Description |
|-------|------|-------------|
| transactionId | uuid.UUID | Transaction identifier |
| worldId | world.Id | World identifier |
| characterId | uint32 | Character identifier |
| type | string | Event type |
| body | object | Event body |

**Event Types:**

- `STARTED`: Character has started an instance transport
- `COMPLETED`: Character has arrived at destination
- `CANCELLED`: Character's transport was cancelled

**STARTED Body:**

| Field | Type | Description |
|-------|------|-------------|
| routeId | uuid.UUID | Route identifier |
| instanceId | uuid.UUID | Instance identifier |

**COMPLETED Body:**

| Field | Type | Description |
|-------|------|-------------|
| routeId | uuid.UUID | Route identifier |
| instanceId | uuid.UUID | Instance identifier |

**CANCELLED Body:**

| Field | Type | Description |
|-------|------|-------------|
| routeId | uuid.UUID | Route identifier |
| instanceId | uuid.UUID | Instance identifier |
| reason | string | Cancellation reason (MAP_EXIT, LOGOUT, STUCK) |

**Partition Key:** Character ID integer

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
| instance | uuid.UUID | Target instance ID |
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

### Command[E] (kafka/message/instance_transport/kafka.go)

Generic instance transport command.

### Event[E] (kafka/message/instance_transport/kafka.go)

Generic instance transport event.

### StatusEvent[E] (kafka/message/map/kafka.go)

Generic map status event.

## Transaction Semantics

Messages are buffered and emitted together via the message.Buffer pattern. The producer emits all buffered messages in a single operation.

Header parsing includes:
- SpanHeaderParser: For distributed tracing
- TenantHeaderParser: For multi-tenant context
