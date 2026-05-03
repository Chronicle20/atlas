# Map Domain

## Responsibility

Tracks character presence in maps, manages monster spawn points with cooldown enforcement, coordinates reactor spawning, manages map weather effects, and records character map visit history.

## Core Models

### MapKey

Composite key identifying a map instance.

| Field | Type | Description |
|-------|------|-------------|
| Tenant | tenant.Model | Tenant identifier |
| Field | field.Model | Field identifier (worldId, channelId, mapId, instance) |

### SpawnPoint

Location where monsters can spawn.

| Field | Type | Description |
|-------|------|-------------|
| Id | uint32 | Spawn point identifier |
| Template | uint32 | Monster template ID |
| MobTime | int32 | Spawn timing behavior (negative = non-spawnable) |
| Team | int8 | Team assignment |
| Cy | int16 | Y coordinate for behavior |
| F | uint32 | Spawn flags |
| Fh | int16 | Foothold |
| Rx0 | int16 | Left spawn boundary |
| Rx1 | int16 | Right spawn boundary |
| X | int16 | X coordinate |
| Y | int16 | Y coordinate |

### CooldownSpawnPoint

Spawn point with cooldown tracking.

| Field | Type | Description |
|-------|------|-------------|
| SpawnPoint | SpawnPoint | Embedded spawn point |
| NextSpawnAt | time.Time | Cooldown expiry time |

### Reactor Model

Reactor instance in a map. Immutable with Builder pattern.

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Reactor identifier |
| f | field.Model | Field identifier (worldId, channelId, mapId, instance) |
| classification | uint32 | Reactor classification |
| name | string | Reactor name |
| state | int8 | Current state |
| eventState | byte | Event state |
| delay | uint32 | Delay value |
| direction | byte | Direction |
| x | int16 | X coordinate |
| y | int16 | Y coordinate |
| updateTime | time.Time | Last update time |

### Data Reactor Model

Reactor data from atlas-data service.

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Reactor identifier |
| classification | uint32 | Reactor classification |
| name | string | Reactor name |
| x | int16 | X coordinate |
| y | int16 | Y coordinate |
| delay | uint32 | Delay value |
| direction | byte | Direction |

### Data Script Model

Map script data from atlas-data service.

| Field | Type | Description |
|-------|------|-------------|
| onFirstUserEnter | string | Script name for first user entry |
| onUserEnter | string | Script name for every user entry |

### FieldKey

Composite key identifying a map instance for weather tracking.

| Field | Type | Description |
|-------|------|-------------|
| Tenant | tenant.Model | Tenant identifier |
| Field | field.Model | Field identifier (worldId, channelId, mapId, instance) |

### WeatherEntry

Active weather effect in a map.

| Field | Type | Description |
|-------|------|-------------|
| ItemId | uint32 | Weather item identifier |
| Message | string | Weather message |
| ExpiresAt | time.Time | Expiry time |

### Visit

Character map visit record. Immutable.

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character identifier |
| mapId | map.Id | Map identifier |
| firstVisitedAt | time.Time | Timestamp of first visit |

### Data Map Info Model

Map information from atlas-data service.

| Field | Type | Description |
|-------|------|-------------|
| id | map.Id | Map identifier |
| timeLimit | int32 | Map stay duration in seconds |
| forcedReturnMapId | map.Id | Map to which a character is returned when the time limit elapses |

### Map Timer Entry

Per-character map-stay timer record. Immutable, constructed via Builder.

| Field | Type | Description |
|-------|------|-------------|
| tenant | tenant.Model | Tenant identifier |
| characterId | uint32 | Character identifier |
| field | field.Model | Field where the character is currently tracked |
| forcedReturnMapId | map.Id | Map id used when the timer expires or is force-returned |
| seconds | uint32 | Configured map stay duration |
| token | UUID | Per-registration token used to detect stale timer callbacks |
| expiresAt | time.Time | Wall clock time at which the timer is scheduled to fire |
| timer | *time.Timer | Per-entry Go timer handle |

## Invariants

- SpawnPoints with MobTime < 0 are not spawnable
- Each MapKey maintains a separate character registry
- Each MapKey maintains a separate spawn point registry
- Spawn points with MobTime == 0 have a 5-second cooldown after spawning; spawn points with MobTime > 0 use MobTime seconds as cooldown
- Only spawn points with NextSpawnAt <= now are eligible for spawning
- When a monster with MobTime > 0 is killed, spawn point cooldown is reset to MobTime seconds from the kill time
- Reactor name cannot be empty
- Reactor classification must be positive
- Visit records are unique per tenant, character, and map (upsert via FirstOrCreate)
- Weather entries are keyed by FieldKey (tenant + field)
- Weather duration is capped at 20 seconds
- Weather entries are automatically removed after ExpiresAt
- A Map Info Model is time-limited when timeLimit > 0 and forcedReturnMapId is not the sentinel 999999999
- Map info is cached per (tenant, mapId)
- Map timer entries are keyed by (tenant, characterId); registering a new entry replaces and stops any prior entry for the same key
- Timer callbacks act on an entry only when their token matches the entry's current token
- ForceReturnIfTracked removes the entry unconditionally (token is not checked)

## State Transitions

### Map Timer Lifecycle

- On MAP_CHANGED, any prior timer for the character is cancelled. If the target map is time-limited, a new timer is registered for the configured seconds and MAP_TIMER_STARTED is emitted.
- On CHANNEL_CHANGED, any tracked timer for the character is force-returned (entry is removed and CHANGE_MAP is emitted to the forced-return map).
- On SESSION_DESTROYED for a character, any tracked timer is force-returned (entry is removed and CHANGE_MAP is emitted to the forced-return map).
- On timer expiry, the entry is claimed (only when the token matches) and CHANGE_MAP is emitted to the forced-return map.

## Processors

### Map Processor

Coordinates character entry and exit from maps.

- Enter: Registers character in map, records visit, triggers monster and reactor spawning, emits CHARACTER_ENTER event. On first visit, emits onFirstUserEnter map action. On every entry, emits onUserEnter map action if configured.
- EnterAndEmit: Executes Enter with Kafka emission
- Exit: Removes character from map, emits CHARACTER_EXIT event
- ExitAndEmit: Executes Exit with Kafka emission
- TransitionMap: Exits old map and enters new map
- TransitionMapAndEmit: Executes TransitionMap with Kafka emission
- TransitionChannel: Exits old channel and enters new channel
- TransitionChannelAndEmit: Executes TransitionChannel with Kafka emission
- GetCharactersInMap: Returns character IDs in a map instance
- GetCharactersInMapAllInstances: Returns character IDs across all instances of a map

### Character Processor

Manages in-memory character registry per map.

- GetCharactersInMap: Returns character IDs in a map instance
- GetCharactersInMapAllInstances: Returns character IDs across all instances of a map
- GetMapsWithCharacters: Returns all maps with active characters
- Enter: Adds character to registry
- Exit: Removes character from registry

### Monster Processor (map/monster)

Manages monster spawn points with cooldown enforcement.

- SpawnMonsters: Spawns monsters based on character count and spawn point availability
  - Initializes spawn point registry in Redis on first access (from atlas-data)
  - Filters spawn points by cooldown eligibility via Lua script (NextSpawnAt <= now)
  - Calculates spawn count: ceil((0.70 + 0.05 * min(6, characterCount)) * spawnPointCount) - currentMonsters
  - Randomly selects from eligible spawn points
  - Updates cooldown after spawning: 5 seconds for normal monsters (MobTime == 0), MobTime seconds for boss monsters (MobTime > 0)
  - Batch updates cooldowns atomically in Redis

### Monster Processor (monster)

Interacts with external monster service.

- CountInMap: Gets monster count via REST
- CreateMonster: Creates monster via REST

### Reactor Processor

Manages reactor spawning.

- InMapModelProvider: Provides reactor models in map via REST
- GetInMap: Gets reactors in map via REST
- Spawn: Creates reactors that do not exist in map
- SpawnAndEmit: Spawns reactors and emits Kafka messages

### Weather Processor

Manages in-memory weather effects per map instance.

- Start: Registers a weather entry in the registry with an expiry time
- GetActive: Returns the active weather entry for a map instance, if any

### Visit Processor

Manages character map visit records in PostgreSQL.

- RecordVisit: Records a character visiting a map (upsert)
- ByCharacterIdProvider: Provides all visits for a character
- ByCharacterIdAndMapIdProvider: Provides a specific visit for a character and map
- DeleteByCharacterId: Deletes all visit records for a character

### Data Processors

#### Data Monster Processor

Retrieves spawn point data from atlas-data service.

- SpawnPointProvider: Provides all spawn points for a map
- SpawnableSpawnPointProvider: Provides spawn points where MobTime >= 0
- GetSpawnPoints: Gets all spawn points for a map
- GetSpawnableSpawnPoints: Gets spawn points where MobTime >= 0

#### Data Reactor Processor

Retrieves reactor data from atlas-data service.

- InMapProvider: Gets reactor data for a map

#### Data Script Processor

Retrieves map script data from atlas-data service.

- GetScripts: Gets script names (onFirstUserEnter, onUserEnter) for a map

#### Data Map Info Processor

Retrieves map info from atlas-data service.

- GetById: Returns the Map Info Model for a map id, cached per (tenant, mapId)

### Map Timer Processor

Manages per-character map-stay timers.

- Register: Inserts (or replaces) the timer entry for a character, schedules a per-entry time.Timer for the configured seconds, and emits MAP_TIMER_STARTED. Replaces and stops any prior entry for the same (tenant, characterId).
- CancelIfTracked: Atomically removes any tracked entry for the character and stops its timer. Returns whether an entry was removed.
- ForceReturnIfTracked: Atomically removes any tracked entry for the character regardless of token, stops its timer, and emits CHANGE_MAP to the entry's forcedReturnMapId.
- handleExpire (timer callback): Atomically claims the entry only if its current token matches; on match emits CHANGE_MAP to the forcedReturnMapId.
