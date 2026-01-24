# Fame Domain

## Responsibility

The fame domain tracks and validates fame transactions between characters.

## Core Models

### Fame Model

| Field | Type | Description |
|-------|------|-------------|
| tenantId | uuid.UUID | Tenant identifier |
| id | uuid.UUID | Fame log entry identifier |
| characterId | uint32 | Character giving fame |
| targetId | uint32 | Character receiving fame |
| amount | int8 | Fame amount (+1 or -1) |
| createdAt | time.Time | Timestamp of fame transaction |

### Character Model

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Character identifier |
| name | string | Character name |
| level | byte | Character level |

## Invariants

- A character must be at least level 15 to give fame
- A character can only give fame once per day
- A character can only give fame to a specific target once per month
- The target character must exist

## Processors

### Fame Processor

- `GetByCharacterIdLastMonth`: Retrieves fame logs for a character within the last month
- `RequestChange`: Validates and processes a fame change request
- `RequestChangeAndEmit`: Validates, processes, and emits messages for a fame change

### Character Processor

- `GetById`: Retrieves character data by ID from the character service
- `RequestChangeFame`: Emits a command to change a character's fame value
- `RequestChangeFameAndEmit`: Emits a command message to change fame
