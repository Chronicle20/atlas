# Buff Domain

## Responsibility

The buff domain manages temporary stat modifications for game characters, including application, cancellation, and automatic expiration of buffs.

## Core Models

### buff.Model

Immutable representation of a buff.

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Unique buff identifier |
| sourceId | int32 | Source identifier (skill/item ID) |
| duration | int32 | Duration in seconds |
| changes | []stat.Model | Stat modifications |
| createdAt | time.Time | Creation timestamp |
| expiresAt | time.Time | Expiration timestamp |

### stat.Model

Immutable representation of a stat modification.

| Field | Type | Description |
|-------|------|-------------|
| statType | string | Stat type identifier |
| amount | int32 | Modification amount |

### character.Model

Represents a character with active buffs.

| Field | Type | Description |
|-------|------|-------------|
| tenant | tenant.Model | Tenant context |
| worldId | byte | World identifier |
| characterId | uint32 | Character identifier |
| buffs | map[int32]buff.Model | Active buffs keyed by sourceId |

## Invariants

- Each buff has a unique UUID generated at creation
- Buffs are keyed by sourceId within a character's buff map
- Applying a buff with an existing sourceId replaces the previous buff
- Expiration time is calculated as createdAt + duration seconds
- A buff is expired when expiresAt is before current time

## Processors

### Processor

Primary domain processor for buff operations.

| Method | Description |
|--------|-------------|
| GetById | Retrieve character with buffs by character ID |
| Apply | Apply buff to character and emit applied event |
| Cancel | Cancel buff by sourceId and emit expired event |
| CancelAll | Cancel all buffs for character and emit expired events |
| ExpireBuffs | Process and emit events for all expired buffs |

### Registry

In-memory buff storage (singleton). Thread-safe with per-tenant locking.

| Method | Description |
|--------|-------------|
| Apply | Add or replace buff for character |
| Get | Retrieve character by ID |
| GetTenants | Get all tenants with registered characters |
| GetCharacters | Get all characters for a tenant |
| Cancel | Remove buff by sourceId |
| CancelAll | Remove all buffs for character |
| GetExpired | Remove and return expired buffs for character |

## Background Tasks

### Expiration Task

Runs on configurable interval (default 10000ms) to:
1. Iterate all tenants with active buffs
2. For each tenant, process all characters
3. Remove expired buffs and emit expired events
