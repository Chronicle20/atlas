# Chalkboard Domain

## Responsibility

The chalkboard domain manages chalkboard messages for game characters, including setting, clearing, and querying messages.

## Core Models

### chalkboard.Model

Represents a chalkboard message for a character.

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Character identifier |
| message | string | Chalkboard message content |

### character.MapKey

Composite key for character location tracking.

| Field | Type | Description |
|-------|------|-------------|
| Tenant | tenant.Model | Tenant context |
| WorldId | world.Id | World identifier |
| ChannelId | channel.Id | Channel identifier |
| MapId | map.Id | Map identifier |

## Invariants

- A character can have at most one chalkboard message at a time
- Setting a new message replaces any existing message
- Chalkboard is automatically cleared on character logout, map change, or channel change

## Processors

### chalkboard.Processor

Primary processor for chalkboard operations.

| Method | Description |
|--------|-------------|
| GetById | Retrieve chalkboard message for character |
| Set | Set chalkboard message for character |
| Clear | Clear chalkboard message for character |

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

## Registries

### chalkboard.Registry

In-memory chalkboard message storage (singleton). Thread-safe.

| Method | Description |
|--------|-------------|
| Get | Get message for character |
| Set | Set message for character |
| Clear | Remove message for character |

### character.Registry

In-memory character location storage (singleton). Thread-safe with per-map locking.

| Method | Description |
|--------|-------------|
| AddCharacter | Add character to map |
| RemoveCharacter | Remove character from map |
| GetInMap | Get all characters in map |
