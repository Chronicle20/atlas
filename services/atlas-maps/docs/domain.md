# Map Domain

## Responsibility

Tracks character presence in maps, manages monster spawn points with cooldown enforcement, and coordinates reactor spawning.

## Core Models

### MapKey

Composite key identifying a map instance.

| Field | Type | Description |
|-------|------|-------------|
| Tenant | tenant.Model | Tenant identifier |
| WorldId | world.Id | World identifier |
| ChannelId | channel.Id | Channel identifier |
| MapId | map.Id | Map identifier |

### SpawnPoint

Location where monsters can spawn.

| Field | Type | Description |
|-------|------|-------------|
| Id | uint32 | Spawn point identifier |
| Template | uint32 | Monster template ID |
| MobTime | uint32 | Spawn timing behavior |
| Team | int32 | Team assignment |
| Cy | int16 | Y coordinate for behavior |
| F | uint32 | Spawn flags |
| Fh | uint16 | Foothold |
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

Reactor instance in a map.

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Reactor identifier |
| worldId | byte | World identifier |
| channelId | byte | Channel identifier |
| mapId | uint32 | Map identifier |
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

## Invariants

- SpawnPoints with MobTime < 0 are not spawnable
- Each MapKey maintains a separate character registry
- Each MapKey maintains a separate spawn point registry
- Spawn points have a 5-second cooldown after spawning
- Only spawn points with NextSpawnAt <= now are eligible for spawning
- Reactor name cannot be empty
- Reactor classification must be positive

## Processors

### Map Processor

Coordinates character entry and exit from maps.

- Enter: Registers character in map, triggers monster and reactor spawning, emits CHARACTER_ENTER event
- EnterAndEmit: Executes Enter with Kafka emission
- Exit: Removes character from map, emits CHARACTER_EXIT event
- ExitAndEmit: Executes Exit with Kafka emission
- TransitionMap: Exits old map and enters new map
- TransitionMapAndEmit: Executes TransitionMap with Kafka emission
- TransitionChannel: Exits old channel and enters new channel
- TransitionChannelAndEmit: Executes TransitionChannel with Kafka emission
- GetCharactersInMap: Returns character IDs in a map

### Character Processor

Manages in-memory character registry per map.

- GetCharactersInMap: Returns character IDs in a map
- GetMapsWithCharacters: Returns all maps with active characters
- Enter: Adds character to registry
- Exit: Removes character from registry

### Monster Processor (map/monster)

Manages monster spawn points with cooldown enforcement.

- SpawnMonsters: Spawns monsters based on character count and spawn point availability
  - Initializes spawn point registry on first access
  - Filters spawn points by cooldown eligibility
  - Calculates spawn count: ceil((0.70 + 0.05 * min(6, characterCount)) * spawnPointCount) - currentMonsters
  - Randomly selects from eligible spawn points
  - Updates cooldown (NextSpawnAt = now + 5 seconds) after spawning

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
