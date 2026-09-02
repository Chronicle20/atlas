# Dragon Domain

## Responsibility

Manages the lifecycle of Evan's dragon (CDragon): creating it for a dragon-bearing character, destroying it, and relaying its movement. A dragon is 1:1 with its owning character; there is no separate dragon id.

## Core Models

### Model

Represents an active dragon. Immutable; constructed via `Builder` (`NewBuilder`, `Clone`).

| Field | Type | Description |
|-------|------|-------------|
| ownerCharacterId | uint32 | Owner character id (the dragon's identity) |
| fld | field.Model | Field the dragon occupies (world, channel, map, instance) |
| x | int32 | Dragon X coordinate |
| y | int32 | Dragon Y coordinate |
| stance | byte | Dragon stance |
| jobId | job.Id | Owner's job id at the time the dragon was created/refreshed |

`Move(x, y)` returns a copy with the position updated; stance is left unchanged.

### HasDragon

`HasDragon(t tenant.Model, wireJobId job.Id) bool` reports whether `wireJobId` resolves, on the tenant's client version, to an Evan growth stage (`EvanStage1`..`EvanStage10`). The Evan beginner job id is excluded.

## Invariants

- A character has at most one dragon; the owner character id is the registry's primary key, so this is a property of the key space, not an enforced check.
- `Create` is the only place the job gate (`HasDragon`) is enforced.
- `Create` emits `CREATED` only on the absent-to-present transition; if a dragon already existed for the owner, the stored state is refreshed (field/position/stance/jobId) and no event is emitted.
- `Create` is a no-op (returns nil, no event) when the owner character is not found (`requests.ErrNotFound`) or does not currently resolve to a dragon-bearing job.
- `Destroy` on an absent dragon is a silent no-op returning nil.
- `Destroy` emits `DESTROYED` only when a dragon existed and was removed.
- `Move` never creates a dragon as a side effect; a move for a character with no dragon is dropped with a warning and no event.
- `Move` does not persist the `stance` parameter it receives; the dragon's stored stance is only ever set by `Create`. The `stance` parameter is retained on the method signature because it is part of the Kafka `MoveCommandBody` contract but is deliberately unused for persistence.

## State Transitions

### Dragon Lifecycle

1. **Create**: The owner's job and position are fetched from `atlas-character`. If the character is not found, or its job does not resolve to a dragon-bearing Evan stage, this is a no-op. Otherwise the dragon is written to the registry (creating it or refreshing its state) in the given field, and `CREATED` is emitted only when the dragon did not previously exist.
2. **Move**: If the owner has no stored dragon, the move is dropped. Otherwise the dragon's position is updated (stance untouched) and `MOVED` is emitted, carrying the raw movement blob.
3. **Destroy**: If the owner has no stored dragon, this is a no-op. Otherwise the dragon is removed from the registry and `DESTROYED` is emitted.

### Job Change

A `JOB_CHANGED` character-status event whose new job id no longer resolves to a dragon-bearing Evan stage triggers `Destroy`. A stage-up within the Evan range is left alone.

### Field Change (Map/Channel Change)

A `MAP_CHANGED` or `CHANNEL_CHANGED` character-status event triggers `Destroy` in the old field followed by `Create` in the new field (destroy-then-create, not a field update), so the old field receives a `DESTROYED` broadcast and the new field receives a `CREATED` broadcast.

## Processors

### Processor

Interface defining dragon processing operations. Constructed via `NewProcessor(l, ctx)`, which wires the Kafka emitter and the character REST processor.

**Queries:**
- `GetByCharacterId`: Retrieves the dragon owned by a character.
- `GetInField`: Retrieves all dragons in a field.

**Commands:**
- `Create`: Fetches the owner's job/position, applies the job gate, and writes the dragon to the registry; emits `CREATED` only on absent-to-present.
- `Destroy`: Removes the owner's dragon if one exists; emits `DESTROYED`.
- `Move`: Updates the owner's dragon position if one exists; emits `MOVED` with the raw movement blob.

# Character Domain

## Responsibility

Provides atlas-dragons' read-only view of a character: the job id (to decide whether a dragon exists) and the position (to seed a newly created dragon's spawn coordinates). Fetched via a sparse fieldset from atlas-character; no other character data is retrieved.

## Core Models

### Model

| Field | Type | Description |
|-------|------|--------------|
| id | uint32 | Character id |
| jobId | job.Id | Character's job id |
| x | int16 | Character X coordinate |
| y | int16 | Character Y coordinate |
| stance | byte | Character stance |

Constructed via `Builder` (`NewBuilder`); `id` is required.

## Invariants

- `Builder.Build` fails with an error when `id` is zero.

## Processors

### Processor

Interface defining character retrieval. Constructed via `NewProcessor(l, ctx)`.

- `GetById`: Retrieves a character by id via the atlas-character REST API. Returns `requests.ErrNotFound` when the character no longer exists; callers must treat that as "gone," not as a fetch failure.
