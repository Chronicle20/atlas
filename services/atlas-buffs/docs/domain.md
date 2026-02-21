# Buff Domain

## Responsibility

The buff domain manages temporary stat modifications for game characters, including application, cancellation, automatic expiration of buffs, disease immunity checks, and periodic poison damage processing.

## Core Models

### buff.Model

Immutable representation of a buff.

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Unique buff identifier |
| sourceId | int32 | Source identifier (skill/item ID) |
| level | byte | Buff level |
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
| worldId | world.Id | World identifier |
| channelId | channel.Id | Channel identifier |
| characterId | uint32 | Character identifier |
| buffs | map[int32]buff.Model | Active buffs keyed by sourceId |

### PoisonTickEntry

Represents a character with an active poison buff for tick processing.

| Field | Type | Description |
|-------|------|-------------|
| Tenant | tenant.Model | Tenant context |
| WorldId | world.Id | World identifier |
| ChannelId | channel.Id | Channel identifier |
| CharacterId | uint32 | Character identifier |
| Amount | int32 | Poison damage amount |

## Invariants

- Each buff has a unique UUID generated at creation
- Buffs are keyed by sourceId within a character's buff map
- Applying a buff with an existing sourceId replaces the previous buff
- Expiration time is calculated as createdAt + duration seconds
- A buff is expired when expiresAt is before current time
- Duration must be positive (ErrInvalidDuration)
- Changes must not be empty (ErrEmptyChanges)
- Disease buffs (STUN, POISON, SEAL, DARKNESS, WEAKEN, CURSE, SEDUCE, CONFUSE, UNDEAD, SLOW, STOP_PORTION) are blocked if the character has a HOLY_SHIELD buff active
- Poison ticks enforce a minimum 1-second interval between ticks per character
- Poison tick damage is applied as negative HP change (amount negated to int16)

## Processors

### Processor

Primary domain processor for buff operations.

| Method | Description |
|--------|-------------|
| GetById | Retrieve character with buffs by character ID |
| Apply | Apply buff to character with disease immunity check, emit applied event |
| Cancel | Cancel buff by sourceId and emit expired event |
| CancelAll | Cancel all buffs for character and emit expired events |
| ExpireBuffs | Process and emit events for all expired buffs |
| ProcessPoisonTicks | Find characters with poison buffs and emit HP change commands |

### Registry

Redis-backed buff storage (singleton). Per-tenant key isolation via TenantRegistry.

| Method | Description |
|--------|-------------|
| Apply | Add or replace buff for character |
| Get | Retrieve character by ID |
| GetTenants | Get all tenants with registered characters |
| GetCharacters | Get all characters for a tenant |
| Cancel | Remove buff by sourceId |
| CancelAll | Remove all buffs for character |
| GetExpired | Remove and return expired buffs for character |
| HasImmunity | Check if character has HOLY_SHIELD buff active |
| GetPoisonCharacters | Find all characters with active non-expired POISON buffs |
| GetLastPoisonTick | Get last poison tick timestamp for character |
| UpdatePoisonTick | Record poison tick timestamp for character |
| ClearPoisonTick | Remove poison tick state for character |

## Background Tasks

### Expiration Task

Runs on configurable interval (default 10000ms) to:
1. Iterate all tenants with active buffs
2. For each tenant, process all characters
3. Remove expired buffs and emit expired events

### PoisonTick Task

Runs on configurable interval (default 1000ms) to:
1. Iterate all tenants with active buffs
2. For each tenant, find characters with active POISON buffs
3. Enforce minimum 1-second interval between ticks per character
4. Emit HP change commands for poison damage
