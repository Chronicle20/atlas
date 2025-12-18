# atlas-maps

Mushroom game maps Service

## Overview

A RESTful resource which provides maps services, including character tracking, monster spawning, and reactor management.

### Features

- **Character Tracking**: Monitor character locations and movements across maps
- **Monster Spawning**: Intelligent spawn point management with cooldown tracking
- **Reactor Management**: Handle interactive map objects and their states
- **Multi-tenant Architecture**: Separate data isolation per tenant/world/channel/map
- **Real-time Events**: Kafka-based event streaming for character and map status changes

## Environment Variables

### General Configuration
- JAEGER_HOST - Jaeger [host]:[port] for distributed tracing
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- REST_PORT - [port] of the REST interface
- BOOTSTRAP_SERVERS - Kafka [host]:[port]
- BASE_SERVICE_URL - [scheme]://[host]:[port]/api/

### Kafka Topics
- EVENT_TOPIC_CHARACTER_STATUS - Kafka Topic for transmitting character status events
- EVENT_TOPIC_MAP_STATUS - Kafka Topic for transmitting map status events
- EVENT_TOPIC_CASH_SHOP_STATUS - Kafka Topic for transmitting cash shop status events
- COMMAND_TOPIC_REACTOR - Kafka Topic for transmitting reactor commands

## REST API

### Header

All RESTful requests require the supplied header information to identify the server instance.

```
TENANT_ID:083839c6-c47c-42a6-9585-76492795d123
REGION:GMS
MAJOR_VERSION:83
MINOR_VERSION:1
```

### Endpoints

#### Map Characters
- `GET /{worldId}/channels/{channelId}/maps/{mapId}/characters` - Get all characters in a specific map

### Requests

Detailed API documentation is available via Bruno collection.

## Kafka Message API

### Character Status Events
Messages published to `EVENT_TOPIC_CHARACTER_STATUS` with the following types:
- `LOGIN` - Character logged in
- `LOGOUT` - Character logged out
- `CHANNEL_CHANGED` - Character changed channels
- `MAP_CHANGED` - Character changed maps

### Map Status Events
Messages published to `EVENT_TOPIC_MAP_STATUS` with the following types:
- `CHARACTER_ENTER` - Character entered a map
- `CHARACTER_EXIT` - Character exited a map

### Cash Shop Status Events
Messages published to `EVENT_TOPIC_CASH_SHOP_STATUS` with the following types:
- `CHARACTER_ENTER` - Character entered the cash shop
- `CHARACTER_EXIT` - Character exited the cash shop

### Reactor Commands
Messages published to `COMMAND_TOPIC_REACTOR` with the following types:
- `CREATE` - Create a reactor

All Kafka messages include a transaction ID (UUID) to track message flow through the system.

## Spawn Point Cooldown Mechanism

The atlas-maps service implements an advanced spawn point cooldown system to prevent over-spawning and ensure balanced monster distribution across maps.

### Overview

The spawn point cooldown mechanism maintains an in-memory registry of spawn points per map instance, where each spawn point tracks its cooldown state to prevent immediate re-spawning.

### Key Features

- **5-Second Cooldown**: Each spawn point enforces a 5-second cooldown period after monster spawning
- **Per-Map Registry**: Separate spawn point registries scoped by MapKey (tenant/world/channel/map)
- **Lazy Initialization**: Registry is populated on first access from the spawn point REST provider
- **Thread Safety**: Per-map RWMutex ensures safe concurrent access across multiple maps
- **Multi-tenant Support**: Complete isolation between different tenant/world/channel/map combinations

### Architecture

#### Data Structures

- **`SpawnPoint`**: Basic spawn point data including position, template, and timing information
- **`CooldownSpawnPoint`**: Extends SpawnPoint with `NextSpawnAt` timestamp for cooldown tracking
- **Registry**: In-memory map of `character.MapKey` to `[]*CooldownSpawnPoint` arrays

#### Spawn Process

1. **Registry Access**: Get or initialize spawn point registry for the target map
2. **Cooldown Filtering**: Filter spawn points where `NextSpawnAt.Before(now)` (eligible points)
3. **Spawn Calculation**: Calculate required spawns based on character count and monster limits
4. **Random Selection**: Randomly shuffle and select from eligible spawn points
5. **Monster Creation**: Spawn monsters asynchronously via REST API calls
6. **Cooldown Update**: Set `NextSpawnAt = now + 5 seconds` for used spawn points

#### Thread Safety

- **Per-Map Mutexes**: Each MapKey has its own `sync.RWMutex` for thread-safe operations
- **Concurrent Access**: Multiple maps can be processed simultaneously without interference
- **Reader/Writer Locks**: RLock for reading during filtering, Lock for updating cooldowns

### Usage

The cooldown mechanism is transparent to existing code. The `SpawnMonsters()` method maintains the same interface while adding cooldown enforcement internally.

```go
// Example usage (internal to the service)
processor := monster.NewProcessor(logger, ctx)
spawnFunc := processor.SpawnMonsters(transactionId)
err := spawnFunc(worldId)(channelId)(mapId)
```

### Benefits

- **Prevents Over-spawning**: Ensures monsters aren't spawned too frequently from the same points
- **Balanced Distribution**: Encourages use of different spawn points across the map
- **Performance**: In-memory registry provides fast access without external API calls
- **Scalability**: Per-map isolation allows independent scaling across different maps
- **Reliability**: Thread-safe implementation supports high-concurrency environments

### Monitoring

The system includes comprehensive logging for:
- Registry initialization events
- Spawn attempts and cooldown status
- Eligibility filtering results
- Concurrent access patterns

Logs capture when spawn points are used and when they are skipped due to cooldown, providing visibility into the spawning behavior.
