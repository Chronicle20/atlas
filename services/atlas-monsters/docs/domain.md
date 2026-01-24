# Monster Domain

## Responsibility

Manages monster instances within game maps, including creation, movement, damage tracking, control assignment, and destruction.

## Core Models

### Model

Represents an active monster instance in a map.

| Field | Type | Description |
|-------|------|-------------|
| uniqueId | uint32 | Unique identifier for this monster instance |
| worldId | byte | World identifier |
| channelId | byte | Channel identifier |
| mapId | uint32 | Map identifier |
| monsterId | uint32 | Monster type identifier |
| maxHp | uint32 | Maximum hit points |
| hp | uint32 | Current hit points |
| maxMp | uint32 | Maximum magic points |
| mp | uint32 | Current magic points |
| controlCharacterId | uint32 | Character ID controlling this monster (0 if uncontrolled) |
| x | int16 | X coordinate |
| y | int16 | Y coordinate |
| fh | int16 | Foothold |
| stance | byte | Animation stance |
| team | int8 | Team assignment |
| damageEntries | []entry | List of damage dealt by characters |

### DamageSummary

Represents the result of applying damage to a monster.

| Field | Type | Description |
|-------|------|-------------|
| CharacterId | uint32 | Character who dealt damage |
| Monster | Model | Updated monster state |
| VisibleDamage | uint32 | Damage shown to clients |
| ActualDamage | int64 | Actual damage applied |
| Killed | bool | Whether the monster was killed |

### entry

Tracks damage dealt by a character.

| Field | Type | Description |
|-------|------|-------------|
| CharacterId | uint32 | Character who dealt damage |
| Damage | uint32 | Amount of damage |

### MapKey

Composite key for map-scoped monster lookups.

| Field | Type | Description |
|-------|------|-------------|
| Tenant | tenant.Model | Tenant context |
| WorldId | byte | World identifier |
| ChannelId | byte | Channel identifier |
| MapId | uint32 | Map identifier |

### MonsterKey

Composite key for monster lookups.

| Field | Type | Description |
|-------|------|-------------|
| Tenant | tenant.Model | Tenant context |
| MonsterId | uint32 | Monster unique identifier |

## Invariants

- Monster uniqueId values range from 1000000000 to 2000000000 per tenant
- A monster is alive when hp > 0
- A monster is controlled when controlCharacterId != 0
- Damage entries accumulate over the monster's lifetime
- The damage leader is the character with the highest total damage dealt

## State Transitions

### Monster Lifecycle

1. **Created**: Monster spawned in map with initial HP/MP from monster information
2. **Controlled**: Character assigned as controller
3. **Damaged**: HP reduced, damage entries recorded
4. **Control Transferred**: Controller changed based on damage leadership
5. **Killed**: HP reaches 0, monster removed from registry
6. **Destroyed**: Monster removed from registry (manual destruction)

### Control Assignment

- When a monster is created, the service attempts to assign a controller from characters in the map
- The controller candidate is the character controlling the fewest monsters in that map
- When the current controller exits the map, control stops and a new controller is assigned
- When a character becomes the damage leader and is not the current controller, control transfers to them

## Processors

### Processor

Interface defining monster processing operations.

**Providers:**
- `ByIdProvider`: Provides a monster by unique ID
- `ByMapProvider`: Provides all monsters in a map
- `ControlledInMapProvider`: Provides controlled monsters in a map
- `NotControlledInMapProvider`: Provides uncontrolled monsters in a map
- `ControlledByCharacterInMapProvider`: Provides monsters controlled by a specific character in a map

**Queries:**
- `GetById`: Retrieves a monster by unique ID
- `GetInMap`: Retrieves all monsters in a map

**Commands:**
- `Create`: Creates a monster in a map, assigns controller, emits created status event
- `StartControl`: Assigns a character as controller, emits start control status event
- `StopControl`: Removes controller assignment, emits stop control status event
- `FindNextController`: Finds and assigns the next controller for a monster
- `Damage`: Applies damage to a monster, may transfer control or kill monster
- `Move`: Updates monster position and stance
- `Destroy`: Removes monster from registry, emits destroyed status event
- `DestroyInMap`: Destroys all monsters in a map

### Registry

Singleton in-memory store for monster instances.

**Operations:**
- `CreateMonster`: Creates and stores a new monster instance
- `GetMonster`: Retrieves a monster by tenant and unique ID
- `GetMonstersInMap`: Retrieves all monsters in a map
- `MoveMonster`: Updates monster position
- `ControlMonster`: Assigns a controller to a monster
- `ClearControl`: Removes controller assignment
- `ApplyDamage`: Applies damage and returns damage summary
- `RemoveMonster`: Removes a monster from the registry
- `GetMonsters`: Returns all monsters grouped by tenant
- `Clear`: Clears all registry data

### RegistryAudit

Periodic task that logs registry statistics (maps tracked, monsters tracked).
