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

**Headers:** SpanHeaderParser, TenantHeaderParser

### EVENT_TOPIC_CHANNEL_STATUS

Channel status events for tracking active channels.

| Field | Type | Description |
|-------|------|-------------|
| type | string | Event type (channel.StatusType) |
| worldId | world.Id | World identifier |
| channelId | channel.Id | Channel identifier |
| ipAddress | string | Channel IP address |
| port | int | Channel port |

**Event Types:**

- `STARTED`: Channel started event
- `SHUTDOWN`: Channel shutdown event

**Consumer Group:** Transport Service

**Headers:** SpanHeaderParser, TenantHeaderParser

### EVENT_TOPIC_MAP_STATUS

Map status events for handling character enter and exit on transit maps.

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

- `CHARACTER_ENTER`: Character entered a map
- `CHARACTER_EXIT`: Character exited a map

**CHARACTER_ENTER Body:**

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character identifier |

**CHARACTER_EXIT Body:**

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character identifier |

**Consumer Group:** Transport Service

**Consumer Start Offset:** Latest (LastOffset)

**Headers:** SpanHeaderParser, TenantHeaderParser

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

**Headers:** SpanHeaderParser, TenantHeaderParser

### EVENT_TOPIC_CONFIGURATION_STATUS

Configuration change events for reloading route and vessel configurations.

| Field | Type | Description |
|-------|------|-------------|
| tenantId | uuid.UUID | Tenant identifier |
| type | string | Event type |
| resourceType | string | Resource type that changed |
| resourceId | string | Resource identifier |

**Resource Types Handled:**

- `route`, `vessel`: Triggers reload of scheduled routes for the tenant
- `instance-route`: Triggers reload of instance routes for the tenant

**Consumer Group:** Transport Service

**Headers:** SpanHeaderParser, TenantHeaderParser

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
- `VOYAGE_DEPARTED`: A concrete trip on the route has departed
- `VOYAGE_ARRIVED`: A concrete trip on the route has arrived

**ARRIVED Body:**

| Field | Type | Description |
|-------|------|-------------|
| mapId | map.Id | Observation map ID |

**DEPARTED Body:**

| Field | Type | Description |
|-------|------|-------------|
| mapId | map.Id | Observation map ID |

**VOYAGE_DEPARTED / VOYAGE_ARRIVED Body (VoyageStatusEventBody):**

| Field | Type | Description |
|-------|------|-------------|
| voyageId | uuid.UUID | Derived identity of the trip (`transport.VoyageId`) |
| worldId | world.Id | World identifier |
| channelId | channel.Id | Channel identifier |
| stagingMapId | map.Id | Staging map ID |
| enRouteMapIds | []map.Id | En-route map IDs |
| destinationMapId | map.Id | Destination map ID |
| observationMapId | map.Id | Observation map ID |
| departedAt | time.Time | Departure instant of the voyage |

One VOYAGE_DEPARTED/VOYAGE_ARRIVED event is emitted per (world, channel).

**Partition Key:** Route ID string for ARRIVED/DEPARTED. Voyage ID string for VOYAGE_DEPARTED/VOYAGE_ARRIVED.

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
- `TRANSIT_ENTERED`: Character has entered the transit map
- `COMPLETED`: Character has arrived at destination
- `CANCELLED`: Character's transport was cancelled

**STARTED Body:**

| Field | Type | Description |
|-------|------|-------------|
| routeId | uuid.UUID | Route identifier |
| instanceId | uuid.UUID | Instance identifier |

**TRANSIT_ENTERED Body:**

| Field | Type | Description |
|-------|------|-------------|
| routeId | uuid.UUID | Route identifier |
| instanceId | uuid.UUID | Instance identifier |
| channelId | channel.Id | Channel identifier |
| durationSeconds | uint32 | Transit duration in seconds |
| message | string | Transit message |

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
| reason | string | Cancellation reason (MAP_EXIT, LOGOUT, STUCK, TIMEOUT) |

**Cancellation Reasons:**

| Reason | Description |
|--------|-------------|
| MAP_EXIT | Character entered a non-transit map while in transport |
| LOGOUT | Character logged out during transport |
| STUCK | Instance exceeded max lifetime; characters were force-warped to the route start map |
| TIMEOUT | The travel timer expired on a route declaring a forced return; the character was sent back rather than delivered |

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

### COMMAND_TOPIC_CONSUMABLE

Emitted for routes that declare `effectItemIds`. Applies the declared effects
when a character boards an instance transport, and cancels them on every
terminal path (travel-timer arrival, entering a non-transit map, logout, stuck
timeout, graceful shutdown). Routes declaring no effects emit nothing.

`transactionId` is always the nil UUID: these are not saga-driven applications,
and atlas-saga-orchestrator skips saga completion for a nil transaction id.
`mapId`/`instance` are left zero — APPLY resolves the character's live map
itself, and CANCEL's field reaches atlas-buffs' `Cancel`, which reads only
`worldId`.

| Command Type | Body | Purpose |
|--------------|------|---------|
| APPLY_CONSUMABLE_EFFECT | `{itemId}` | Apply a route's declared transit effect on boarding |
| CANCEL_CONSUMABLE_EFFECT | `{itemId}` | Remove it on every terminal path |

**Partition Key:** Character ID integer

## Message Types

### StatusEvent[E] (kafka/message/transport/kafka.go)

Generic transport status event.

### VoyageStatusEventBody (kafka/message/transport/kafka.go)

Body used for both VOYAGE_DEPARTED and VOYAGE_ARRIVED; the envelope's `type` discriminates.

### Command[E] (kafka/message/character/kafka.go)

Generic character command.

### StatusEvent[E] (kafka/message/character/kafka.go)

Generic character status event.

### StatusEvent (kafka/message/channel/kafka.go)

Channel status event.

### Command[E] (kafka/message/instance_transport/kafka.go)

Generic instance transport command.

### Event[E] (kafka/message/instance_transport/kafka.go)

Generic instance transport event.

### Command[E] (kafka/message/consumable/kafka.go)

Generic consumable effect command.

### StatusEvent[E] (kafka/message/map/kafka.go)

Generic map status event.

### StatusEvent (kafka/message/configuration/kafka.go)

Configuration status event.

## Transaction Semantics

Messages are buffered and emitted together via the message.Buffer pattern. The producer emits all buffered messages in a single operation.

Header parsing includes:
- SpanHeaderParser: For distributed tracing
- TenantHeaderParser: For multi-tenant context
