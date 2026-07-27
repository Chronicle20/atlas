# Domain

## Card

### Responsibility

The Card domain tracks per-character ownership and level of individual monster-book cards.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| tenantId | uuid.UUID | Tenant identifier |
| characterId | character.Id | Owning character |
| cardId | item.Id | Card item identifier |
| level | uint8 | Card level (1-5) |
| isSpecial | bool | Whether the card is in the special-card range |
| lastEventId | *uuid.UUID | Event id of the last applied CARD_PICKED_UP command, for idempotency |
| firstAcquiredAt | time.Time | Timestamp the card was first acquired |
| updatedAt | time.Time | Timestamp of the last update |

### Invariants

- characterId is required (cannot be 0)
- cardId must classify as `item.ClassificationConsumableMonsterCard` (`IsCardId`)
- level must be in the range [1, MaxLevel] where MaxLevel is 5
- isSpecial is derived from cardId: true when `cardId / 1000 >= 2388` (`IsSpecialCard`)

### State Transitions

- A card is inserted at level 1 on first pickup.
- A repeat pickup of an already-owned card increments its level by 1, capped at MaxLevel.
- Each transition is guarded by the command's eventId against the card's lastEventId: a pickup carrying an eventId equal to the stored lastEventId is treated as a duplicate and produces no level change.
- Once a card is at MaxLevel, further pickups persist the new lastEventId (so future replays of that same eventId still no-op) but leave the level at MaxLevel.

### Processors

#### Card Processor

| Method | Description |
|--------|-------------|
| GetByCharacterId | Gets all cards owned by a character |
| GetByCharacterIdAndCardId | Gets a single card owned by a character |
| GetByCharacterIdAndIsSpecial | Gets a character's cards filtered by special/normal classification |
| Add | Applies a card pickup within a caller-supplied message buffer, buffering a CARD_ADDED status event unless the pickup was a duplicate |
| AddAndEmit | Applies a card pickup in its own database transaction and emits the resulting status event(s) |
| DeleteByCharacterId | Deletes all of a character's cards |
| WithTransaction | Returns a Processor bound to the given transaction |

## Collection

### Responsibility

The Collection domain aggregates a character's monster-book statistics (book level, normal/special card counts, EXP bonus percent) and tracks the collection's selected cover card.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| tenantId | uuid.UUID | Tenant identifier |
| characterId | character.Id | Owning character |
| coverCardId | item.Id | Selected cover card item id (0 = no cover) |
| coverMobId | monster.Id | Monster id represented by the cover card, resolved via atlas-data (0 if unresolved or no cover) |
| bookLevel | uint16 | Monster book level |
| normalCount | uint16 | Count of owned normal (non-special) cards |
| specialCount | uint16 | Count of owned special cards |
| expBonusPercent | uint16 | Party EXP bonus percentage granted by the book |
| lastCoverEventId | *uuid.UUID | Event id of the last applied SET_COVER command, for idempotency |
| createdAt | time.Time | Timestamp the collection row was created |
| updatedAt | time.Time | Timestamp of the last update |

`TotalUniqueCards()` is a derived getter equal to `normalCount + specialCount`.

### Invariants

- characterId is required (cannot be 0)
- `GetByCharacterId` synthesizes a default in-memory Model with bookLevel=1 and all counts/cover fields zero-valued when no row exists for the character; it does not persist this default
- SetCoverAndEmit rejects a non-zero coverCardId that does not classify as a card item (`ErrCardIdOutOfRange`) or that the character does not own at level >= 1 (`ErrCoverNotOwned`); coverCardId 0 is always allowed and clears the cover
- Cover-mob-id resolution (`resolveCoverMobId`) never fails the SetCover operation: any atlas-data lookup error, a card not marked `monsterBook`, or a resolved `monsterId` of 0 results in a stored coverMobId of 0

### State Transitions

- `RecomputeAndEmit` recomputes normalCount, specialCount, bookLevel, and expBonusPercent from the character's current cards and upserts the collection row. bookLevel is computed by `computeBookLevel`: starting from level 0 and expToNext=1, level is incremented and `level*10` added to expToNext while the running expToNext is still less than or equal to totalUniqueCards; the first level for which the total falls short is returned. expBonusPercent equals bookLevel (`computeExpBonusPercent`). A STATS_CHANGED status event and an EXPERIENCE_CHANGED distribution event are only buffered when normalCount, specialCount, bookLevel, or expBonusPercent actually changed from the prior stored values.
- `SetCoverAndEmit` validates cover ownership, resolves the cover's mob id, and updates coverCardId/coverMobId within a database transaction, guarded by lastCoverEventId: a call whose eventId matches the stored lastCoverEventId is a no-op and does not emit a COVER_CHANGED event.

### Processors

#### Collection Processor

| Method | Description |
|--------|-------------|
| GetByCharacterId | Gets a character's collection stats, synthesizing a level-1 default if no row exists |
| SetCoverAndEmit | Validates and sets the cover card, emitting COVER_CHANGED on change |
| RecomputeAndEmit | Recomputes and persists stats from current card ownership, emitting STATS_CHANGED and EXPERIENCE_CHANGED on change |
| DeleteByCharacterId | Deletes a character's collection row |
| WithTransaction | Returns a Processor bound to the given transaction |

## Consumable (data client)

### Responsibility

The Consumable domain is a client-side representation of a consumable item fetched from the atlas-data service, used only to resolve a monster-book cover card to the monster id it represents.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| monsterBook | bool | Whether the item is a monster-book card |
| monsterId | uint32 | Monster id represented by the card |

### Processors

#### Consumable Processor

| Method | Description |
|--------|-------------|
| GetById | Gets a consumable by item id from the atlas-data service |
