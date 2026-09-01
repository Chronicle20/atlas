# Chair Domain

## Responsibility

The chair domain manages chair usage by game characters, including sitting on and clearing chairs, with validation based on chair type.

## Core Models

### chair.Model

Represents a chair being used by a character.

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Chair identifier |
| chairType | string | Chair type (FIXED or PORTABLE) |
| lastHpRecoveryAt | int64 | Unix-milli of last honored HP recovery tick; zero if never |
| lastMpRecoveryAt | int64 | Unix-milli of last honored MP recovery tick; zero if never |

### character.MapKey

Composite key for character location tracking.

| Field | Type | Description |
|-------|------|-------------|
| Tenant | tenant.Model | Tenant context |
| Field | field.Model | Field location (world, channel, map, instance) |

### data/map.Model

Map data retrieved from external data service.

| Field | Type | Description |
|-------|------|-------------|
| seats | uint32 | Number of fixed seats in map |

### data/setup.Model

Chair setup data retrieved from external data service.

| Field | Type | Description |
|-------|------|-------------|
| recoveryHP | uint32 | HP recovery amount honored per tick for the chair item |
| recoveryMP | uint32 | MP recovery amount honored per tick for the chair item |

## Invariants

- A character can only sit on one chair at a time
- Fixed chairs must have chairId less than map's seat count
- Portable chairs must have item category 301 (chairId / 10000 == 301)
- Portable chairs require character ownership of the corresponding item
- Chair is automatically cleared on character logout, map change, or channel change
- A recovery tick is honored against a portable chair's setup data only while the character is seated on that chair
- A recovery tick is honored at most once per stat (HP, MP) per 4000ms (minRecoveryTickIntervalMillis) per character
- When a recovery tick is honored, the chair item's recovery value is applied; the character-claimed recovery value is not applied
- When the chair is not portable, has no recovery stats, or setup data cannot be retrieved, the character-claimed HP/MP values are passed through unchanged

## Chair Types

| Type | Description |
|------|-------------|
| FIXED | Map-fixed chairs validated against map seat count |
| PORTABLE | Item-based chairs validated by item category and ownership |

## Processors

### chair.Processor

Primary processor for chair operations.

| Method | Description |
|--------|-------------|
| GetById | Retrieve chair for character |
| Set | Sit character on chair with validation |
| Clear | Clear chair for character |
| RecoverAndEmit | Process an HP/MP recovery tick for character and emit corresponding character HP/MP change commands |

### character.Processor

Tracks character locations for map-based queries.

| Method | Description |
|--------|-------------|
| InMapProvider | Provider for characters in a map |
| GetCharactersInMap | Retrieve characters in a map |
| Enter | Register character in map |
| Exit | Remove character from map |
| TransitionMap | Move character between maps |
| TransitionChannel | Move character between channels |

### data/map.Processor

Retrieves map data from external data service.

| Method | Description |
|--------|-------------|
| GetById | Retrieve map data by map ID |

### data/setup.Processor

Retrieves chair setup data (HP/MP recovery values) from external data service.

| Method | Description |
|--------|-------------|
| GetById | Retrieve setup data by chair item ID |

### validation.Processor

Validates character item ownership via external query aggregator service.

| Method | Description |
|--------|-------------|
| HasItem | Check if character owns at least one of the specified item |

## Registries

### chair.Registry

Redis-backed chair assignment storage (singleton). Tenant-scoped.

| Method | Description |
|--------|-------------|
| Get | Get chair for character |
| Set | Assign chair to character |
| Clear | Remove chair assignment |

### character.Registry

Redis-backed character location storage (singleton). Tenant-scoped.

| Method | Description |
|--------|-------------|
| AddCharacter | Add character to map |
| RemoveCharacter | Remove character from map |
| GetInMap | Get all characters in map |

## Error Types

| Code | Description |
|------|-------------|
| INTERNAL | Internal system error |
| ALREADY_SITING | Character already sitting on a chair |
| DOES_NOT_EXIT | Chair does not exist |
| NOT_OWNED | Character does not own the portable chair item |
