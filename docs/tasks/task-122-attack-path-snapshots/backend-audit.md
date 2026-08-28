# Backend Audit — task-122-attack-path-snapshots

- **Scope:** changed Go packages in `services/atlas-channel/atlas.com/channel` and `services/atlas-character/atlas.com/character`, merge-base `2537d8a6a` → HEAD `74d07bbbf`
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-27
- **Build:** PASS (both modules)
- **Tests:** PASS (both modules), 0 failed
- **Overall:** NEEDS-WORK

## Build & Test Results

```
cd services/atlas-channel/atlas.com/channel && go build ./...     -> exit 0, no output
cd services/atlas-channel/atlas.com/channel && go test ./... -count=1  -> all `ok`, no FAIL
cd services/atlas-character/atlas.com/character && go build ./... -> exit 0, no output
cd services/atlas-character/atlas.com/character && go test ./... -count=1 -> all `ok`, no FAIL (character/pending_change 230s, character/kafka/consumer/character 11-12s — pre-existing slow suites, unrelated to this diff)
```

`tools/goroutine-guard.sh` (flagless): exit 0 — `goroutineguard: 91 module(s), 8 parallel` self-test passed, no unguarded bare `go`.

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| FILE-01..06 | Yes | Every changed package audited regardless of classification (no exemption clause). |
| DOM structure (DOM-01..05,11,16) | Yes | `character/skill/builder.go` new (package has pre-existing `model.go`); no `entity.go`/`provider.go`/`administrator.go` touched. |
| SUB-01..04 | N/A | No changed package has `resource.go` without `model.go`. |
| REST (DOM-06..09,12..15,17..19,32) | N/A | No changed package has `resource.go`/registers HTTP routes; `processor.go` files present are non-REST (`snapshot`, `data/skill`) — see DOM-06 note below, evaluated under FILE-01 instead since these processors aren't REST-fronted. |
| Constants reuse (DOM-21) | Yes | New consts (`componentCore` etc., cache TTL bounds, `liveMirrorSweepInterval`) checked against `libs/atlas-constants` — none duplicate a shared constant. |
| Testing (DOM-10,20,24,33) | Yes | Diff adds/changes many `_test.go` files; `stat_values_test.go` opens a GORM DB directly. |
| Cache (DOM-29) | Yes | `data/skill/cache.go` new; `character/snapshot/registry.go` and `monster/live_mirror.go` hold cached/mirrored state. |
| Messaging (DOM-30) | N/A | No changed package emits Kafka messages via `AndEmit`/`message.Emit`/`producer.ProviderImpl` from a new write path (all snapshot mutators are pure in-memory projections fed by consumers; the character-service diff only enriches existing `statChangedProvider` payloads, an existing emit call site, unchanged shape). |
| Multi-tenancy (DOM-31) | Yes | Every registry/cache/mirror mutator takes `tenant.Model` from `tenant.MustFromContext(ctx)`; no REST model, request body, or query parameter carries a tenant id. |
| Migration hygiene (DOM-34,35) | N/A | No symbols moved between a service and `libs/atlas-*`. |
| Deploy & topics (DOM-22,23) | N/A | No `libs/atlas-*` module added; `skill/kafka.go` adds a new event *type* (`DELETED`) on an existing topic, not a new topic env var. |
| Runtime safety (DOM-26) | Yes | Every non-test file changed is audited; one bare `go` found (`monster/live_mirror.go:60`), carries a `//goroutine-guard:allow` marker. |
| Channel wire values (DOM-25) | Yes (touches atlas-channel) | No new client-interpreted byte/opcode literals added; all snapshot/mirror code is server-side state, no new `.Encode`/writer byte literal introduced. |
| Resilience (DOM-27,28) | N/A | No `model.Decorator` changed; snapshot fallback paths are new code, not the enrichment/decorator pattern DOM-27/28 target (DB-backed handler error branches also N/A — no changed `resource.go`). |
| External clients (EXT-01..04) | Yes | `character/snapshot/processor.go`/`shadow.go` call existing `character.NewProcessor`/`inventory.NewProcessor`/`skill.NewProcessor`/`buff.NewProcessor` (unchanged `GetById`/`GetByCharacterId`, unchanged `requests.go`); `data/skill/cache.go` wraps the pre-existing `requests.Provider` call in `data/skill/processor.go` (unchanged `requests.go`). No new `requests.*Request[T]` call site introduced. |
| Scaffolding (SCAFFOLD-01..09) | N/A | No new service directory, writer/handler registration is to existing consumers, `routes.conf` untouched. |
| Security (SEC-01..04) | N/A | Neither service handles auth/tokens/redirects/secrets in this diff. |
| `patterns-provider.md` (foundational) | Yes | `snapshot.Processor.BuffsProvider` composes `model.Provider[[]buff.Model]`. |
| `patterns-functional.md` (foundational) | Yes | `applyStat`, curried consumer handler builders (`func(sc, wp) func(l, ctx, e)`) throughout. |

## Checklist Results

### character/snapshot (support — no `model.go`; holds cached/projected state)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor in `processor.go` | PASS | `character/snapshot/processor.go:39-45` — `Processor` struct + `NewProcessor` only there. |
| FILE-06 | No catch-all file with ≥2 responsibilities | PASS | `registry.go`, `processor.go`, `shadow.go`, `metrics.go` each single-purpose. |
| DOM-26 | Goroutines via `routine.Go` | PASS | `shadow.go:73` uses `routine.Go(l, ctx, fn)`. |
| DOM-29 | Singleton cache/registry via accessor, not per-instance | PASS | `registry.go:57-67` `sync.Once` + `GetRegistry()`; `processor.go:56` calls `GetRegistry()`, never constructs its own. |
| DOM-31 | Tenant travels in context only | PASS | Every `Registry` method takes `tenant.Model` sourced from `tenant.MustFromContext(ctx)` at call sites (e.g. `kafka/consumer/character/consumer.go:561`), never a REST/body/query field — package has no `rest.go`. |
| Immutability (foundational, `patterns-functional.md`) | Component models are not mutated in place | PASS | `registry.go:184-189` `BackfillSkills`/`UpsertSkill` copy the slice (`append([]skill.Model(nil), ms...)`, `out := make(...)`) before storing; `replaceCompartment` (`registry.go:455-465`) rebuilds via a fresh `inventory.NewBuilder` specifically because `CloneModel` shares the compartments map (comment explains the aliasing hazard it avoids); `TestRegistry_AssetMutationDoesNotAliasPriorReads` (`registry_test.go:293`) asserts this. |
| DOM-20 | Table-driven tests | **FAIL** | `character/snapshot/processor_test.go` (0 `[]struct{`/`t.Run`) and `character/snapshot/shadow_test.go` (0 `[]struct{`/`t.Run`) and `character/snapshot/registry_test.go` (17 `func Test...`, only 1 uses `t.Run`/table shape) are all-new files with no table-driven pattern; no packet-fixture playbook applies here to substitute. |
| DOM-24 | Producer stub installed where an emit path is reached | N/A | Neither `processor_test.go` nor `shadow_test.go` reaches `AndEmit`/`message.Emit`/`producer.ProviderImpl` — `maybeShadow`/`shadowCompare` only issue REST fetches, never Kafka emits. |
| DOM-01 (via `character/skill`, called from snapshot) | see below | — | see `character/skill` section |

### character/skill (domain — has `model.go`; `builder.go` newly added)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` has `NewBuilder()`, fluent setters, validating `Build()` | **FAIL** | `character/skill/builder.go:21-23` defines only `NewModelBuilder(id skillconst.Id) *modelBuilder` — no `NewBuilder()` constructor exists anywhere in the package. `Build()`/`MustBuild()`/setters are otherwise present and validate (`builder.go:48-51` rejects `id == 0`). |
| FILE-05 | Builder in `builder.go` | PASS | Confirmed above — placement is correct, only the constructor name deviates from the rule's named symbol. |

### data/skill (support — no `model.go`; new `cache.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-29 | Singleton cache via `sync.Once` + accessor | PASS | `cache.go:104-117` `skillCacheOnce.Do` + `getSkillCache()`; `processor.go:31-33` `ProcessorImpl.GetById` calls the package function `getByIdCached`, never builds its own cache. |
| DOM-31 | Tenant from context only | PASS | `getByIdCached` (`cache.go:177-201`) derives `t := tenant.MustFromContext(ctx)`; `perTenant map[uuid.UUID]map[uint32]cacheEntry` keyed by `t.Id()` (`cache.go:101,137,151`); `EvictTenant` (`cache.go:169-174`) drops one tenant's map on tenant drain, wired from `main.go` (`main.go` diff: `dataskill.EvictTenant(tid)`). |
| EXT-03 | Only genuine 404 → domain not-found; other errors bubble | PASS | `cache.go:193-198` — negative-caches only `errors.Is(err, requests.ErrNotFound)`; any other error (`transport`/decode/5xx) is returned unmodified and uncached. |
| DOM-20 | Table-driven tests | **FAIL** | `data/skill/cache_test.go` has 0 `[]struct{`/`t.Run` occurrences — all-new file, no table-driven tests. |
| FILE-06 | No catch-all file | PASS | `cache.go` is single-purpose; the deleted `registry.go`/`registry_test.go` (replaced by `cache.go`) confirms a clean 1:1 swap, not a merge into an existing catch-all. |

### monster (live mirror addition, package pre-existing)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-26 | Bare `go` justified | PASS | `monster/live_mirror.go:59-60` — `//goroutine-guard:allow` marker present with rationale (process-lifetime sweeper on a `sync.Once` singleton); `tools/goroutine-guard.sh` exits 0. |
| DOM-31 | Tenant from context, keyed per tenant | PASS | `LiveMirror.perTenant map[uuid.UUID]map[uint32]LiveEntry` (`live_mirror.go:44-46`); every method takes `tenant.Model`; `EvictTenant` wired from `main.go`. |
| DOM-29 | Singleton via accessor | PASS | `live_mirror.go:49-63` `sync.Once` + `GetLiveMirror()`. |
| DOM-20 | Table-driven tests | PASS | `monster/live_mirror_test.go` uses `[]struct{` + `t.Run` (1 occurrence covering the sweep/eviction cases). |

### kafka/consumer/{character,asset,buff,compartment,skill} (sub-domain, action-event consumers)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-31 | Tenant scoping before every snapshot mutation | PASS | Every new `handleSnapshot*` handler gates on `t := tenant.MustFromContext(ctx)` then `sc.IsWorld(t, ...)`/`t.Is(sc.Tenant())` before calling into `snapshot.GetRegistry()` — e.g. `kafka/consumer/character/consumer.go:561-566`, `kafka/consumer/asset/consumer.go:623-627`, `kafka/consumer/buff/consumer.go:641-651`, `kafka/consumer/compartment/consumer.go:217-221`, `kafka/consumer/skill/consumer.go:205-209`. |
| DOM-24 | Producer stub where emit reached | N/A | New `handleSnapshot*` tests (`buff/consumer_test.go:41-116`, `character/consumer_test.go:49-168`, `asset/consumer_test.go:294-431`, `compartment/consumer_test.go`, `skill/consumer_test.go`) call the snapshot-projection handlers with `wp = nil`; none of these code paths reach a Kafka emit (registry mutation only). |
| DOM-20 | Table-driven tests | **FAIL** (buff, character, skill) / PASS (asset, compartment) | `kafka/consumer/buff/consumer_test.go`, `kafka/consumer/character/consumer_test.go`, `kafka/consumer/skill/consumer_test.go` have 0 `[]struct{`/`t.Run`; `kafka/consumer/asset/consumer_test.go:41-84` and `kafka/consumer/compartment/consumer_test.go` do use the table-driven shape. |
| DOM-21 | New event type/body against `libs/atlas-constants` | PASS | `kafka/message/skill/kafka.go` adds `StatusEventTypeDeleted`/`StatusEventDeletedBody` — string literal + empty struct, not a redeclaration of any `libs/atlas-constants` symbol. |

### movement (processor.go changes)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-26 | Goroutines via `routine.Go`, no new bare `go` | PASS | All new/changed goroutine spawns keep `routine.Go(p.l, p.ctx, func(_ context.Context) {...})` (`movement/processor.go` diff, unchanged call shape). |
| DOM-31 | Tenant from `p.t`, not request data | PASS | `snapshot.GetRegistry().SetPosition(p.t, characterId, ms.X, ms.Y)` and `monster.GetLiveMirror().UpdatePosition(p.t, objectId, ms.X, ms.Y)` — `p.t` is the processor's tenant, sourced from context at construction. |
| DOM-20 | Table-driven tests | **FAIL** | `movement/processor_test.go` and `movement/teleport_test.go` have 0 `[]struct{`/`t.Run`. |

### socket/handler (character_attack_common.go)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-31 | Tenant sourced from context for monster resolvers | PASS | `buildMonsterResolver`/`buildMonsterModelResolver` (`character_attack_common.go:107-155`) take `t tenant.Model` passed down from the handler's `tenant.MustFromContext(ctx)`-derived `t`, never re-derived from request data. |
| Immutability | Per-swing resolver memoization does not mutate shared state | PASS | `resolved := make(map[uint32]monster.LiveEntry)` / `map[uint32]monster.Model` (`character_attack_common.go:112,141`) are local to one `buildMonsterResolver`/`buildMonsterModelResolver` call, captured by the closure only — not shared across swings or goroutines (comment at `character_attack_common.go:110-111` states "Not goroutine-safe — processAttack runs single-goroutine per packet"). |
| DOM-20 | Table-driven tests | PASS (mixed) | `character_attack_common_test.go`, `character_attack_drain_test.go`, `character_attack_mortal_blow_test.go` all use `[]struct{`/`t.Run`. |

### atlas-character/character (processor.go — absolute Values on STAT_CHANGED)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-31 | No tenant/trace leakage introduced | PASS | Changes only populate `map[string]interface{}` value payloads already carried on the existing `statChangedProvider` call; no new field added to any REST model or request body. |
| DOM-33 | Mocks updated for any interface change | N/A | No `Processor`/`Provider`/`Administrator` interface method added, removed, or re-signed — `ChangeJob`, `ChangeHair`, etc. keep their existing signatures; only their internal `statChangedProvider(...)` call's `values` argument changed from `nil` to a populated map. |
| DOM-10 | Test DB setup calls `database.RegisterTenantCallbacks` | Not evaluable from the diff | `stat_values_test.go` reuses `databasetest.NewInMemoryTenantDB` (a shared library helper, not defined in this diff); whether that helper itself calls `database.RegisterTenantCallbacks` is in `libs/atlas-database/databasetest`, outside this diff's changed-file surface. |
| DOM-20 | Table-driven tests | **FAIL** | `stat_values_test.go` has 0 `[]struct{`/`t.Run` — all-new file. |

## Known/accepted items (per task brief, not re-litigated as findings)

- `tools/verify.sh` (flagless) passes at HEAD — confirmed out of scope for this audit per brief; independently, the narrower gates run here (`go build`, `go test`, `tools/goroutine-guard.sh`) all pass.
- Four attack-path REST reads remain by disclosed design (`docs/tasks/task-122-attack-path-snapshots/context.md`) — not re-flagged.
- `character/snapshot/shadow.go:73` — `routine.Go(l, ctx, func(ctx context.Context) {` — every other `routine.Go` call site in the service (39 call sites checked) uses `func(_ context.Context)`; this is the sole named-parameter deviation introduced by this branch. **Ruling: non-blocking style deviation** — the parameter is unused in the closure body exactly like the `_` sites, so there is no functional difference, only an inconsistent naming convention. Recorded here per instruction, not re-discovered as new.
- `character/snapshot/shadow.go:105` — the buffs-skip-comparison note (`l.WithField("component", componentBuffs).Debug(...)`) logs at Debug level, so a real-world "shadow never got real served buffs" condition would not surface in a production logrus config running at Info or above. **Ruling: non-blocking** — shadow verification is itself an opt-in, env-gated diagnostic path (default rate 0, `shadow.go:25`), so a silent-in-prod Debug log narrows an already-optional diagnostic rather than hiding a user-facing defect. Recorded here per instruction, not re-discovered as new.

## Not evaluable from the diff

- DOM-10 (`character/stat_values_test.go`): `newStatValuesFixture` calls `databasetest.NewInMemoryTenantDB(t, Migration, outbox.Migration)`, a `libs/atlas-database/databasetest` helper not touched by this diff. Confirming it internally calls `database.RegisterTenantCallbacks(l, db)` requires reading that library file, which is outside the changed-file surface for this review.

## Summary

### Blocking (must fix)

- DOM-01: `services/atlas-channel/atlas.com/channel/character/skill/builder.go:21` — new `builder.go` defines only `NewModelBuilder(id)`, no `NewBuilder()` constructor as the rule requires.
- DOM-20 (table-driven tests, widespread in this diff — grading each new/changed test file independently, not curved by the files that do comply):
  - `services/atlas-channel/atlas.com/channel/character/snapshot/processor_test.go` — no table-driven tests.
  - `services/atlas-channel/atlas.com/channel/character/snapshot/shadow_test.go` — no table-driven tests.
  - `services/atlas-channel/atlas.com/channel/character/snapshot/registry_test.go` — 17 test functions, only 1 uses the table-driven shape.
  - `services/atlas-channel/atlas.com/channel/data/skill/cache_test.go` — no table-driven tests.
  - `services/atlas-channel/atlas.com/channel/kafka/consumer/buff/consumer_test.go` — no table-driven tests.
  - `services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer_test.go` — no table-driven tests.
  - `services/atlas-channel/atlas.com/channel/kafka/consumer/skill/consumer_test.go` — no table-driven tests.
  - `services/atlas-channel/atlas.com/channel/movement/processor_test.go` — no table-driven tests.
  - `services/atlas-channel/atlas.com/channel/movement/teleport_test.go` — no table-driven tests.
  - `services/atlas-character/atlas.com/character/character/stat_values_test.go` — no table-driven tests.

### Non-Blocking (should fix)

- `character/snapshot/shadow.go:73` — `func(ctx context.Context)` instead of the service-wide `func(_ context.Context)` convention (unused parameter, cosmetic only).
- `character/snapshot/shadow.go:105` — buffs-skip note logs at Debug; would not surface in a production logrus config above Debug level for this already-optional, env-gated diagnostic path.

### not_evaluable

- DOM-10: `stat_values_test.go`'s DB bootstrap delegates to a `libs/atlas-database/databasetest` helper outside the changed-file surface.
