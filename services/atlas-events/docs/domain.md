# Event Domain

## definition

### Responsibility

Represents the static configuration of an event: its type, name, enabled flag, and an opaque configuration payload interpreted only by the registered handler for that type.

### Core Models

**definition.Model** — Immutable; constructed via `Builder` (`NewBuilder`, `Model.Builder`).

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Definition id |
| theType | string | Event type; the registry key |
| name | string | Human-readable name |
| enabled | bool | Whether the definition is active |
| configuration | json.RawMessage | Opaque, handler-interpreted configuration |
| createdAt | time.Time | Creation timestamp |
| updatedAt | time.Time | Last-update timestamp |

### Invariants

- `type`, `name` are required to `Build()`.
- `configuration`, when set, must be valid JSON.
- A definition's configuration is validated against its registered handler (`registry.Get(type).ValidateConfiguration`) before it is ever persisted — both the REST `Create` path and the seeder's `BulkCreate` path call this; a type with no registered handler is rejected.
- `SetEnabled` changes only the `enabled` flag and `updated_at`; it never touches occurrences or scheduled work.
- Every seeded definition is created disabled; enabling one is a separate administrative action.

### Processors

**definition.Processor** — Interface + `ProcessorImpl`. Created via `NewProcessor(l, ctx, db)`, tenant-scoped through the attached `*gorm.DB` context.

- `GetById(id)`
- `GetAllPaged(page)`
- `GetByType(theType)`
- `GetEnabledByType(theType)`
- `SetEnabled(id, enabled)` — toggles the flag only (never touches occurrences or scheduled work)
- `Create(m)` — validates configuration against the registered handler before persisting

## occurrence

### Responsibility

Represents a live instance of an event: one row per occurrence, its map scope (which maps it is visually active on), and its monster set (which spawned monsters belong to it and whether each is still alive). Pairs every state or stage change with a `transition` row in one transaction.

### Core Models

**occurrence.Model** — Immutable; constructed via `Builder` (`NewBuilder`, `Model.Builder`).

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Occurrence id |
| definitionId | uuid.UUID | Owning definition |
| theType | string | Event type (mirrors the definition's type) |
| state | string | ACTIVE / COMPLETED / CANCELLED / FAILED |
| stage | string | Handler-defined stage name |
| context | json.RawMessage | Opaque, handler-interpreted occurrence context |
| worldId | world.Id | World scope |
| channelId | channel.Id | Channel scope |
| voyageId | uuid.UUID | Voyage scope; `uuid.Nil` when the event has no voyage scope |
| concurrencyKey | string | Gameplay slot this occurrence occupies; empty means unbounded |
| startedAt | time.Time | Start timestamp |
| nextTransitionAt | *time.Time | When the next `OCCURRENCE_TRANSITION` work should fire, if any |
| completedAt | *time.Time | Completion timestamp, if completed |
| completionReason | string | Why the occurrence completed |

Occurrence states: `ACTIVE`, `COMPLETED`, `CANCELLED`, `FAILED`.

Generic completion reasons declared in this package (shared by any event type that spawns monsters or is voyage-scoped, rather than invented per event type):

- `MONSTERS_ELIMINATED` — every spawned monster (the MonsterTally SET) is accounted for and none alive.
- `VESSEL_ARRIVED` — the voyage the occurrence was scoped to reached its destination before every spawned monster was eliminated.

**occurrence.ListFilters** — Zero-value-means-unset filter set for `Processor.ListPaged`: `DefinitionId`, `Type`, `State`, `WorldId`, `ChannelId`, `MapId`, `VoyageId`, `StartedAtFrom`, `StartedAtTo`.

### Invariants

- Every state or stage change writes the occurrence row and a `transition` row in one transaction (`CreateFromSeed`, `ApplyProgress`, `Complete`); there is no path that writes one without the other.
- An occurrence's concurrency key, when non-empty, is unique among ACTIVE occurrences for the same definition — a `CreateFromSeed` call that collides returns `ErrConcurrencyKeyTaken`. An empty key opts out of this constraint (unbounded).
- Completion is a guarded update (`WHERE id = ? AND state = 'ACTIVE'`), not a lock: a losing caller (another path already completed the same occurrence) gets `won == false` (from `Complete`) or `ErrAlreadyCompleted` (from `ApplyProgress`'s terminal branch) and must skip its own cleanup.
- `ObserveMonsterSpawned` is insert-if-absent, never an upsert: a `KILLED` observation that arrives before its `CREATED` counterpart already recorded the monster dead, and a late `CREATED` must not resurrect it (the two events share a topic with no cross-partition ordering guarantee).
- `ObserveMonsterGone` is an upsert to `alive=false`, idempotent regardless of arrival order relative to `ObserveMonsterSpawned`.
- `MonsterTally` reports counts derived from the monster SET (total observed, how many alive), not a counter.
- `VisualsInMap` and the map-scope query never filter on event type: the generic layer answers "what visuals are active in this map", not "what CRIMSON_BALROG visuals".

### State Transitions

1. **Create** (`CreateFromSeed`): a handler's `Evaluate` result (a `registry.Seed`) becomes a new ACTIVE occurrence, its map-scope rows, and an `OCCURRENCE_CREATED` transition row, in one transaction.
2. **Progress** (`ApplyProgress`): a handler's `Start`/`Advance` result (a `registry.Progress`) updates the occurrence's stage/context/next-transition-at and writes a paired transition row. When `Progress.Terminal` is true, the same call also completes the occurrence (state COMPLETED, completedAt, completionReason) using the same guarded update `Complete` uses.
3. **Complete** (`Complete`): an externally driven completion (not from a handler's `Progress`) — guarded update to COMPLETED with a caller-supplied reason, trigger type, and trigger reference.

## transition

### Responsibility

Records the history of stage transitions for an occurrence: one row per transition. Read-only from this package's own perspective — every write happens inside `occurrence`'s paired transaction; this package exposes no write path of its own.

### Core Models

**transition.Model** — Immutable; constructed via `Builder` (`NewBuilder`, `Model.Builder`).

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Transition id |
| occurrenceId | uuid.UUID | Owning occurrence |
| fromStage | string | Prior stage; empty for the creation row |
| toStage | string | Stage (or state, when the event has no staged component) reached |
| occurredAt | time.Time | When the transition occurred |
| triggerType | string | What caused the transition |
| triggerReference | string | Reference tied to the trigger (e.g. a scheduled-work id) |

Trigger types: `OCCURRENCE_CREATED`, `OCCURRENCE_START`, `SCHEDULED_WORK`, `MONSTER_KILLED`, `VOYAGE_ARRIVED`.

### Invariants

- `toStage` and `triggerType` are required to `Build()`.
- `fromStage` is empty only for the creation row.
- Transitions are ordered oldest-first when read.

### Processors

**transition.Processor** — Interface + `ProcessorImpl`. Created via `NewProcessor(l, ctx, db)`.

- `GetByOccurrenceId(occurrenceId)` — full history, oldest-first. There is no write method; adding one would open a second path to the table that `occurrence`'s paired transaction already owns exclusively.

## scheduling

### Responsibility

Represents and drives durable, delayed event work: one row per unit of work (a trigger evaluation or an occurrence-transition advance), claimed and executed by a background poller across every tenant.

### Core Models

**scheduling.Model** — Immutable; constructed via `Builder` (`NewBuilder`, `Model.Builder`).

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Work row id |
| definitionId | uuid.UUID | Owning definition |
| occurrenceId | uuid.UUID | Associated occurrence; `uuid.Nil` for work that predates one |
| theType | string | `TRIGGER_EVALUATION` or `OCCURRENCE_TRANSITION` |
| context | json.RawMessage | Opaque, handler-interpreted work context |
| executeAt | time.Time | When the work becomes eligible to run |
| state | string | PENDING / PROCESSING / COMPLETED / CANCELLED / FAILED |
| attempts | int | Number of times this work has been attempted |
| lastError | string | Error from the most recent failed attempt |
| dedupeKey | string | Collapses redelivered scheduling requests for the same logical work; empty opts out of dedup |
| tenantId, tenantRegion, tenantMajor, tenantMinor | — | Owning tenant's full identity, carried on the row so the cross-tenant poller can rebuild a tenant context for a single claimed row without a separate lookup |

Work types: `TRIGGER_EVALUATION`, `OCCURRENCE_TRANSITION`.
States: `PENDING`, `PROCESSING`, `COMPLETED`, `CANCELLED`, `FAILED`.

### Invariants

- `definitionId`, `theType`, and a non-zero `executeAt` are required to `Build()`.
- `Schedule` is deduplicated by `dedupeKey` among PENDING/PROCESSING rows: a redelivered schedule request for the same key is a no-op that returns the existing row (`created == false`), never a second row or an error. An empty key opts out entirely.
- `ClaimBatch` atomically moves due PENDING rows to PROCESSING (`SKIP LOCKED`, so replicas do not block each other), stamping `claimed_by`/`claimed_at`.
- `Reclaim` returns rows stuck PROCESSING past the configured lease back to PENDING, incrementing `attempts`.
- `ExecuteOne`'s outcome policy: success completes the row (COMPLETED); an unregistered handler type fails permanently (FAILED, no retry); any other error retries with backoff up to `MaxAttempts`, after which the row is FAILED.
- `dispatch` never switches on a type constant; `registry.Get` is the only seam into event-specific behavior.
- `CancelPendingForDefinition` cancels only PENDING rows for a definition; a PROCESSING row (already claimed) is left alone.
- The poller is deliberately cross-tenant: it must see every tenant's due work, so its queries bypass the standard tenant filter, then re-enter a tenant-scoped context per claimed row before invoking any handler or write.
- When an ownership predicate is installed (multi-deployment/ephemeral-environment support), `ClaimBatch` and `Reclaim` only touch rows belonging to tenants this deployment serves; an unowned row is left PENDING for its owning deployment.

### Processors

**scheduling.Administrator** — Entry point onto write operations. Constructed via `NewAdministrator(l, ctx, db)`, tenant-scoped.

- `Schedule(m)` — inserts a new row, or returns the existing active row for the same dedupe key.
- `SetState(id, newState, lastError)` — transitions a row's state.
- `CancelPendingForDefinition(db)(definitionId)` — bulk-cancels a definition's PENDING rows.

**scheduling.Processor** — The poller-facing entry point onto claiming, lease reclaim, and execution. NOT tenant-scoped (the one deliberate exception in this service). Constructed via `NewProcessor(l, ctx, db)`.

- `ClaimBatch(instanceId, limit)` — claims due work.
- `Reclaim(lease)` — returns abandoned work to PENDING.
- `ExecuteOne(m)` — dispatches a claimed row through the registered handler (`Evaluate`/`Start` for `TRIGGER_EVALUATION`, `Advance` for `OCCURRENCE_TRANSITION`) and applies the outcome.

**scheduling.Poller** — A plain ticker (not leader-elected) over `Reclaim` + `ClaimBatch` + `ExecuteOne`, driven by `Config` (interval, lease, batch size, max attempts, backoff). Constructed via `NewPoller(l, ctx, db, cfg)`; started via `Run` or `Start`.

## registry

### Responsibility

The single seam between the generic event infrastructure (definition/occurrence/transition/scheduling) and event-specific behavior. The generic layer reaches event behavior only through `Handler`, resolved by type string; it never switches on a type constant.

### Core Models

**registry.Handler** — Interface every concrete event type implements and registers once via `Register`.

- `Type() string` — the registry key.
- `ValidateConfiguration(raw json.RawMessage) error` — rejects a configuration this handler cannot interpret.
- `ConcurrencyKey(ctx, workContext) (string, error)` — names the gameplay slot an occurrence would occupy; empty means unlimited.
- `ConcurrencyKeyIsConstant() bool` — whether `ConcurrencyKey`'s result never varies with its input (a single, per-type gameplay slot vs. one per voyage/world/channel/etc.); declared directly by the handler, not inferred by probing.
- `Evaluate(ctx, d, w) (*Seed, error)` — decides whether a `TRIGGER_EVALUATION` should produce an occurrence; `(nil, nil)` is the ordinary "no occurrence" outcome.
- `Start(ctx, o) (Progress, error)` — orchestrates the side effects of a newly created occurrence.
- `Advance(ctx, o, w) (Progress, error)` — handles a due `OCCURRENCE_TRANSITION` row, including completion side effects on the terminal branch.

**registry.Seed** — A handler's request to create an occurrence: `Stage`, `Context`, `WorldId`, `ChannelId`, `VoyageId` (`uuid.Nil` for no voyage scope), `ConcurrencyKey`, `Maps` (`[]MapScope`), `NextTransitionAt`.

**registry.Progress** — What a handler settles an occurrence into after `Start`/`Advance`: `Stage`, `NextTransitionAt`, `Terminal`, `CompletionReason`.

**registry.Definition** / **registry.Occurrence** / **registry.Work** — Read-only, generic-layer views of a definition, an occurrence, and the due scheduled row, respectively, passed into handler methods.

### Invariants

- `Register` panics on a duplicate type registration (a programming error, not a runtime condition).
- `Get` returning `(_, false)` is a failure the caller must surface (a work row fails with "no handler for type X"), never a fallback to a default handler.
- Domain reactions to Kafka events (a monster died, a voyage arrived, a character logged in) do not go through `Handler` — they are ordinary Kafka consumers owned by the event's own package.

## orchestration

### Responsibility

Thin composition point between `definition` and `scheduling`: owns the "enabling a definition also schedules work" behavior so that `definition.Processor.SetEnabled` stays toggle-only and this business rule does not live in the REST handler. Exists as its own package because `definition` cannot import `scheduling` (which already imports `definition` to resolve a claimed row's definition) without a cycle.

### State Transitions

`SetEnabled(l, ctx, db)(id, enabled)`:

1. Toggles the definition's `enabled` flag via `definition.Processor.SetEnabled`.
2. On a false→true transition only, schedules one generic `TRIGGER_EVALUATION` row at `time.Now()`, empty context, dedupe key `enable:<definitionId>`.
3. A false→false, true→true, or true→false transition schedules nothing.

The scheduled row is entirely generic; what an empty-context evaluation means is decided entirely by the enabled definition's own handler.

## anniversary (ANNIVERSARY)

### Responsibility

A scheduled, server-wide window during which EXP/drop rates are multiplied and every online character receives a buff for the duration. Has no voyage scope; at most one occurrence can be active tenant-wide.

### Core Models

**anniversary.Config** — Definition configuration: `ScheduledStart`, `ScheduledEnd` (time.Time), `ExpMultiplier`, `DropMultiplier` (float64), `BuffSourceId` (int32).

**anniversary.OccurrenceContext** — Occurrence context: `ScheduledEnd`, `ExpMultiplier`, `DropMultiplier`, `BuffSourceId`.

### Invariants

- `ScheduledEnd` must be after `ScheduledStart`; `ExpMultiplier` and `DropMultiplier` must both be greater than zero.
- `ConcurrencyKey` is the constant string `"anniversary"`; `ConcurrencyKeyIsConstant()` is true.
- A definition whose whole window has already elapsed schedules and evaluates to nothing (no retroactive occurrence).
- A definition enabled before its window opens schedules the trigger evaluation for `ScheduledStart`, not immediately.
- The buff granted has no expiry duration; the occurrence's existence is the authoritative fact that the window is active, and completion cancels the buff explicitly by correlation (the occurrence id).

### State Transitions

1. **Evaluate**: if the window has fully elapsed, no occurrence. If it has not opened yet, the start is (re)scheduled and no occurrence is produced now. Otherwise, an occurrence is seeded with the window's end and multipliers.
2. **Start**: settles `NextTransitionAt` to the window's scheduled end; grants no buffs itself.
3. **Advance** (fires at the window's end): emits one `CANCEL_BY_CORRELATION` buff command (sweeping every buff granted under this occurrence's id, tenant-wide) and completes the occurrence with reason `SCHEDULED_END`.
4. **Login reaction** (independent of the above): on every character login, every active ANNIVERSARY occurrence's buff (exp/drop multipliers) is granted to the logging-in character. This is a best-effort reaction — atlas-events being unavailable delays the buff, never the login.

### Processors

- **anniversary.Handler** — the `registry.Handler` implementation (`NewHandler`, `NewHandlerWith`).
- **anniversary.LoginProcessor** — `OnLogin(e)`: grants every active occurrence's buff to the logging-in character.
- **anniversary.Scheduler** — `OnDefinitionEnabled(d)`: schedules the definition's start row, applying the same "already elapsed" and "not yet open" rules `Evaluate` applies.

## crimsonbalrog (CRIMSON_BALROG)

### Responsibility

A chance-based monster attack that may occur on an Orbis/Ellinia boat voyage: on voyage departure, a delayed evaluation rolls whether an attack happens; if it does, an occurrence spawns monsters on the boat's attack maps until either every monster is eliminated or the voyage arrives.

### Core Models

**crimsonbalrog.Config** — Definition configuration: `ApplicableRouteIds` ([]string, route slugs), `TriggerDelay`/`TriggerDelayJitter` (Duration), `AttackProbability` (float64), `MonsterId` (monster.Id), `MonsterCount` (uint32), `AttackMaps` ([]AttackMap), `RelatedMapIds` ([]_map.Id), `BackgroundMusic` (string), `Visual` (VisualConfig).

**crimsonbalrog.AttackMap** — One deck an occurrence may spawn on: `MapId`, `SpawnPositions` ([]Position).

**crimsonbalrog.WorkContext** — Per-work-row context carried from `OnVoyageDeparted` through to `Evaluate`: `VoyageId`, `RouteId`, `WorldId`, `ChannelId`, `StagingMapId`, `DestinationMapId`, `ObservationMapId`, `EnRouteMapIds`, `DepartedAt`.

**crimsonbalrog.OccurrenceContext** — Occurrence context: `RouteId`, `VoyageId`, `WorldId`, `ChannelId`, `AttackMaps`, `RelatedMapIds`, `MonsterId`, `MonsterCount`, `BackgroundMusic`, `Visual`.

Occurrence stage: `ATTACKING` — the only stage this event type reaches.

### Invariants

- `ApplicableRouteIds` must contain at least one non-empty route slug; `AttackProbability` must be in `[0,1]`; `MonsterCount` must be greater than zero; `TriggerDelay`/`TriggerDelayJitter` must not be negative; `AttackMaps` must contain at least one map, each with at least `MonsterCount` spawn positions; `Visual.Name` must be set.
- `ConcurrencyKey` scopes an occurrence to one voyage in one channel of one world (`voyageId|worldId|channelId`); `ConcurrencyKeyIsConstant()` is false — several occurrences can be active concurrently, one per voyage.
- `Advance` is unreachable in normal operation: this event type never schedules an `OCCURRENCE_TRANSITION` (completion is driven externally, by monster elimination or voyage arrival); a call to it is treated as a bug and fails the work row.
- A character in a related ("cabin") map counts as "aboard" for the evaluation gate, same as one in an attack map; only attack maps get the visual and spawns.
- `OnVoyageDeparted` schedules delayed work (no occurrence yet) for every enabled, applicable definition, with the trigger-delay jitter rolled at scheduling time (not at execution) so the delay survives a restart.
- Monster CREATED observation is insert-if-absent; KILLED/DESTROYED observation is upsert-to-dead; completion by elimination fires only once every spawned monster (matched by provenance) is accounted for and none alive.
- A monster is attributed to an occurrence only via its provenance pair (`spawnSourceType == "EVENT"`, `spawnSourceId == occurrenceId`), echoed back on the monster status event.
- Voyage arrival completes any still-ACTIVE occurrence scoped to that voyage, racing safely against the elimination path via the same guarded completion.
- Completion cleanup (despawn everything the occurrence owns, hide the visual) runs only on the path that won the guarded completion, never on both.

### State Transitions

1. **Trigger** (`OnVoyageDeparted`, reacting to a `VOYAGE_DEPARTED` transport event): for every enabled definition whose `ApplicableRouteIds` matches the departed route, schedules one `TRIGGER_EVALUATION` row at `departedAt + triggerDelay + jitter`, deduplicated per (definition, voyage, world, channel).
2. **Evaluate**: gates in order — voyage still underway, definition still enabled, probability roll, someone aboard (attack or related maps). Any gate failing is the ordinary "no occurrence" outcome, logged with a named reason. All gates passing seeds a new occurrence at stage `ATTACKING`.
3. **Start**: for each attack map, emits a SHOW visual event, then spawns `MonsterCount` monsters at that map's configured positions, tagged with the occurrence's provenance. No `NextTransitionAt` — nothing about this occurrence is timed.
4. **Complete by elimination** (reacting to monster status events): tracks the occurrence's monster set; once every spawned monster is accounted for and none alive, completes the occurrence (reason `MONSTERS_ELIMINATED`) and hides the visual on every attack map.
5. **Complete by arrival** (reacting to a `VOYAGE_ARRIVED` transport event): completes any still-ACTIVE occurrence scoped to the arriving voyage (reason `VESSEL_ARRIVED`), then despawns everything it owns and hides its visual on every attack map.

### Processors

- **crimsonbalrog.Handler** — the `registry.Handler` implementation (`NewHandler`).
- **crimsonbalrog.TriggerProcessor** — `OnVoyageDeparted(e)`: schedules delayed evaluation work.
- **crimsonbalrog.MonsterProcessor** — `OnMonsterStatus(e)`: tracks the occurrence's monster set and completes on full elimination.
- **crimsonbalrog.ArrivalProcessor** — `OnVoyageArrived(e)`: completes and cleans up any occurrence scoped to the arriving voyage.
