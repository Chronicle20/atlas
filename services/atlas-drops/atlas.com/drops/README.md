# Atlas-Drops Service

In-memory drop state management service for the Atlas game world.

## Overview

This service manages transient drop state (items, meso, equipment dropped in the game world). Drops are stored in-memory with TTL-based expiration, making database persistence unnecessary for this short-lived game state.

### Key Characteristics

- **In-memory storage**: Drops are stored in a singleton registry with thread-safe access
- **TTL expiration**: Drops automatically expire after a configurable time period
- **Multi-tenant**: Drops are isolated by tenant ID
- **Stateless deployment**: Service can be restarted without data migration (drops are ephemeral)

## REST API

### Get Drop by ID

```
GET /drops/{id}
```

Returns a single drop by its unique identifier.

**Response:** JSON:API formatted drop resource

### Get Drops in Map

```
GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/drops
```

Returns all drops currently on a specific map.

**Path Parameters:**
- `worldId` - World identifier
- `channelId` - Channel identifier
- `mapId` - Map identifier

**Response:** JSON:API formatted array of drop resources

## Kafka Topics

### Command Topic (Consumed)

**Environment Variable:** `COMMAND_TOPIC_DROP`

| Command Type | Description |
|--------------|-------------|
| `SPAWN` | Create a new drop from a monster |
| `SPAWN_FROM_CHARACTER` | Create a new drop from a character (with existing equipment ID) |
| `REQUEST_RESERVATION` | Request to reserve a drop for pickup |
| `CANCEL_RESERVATION` | Cancel an existing reservation |
| `REQUEST_PICK_UP` | Request to pick up a reserved drop |

### Status Event Topic (Produced)

**Environment Variable:** `EVENT_TOPIC_DROP_STATUS`

| Event Type | Description |
|------------|-------------|
| `CREATED` | Drop was spawned in the world |
| `RESERVED` | Drop was successfully reserved for a character |
| `RESERVATION_FAILURE` | Reservation request failed (already reserved) |
| `PICKED_UP` | Drop was successfully picked up |
| `EXPIRED` | Drop expired due to TTL |

## Cross-Service Dependencies

### Equipment Service (EQUIPABLES)

When spawning equipment-type drops via the `SPAWN` command, this service makes REST calls to the equipment service to create the equipment record. The equipment ID is then associated with the drop.

When drops expire, equipment records are deleted via REST call.

## Domain Model

### Drop States

| Status | Description |
|--------|-------------|
| `AVAILABLE` | Drop can be picked up by any eligible character |
| `RESERVED` | Drop is reserved for a specific character |

### Drop Lifecycle

```
SPAWN → AVAILABLE → RESERVED → PICKED_UP
                 ↘          ↗
                   EXPIRED
```

## Architecture Notes

### Why In-Memory?

Drops are short-lived objects (typically <5 minutes) that don't require:
- Historical queries
- Cross-service persistence
- Recovery after restart

In-memory storage provides lower latency for high-frequency operations like spawn, reserve, and pickup.

### Thread Safety

The registry uses a multi-level locking strategy:
- Global RWMutex for map access
- Per-drop mutexes for individual drop operations
- Per-map mutexes for map-scoped queries

### Unique ID Generation

Drop IDs are generated using an atomic counter starting at 1,000,000,001 and wrapping at 2,000,000,000. This range is chosen to avoid conflicts with other entity IDs in the system.
