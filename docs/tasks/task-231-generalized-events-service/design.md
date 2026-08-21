# Generalized Events Service (`atlas-events`) — Design

Task: task-231
PRD: [`prd.md`](prd.md) (v1, approved)
Created: 2026-08-15
Status: Draft for review

---

## 0. How to read this document

The PRD fixed *what* must be true. This document fixes *how*, and resolves the
seven open questions in PRD §19 against the code rather than against
recollection. Every claim about existing behavior below carries a `file:line`
or a quoted symbol; where something could not be verified it is labelled
**unverified** rather than guessed.

§1 lists seven findings that correct or sharpen the PRD's grounding notes —
read it before anything else, because three of them change the shape of the
work. §2–§13 are the design proper. §14 records alternatives that were
considered and rejected. §15 answers PRD §19 one question at a time. §16 is the
architecture-validation walkthrough PRD §20.9 requires.

---

## 1. Grounding corrections (G1–G7)

These were found by reading the current tree. They supersede the corresponding
PRD statements where they disagree.

**G1 — `ARRIVED`/`DEPARTED` are emitted once per route transition, not once per
channel.** PRD FR-V4 says the new voyage events are "emitted once per channel,
matching the existing per-channel fan-out at `processor.go:150-186`." The
per-channel fan-out at that location is the *warp* loop; the status emits sit
outside it:

- `transport/processor.go:164` — `mb.Put(transport.EnvEventTopicStatus, ArrivedStatusEventProvider(r.Id(), r.ObservationMapId()))`, once, on `OpenEntry`.
- `transport/processor.go:182` — `DepartedStatusEventProvider(...)`, once, on `InTransit`.

Both bodies carry only a map id (`kafka/message/transport/kafka.go`,
`ArrivedStatusEventBody{MapId}`). `atlas-channel` recovers world/channel from
its own `server.Model`, which is why one emit suffices today. The new
`VOYAGE_DEPARTED` / `VOYAGE_ARRIVED` events **do** need per-channel emission,
because `atlas-events` has no channel identity of its own — but this is a new
emission shape, not a mirror of an existing one. Consequence: the design adds a
per-channel loop around the new emits, and the existing single emits stay
exactly where they are.

**G2 — FR-V6 requires no `atlas-channel` code change.** Both handlers already
guard on type before doing anything:
`kafka/consumer/route/consumer.go:62` (`if e.Type != route2.EventStatusArrived { return }`) and `:82`.
Unknown types on `EVENT_TOPIC_TRANSPORT_STATUS` are already ignored. The work
FR-V6 implies is therefore **a regression test that pins the behavior**, not a
consumer edit. This is exactly the class of seam the project's "green
verify.sh ≠ correct" rule warns about, so the test is mandatory, not optional.

**G3 — A trip already has an identity; a voyage can be derived from it.**
`TripScheduleModel` carries `tripId uuid.UUID` (`transport/model.go:290-297`),
assigned when `NewScheduler(...).ComputeSchedule()` runs. What it does not have
is *per-occurrence* identity: the schedule is time-of-day recurrent and
`Evaluate` re-derives state from the clock every tick
(`transport/model.go:149-229`), so the same `tripId` recurs daily. This makes
FR-V1 satisfiable by **derivation** rather than storage — see §7 — which in turn
answers PRD open question 3 without adding a table or a Redis key.

**G4 — `COMMAND_TOPIC_MONSTER` fans every message to every handler.** The
service's own comments state it twice, at `catchCommandBody` and
`killCommandBody` in `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go`:

> every handler on this shared command topic json-unmarshals every message, so a
> field name whose type disagrees with a sibling body produces one spurious
> unmarshal error per message

The two new `SPAWN_FIELD` fields (FR-P1) and the new `DESTROY_BY_SOURCE` body
must therefore use field names that either do not appear in any sibling body, or
appear with an identical Go type. `spawnSourceType string` and
`spawnSourceId string` appear in no sibling body today (verified by reading the
full body list in that file), so they are safe — but the constraint has to be
restated in the plan, because it is invisible at the call site.

**G5 — `atlas-data` does not expose Map.wz `info/bgm`.** `grep -n "bgm|Bgm"`
over `services/atlas-data/atlas.com/data/map/*.go` (excluding tests) returns
nothing. No service can currently name a map's default background music. This
decides PRD open question 4 — see §15.4.

**G6 — The map-entry hook already exists and already has this exact shape.**
`atlas-channel`'s `SpawnForSelf`
(`kafka/consumer/map/consumer.go:167`, a run of `routine.Go` blocks from `:188` to `:350`) is a sequence of
independent `routine.Go` blocks, each performing a best-effort lookup for the
entering character and announcing a packet on success. One of them **already
REST-queries `atlas-transports`**:

```go
routine.Go(l, ctx, func(_ context.Context) {
    hasShip, err := route.NewProcessor(l, ctx).IsBoatInMap(f.MapId())
    ...
})
```

and another queries `weather.NewProcessor(l, ctx).GetActive(f)` and then
announces `FieldEffectBackgroundMusicBody(ci.BgmPath)`. FR-B14/B15/B16 is one
more block in that list. Non-blocking and fail-open are properties of the
existing shape — a panic-recovering goroutine whose error path only logs — not
new machinery we have to invent.

**G7 — All three candidate Anniversary stats exist on v83, and one of them is a
cross-version footgun.** In
`libs/atlas-packet/model/character_temporary_stat.go`,
`MESO_UP_BY_ITEM` (:142), `ITEM_UP_BY_ITEM` (:146), `EVENT_RATE` (:161) and
`EXP_BUFF_RATE` (:168) are all registered in the *unconditional* pre-`SoulStone`
run — the version gates begin only after `SoulStone` (:184 onwards,
`gmsV95Plus`/`post87`/`extended`). All four use `NoOpForeignValueWriter`, so on the remote
path they are mask-only and add no bytes. However, `EVENT_RATE` **is** a member
of the JMS movement-affecting set (`movementAffectingStatNames`, JMS branch, `:799`), which gates whether the client reads the reset/give trailing byte.
Choosing it would make a JMS tenant's Anniversary buff interact with the
movement filter for no benefit. See §10.3.

---

## 2. Architecture overview

```
 EVENT_TOPIC_TRANSPORT_STATUS ─┐
 EVENT_TOPIC_MONSTER_STATUS  ──┤     ┌──────────────────────────────────┐
 EVENT_TOPIC_CHARACTER_STATUS ─┴────►│           atlas-events           │
                                     │                                  │
                                     │  event/definition                │
                                     │  event/occurrence   ← generic    │
                                     │  event/transition                │
                                     │  event/scheduling                │
                                     │  event/registry  (type→handler)  │
                                     │  ────────────────────────────    │
                                     │  events/crimsonbalrog            │
                                     │  events/anniversary  ← specific  │
                                     └───────┬─────────────────┬────────┘
                                             │ commands        │ REST
                     ┌───────────────────────┼─────────┐       │
                     ▼                       ▼         ▼       ▼
             COMMAND_TOPIC_MONSTER   COMMAND_TOPIC   EVENT_TOPIC   GET /events/
              SPAWN_FIELD(+source)    _CHARACTER_     _EVENT_       occurrences
              DESTROY_BY_SOURCE        BUFF           VISUAL      (map entry,
                     │                   │              │          admin UI)
                     ▼                   ▼              ▼
              atlas-monsters        atlas-buffs    atlas-channel
                                         │
                                         ▼ EVENT_TOPIC_CHARACTER_BUFF_STATUS
                                    atlas-rates
```

Three structural commitments:

1. **The generic layer never names an event type.** It owns rows, state
   machines, the poller, and REST. It reaches event behavior only through a
   `Handler` interface resolved out of a registry keyed by `type` (§3.2). This
   is FR-X3, and it is enforceable by a test (§17.4).
2. **`atlas-events` issues commands; it never renders.** The enemy-ship visual
   is a new Kafka event that `atlas-channel` translates into `CONTI_MOVE`. No
   packet code changes (F3).
3. **Everything correctness-critical is a durable row.** No timer, ticker, or
   in-memory map is the system of record for anything (FR-N1). The only
   in-memory component is the poller loop itself, which is stateless.

---

## 3. `atlas-events` service structure

### 3.1 Package layout

```
services/atlas-events/atlas.com/events/
  main.go
  event/
    definition/    model, builder, entity, provider, administrator,
                   processor, rest, resource, subdomain (seeding), validate
    occurrence/    model, builder, entity, provider, administrator,
                   processor, rest, resource, scope (map child table)
    transition/    model, builder, entity, provider, administrator, rest
    scheduling/    model, builder, entity, provider, administrator,
                   processor, poller
    registry/      Handler interface, type registry, registration
  events/
    crimsonbalrog/ config, handler, trigger, occurrence, cleanup
    anniversary/   config, handler, trigger, occurrence
  kafka/
    consumer/{transport,monster,character}/
    message/{transport,monster,character,buff,event}/
  rest/handler.go
```

This follows the Atlas conventions verbatim: immutable domain models with
builders, `provider.go` for reads, `administrator.go` for writes,
`processor.go` for business logic, separate `rest.go` REST representations
(FR-N17, FR-N18). The `definition` package mirrors
`services/atlas-party-quests/atlas.com/party-quests/definition/` closely enough
that its `entity.go` / `subdomain.go` are the reference implementations to copy
from — including `seeder.Subdomain` for FR-D7.

### 3.2 The event handler registry (FR-X3)

The single seam between generic and specific:

```go
// event/registry/handler.go
type Handler interface {
    // Type is the definition type this handler owns, e.g. "CRIMSON_BALROG".
    Type() string

    // ValidateConfiguration rejects a definition whose configuration this
    // handler cannot interpret (FR-D6). Returns a field-scoped error.
    ValidateConfiguration(raw json.RawMessage) error

    // ConcurrencyKey names the gameplay slot an occurrence would occupy.
    // The generic layer enforces at most one non-terminal occurrence per
    // (tenant, definition, key) — see §5.3. Empty string means unlimited.
    ConcurrencyKey(ctx context.Context, workContext json.RawMessage) (string, error)

    // Evaluate decides whether a TRIGGER_EVALUATION should produce an
    // occurrence. Returning (nil, nil) is the ordinary "no occurrence"
    // outcome (FR-B7, FR-B8), not an error.
    Evaluate(ctx context.Context, d definition.Model, work scheduling.Model) (*occurrence.Seed, error)

    // Start orchestrates the side effects of a newly created occurrence
    // (FR-B11) and returns the stage/next-transition it settles into.
    Start(ctx context.Context, o occurrence.Model) (occurrence.Progress, error)

    // Advance handles a due OCCURRENCE_TRANSITION work row (FR-A14).
    Advance(ctx context.Context, o occurrence.Model, work scheduling.Model) (occurrence.Progress, error)

    // Complete performs cleanup for a terminal transition (FR-B18, FR-B19,
    // FR-A15). Must be idempotent (FR-B20).
    Complete(ctx context.Context, o occurrence.Model, reason string) error
}
```

Domain reactions (a monster died, a voyage arrived, a character logged in) do
**not** go through this interface. They are ordinary Kafka consumers owned by
the event package, registered from that package's own `InitConsumers`. The
generic layer never sees them. This is the difference between "a registry
mapping type to a handler" (allowed by FR-X3) and "a central dispatch table
containing event logic" (forbidden).

Registration is by import side effect in `main.go`:

```go
registry.Register(crimsonbalrog.NewHandler())
registry.Register(anniversary.NewHandler())
```

`registry.Get(type)` returning `(Handler, false)` is a **failure**, not a
fallback: a definition row whose type has no handler makes its work rows fail
with `lastError = "no handler for type X"` rather than silently succeeding.

---

## 4. Data model

New Postgres database `atlas-events`, GORM `AutoMigrate` per entity in the
`Migration` pattern already used by
`services/atlas-party-quests/.../definition/entity.go:MigrateTable`.

| Table | Notes |
|---|---|
| `event_definition` | `id`, `tenant_id`, `type`, `name`, `enabled`, `configuration jsonb`, `created_at`, `updated_at` |
| `event_occurrence` | `id`, `tenant_id`, `event_definition_id`, `type`, `state`, `stage`, `context jsonb`, `world_id`, `channel_id`, `voyage_id`, `concurrency_key`, `started_at`, `next_transition_at`, `completed_at`, `completion_reason` |
| `event_occurrence_map` | `occurrence_id`, `map_id`, `visual bool` — FR-API8 |
| `event_occurrence_monster` | `occurrence_id`, `unique_id`, `monster_id`, `alive bool`, `observed_at` — §9.5 |
| `event_occurrence_transition` | `id`, `tenant_id`, `occurrence_id`, `from_stage`, `to_stage`, `occurred_at`, `trigger_type`, `trigger_reference` |
| `scheduled_event_work` | `id`, `tenant_id`, `event_definition_id`, `event_occurrence_id`, `type`, `context jsonb`, `execute_at`, `state`, `claimed_by`, `claimed_at`, `attempts`, `last_error`, `dedupe_key` |

`event_occurrence_monster` is not in the PRD's §14 table. It is the durable form
of "are any left?" (FR-B18) and is justified in §9.5 — a counter cannot be made
idempotent under Kafka redelivery; a set can.

`concurrency_key` is likewise an addition, and is what makes PRD open question 5
answerable generically (§15.5).

### 4.1 Indexes

```sql
-- FR-S4 poller hot path. PARTIAL on state: this is what makes the poll cost
-- independent of how many COMPLETED rows have accumulated (FR-N16), and is
-- why no retention policy is needed in this task (§15.7).
CREATE INDEX ix_sew_pending_due ON scheduled_event_work (execute_at)
    WHERE state = 'PENDING';

-- FR-S7 lease reclaim sweep.
CREATE INDEX ix_sew_processing_claimed ON scheduled_event_work (claimed_at)
    WHERE state = 'PROCESSING';

-- FR-B4 dedup. Partial so a cancelled/failed row does not block a retry.
CREATE UNIQUE INDEX ux_sew_dedupe ON scheduled_event_work (tenant_id, dedupe_key)
    WHERE state IN ('PENDING','PROCESSING');

-- FR-API7 second query (Anniversary "is it happening?").
CREATE INDEX ix_occ_type_state ON event_occurrence (tenant_id, type, state);

-- §5.3 concurrency policy.
CREATE UNIQUE INDEX ux_occ_concurrency ON event_occurrence
    (tenant_id, event_definition_id, concurrency_key)
    WHERE state = 'ACTIVE' AND concurrency_key <> '';

-- FR-B15 map-entry query. Leading map_id because that is the most selective
-- column at the call site; world/channel/state are filtered from the joined
-- occurrence row.
CREATE INDEX ix_occ_map ON event_occurrence_map (map_id, occurrence_id)
    WHERE visual = true;
CREATE INDEX ix_occ_active_scope ON event_occurrence (tenant_id, world_id, channel_id, state)
    WHERE state = 'ACTIVE';
```

The map-entry query becomes:

```sql
SELECT o.* FROM event_occurrence o
JOIN event_occurrence_map m ON m.occurrence_id = o.id
WHERE m.map_id = $1 AND m.visual
  AND o.tenant_id = $2 AND o.world_id = $3 AND o.channel_id = $4
  AND o.state = 'ACTIVE' AND o.type = $5;
```

Both sides are index-covered; nothing scans jsonb. The plan task that adds these
indexes must include an `EXPLAIN` against a seeded table rather than asserting
the shape — PRD §14 explicitly asks for that confirmation and it is cheap.

### 4.2 Tenancy

Every table carries `tenant_id` and every REST/processor path derives it from
the request context, per the existing `atlas-database` tenant filter. The **one**
deliberate exception is the poller, which must see all tenants: it uses
`database.WithoutTenantFilter(ctx)` — the same escape hatch
`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/store.go:242`
uses for its timeout reaper — and then re-enters a tenant-scoped context per
claimed row before invoking any handler. A handler never runs in an
unfiltered context (FR-N13, FR-N14).

---

## 5. Scheduling

### 5.1 Claim

The poller is a plain ticker calling `ClaimBatch` (FR-S4). The claim is one
transaction:

```go
db.Transaction(func(tx *gorm.DB) error {
    var rows []Entity
    err := tx.WithContext(database.WithoutTenantFilter(ctx)).
        Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
        Where("state = ? AND execute_at <= ?", StatePending, time.Now()).
        Order("execute_at ASC").Limit(batchSize).Find(&rows).Error
    ...
    return tx.Model(&Entity{}).Where("id IN ?", ids).
        Updates(map[string]any{"state": StateProcessing, "claimed_by": instanceId, "claimed_at": now}).Error
})
```

This is FR-S6 verbatim, and it is the pattern already proven twice in the repo
(`libs/atlas-outbox/drainer.go:223`, saga store `:242`). Two replicas cannot
claim the same row; `SKIP LOCKED` means the loser takes the next rows instead of
blocking.

**Why not leader election.** `libs/atlas-lock` already provides
`LeaderElection.Run`, and gating the poller behind it would also prevent double
execution. It is rejected because it makes the scheduler single-replica by
construction — a leader that is slow or wedged stalls *all* events, which
directly contradicts FR-N6 ("a single event implementation failing must not
stall the scheduler for other events"). `SKIP LOCKED` keeps every replica
working and isolates a stuck row to itself. Note also `libs/atlas-lock`'s own
`EnvVar` warning: a lease name unscoped by `ATLAS_ENV` is global across
deployments sharing a Redis, which is a whole class of failure the DB-lock
approach simply does not have.

### 5.2 Lease reclaim and retry

A sweeper (same poller loop, second query) moves rows whose
`state = 'PROCESSING' AND claimed_at < now() - lease` back to `PENDING`,
incrementing `attempts` (FR-S7). Lease timeout, poll interval, and batch size
are configuration (FR-N16).

Execution outcome per row:

| Outcome | Row transition |
|---|---|
| Handler returned normally | `COMPLETED` |
| Handler returned an error, `attempts < max` | `PENDING`, `execute_at = now + backoff`, `last_error` set |
| Handler returned an error, `attempts >= max` | `FAILED`, `last_error` retained (FR-S9) |
| Definition disabled / no handler | `COMPLETED` with no occurrence, or `FAILED` respectively |

A `FAILED` row is terminal and, because the poll index is partial on
`state = 'PENDING'`, invisible to the poller — it cannot block anything
(FR-S9, FR-N6).

### 5.3 Idempotency (FR-S8, FR-N2)

Three independent guards, because any one of them alone has a hole:

1. **Dedup on creation.** Trigger work rows carry a `dedupe_key`. For Crimson
   Balrog it is `balrog:<definitionId>:<voyageId>:<worldId>:<channelId>`
   (FR-B4); for Anniversary, `anniversary:<definitionId>:<scheduledStart>`. The
   partial unique index makes a redelivered Kafka message a no-op insert.
2. **The work row's own state transition is the commit point.** A row already
   `COMPLETED` is never re-executed, so a redelivered *execution* cannot happen.
3. **The occurrence concurrency key.** Even if 1 and 2 both fail — two
   definitions, a hand-inserted row, a clock skew — `ux_occ_concurrency`
   makes the second occurrence insert fail rather than producing two live
   Balrog attacks on one voyage. The handler treats that unique violation as
   "someone else already created it" and completes the work row successfully.

Guard 3 is the generic answer to PRD open question 5: the *mechanism* is
generic, the *policy* is the handler's, supplied by `ConcurrencyKey`.

---

## 6. Occurrence lifecycle

States are `ACTIVE`, `COMPLETED`, `CANCELLED`, `FAILED` (FR-O2); `stage` is an
opaque string the handler owns (FR-O3).

Every state or stage change is one transaction that writes both the occurrence
row and an `event_occurrence_transition` row (FR-O6, FR-T2). There is no path
that updates one without the other; the administrator exposes only the paired
operation. `trigger_type` / `trigger_reference` are parameters of that
operation, so FR-T3's traceability is structural rather than a convention
someone has to remember.

Recovery on startup (FR-N5) needs no special code: active occurrences are rows,
and pending/overdue work is picked up by the ordinary poll on the first tick.
The one thing startup does do is run the lease sweeper immediately rather than
waiting a full lease interval, so work orphaned by the previous process's death
resumes promptly.

---

## 7. Transport voyage identity (§9 of the PRD)

### 7.1 Derivation, not storage

Given G3, a voyage is uniquely determined by
`(tenantId, routeId, tripId, departure date in the route's own frame)`. The
design assigns:

```go
// transport/voyage.go
var voyageNamespace = uuid.MustParse("<fixed constant, generated once>")

func VoyageId(t tenant.Model, routeId uuid.UUID, tripId uuid.UUID, departedOn time.Time) uuid.UUID {
    return uuid.NewSHA1(voyageNamespace,
        []byte(t.Id().String()+"|"+routeId.String()+"|"+tripId.String()+"|"+departedOn.Format("2006-01-02")))
}
```

`departedOn` is the materialized departure instant (`Transition.NextAt`'s frame
— `transport/model.go:135-143` already projects a time-of-day boundary onto an
absolute instant in `now`'s date and location), truncated to the day.

Properties this buys, all of which FR-V5 asks for and none of which needs a
write:

- **Restart-safe.** Recomputed identically after a process restart mid-trip.
- **Redis-flush-safe.** The route registry (`transport/route_registry.go`) is a
  Redis `TenantRegistry`; a flush loses routes, but once they are re-seeded the
  voyage id for an in-flight trip is the same value.
- **Replica-safe.** Two `atlas-transports` pods derive the same id.
- **Correlatable without a query.** `atlas-events` holding a `voyageId` from
  `VOYAGE_DEPARTED` can match `VOYAGE_ARRIVED` by equality alone.

The one property it does *not* have: a trip that legitimately runs twice on the
same calendar day with the same `tripId` would collide. It cannot —
`ComputeSchedule` emits one row per `tripId` per day and `Evaluate` selects at
most one in-transit trip. A test pins this (§17.2).

### 7.2 New events

`kafka/message/transport/kafka.go` gains:

```go
EventStatusVoyageDeparted = "VOYAGE_DEPARTED"
EventStatusVoyageArrived  = "VOYAGE_ARRIVED"

type VoyageStatusEventBody struct {
    VoyageId         uuid.UUID  `json:"voyageId"`
    WorldId          world.Id   `json:"worldId"`
    ChannelId        channel.Id `json:"channelId"`
    StagingMapId     _map.Id    `json:"stagingMapId"`
    EnRouteMapIds    []_map.Id  `json:"enRouteMapIds"`
    DestinationMapId _map.Id    `json:"destinationMapId"`
    ObservationMapId _map.Id    `json:"observationMapId"`
    DepartedAt       time.Time  `json:"departedAt"`
}
```

One body type serves both event types — the `Type` discriminator distinguishes
them, and `VOYAGE_ARRIVED` carrying the departure instant is useful for
computing voyage duration in logs. The Kafka key is the `voyageId` string, so
all events for one voyage land on one partition and `VOYAGE_ARRIVED` cannot
overtake `VOYAGE_DEPARTED` for the same voyage.

### 7.3 Emission

In `UpdateRoute` (`transport/processor.go:136-191`):

- On `InTransit`, inside the existing `model.ForEachSlice(... p.chanP.GetAll() ...)`
  loop that already warps per channel (`:173-177`), also `mb.Put` a
  `VOYAGE_DEPARTED` for that `(worldId, channelId)`. The existing single
  `DEPARTED` emit at `:182` is untouched.
- On `AwaitingReturn`, add the same per-channel loop for `VOYAGE_ARRIVED`. This
  branch (`:149-162`) currently emits nothing — F1's central point.

Both need the departure instant and the trip id. `Evaluate` already computes the
selected trip internally; the design exposes it by having `UpdateState` return
the `Transition` (it already computes one — `model.go:100`, `processStateChange`
discards everything but `State`) plus the selected `TripScheduleModel`. This is a
small widening of an existing return, not new state.

For `AwaitingReturn` the *departure* instant is the trip's departure, materialized
onto the current day — and for a midnight-crossing trip
(`model.go:165-171, 207-217`) onto the **previous** day. This asymmetry is the
one genuinely fiddly part of §7 and gets its own table-driven test (§17.2).

### 7.4 REST

`transport/rest.go`'s route RestModel gains a nullable `voyageId`, populated only
when `state == in_transit`. `atlas-events` uses it for FR-B5 step 1 ("is the
voyage still underway?"): fetch the route, and the voyage is underway iff the
route is `in_transit` **and** its `voyageId` equals the one on the work row. A
route that has since departed on the *next* trip reports a different voyage id,
which is the correct "no longer underway" answer and is why comparing state
alone would be wrong.

---

## 8. Generic spawn provenance (§12 of the PRD)

### 8.1 Wire changes

`spawnFieldCommandBody` gains two optional string fields:

```go
type spawnFieldCommandBody struct {
    MonsterId       uint32 `json:"monsterId"`
    X               int16  `json:"x"`
    Y               int16  `json:"y"`
    Fh              int16  `json:"fh"`
    Team            int8   `json:"team"`
    SpawnSourceType string `json:"spawnSourceType,omitempty"`
    SpawnSourceId   string `json:"spawnSourceId,omitempty"`
}
```

Both names are absent from every sibling body on the topic (G4), so the
fan-to-every-handler unmarshal stays clean. `omitempty` keeps existing
producers' messages byte-identical, which is what makes FR-P5 checkable by
diffing an existing producer test's golden output.

Empty `SpawnSourceType` normalizes to `CYCLIC` **at the boundary**, in the
consumer, not at every read site — so the rest of the service only ever sees a
populated value and FR-P1's backward-compatibility rule has one enforcement
point.

`DESTROY_BY_SOURCE` is a new `fieldCommand` type (it is field-scoped — §15.6):

```go
CommandTypeDestroyBySource = "DESTROY_BY_SOURCE"

type destroyBySourceCommandBody struct {
    SpawnSourceType string `json:"spawnSourceType"`
    SpawnSourceId   string `json:"spawnSourceId"`
}
```

### 8.2 Persistence and echo

`monster.Model` gains `spawnSourceType string` / `spawnSourceId string` with
builder setters, and `storedMonster`
(`services/atlas-monsters/atlas.com/monsters/monster/registry.go:25-52`) gains
the matching JSON fields. Old Redis payloads unmarshal with both empty, which
normalizes to `CYCLIC` — no migration, no backfill (FR-P2).

For the echo (FR-P3), the fields go on the **envelope**
(`monster/kafka.go:60-69`, `statusEvent[E]`), not per-body. `statusEventFromField`
is the single constructor for every status event, so one change echoes the
provenance on all types — a superset of the CREATED/KILLED/DESTROYED that FR-P3
requires, at no extra cost and with no risk of one type being forgotten.

### 8.3 Destroy-by-source

`Registry.GetMonstersInMap(tenant, field)` already exists
(`monster/registry.go:376`). The handler enumerates the field, filters on the
source pair, and calls the existing destroy path per match. Zero matches is
success (FR-P4, FR-B20). `atlas-monsters` never interprets `spawnSourceId` —
it stores it, echoes it, and compares it for equality (FR-P6).

---

## 9. Event 1: Crimson Balrog

### 9.1 Configuration

Exactly PRD §10.2, decoded by `crimsonbalrog.Config` and validated by the
handler's `ValidateConfiguration`. Every value is configuration; the package
contains no `2`, no `0.42`, no `8150000` outside of the seed JSON (FR-B1). The
seed file (`seeder.Subdomain`, per party-quests' `definition/subdomain.go`)
carries the reference values from PRD F6 and ships **disabled** (FR-D7).

`spawnPosition` is per attack map, so the config models it as
`attackMaps: [{mapId, spawnPositions: [{x,y}...]}]` rather than a flat
`attackMapIds` plus a single position — otherwise two decks with different
geometry cannot both be configured. `relatedMapIds` stays flat: cabins get
nothing, so they need no position.

### 9.2 Departure → scheduled evaluation

Consumer on `EVENT_TOPIC_TRANSPORT_STATUS`, type-guarded to `VOYAGE_DEPARTED`.
For each enabled `CRIMSON_BALROG` definition whose `applicableRouteIds` contains
the event's `routeId`, insert one `TRIGGER_EVALUATION` row with
`execute_at = departedAt + triggerDelay + rand(0, triggerDelayJitter)` and the
full voyage scope in `context` (FR-B2). No occurrence yet (FR-B3). The
`dedupe_key` makes redelivery a no-op (FR-B4).

The jitter roll happens **here**, at scheduling time, and is persisted in
`execute_at`. Rolling it at execution time would make the delay non-durable
across restart, which FR-B5 and acceptance criterion "the configured delay
survives restart" both forbid.

### 9.3 Evaluation

`Handler.Evaluate`, in PRD FR-B5's order — the order matters, because each step
is cheaper than the next:

1. **Voyage still underway?** REST to `atlas-transports` (§7.4). No → complete
   the work, no occurrence.
2. **Definition still enabled?** Row read. No → complete, no occurrence.
3. **Probability roll.** Fail → complete, no occurrence (FR-B7).
4. **Anyone aboard?** REST to `atlas-maps` for character ids in each of
   `attackMapIds ∪ relatedMapIds` for this `(world, channel)`. The union is the
   point: a character in the cabin counts (FR-B6). Empty → complete, no
   occurrence (FR-B8).

Success creates the occurrence, `ACTIVE`/`ATTACKING`, with the map child rows —
attack maps `visual = true`, related maps `visual = false` (FR-B9, FR-B10,
FR-API8). The `visual` flag on the child table is what makes FR-B13/FR-B14's
deck-vs-cabin distinction a query predicate rather than a branch in the channel.

Steps 1 and 4 are the only network calls, and both are cheap. If either is
unreachable the work row **retries** rather than completing — an unreachable
`atlas-maps` must not be silently read as "nobody aboard."

### 9.4 Start

`Handler.Start` emits, for the attack maps of this channel only:

- `EVENT_TOPIC_EVENT_VISUAL` / `SHOW`, carrying `(worldId, channelId, mapId,
  occurrenceId, visual = CONTI_MOVE, state = 10, subState = 4, bgm = <configured>)`.
- `COMMAND_TOPIC_MONSTER` / `SPAWN_FIELD` × `monsterCount`, each with
  `spawnSourceType = EVENT`, `spawnSourceId = <occurrenceId>` (FR-B22).

`atlas-events` constructs no packets (FR-B12). The `state`/`subState` bytes are
in the event because they are gameplay content (which visual), not encoding —
`atlas-channel` maps the event to `ContiMoveBody(state, subState)`
(`socket/writer/conti_move.go`) and `FieldEffectBackgroundMusicBody(bgm)`
(both already registered: `main.go:778`, `:803`).

### 9.5 Tracking event-owned monsters

`event_occurrence_monster` rows, driven by the echoed provenance (§8.2):

- `CREATED` with our `spawnSourceId` → **insert if absent** `(occurrence_id,
  unique_id, alive = true)`.
- `KILLED` / `DESTROYED` with our `spawnSourceId` → **upsert** with
  `alive = false`.

Set semantics, not a counter. A counter cannot be made idempotent: a redelivered
`KILLED` decrements twice. The upsert is idempotent by construction, and it also
survives `KILLED` arriving before `CREATED` — the kill upserts a dead row and
the later `CREATED` must not resurrect it, which is why `CREATED` is
insert-if-absent rather than upsert. (The two events share a topic but the
envelope has no ordering guarantee across partitions, so this is a real case,
not a hypothetical.)

Completion on the eliminate path fires when
`count(rows) == monsterCount AND count(alive) == 0` (FR-B18). The first
conjunct is what stops a completion firing in the window after the first
spawn's `CREATED` but before the second's.

### 9.6 Completion

Both paths converge on one guarded transition:

```
UPDATE event_occurrence SET state='COMPLETED', completion_reason=$1, completed_at=now()
WHERE id=$2 AND state='ACTIVE'
```

`RowsAffected == 0` means someone else completed it first; the handler then does
**not** run cleanup a second time and returns success. This is FR-B20's
"racing paths produce exactly one completion" as a database predicate rather
than a lock.

- *Monsters eliminated* → `MONSTERS_ELIMINATED`. Cleanup: emit
  `EVENT_TOPIC_EVENT_VISUAL` / `HIDE` (`CONTI_MOVE(10, 5)` per F6). BGM is
  **not** restored — see §15.4.
- *Voyage arrived* → `VESSEL_ARRIVED`. Cleanup: `DESTROY_BY_SOURCE` for
  `(EVENT, occurrenceId)` in each attack map, then the same `HIDE`. Zero
  surviving monsters is success (FR-P4), so arrival-after-everything-died is the
  ordinary case, not an error path.

### 9.7 Map entry

Per G6, one more `routine.Go` block in `SpawnForSelf`:

```go
routine.Go(l, ctx, func(_ context.Context) {
    vs, err := events.NewProcessor(l, ctx).ActiveVisualsInMap(f.WorldId(), f.ChannelId(), f.MapId())
    if err != nil { l.WithError(err).Debugf(...); return }   // fail open (FR-B16)
    for _, v := range vs {
        _ = session.Announce(l)(ctx)(wp)(fieldcb.ContiMoveWriter)(fieldpkt.ContiMoveBody(v.State, v.SubState))(s)
        if v.Bgm != "" {
            _ = session.Announce(l)(ctx)(wp)(fieldcb.FieldEffectWriter)(fieldpkt.FieldEffectBackgroundMusicBody(v.Bgm))(s)
        }
    }
})
```

It is a goroutine, so it cannot block map entry; its error path only logs, so an
unreachable `atlas-events` costs the visual and nothing else (FR-B16, FR-N15).
The cabin returns no rows because its child row has `visual = false` (FR-B13).

The response type is deliberately a small `visual` projection rather than the
full occurrence resource — the channel needs four fields and should not be
coupled to the occurrence schema.

---

## 10. Event 2: Anniversary

### 10.1 Scheduling

On definition create/enable, the handler schedules one `TRIGGER_EVALUATION` at
`scheduledStart` — or **immediately**, if `scheduledStart` is past and
`scheduledEnd` is future (FR-A6). "Immediately" is `execute_at = now()`, i.e.
still a durable row picked up by the ordinary poll; there is no special
synchronous path. On occurrence creation, one `OCCURRENCE_TRANSITION` at
`scheduledEnd` (FR-A3). Both survive restart because both are rows (FR-A4).

A start whose instant passed during downtime executes late (FR-S5), and
`Evaluate` re-checks `scheduledEnd > now` — an event whose entire window elapsed
during an outage completes the work with no occurrence rather than starting a
retroactive one.

### 10.2 Login

Reaction to `EVENT_TOPIC_CHARACTER_STATUS` / `LOGIN`
(`StatusEventLoginBody{ChannelId, MapId, Instance}`), the same event and the
same handler shape `atlas-buffs` already uses for berserk tracking
(`services/atlas-buffs/atlas.com/buffs/kafka/consumer/characterstatus/consumer.go:52`).
`atlas-events` looks up an active `ANNIVERSARY` occurrence for the tenant
(index `ix_occ_type_state`) and, if present, issues a buff-apply command
(FR-A7, FR-A8).

This resolves PRD open question 2 in the "reaction alone suffices" direction, and
critically it means **login never touches `atlas-events`** — there is no
synchronous query anywhere in the login path, so `atlas-events` being down
delays the buff rather than the login. Buff propagation across channel changes
is already `atlas-buffs`' responsibility (it consumes `CHANNEL_CHANGED` on the
same topic), so no additional handling is needed here.

### 10.3 Stat selection (PRD open question 1)

**Chosen: `EXP_BUFF_RATE` for exp, `ITEM_UP_BY_ITEM` for drop, both
`ConversionDirect`.**

Evidence (all from `libs/atlas-packet/model/character_temporary_stat.go`):

| Fact | Where |
|---|---|
| `ITEM_UP_BY_ITEM` allocated unconditionally, all versions incl. v83 | `:146` |
| `EXP_BUFF_RATE` allocated unconditionally, all versions incl. v83 | `:168` |
| Both use `NoOpForeignValueWriter` — mask-only on the remote path, no extra bytes | `:146`, `:168` |
| Version gating begins only after `SoulStone`; these slots precede it | `:184` |
| Neither appears in the GMS movement-affecting set | `:778`, GMS branch `:807-827` |
| `EVENT_RATE` **does** appear in the JMS movement-affecting set | `:799` |

`EVENT_RATE` is therefore rejected: on a JMS tenant it would place the buff
inside the mask the client tests before reading the reset/give trailing byte
(`movementAffectingStatNames`' documented purpose), for no gameplay benefit over
`EXP_BUFF_RATE`. This satisfies FR-A13's "must not regress any verified packet
cell" by avoiding the interaction rather than reasoning about it.

`ITEM_UP_BY_ITEM` is additionally the only drop-rate stat in the table, and its
sibling `MESO_UP_BY_ITEM` (`:142`) is the established meso analogue — the
coupon-shaped pair.

**Conversion.** `buffToRateMappings`
(`services/atlas-rates/atlas.com/rates/kafka/message/buff/kafka.go:72-76`) gains:

```go
character.TemporaryStatTypeExpBuffRate:  {RateType: "exp",       Conversion: ConversionDirect},
character.TemporaryStatTypeItemUpByItem: {RateType: "item_drop", Conversion: ConversionDirect},
```

`ConversionDirect` is `amount / 100.0` (`:57-59`), so a configured
`expMultiplier: 2.0` is carried as `amount = 200` and read back as `2.0x`. The
mapping is exactly invertible, which is what lets FR-A1 keep the multiplier in
configuration and FR-UI8 display it. `MESO_UP` already uses `ConversionDirect`
with the same percent-of-base meaning (`:74`), so this is the established
convention, not a new one. The `item_drop` entry is the first in the table
(FR-A10) — `rate.TypeItemDrop` is already consumed by the calculator, so
nothing downstream changes.

**Unverified and explicitly out of scope:** what icon, if any, the v83 client
renders for these two slots. The client computes neither EXP nor drop rate —
both are server-side in `atlas-rates` — so a missing icon would be cosmetic. The
`GIVE_BUFF` mask position is what must be right, and that is what the table
above establishes.

### 10.4 Correlation and cancellation

The buff-apply command carries `eventOccurrenceId` as a correlation field
(FR-A12); `atlas-buffs` stores it alongside the buff and supports
cancel-by-correlation. At `scheduledEnd` the `OCCURRENCE_TRANSITION` fires,
completes the occurrence with `SCHEDULED_END` (FR-A14), and issues one
cancel-by-correlation for `eventOccurrenceId` — one command, not one per
character, so completion cost does not scale with the online population
(FR-A15). Characters logging in afterwards find no active occurrence and get
nothing (FR-A16).

---

## 11. `atlas-channel` integration

Three additions, all small:

1. **`EVENT_TOPIC_EVENT_VISUAL` consumer** — `SHOW` broadcasts
   `ContiMoveBody(state, subState)` (and `FieldEffectBackgroundMusicBody` when
   the event carries one) to the map via the existing
   `_map.NewProcessor(l, ctx).ForSessionsInMap`; `HIDE` broadcasts
   `ContiMoveBody(state, subState)` with the hide pair. Both guard on
   `sc.Is(tenant, worldId, channelId)` like every other channel consumer.
2. **`SpawnForSelf` block** — §9.7.
3. **A test pinning FR-V6** — the existing route consumer already ignores the
   new types (G2); the test stops a future edit from breaking that.

No packet work (F3, confirmed: `ContiMoveWriter` at `main.go:778`,
`FieldEffectWriter` at `:803`, both already routed).

**`atlas-configurations`:** PRD §15 says "confirm during design; if any template
lacks `ContiMove`, add it." F3 states all nine templates route it. This design
does not re-verify all nine by hand — the plan carries an explicit task to run
the repo's own template/operations check rather than trusting either statement,
because a missing route is a silent drop (the standing "missing-opcode drops"
failure mode) and the check is cheap.

---

## 12. REST API

Exactly PRD §13. Two notes:

- `GET /events/occurrences/{id}` returns transitions as a JSON:API **included**
  relationship rather than inline, matching how the rest of the repo handles
  child collections (FR-API5).
- The map-entry query gets its own narrow projection endpoint rather than reusing
  the general occurrence filter — §9.7. The general filtered list still exists
  and still satisfies FR-API6/FR-API7; the projection is an additional, cheaper
  read for the one call on the hot path.

---

## 13. Web UI

Two pages plus a detail view, following the existing flat layout in
`services/atlas-ui/src/pages/` (`XPage.tsx` + `x-columns.tsx` + `XDetailPage.tsx`,
routed from `App.tsx`, API client under `src/services/`):

- `EventDefinitionsPage.tsx` + `event-definitions-columns.tsx` — list, enable/disable toggle (FR-UI2, FR-UI3).
- `EventOccurrencesPage.tsx` + `event-occurrences-columns.tsx` — list with the FR-UI6 filters (FR-UI5).
- `EventOccurrenceDetailPage.tsx` — context, transition history, and a
  type-specific panel (FR-UI7, FR-UI8).

FR-UI4 — never render "enabled" as "occurring" — is handled by making the
definition row's second column derive from the handler's concurrency policy:
where `ConcurrencyKey` is a constant (Anniversary: at most one), the row shows
the live occurrence state; where it varies (Crimson Balrog: one per
voyage/channel), the row shows a count linking to the filtered occurrence list.
The UI reads this from a `singleOccurrence: boolean` field on the definition
resource so it does not switch on `type` — the FR-X3 rule applies to the
frontend too.

---

## 14. Alternatives considered and rejected

**A1 — Push the visual to entering characters instead of querying on entry.**
`atlas-events` already consumes `EVENT_TOPIC_CHARACTER_STATUS`; it could react to
`MAP_CHANGED` and emit a character-targeted visual event, removing the map-entry
query entirely. Genuinely attractive: no REST call on the hot path at all.
Rejected because FR-B15 specifies the query, and because the push variant makes
the visual's arrival depend on `atlas-events` consuming a Kafka event promptly —
a character who walks in during consumer lag sees nothing and has no second
chance, whereas the query is evaluated at exactly the right moment. The existing
`IsBoatInMap` precedent (G6) shows the query shape is already accepted here for
precisely this kind of per-map state.

**A2 — Leader-elected scheduler via `libs/atlas-lock`.** Rejected in §5.1: it
contradicts FR-N6 and inherits the `ATLAS_ENV` lease-scoping hazard the library's
own documentation describes at length.

**A3 — Store the voyage id in the Redis route registry.** The PRD's open
question 3 offered this or a table. Both are rejected in favor of derivation
(§7.1), which is strictly stronger: it survives a Redis flush, needs no write
path, and cannot diverge between replicas. The cost is that the derivation
inputs must be reproducible, which §7.3 handles and §17.2 tests.

**A4 — Count remaining monsters instead of tracking a set.** Rejected in §9.5:
a counter is not idempotent under redelivery and cannot tolerate
`KILLED`-before-`CREATED`.

**A5 — Extend the existing party-quest instance machinery instead of a new
service.** `atlas-party-quests` already has definitions, stages, instances, and
Kafka reactions, and the overlap is real. Rejected per PRD §2 non-goals: party
quests are party-scoped and instance-scoped, events are world/channel/global, and
merging them now would couple two domains before either's abstraction has
settled. The design deliberately *mirrors* party-quests' package and seeding
patterns so a later extraction is a move rather than a rewrite.

**A6 — Global `DESTROY_BY_SOURCE`.** §15.6.

---

## 15. PRD §19 open questions, resolved

**15.1 — Anniversary stat selection.** `EXP_BUFF_RATE` (exp) and
`ITEM_UP_BY_ITEM` (drop), both `ConversionDirect`, amount = multiplier × 100.
`EVENT_RATE` rejected. Full evidence table in §10.3.

**15.2 — Login timing.** Reaction to `EVENT_TOPIC_CHARACTER_STATUS` / `LOGIN` is
sufficient; no fallback query, and therefore no dependency of login on
`atlas-events` availability. The event already carries `channelId`/`mapId`, and
`atlas-buffs` consumes the same event for berserk tracking today (§10.2).

**15.3 — Voyage identity storage.** Neither Redis nor a table: derive it
(§7.1). Redis-flush loss stops being a question rather than being answered.

**15.4 — Music on the monsters-eliminated path.** **Leave the BGM until
arrival.** Three reasons, in order of weight: (a) per G5, `atlas-data` does not
expose Map.wz `info/bgm`, so no service can currently name the map's default —
"restore the default" would mean hard-coding a guessed string, which the
project's no-invention rule forbids; (b) the reference implementation never
restores it (PRD F6); (c) the deck is emptied minutes later at arrival anyway.
The visual **is** removed on this path (`CONTI_MOVE(10, 5)`), which is the part
players actually see. If BGM restoration is later wanted, the prerequisite is
exposing `info/bgm` through `atlas-data` — a clean, separable piece of work with
an obvious owner, and not a hidden blocker inside this task.

**15.5 — Concurrent-occurrence policy.** The generic layer enforces a
`concurrency_key` uniqueness constraint among `ACTIVE` occurrences; the handler
supplies the key (§3.2, §5.3). Anniversary returns a constant, Crimson Balrog
returns `voyage|world|channel`, a future event returning `""` opts out.
Mechanism generic, policy specific — and it is what FR-UI4's rendering decision
reads from (§13).

**15.6 — `DESTROY_BY_SOURCE` scope.** **Field-scoped.**
`Registry.GetMonstersInMap` (`monster/registry.go:376`) makes a field-scoped
sweep a single existing call; there is no index from source → monsters, so a
global variant would require a new Redis secondary index maintained on every
spawn and every death. That cost is real and buys nothing this task needs
(FR-B19 always knows its maps). Recorded here so a future GM-tooling task knows
exactly what it would have to add.

**15.7 — Scheduled work retention.** **Deferred, and safe to defer.** The
poller's index is partial on `state = 'PENDING'` (§4.1), so completed and failed
rows are not in it and the poll's cost does not grow with them — which is
precisely what FR-N16 asks for. The only cost of accumulation is disk. A
retention job is a straightforward later addition and is called out in §19.

---

## 16. Architecture validation: adding a third event (FR-20.9)

The walkthrough PRD §20.9 requires. Third event: **Mysterious Merchant** — a
travelling NPC that appears in one randomly chosen town map for a bounded window,
several times a day. It is deliberately unlike both shipped events: recurring
rather than one-shot or externally triggered, spatially scoped to a map it picks
itself rather than one it is handed, and it commands a domain (`atlas-npcs`)
neither shipped event touches.

What it needs:

1. `events/mysteriousmerchant/config.go` — `candidateMapIds`, `appearancesPerDay`,
   `duration`, `npcId`.
2. `events/mysteriousmerchant/handler.go` implementing `registry.Handler`:
   - `ValidateConfiguration` — non-empty candidates, positive duration.
   - `ConcurrencyKey` — a constant, so at most one merchant is abroad at a time.
   - `Evaluate` — pick a map, return an `occurrence.Seed` with it; also schedule
     the day's next `TRIGGER_EVALUATION`, which is how "recurring" is expressed
     without any new scheduling primitive.
   - `Start` — emit an NPC-spawn command; return `Progress` with
     `nextTransitionAt = now + duration`.
   - `Advance` — terminal; complete with `WINDOW_ELAPSED`.
   - `Complete` — emit an NPC-despawn command.
3. One line in `main.go`: `registry.Register(mysteriousmerchant.NewHandler())`.
4. A seed JSON file.
5. A new NPC command in `kafka/message/npc/` — a domain command, owned by the
   event package.

What it does **not** need, which is the actual claim: no change to
`event/definition`, `event/occurrence`, `event/transition`, `event/scheduling`,
the poller, the REST resources, the UI page structure, the database schema, or
any gameplay service other than the one it commands. The occurrence's map scope
rides `event_occurrence_map`; its recurrence rides `TRIGGER_EVALUATION`; its
window rides `OCCURRENCE_TRANSITION`; its exclusivity rides `concurrency_key`.
Each is a generic mechanism the shipped events already exercise from a different
angle.

The one thing it would add to the generic layer is a UI detail panel, and that is
by design a per-type component (§13) rather than an edit to a shared switch.

---

## 17. Testing strategy

**17.1 Scheduler.** Two `atlas-events` instances against one Postgres, N work
rows, assert each row executed exactly once — the acceptance criterion
explicitly demands a concurrent test rather than inspection (PRD §20.3). Plus:
overdue-on-startup recovery, lease reclaim after a simulated dead claimer,
retry-to-`FAILED` with `lastError` retained, and a `FAILED` row not blocking a
`PENDING` sibling.

**17.2 Voyage derivation.** Table-driven over the `Evaluate` branches, including
the midnight-crossing trip (`model.go:165-171, 207-217`): same trip → same id
across two derivations; consecutive days → different ids; the arrival-side
derivation of the departure date agreeing with the departure-side one.

**17.3 Crimson Balrog.** Each FR-B rejection path (arrived, disabled, roll
failed, nobody aboard) asserting *no occurrence row*. Cabin-counts-as-aboard.
Exactly `monsterCount` spawns, attack map only, with the provenance pair.
`KILLED`-before-`CREATED`. Both completion paths racing → one `COMPLETED`, one
reason. Arrival cleanup with zero surviving monsters.

**17.4 Boundary.** A test that walks the generic packages' ASTs and fails on any
`switch` or `if` comparing against a known event type constant (FR-X3, PRD
§20.9). Cheap, and it is the only way this property survives contact with the
next feature.

**17.5 Cross-service seams.** Per the project's "green verify.sh ≠ correct" rule,
one test per seam asserting the **new** contract: `atlas-channel`'s route
consumer ignoring the two new transport types (G2); a monster spawn without
provenance producing byte-identical output to today (FR-P5); the
`buffToRateMappings` entries producing `2.0x` end-to-end from `amount = 200`.

**17.6 Builders.** All test setup uses the project's builder pattern; no
`*_testhelpers.go` (FR-N20).

---

## 18. Risks

| Risk | Mitigation |
|---|---|
| Midnight-crossing voyage derives the wrong departure date, so `VOYAGE_ARRIVED` never matches its `VOYAGE_DEPARTED` and Balrog occurrences never complete on arrival | §17.2's table-driven test is the gate; the eliminate path still completes, so the failure degrades rather than wedges |
| New monster command fields collide with a sibling body on the fan-to-all topic (G4) | Names verified absent today; the plan restates the constraint, and the FR-P5 byte-identity test would catch a regression |
| `EXP_BUFF_RATE` / `ITEM_UP_BY_ITEM` render no client icon on v83 | Cosmetic only — rates are server-side (§10.3). Flagged as unverified rather than asserted |
| Map-entry query becomes a hot-path cost as occurrence history grows | Partial indexes on `state = 'ACTIVE'` and `visual = true` (§4.1); confirmed with `EXPLAIN`, not assumed |
| New service registration missed in an overlay | `docs/adding-a-new-service.md` checklist is a plan task in its own right; `tools/verify.sh`'s bake step covers the Dockerfile `COPY libs/...` class |

---

## 19. Explicitly out of scope

Carried forward from the PRD, plus what this design added to the list:

- Administrative termination of a live occurrence (FR-D5).
- Configuration editing beyond enable/disable (FR-API2).
- Scheduled-work retention (§15.7) — safe to defer; the poller does not degrade.
- BGM restoration on the monsters-eliminated path (§15.4) — blocked on
  `atlas-data` exposing `info/bgm`, which is separable work with a clear owner.
- Global `DESTROY_BY_SOURCE` (§15.6) — field-scoped suffices here.
- A third event. §16 is a walkthrough, not an implementation.

---

## 20. Implementation sequencing

The dependency order the plan should follow. Steps 1–3 are independent of each
other and of `atlas-events`, so they can land first and be verified on their own.

1. **`atlas-monsters` spawn provenance** (§8) — self-contained, no consumer yet.
2. **`atlas-transports` voyage identity and events** (§7) — self-contained; the
   new events have no consumer yet, and G2 means nothing downstream breaks.
3. **`atlas-rates` mappings** (§10.3) — two map entries plus tests.
4. **`atlas-events` generic core** (§3–§6) — definitions, occurrences,
   transitions, scheduler, REST, seeding. Verifiable end-to-end with no event
   implementation registered.
5. **Crimson Balrog** (§9) + the `atlas-channel` visual consumer and
   `SpawnForSelf` block (§11).
6. **Anniversary** (§10) + `atlas-buffs` correlation.
7. **Web UI** (§13).
8. **Service registration** — `docs/adding-a-new-service.md` checklist, then
   flagless `tools/verify.sh`, then the three-reviewer code-review pass
   (PRD §20.10).
