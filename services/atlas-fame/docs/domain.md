# Domain

## Fame

### Responsibility

The Fame domain manages character fame (reputation) transaction logs. It tracks which character gave fame to which target, the amount (+1 or -1), and when the transaction occurred.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| tenantId | uuid.UUID | Tenant identifier |
| id | uuid.UUID | Fame log entry identifier |
| characterId | uint32 | Character who gave fame |
| targetId | uint32 | Character who received fame |
| amount | int8 | Fame amount (+1 or -1) |
| createdAt | time.Time | Timestamp of fame transaction |

### Invariants

- tenantId is required (cannot be uuid.Nil)
- characterId is required (cannot be 0)
- targetId is required (cannot be 0)
- amount must be exactly 1 or -1

### Processors

#### Fame Processor

| Method | Description |
|--------|-------------|
| GetByCharacterIdLastMonth | Gets all fame logs for a character in the last month |
| ByCharacterIdLastMonthProvider | Returns a provider for fame logs for a character in the last month |
| RequestChange | Requests a fame change with validation |
| RequestChangeAndEmit | Requests a fame change and emits messages |
| DeleteByCharacterId | Deletes all fame logs involving a character (as giver or receiver) |

Fame change validation rules:
- Source character must exist
- Target character must exist
- Source character must be level 15 or higher
- Source character cannot have given fame today
- Source character cannot have given fame to this target in the last month

## Character

### Responsibility

The Character domain is a client-side representation of characters fetched from the atlas-character service. It provides character lookup and fame change request functionality.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Character identifier |
| name | string | Character name |
| level | byte | Character level |

### Processors

#### Character Processor

| Method | Description |
|--------|-------------|
| GetById | Gets a character by ID from atlas-character service |
| ByIdProvider | Returns a provider for a character by ID |
| RequestChangeFame | Requests a fame change via Kafka command |
| RequestChangeFameAndEmit | Requests a fame change and emits the message |
