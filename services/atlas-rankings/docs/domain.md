# Ranking Domain

## Responsibility

Computes per-world overall and job-category character rankings for a tenant, tracks rank movement between recompute cycles, and records cycle progress for observability and due-ness scheduling.

## Core Models

### Model

Represents one character's current ranking. Immutable; constructed via `Make(Entity)` or `ModelBuilder` (`NewBuilder`).

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character id |
| name | string | Character name |
| worldId | world.Id | World id |
| jobCategory | uint16 | Job category (`jobId / 100`) |
| level | byte | Character level |
| jobId | job.Id | Character job id |
| overallRank | uint32 | 1-based rank within the world |
| overallRankMove | int32 | Change from the previous cycle's overall rank (positive = moved up) |
| jobRank | uint32 | 1-based rank within the world and job category |
| jobRankMove | int32 | Change from the previous cycle's job rank (positive = moved up) |
| computedAt | time.Time | Timestamp of the cycle that produced this rank |

### CycleModel

Represents recompute cycle progress for a tenant.

| Field | Type | Description |
|-------|------|-------------|
| lastStartedAt | time.Time | When the most recent cycle began |
| lastCompletedAt | *time.Time | When the most recent cycle finished (nil if never completed) |
| charactersRanked | uint32 | Number of characters ranked in the most recent completed cycle |
| durationMs | uint32 | Wall-clock duration of the most recent completed cycle, in milliseconds |

### Input

One eligible (non-GM) character snapshot supplied to `Rank`.

| Field | Type | Description |
|-------|------|-------------|
| CharacterId | uint32 | Character id |
| Name | string | Character name |
| WorldId | world.Id | World id |
| JobId | job.Id | Job id |
| Level | byte | Character level |
| Experience | uint32 | Character experience |

### Ranked

The computed placement for one character, produced by `Rank`.

| Field | Type | Description |
|-------|------|-------------|
| CharacterId | uint32 | Character id |
| Name | string | Character name |
| WorldId | world.Id | World id |
| JobCategory | uint16 | Job category (`jobId / 100`) |
| Level | byte | Character level |
| JobId | job.Id | Job id |
| OverallRank | uint32 | 1-based rank within the world |
| JobRank | uint32 | 1-based rank within the world and job category |

### character.Model

Trimmed, immutable read model of an atlas-character character, carrying only the attributes ranking computation needs: id, name, worldId, jobId, level, experience, gm.

## Invariants

- 0 is never stored as a rank; an unranked character simply has no row.
- Ranks are 1-based and unique within their scope. The characterId tiebreak makes the sort order a strict total order, so dense and ordinal ranking coincide.
- `JobCategory` buckets a job id into its top-level division via `jobId / 100`.
- Rank ordering is level DESC, experience DESC, characterId ASC. Job ranks reuse the same sorted order restricted to each category.
- `Move` returns `previousRank − newRank` (positive = moved up); a character with no previous entry (prev == 0) moves 0.
- Characters with `gm > 0` are excluded entirely from ranking: not ranked, not counted, and their stale row is pruned once no longer restamped.
- A recompute cycle is idempotent and convergent: a crashed run's ranks are fully repaired by the next run. The move fields are not self-healing within a single cycle — a crash between the upsert and cycle-completion steps, or a back-to-back double run, makes the next cycle compute moves against its own freshly-written ranks, reading 0 for every unchanged character for that one cycle; this is structural to the single-row schema (no previous-rank column) and self-heals on the following cycle.
- An entirely empty character scan against a non-empty rankings table skips the prune step for that cycle, to avoid wiping live rankings on a possibly-transient empty scan. A scan that legitimately returns zero eligible (non-GM, still-existing) characters still prunes.
- `IsDue` reports true when no cycle has ever run for the tenant, or when `now - lastStartedAt >= interval`.
- The recompute task re-reads each tenant's configured interval on every tick, never from a boot-time snapshot.

## State Transitions

### Recompute Cycle

1. **Start**: `startCycle` upserts the tenant's cycle row, stamping `lastStartedAt`. `lastCompletedAt`, `charactersRanked`, and `durationMs` are left untouched, so a crash between start and completion preserves the previous cycle's completion stats.
2. **Scan**: The full character list for the tenant is read from atlas-character; GM characters (`gm > 0`) are excluded before ranking.
3. **Rank**: `Rank` computes per-world overall and job-category placements from the eligible character set.
4. **Move**: Each ranked character's move fields are computed against its previous stored rank (0 if none).
5. **Persist**: Ranking rows are upserted in batches of 500, keyed on `(tenant_id, character_id)`.
6. **Prune**: Rows not restamped by this cycle are deleted, unless the character scan came back empty against a non-empty rankings table, in which case the prune is skipped for that cycle.
7. **Complete**: `completeCycle` records `lastCompletedAt`, `charactersRanked`, and `durationMs` for the tenant.

## Processors

### Processor

Interface defining ranking read and recompute operations. Constructed via `NewProcessor(l, ctx, db)`, which wires an atlas-character-backed `CharacterSupplier` by default; `WithCharacterSupplier` allows substituting the character source.

**Queries:**
- `ByCharacterIdProvider` / `GetByCharacterId`: Retrieves a single character's ranking.
- `ByCharacterIdsProvider` / `GetByCharacterIds`: Retrieves rankings for a set of characters.
- `LeaderboardProvider`: Returns one page of ranked characters for a world, ordered overall or within a job category.
- `IsDue`: Reports whether the tenant's recompute interval has elapsed since the last cycle start.

**Commands:**
- `Recompute`: Scans characters, ranks them, upserts rows stamped with the given timestamp, prunes stale rows, and records the cycle.

### RecomputeTask

Periodic task (`tasks` package) that ticks on a base interval (60 seconds), re-enumerates tenants from atlas-tenants, and re-reads each tenant's configured cadence on every tick. For each tenant whose interval has elapsed (`IsDue`), it runs `Recompute`. One tenant's failure to enumerate its due-ness or to recompute is logged and skipped, never fatal to the tick. Context cancellation mid-tick abandons remaining tenants for that tick.

### tenant.Processor

Interface for tenant enumeration. `AllProvider` / `GetAll` drain every page of atlas-tenants' `GET /tenants`.

### character.Processor

Interface for the tenant's character scan. `AllProvider` / `GetAll` drain every page of atlas-character's `GET /characters`.

### configuration.GetRecomputeInterval

Resolves a tenant's configured recompute interval from atlas-tenants' per-tenant rankings configuration. A missing configuration (404) or any other read error falls back to `DefaultRecomputeInterval` (5 minutes); a configured value of 0 also falls back to the default.
