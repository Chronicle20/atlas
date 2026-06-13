# Mount Domain

## mount

### Responsibility

Manages per-character mount progression: level, accumulated experience, and tiredness. There is exactly one mount progression record per character. The domain advances tiredness on a fixed tick for active (tamed) mounts, applies feed actions that reduce tiredness and award experience, and exposes the current progression.

### Core Models

#### Mount Model

Immutable representation of a character's mount progression. Constructed via `NewModelBuilder` / `Clone`; mutated through the builder's setters and `Build()`. Fields are private with read-only getters.

| Field | Type | Description |
|-------|------|-------------|
| tenantId | uuid.UUID | Tenant identifier |
| characterId | uint32 | Owning character identifier |
| id | uuid.UUID | Mount identifier |
| level | int | Mount level |
| exp | int | Cumulative mount experience |
| tiredness | int | Mount tiredness |
| lastTirednessTickAt | *time.Time | Timestamp of the last tiredness tick (nil when never ticked) |

#### Mount Builder

Constructs mount models. Created via `NewModelBuilder(tenantId, characterId, id)` or `Clone(model)`. Defaults applied by `NewModelBuilder`: level 1, exp 0, tiredness 0, nil lastTirednessTickAt.

| Method | Description |
|--------|-------------|
| SetLevel | Sets the level |
| SetExp | Sets the cumulative experience |
| SetTiredness | Sets the tiredness |
| SetLastTirednessTickAt | Sets the last-tick timestamp |
| Build | Returns the constructed Model |

#### Mount Ride Context

The per-character state needed to tick an active (tamed) mount, held in the Redis-backed active-mount registry.

| Field | Type | Description |
|-------|------|-------------|
| WorldId | world.Id | World the mount is in (used for status event emission) |
| SkillId | int32 | Skill identifier of the mount |
| VehicleId | int32 | Vehicle item identifier of the mount |

#### Active Entry

A single active mount carrying enough context for the ticker to iterate active mounts across tenants.

| Field | Type | Description |
|-------|------|-------------|
| Tenant | tenant.Model | Tenant the active mount belongs to |
| CharacterId | uint32 | Riding character identifier |
| Ctx | MountRideContext | Ride context (world, skill, vehicle) |

### Invariants

- There is one mount progression record per character, scoped per tenant.
- A new mount defaults to level 1, exp 0, tiredness 0.
- Tiredness increments by 1 per tick and clamps at 99.
- `CAP` is 31; a mount may level up only while its current level is strictly below `CAP`.
- Exp is a cumulative running total and is never decremented.
- A feed gains at most one level.
- The per-level exp requirement table (`mountExp`) has 29 entries (valid indices 0..28). `ExpNeededForLevel` returns `math.MaxInt32` for levels outside the table (negative or `>= len(mountExp)`).
- Only active (tamed) mounts are present in the active-mount registry; skill-only mounts are never registered and never ticked.

### State Transitions

#### Tiredness Tick

`TickTiredness(t)` advances tiredness by one tick:

| From | To | tooTired |
|------|----|----------|
| t < 99 | t + 1 | false |
| t >= 99 | 99 | true |

The `tooTired` flag is true when the value could not increase further this tick (input was already at the clamp).

#### Feed

`ApplyFeed(FeedInput)` computes the heal → exp → level-up outcome for a single feed. `HealMax` is supplied by the caller. It is a pure function with no I/O and no input mutation.

```
heal      = min(tiredness, healMax)
tiredness = tiredness - heal
if healMax > 0 and heal > 0:
    exp += ceil((heal / healMax) * (2*level + 6))   // float division
if level < CAP and exp >= ExpNeededForLevel(level):  // cumulative threshold
    level++                                          // LevelUp = true; exp NOT reset
```

`FeedInput` carries `Level`, `Exp`, `Tiredness`, `HealMax`. `FeedResult` carries the updated `Level`, `Exp`, `Tiredness`, and a `LevelUp` flag.

#### Per-Level Experience Table

`mountExp` (index = level):

```
1, 24, 50, 105, 134, 196, 254, 263, 315, 367,
430, 543, 587, 679, 725, 897, 1146, 1394, 1701, 2247,
2543, 2898, 3156, 3313, 3584, 3923, 4150, 4305, 4550
```

#### Default On First Read

`GetByCharacterId` creates a default mount record (level 1 / exp 0 / tiredness 0) for the character when no record exists yet, and returns it.

### Processors

#### Mount Processor

The processor is persistence-backed. Emitting methods take a `*message.Buffer` so the caller controls the transaction/emit boundary. `worldId` is always supplied by the caller and is never stored on the model.

| Method | Description |
|--------|-------------|
| GetByCharacterId | Loads the character's mount scoped to the tenant in context; creates and returns a default record on first read |
| ApplyTick | Advances tiredness by one tick, persists the new tiredness and lastTirednessTickAt, and buffers a TICK event; DB write and event share one transaction |
| ApplyFeedAndEmit | Applies the feed math, persists the new level/exp/tiredness, and buffers a FEED event; healMax supplied by the caller |
| EmitSet | Loads (or default-creates) the mount and buffers a SET event carrying current progression; changes no progression state |
| With | Returns a processor copy with options applied (e.g. WithTransaction) |

#### Active-Mount Registry

Redis-backed registry of characters with an active tamed mount, keyed by character identifier and partitioned by tenant. A set of tenants is tracked alongside so the ticker can enumerate active mounts across all tenants.

| Method | Description |
|--------|-------------|
| Add | Records (or overwrites) the active mount for a character within the tenant in context |
| Remove | Clears the active mount for a character within the tenant in context; a no-op when no entry exists |
| GetActive | Returns every active mount across all tenants |

#### Tiredness Task

A 60-second ticker that increments tiredness on every active (tamed) mount. A single task iterates the active-mount registry; there are no per-character goroutines or timers. Each entry carries its own tenant, so the tick is scoped to that tenant before invoking the processor. Skill-only mounts are not in the registry and are never ticked.
