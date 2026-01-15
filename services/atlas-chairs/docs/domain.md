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

### character.MapKey

Composite key for character location tracking.

| Field | Type | Description |
|-------|------|-------------|
| Tenant | tenant.Model | Tenant context |
| WorldId | world.Id | World identifier |
| ChannelId | channel.Id | Channel identifier |
| MapId | map.Id | Map identifier |

### data/map.Model

Map data retrieved from external data service.

| Field | Type | Description |
|-------|------|-------------|
| seats | uint32 | Number of fixed seats in map |

## Invariants

- A character can only sit on one chair at a time
- Fixed chairs must have chairId less than map's seat count
- Portable chairs must have item category 301 (chairId / 10000 == 301)
- Chair is automatically cleared on character logout, map change, or channel change

## Chair Types

| Type | Description |
|------|-------------|
| FIXED | Map-fixed chairs validated against map seat count |
| PORTABLE | Item-based chairs validated by item category |

## Processors

### chair.Processor

Primary processor for chair operations.

| Method | Description |
|--------|-------------|
| GetById | Retrieve chair for character |
| Set | Sit character on chair with validation |
| Clear | Clear chair for character |

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

Retrieves map data from external service.

| Method | Description |
|--------|-------------|
| GetById | Retrieve map data by map ID |

## Registries

### chair.Registry

In-memory chair assignment storage (singleton).

| Method | Description |
|--------|-------------|
| Get | Get chair for character |
| Set | Assign chair to character |
| Clear | Remove chair assignment |

### character.Registry

In-memory character location storage (singleton). Thread-safe with per-map locking.

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
