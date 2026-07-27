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

### Location Model

Persisted last-known field for a character. Immutable, constructed via Builder.

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character identifier |
| worldId | world.Id | World identifier |
| channelId | channel.Id | Channel identifier |
| mapId | map.Id | Map identifier |
| instance | UUID | Instance identifier |

### Mist

Area-of-effect field placed on a map that applies a disease to characters whose position falls within its bounding box. Immutable, constructed via Builder.

| Field | Type | Description |
|-------|------|-------------|
| id | UUID | Mist identifier |
| f | field.Model | Field the mist occupies |
| ownerType | string | Type of entity that owns the mist |
| ownerId | uint32 | Id of the owning entity |
| sourceSkillId | uint32 | Skill id that produced the mist |
| sourceSkillLevel | uint32 | Skill level that produced the mist |
| mistType | int32 | Mist/affected-area type discriminator |
| originX | int16 | X coordinate of the mist's anchor |
| originY | int16 | Y coordinate of the mist's anchor |
| ltX | int16 | Left-top x offset relative to origin |
| ltY | int16 | Left-top y offset relative to origin |
| rbX | int16 | Right-bottom x offset relative to origin |
| rbY | int16 | Right-bottom y offset relative to origin |
| disease | string | Disease name applied by the mist |
| diseaseValue | int32 | Magnitude of the applied disease |
| diseaseDuration | time.Duration | Duration of the applied disease on a target |
| duration | time.Duration | Total lifetime of the mist |
| tickInterval | time.Duration | Interval between disease application ticks |
| createdAt | time.Time | Construction time |
| expiresAt | time.Time | Absolute expiry time |
| lastTick | time.Time | Time of the most recent disease application tick |

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
- A Location Model exists at most once per (tenant, characterId); Set replaces any existing row, Delete is idempotent
- Location.Resolve returns ReasonStayPut when map info for the current map is unavailable or the current map's forcedReturnMapId is the sentinel value; otherwise it returns ReasonForcedReturn with a field pointing at the current map's forcedReturnMapId and a cleared instance
- A Mist's bounding box spans (originX+ltX, originY+ltY) to (originX+rbX, originY+rbY), inclusive of edges
- A Mist is expired once the current time is past its expiresAt
- A Mist ticks (ShouldTick) only when tickInterval is greater than zero and at least tickInterval has elapsed since lastTick
- Mists are keyed by mist id within a tenant-scoped bucket; adding a mist whose id already exists in the bucket fails

## State Transitions

### Map Timer Lifecycle

- On MAP_CHANGED, any prior timer for the character is cancelled. If the target map is time-limited, a new timer is registered for the configured seconds and MAP_TIMER_STARTED is emitted.
- On CHANNEL_CHANGED, any tracked timer for the character is force-returned: the entry is removed and its timer stopped. No CHANGE_MAP is emitted here; forced-return persistence for this case is handled by the Character Location Lifecycle.
- On SESSION_DESTROYED for a character, any tracked timer is force-returned: the entry is removed and its timer stopped. No CHANGE_MAP is emitted here; forced-return persistence is handled by the Character Location Lifecycle at the character's next LOGIN.
- On timer expiry, the entry is claimed (only when the token matches) and CHANGE_MAP is emitted to the forced-return map.

### Character Location Lifecycle

- On CREATED, a Location Model is seeded for the character with channelId 0, since the character is not yet bound to a channel.
- On LOGIN, the Location Model is set to the login field.
- On LOGOUT, the current field is resolved via Location.Resolve; the resolved field (unchanged, or forced-return) is persisted as the Location Model.
- On MAP_CHANGED, the Location Model is set to the new field.
- On CHANNEL_CHANGED, the Location Model is set to the new field.
- On CHANNEL_CHANGE_REQUEST, the target field is resolved via Location.Resolve; the resolved field is persisted as the Location Model.
- On DELETED, the Location Model for the character is removed.
- ChangeMap persists the destination field as the Location Model, using the character's prior Location Model as the old field (defaulting to the destination when no prior Location Model exists).

### Mist Lifecycle

- On MIST_CREATE, a Mist is constructed from the request and registered under the resolved tenant; MIST_CREATED is emitted. If emission fails, the registration is rolled back.
- On MIST_CANCEL, the referenced Mist is removed from the registry and MIST_DESTROYED is emitted with a cancelled reason.
- On periodic tick, a Mist past its expiresAt is removed and MIST_DESTROYED is emitted with an expired reason. A Mist that ShouldTick has its disease reapplied to every character within its bounding box, and its lastTick is advanced.

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
- ForceReturnIfTracked: Atomically removes any tracked entry for the character regardless of token and stops its timer. Does not emit CHANGE_MAP; forced-return persistence for this path is handled by the Location Processor.
- handleExpire (timer callback): Atomically claims the entry only if its current token matches; on match emits CHANGE_MAP to the forcedReturnMapId.

### Location Processor

Manages a character's persisted last-known field.

- GetById: Returns the Location Model for a character
- Set: Upserts the Location Model for a character to the given field
- Delete: Removes the Location Model for a character
- Resolve: Given a current field, returns either the unchanged field (ReasonStayPut) or a field pointing at the current map's forcedReturnMapId with a cleared instance (ReasonForcedReturn)

### Warp Processor

Coordinates a character's authoritative map change. Both the CHANGE_MAP command path and the character-location warp path invoke this processor so the two cannot diverge.

- ChangeMap: Persists the destination field via the Location Processor, emits MAP_CHANGED (using either a target portal id or an exact landing position), and transitions the character's map registries. Uses the character's current Location Model as the old field, defaulting to the destination when no prior Location Model exists. Returns an error only when the durable Set fails; emit and transition failures are logged and the call still succeeds.

### Mist Processor

Manages tenant-scoped Mist lifecycle.

- Create: Constructs a Mist from a creation request, registers it under the resolved tenant, and emits MIST_CREATED. Rolls back the registration if emission fails.
- Destroy: Removes a Mist by id from the resolved tenant's bucket and emits MIST_DESTROYED with the supplied reason. Registry removal is authoritative even if emission fails.
