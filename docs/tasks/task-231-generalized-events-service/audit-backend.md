# Backend Audit — task-231-generalized-events-service

- **Scope:** new service `services/atlas-events` (generic core `event/{definition,occurrence,transition,scheduling,orchestration,registry}` + specific packages `events/{crimsonbalrog,anniversary}`), plus touched seams in `atlas-monsters`, `atlas-transports`, `atlas-buffs`, `atlas-rates`, `atlas-channel`.
- **Guidelines Source:** `.claude/skills/backend-dev-guidelines/resources/*`
- **Date:** 2026-08-15
- **Build:** PASS — `go build ./...` clean for all six touched modules (atlas-events, atlas-monsters, atlas-transports, atlas-buffs, atlas-rates, atlas-channel).
- **Tests:** NOT RUN by this audit — `go test` was explicitly withheld per the task's constraints (a separate `tools/verify.sh` gate was running concurrently against the same worktree). Findings below are grounded in static file:line evidence, including citations to existing `_test.go` files where they demonstrate a claim (e.g. idempotency coverage).
- **Overall:** NEEDS-WORK — build is clean and the generic/specific architectural boundary (the central design goal of this branch) holds, but several guideline violations exist, including two that reproduce a previously-documented anti-pattern (DOM-25) and one that defeats a stated FR (configuration validation bypass on the only production definition-creation path).

---

## 1. Generic core (`event/definition`, `event/occurrence`, `event/transition`, `event/scheduling`, `event/orchestration`, `event/registry`)

### 1.1 Generic/specific boundary — PASS

`grep -rn "crimsonbalrog\|anniversary" event/` returns zero hits. `event/registry/handler.go:1-11` states the AST-enforced rule (Task 39) explicitly, and every dispatch path (`event/scheduling/processor.go` `dispatch`, `event/definition/processor.go:73` `registry.Get(m.Type())`) resolves behavior only through `registry.Get`, never a type switch. The whole point of this service — a third event type needs zero edits to `event/…` — holds structurally.

### 1.2 DOM-08 / anti-pattern — FAIL: manual JSON:API envelope parsing on PATCH

`services/atlas-events/atlas.com/events/event/definition/resource.go:38` registers the PATCH route through `server.RegisterHandler`, not `server.RegisterInputHandler[T]`:
```go
router.HandleFunc("/events/definitions/{definitionId}", registerHandler("update_event_definition", updateDefinitionHandler(db))).Methods(http.MethodPatch)
```
`updateDefinitionHandler` (resource.go:155-214) then hand-rolls the JSON:API envelope:
```go
body, err := io.ReadAll(r.Body)
...
var env patchEnvelope   // type patchEnvelope struct { Data struct { Attributes map[string]interface{} `json:"attributes"` } `json:"data"` } }
if err := json.Unmarshal(body, &env); err != nil { ... }
```
This is exactly the anti-pattern named in `anti-patterns.md:27` ("Manual JSON:API envelope handling") and `ai-guidance.md`'s "Never manually decode JSON with nested structures." The stated reason (rejecting any attribute except `enabled`) does not require bypassing the standard input-handler path — a flat `UpdateRequest{Enabled *bool}` plus explicit rejection of unrecognized JSON keys via a custom `UnmarshalJSON` on the request type would satisfy FR-API2 without hand-parsing the JSON:API `data.attributes` envelope. **Severity: Important.**

### 1.3 DOM-05 — FAIL: no `TransformSlice` in any generic-layer `rest.go`

None of `event/definition/rest.go`, `event/occurrence/rest.go`, `event/transition/rest.go` define `TransformSlice`. Collection handlers use inline `model.SliceMap` loops instead:
- `event/definition/resource.go:103`: `model.SliceMap(func(m Model) (RestModel, error) { return Transform(d.Context(), m) })(...)`
- `event/occurrence/resource.go:144`: `model.SliceMap(Transform)(model.FixedProvider(paged.Items))(...)`

Per `file-responsibilities.md:109` and DOM-05, list handlers should call a `TransformSlice([]Model) ([]RestModel, error)` defined in `rest.go`. **Severity: Minor** (functionally equivalent, just not centralized/testable as a named function) — flagged in all three packages, not just one, so it is a repeated pattern deviation, not a one-off.

### 1.4 DOM-02 — deviation: `ToEntity` is a free function, not a `Model` method

`file-responsibilities.md:14` specifies `Model.ToEntity()`. Both `event/definition/entity.go:49` and `event/occurrence/entity.go:135` instead define a package-level `func ToEntity(m Model, tenantId uuid.UUID) (Entity, error)`. This is a deliberate, defensible deviation (the tenant id must come from the caller, not the immutable model), but it is a literal checklist deviation worth naming. **Severity: Minor.**

### 1.5 Anti-pattern — FAIL: database logic directly in `processor.go` for occurrence monster tracking

`event/occurrence/processor.go:219-258` (`ObserveMonsterSpawned`, `ObserveMonsterGone`, `MonsterTally`) issue `p.db.WithContext(p.ctx).Clauses(clause.OnConflict{...}).Create(...)` and `db.Model(&MonsterEntity{}).Where(...).Count(...)` **directly in the processor**, bypassing the `administrator.go`/`provider.go` split that every other write/read in this same package (`CreateFromSeed`, `ApplyProgress`, `Complete`, `GetById`, etc.) correctly uses. This is the exact anti-pattern named in `anti-patterns.md:15` ("Database logic in processors | Violates functional purity"). **Severity: Important** — three methods, not a one-off slip.

### 1.6 DOM-14 / DOM-16 — FAIL: `event/transition` has no `processor.go`/`administrator.go`; a handler calls its provider directly

`event/transition` has `model.go`, `entity.go`, `builder.go`, `provider.go`, `rest.go` — it qualifies as a domain package (has `model.go`) — but has **no `processor.go` and no `administrator.go`**. Its only write path is `occurrence`'s own `administrator.go` (`createFromSeed`, `applyProgress`, `complete`) directly persisting caller-supplied `transition.Entity` values via `tx.Create(&trans)` — i.e., the `transition` domain never owns its own writes. Its only read path is called directly from another domain's handler:

`services/atlas-events/atlas.com/events/event/occurrence/resource.go:177`:
```go
transEntities, err := transition.ByOccurrenceProvider(occurrenceId)(scopedDB)()
```
This is a handler (`getOccurrenceHandler`) calling a **provider function directly**, the anti-pattern documented at `anti-patterns.md:46-79` ("Handlers Calling Providers Directly"). The "Exception: Cross-Domain Read-Only Views with Circular Dependencies" (`anti-patterns.md:98-131`) requires a comment explaining *why* the circular dependency exists; no such comment is present here, and no circular-dependency justification is evident (there is no `transition.Processor` at all for the handler to have preferred). **Severity: Important** — this is a genuine layering gap, not merely a style nit: `transition` is architecturally a domain package with none of its required write/read encapsulation.

### 1.7 FR-D6 defeated — FAIL: the only production definition-creation path skips configuration validation

`event/definition/processor.go:70-87` (`Processor.Create`) is the ONLY place that calls `registry.Get(m.Type()).ValidateConfiguration(m.Configuration())` before a definition row is written, and its own doc comment claims: *"there is no path by which an unvalidatable definition reaches the table to fail later at trigger time."*

That claim is false. `Processor.Create` is never wired to any HTTP route — `event/definition/resource.go`'s `InitResource` (lines 31-41) registers only GET-all, GET-one, and PATCH; there is no POST route. `grep -rn "\.Create(m)\|p\.Create\b" event/definition/*_test.go` shows `Create` is exercised **only by tests**. The one production path that creates definition rows is `event/definition/subdomain.go`'s `BulkCreate` (lines 58-75, wired to `InitSeedResource` in `main.go:123`), which builds the entity directly:
```go
func (DefinitionSubdomain) BulkCreate(db *gorm.DB, models []Model) error {
    ...
    entity, err := ToEntity(m, tenantId)
    ...
    if result := db.Create(&entity); result.Error != nil { ... }
}
```
No `registry.Get(...).ValidateConfiguration(...)` call anywhere in `subdomain.go`. Every event definition that reaches the database in production goes through the seed-loader path and is written with **zero configuration validation**, directly contradicting FR-D6 and the processor's own stated invariant. **Severity: Important** — this is a real defect in the feature's central safety property, not a style issue.

### 1.8 DOM-10 — FAIL (build-tag-gated, not in default gate): integration test opens Postgres without registering tenant callbacks

`event/occurrence/concurrency_key_integration_test.go:56` (`//go:build integration`):
```go
db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
```
No `database.RegisterTenantCallbacks(l, db)` call follows, unlike every other test in the package (which use the shared `databasetest.NewInMemoryTenantDB` helper — confirmed to call `RegisterTenantCallbacks` internally at `libs/atlas-database/databasetest/testdb.go:39`). Because this test's own assertions rely on a raw Postgres unique-index behavior rather than tenant-scoped reads, it likely still passes, but any query added later against this `db` handle would silently skip tenant filtering. **Severity: Minor** (excluded from the default `go test ./...` gate by its build tag, but still a guideline gap per `patterns-multitenancy-context.md`'s Test Setup section).

### 1.9 Everything else in the generic core — PASS

Verified independently and via a dedicated sub-audit (kafka consumers, scheduling, orchestration, registry, main.go, tenant/, external/):
- Kafka consumer idempotency: `event/scheduling/administrator.go` dedupe-key + `ON CONFLICT DO NOTHING`; `event/occurrence/processor.go:219-239` upsert-based `ObserveMonsterSpawned`/`ObserveMonsterGone`; `event/occurrence/administrator.go`'s guarded `WHERE state='ACTIVE'` completion updates. All correctly idempotent under redelivery.
- DOM-24 (Kafka producer stubbed in tests): not applicable — no test in `event/scheduling`, `event/orchestration`, `event/registry`, `kafka/`, `rest/`, `tenant/`, `external/` exercises an emit path.
- DOM-26 (bare `go` statements): zero hits outside `_test.go`; `main.go:113` and `event/scheduling/poller.go:178-180` both use `routine.Go(l, ctx, fn)`.
- DOM-21 (atlas-constants reuse): `world.Id`, `channel.Id`, `_map.Id` used correctly throughout the generic core and its Kafka envelopes.
- Multi-tenancy: `event/scheduling`'s poller correctly uses `database.WithoutTenantFilter` only for its cross-tenant claim/reclaim queries (documented exception), rebuilding a per-row tenant context before any handler/write runs (`processor.go:196-199`).
- `main.go`: `RegisterTenantCallbacks` correctly absent (relies on `database.Connect`); `server.RegisterTransientErrorClassifier` correctly registered (DOM-27) at `main.go:86-92`.
- No stub/`// TODO`/501 found anywhere in the generic core.

---

## 2. Specific event-type packages (`events/crimsonbalrog`, `events/anniversary`)

### 2.1 DOM-21 — FAIL: `MonsterId` reinvents `monster.Id` as a bare `uint32`

`libs/atlas-constants/monster/constants.go:3` defines `type Id uint32` for exactly this purpose (README: *"Monster IDs and per-monster status flags"*). CRIMSON_BALROG instead uses raw `uint32` throughout:
- `events/crimsonbalrog/config.go:62` — `Config.MonsterId uint32`
- `events/crimsonbalrog/evaluate.go:35` — `OccurrenceContext.MonsterId uint32`
- `events/crimsonbalrog/producer.go:63` — `spawnFieldCommandProvider(..., monsterId uint32, ...)`

**Severity: Important** (DOM-21 is explicitly weighted for this audit per the dispatch brief).

### 2.2 DOM-25 / anti-pattern — FAIL: client wire bytes (`state`/`subState`) flow unresolved from atlas-events config through atlas-channel

`libs/atlas-packet/field/clientbound/conti_move.go:14-29` documents `ContiMove`'s wire layout as a dispatcher: `Decode1(state) selects one of six arms via (state-7)` — precisely the class of "byte the game client feeds through one of its own lookup switches" that `anti-patterns.md:135-164` requires be resolved from a tenant writer-options table at encode time, never carried as a literal, and explicitly rejects the "IDA-verified version-stable" excuse that `conti_move.go`'s own comment supplies (*"Read order verified... found identical in every version checked"*).

This branch introduces exactly that anti-pattern on a new seam:
- `services/atlas-events/atlas.com/events/events/crimsonbalrog/config.go:40-46` — `VisualConfig.ShowState/ShowSubState/HideState/HideSubState byte` lets a seed-data JSON author pick raw wire bytes.
- `services/atlas-events/atlas.com/events/kafka/message/event/kafka.go` — `ShowVisualBody.State/SubState`/`HideVisualBody.State/SubState` carry those bytes verbatim over Kafka as "gameplay content" (per `events/crimsonbalrog/start.go`'s own doc comment, "atlas-events constructs no packets here... the state/subState bytes ride on the event as gameplay content").
- `services/atlas-channel/atlas.com/channel/kafka/consumer/event/consumer.go:87-107,113-129` (`handleVisualShow`/`handleVisualHide`) pass `e.Body.State`/`e.Body.SubState` straight into `writer.ContiMoveBody(state, subState)` with **no per-tenant/per-version writer-options resolution** — the exact opposite of the `WithResolvedCode(...)` pattern the guideline requires.

Per DOM-25(b), this also requires the value to be seeded per-version in every supported client's template — it is not; it is a free-form config field on the event definition. **Severity: Important** — this reproduces a previously-litigated, explicitly-documented anti-pattern (task-102/task-103 uniformity ruling) on a brand-new cross-service seam.

### 2.3 FILE-01 — Minor: `events/anniversary` scatters Kafka producer functions outside a `producer.go`

Unlike `events/crimsonbalrog` (which isolates all Kafka message construction in `producer.go`), `events/anniversary` defines `cancelByCorrelationCommandProvider` inline in `handler.go:146-153` and `applyCommandProvider` inline in `login.go:83-102`. **Severity: Minor** — a file-responsibility inconsistency, not the severe "Processor+RestModel+requests in one file" collapse the checklist warns most strongly against.

### 2.4 Logger anti-pattern — construction-time `logrus.StandardLogger()` instead of a threaded field logger

Both concrete-type handlers bake in the global standard logger at construction time rather than accepting a per-call/tenant-scoped `logrus.FieldLogger`:
- `events/crimsonbalrog/handler.go:34,37,39`:
```go
transports: func(ctx context.Context) transports.Processor {
    return transports.NewProcessor(logrus.StandardLogger(), ctx)
},
maps: func(ctx context.Context) maps.Processor {
    return maps.NewProcessor(logrus.StandardLogger(), ctx)
},
l: logrus.StandardLogger(),
```
- `events/anniversary/handler.go:37`: `return &Handler{db: db, l: logrus.StandardLogger()}`

`h.l` then flows into `message.Emit(h.l, ctx)` at `events/crimsonbalrog/start.go:25` and equivalent call sites. Because `NewHandler()` runs exactly once at process startup (`main.go:71,84`), every subsequent `Start`/`Advance`/`OnLogin` call emits through the same static global logger rather than the field logger the calling Kafka consumer/scheduling dispatch loop actually has scoped to that row's request/trace. This does not violate a literal DOM-* line item (those target `resource.go`), but it is the same defect class — `ai-guidance.md`'s REST-generation guidance forbids `logrus.StandardLogger()` for exactly this reason (log correlation). **Severity: Minor.**

### 2.5 Everything else in the specific packages — PASS

Generic/specific boundary: neither package reimplements occurrence persistence, transition history, definition CRUD, or a scheduling/poller loop — both call into `occurrence.NewProcessor`, `scheduling.NewBuilder`/`NewAdministrator`, `definition.NewProcessor`. Idempotency under redelivery is deliberately engineered and test-covered: `crimsonbalrog/monsters_test.go` `TestRedeliveredFinalKillIsANoOp`, `crimsonbalrog/arrival_test.go` `TestArrivalAfterEliminationIsANoOp`, `crimsonbalrog/trigger_test.go` `TestVoyageDepartedRedeliveryIsANoOp`. No stub/TODO/501. Tenant scoping correct throughout.

---

## 3. Touched sibling services (`atlas-monsters`, `atlas-transports`, `atlas-buffs`, `atlas-rates`, `atlas-channel`) — diff-only review

All new/changed Kafka schema fields are additive and `omitempty` (spawn-provenance fields on `atlas-monsters`' `spawnFieldCommandBody`/`statusEvent[E]`, new `VOYAGE_DEPARTED`/`VOYAGE_ARRIVED` event types on `atlas-transports`, `CorrelationId` + `CancelByCorrelationCommandBody` on `atlas-buffs`' character command envelope, two new `buffToRateMappings` entries on `atlas-rates` using existing `atlas-constants` stat types) — no consumer is orphaned, no existing wire shape narrows.

New consumer logic is idempotent by construction: `atlas-monsters/monster/kafka.go`'s `normalizeSpawnSourceType` treats absent/empty provenance as the legacy `CYCLIC` value (preserving old-producer behavior), and the new `DestroyBySource`/`CancelByCorrelation` handlers are filter-and-clear operations that are no-ops on a second delivery. `atlas-transports/transport/model.go`'s `materializeDeparture` change (midnight-crossing voyage-id derivation fix) is a genuine correctness fix with no atlas-constants duplication.

`atlas-channel`'s new `events/` package (REST client for the atlas-events map-visuals projection) correctly implements the External HTTP Client Checklist: `events/requests.go:15-16` uses `requests.RootUrl("EVENTS")`; `events/rest.go` implements `GetName`/`GetID`/`SetID` and the required `SetToOneReferenceID`/`SetToManyReferenceIDs` no-ops for api2go relationship handling; `events/processor.go:36-39` documents fail-open behavior (an unreachable atlas-events costs the visual, never map entry) per FR-B16/FR-N15. Its Kafka consumer (`kafka/consumer/event/consumer.go`) is where the DOM-25 finding above (§2.2) actually surfaces on the receiving end.

No handler-bypasses-processor regression, no new stub/TODO, and tenant scoping is preserved in all changed queries/emits (`atlas-buffs`' new `CancelByCorrelation` derives `WorldId` per-character from the tenant-scoped registry rather than trusting the command envelope).

---

## 4. Known, deliberately-parked item (not re-litigated)

`event/occurrence/rest.go`'s `RestModel` does not serialize `worldId`/`channelId` even though `Model`/`Entity` carry them (`occurrence/model.go:57-58`, `occurrence/entity.go:26-27`). This is a known wire-contract decision awaiting the user, per the audit brief — no additional finding recorded here. No other REST-shape gap of this kind was found in `event/definition` or `event/transition`'s wire models.

---

## Summary

### Blocking / Important (should fix before merge)

- **DOM-08** — `event/definition/resource.go:38,155-214`: PATCH handler manually parses the JSON:API envelope instead of `RegisterInputHandler[T]`.
- **FR-D6 validation bypass** — `event/definition/subdomain.go:58-75` (`BulkCreate`): the only production definition-creation path (seed loader) skips `registry.Get(...).ValidateConfiguration(...)`, contradicting the processor's own documented invariant. `event/definition/processor.go`'s `Create` (which does validate) is dead code — never wired to a route.
- **Anti-pattern (database logic in processor)** — `event/occurrence/processor.go:219-258`: `ObserveMonsterSpawned`/`ObserveMonsterGone`/`MonsterTally` issue GORM calls directly instead of going through `administrator.go`/`provider.go`.
- **DOM-14 / DOM-16** — `event/occurrence/resource.go:177` calls `transition.ByOccurrenceProvider` (a provider) directly from a handler; `event/transition` has no `processor.go`/`administrator.go` at all despite qualifying as a domain package.
- **DOM-21** — `events/crimsonbalrog/config.go:62`, `evaluate.go:35`, `producer.go:63`: `MonsterId uint32` should be `monster.Id` from `libs/atlas-constants/monster`.
- **DOM-25 / anti-pattern** — `events/crimsonbalrog/config.go:40-46` → `kafka/message/event/kafka.go` → `atlas-channel/kafka/consumer/event/consumer.go`: `ContiMove` `state`/`subState` client wire bytes flow as free-form config, never resolved through a tenant writer-options table, reproducing the exact anti-pattern documented from task-102/task-103.

### Non-Blocking (should fix)

- **DOM-05** — no `TransformSlice` in `event/definition`, `event/occurrence`, or `event/transition`'s `rest.go`; list handlers use inline `model.SliceMap`.
- **DOM-02 deviation** — `ToEntity` is a free function (`ToEntity(m Model, tenantId uuid.UUID)`), not a `Model` method, in `event/definition` and `event/occurrence`.
- **FILE-01** — `events/anniversary` lacks a `producer.go`; Kafka message construction is inline in `handler.go`/`login.go`.
- **Logger anti-pattern** — `events/crimsonbalrog/handler.go:34,37,39` and `events/anniversary/handler.go:37` bake in `logrus.StandardLogger()` at startup instead of a per-call field logger.
- **DOM-10** — `event/occurrence/concurrency_key_integration_test.go:56` opens Postgres directly without `database.RegisterTenantCallbacks` (build-tag `integration`, excluded from default gate).

### Verified PASS (high-confidence, not re-litigated)

Generic/specific architectural boundary; Kafka consumer idempotency across scheduling, occurrence monster-tracking, and both event-type packages (test-covered); DOM-24 (no unstubbed producer test paths); DOM-26 (no bare `go` statements); multi-tenancy scoping including the scheduling poller's documented cross-tenant exception; `main.go` wiring (transient-error classifier, no misplaced `RegisterTenantCallbacks`); backward-compatible Kafka schema evolution across all five touched sibling services; External HTTP Client Checklist compliance in `atlas-channel`'s new `events` package; no stubs/TODOs/501s anywhere in the diff.
