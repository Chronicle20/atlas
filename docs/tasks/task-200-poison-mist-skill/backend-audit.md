# Backend Audit — task-200-poison-mist-skill (Go diff vs origin/main)

- **Scope:** Go files changed by this branch (`git diff --name-only origin/main...HEAD -- '*.go'`), across `atlas-data`, `atlas-channel`, `atlas-maps`, `atlas-monsters`, `libs/atlas-packet`.
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-08-07
- **Build:** PASS (all five modules: atlas-channel, atlas-maps, atlas-monsters, atlas-data, libs/atlas-packet)
- **Tests:** All packages pass, `-count=1`, no failures in any of the five modules
- **Overall:** NEEDS-WORK (two Important findings, two Minor findings; no build/test failures)

## Build & Test Results

```
services/atlas-channel/atlas.com/channel: go build ./... -> clean; go test ./... -count=1 -> all ok
services/atlas-maps/atlas.com/maps:       go build ./... -> clean; go test ./... -count=1 -> all ok
services/atlas-monsters/atlas.com/monsters: go build ./... -> clean; go test ./... -count=1 -> all ok
services/atlas-data/atlas.com/data:       go build ./... -> clean; go test ./... -count=1 -> all ok
libs/atlas-packet:                        go build ./... -> clean; go test ./... -count=1 -> all ok
```

`tools/goroutine-guard.sh` from repo root: exit 0 (no bare `go` statements anywhere in the tree).

## Changed-file inventory

```
libs/atlas-packet/field/clientbound/affected_area_test.go
services/atlas-channel/.../data/skill/effect/model.go, rest.go, rest_dot_test.go
services/atlas-channel/.../kafka/consumer/mist/consumer.go, consumer_test.go
services/atlas-channel/.../kafka/message/mist/kafka.go
services/atlas-channel/.../mist/processor.go, producer.go, producer_test.go
services/atlas-channel/.../skill/handler/poisonmist/poisonmist.go, poisonmist_test.go
services/atlas-channel/.../skill/handler/registrations/registrations.go, registrations_test.go
services/atlas-data/.../skill/effect/model.go, rest.go
services/atlas-data/.../skill/reader.go, reader_test.go
services/atlas-maps/.../kafka/message/mist/kafka.go
services/atlas-maps/.../map/monster/processor_test.go
services/atlas-maps/.../mist/model.go, model_test.go, processor.go, processor_test.go, producer.go, producer_test.go
services/atlas-maps/.../monster/processor.go, processor_rect_test.go, requests.go
services/atlas-maps/.../tasks/mist_tick.go, mist_tick_monster_test.go
services/atlas-monsters/.../kafka/message/mist/kafka.go
services/atlas-monsters/.../monster/processor.go, processor_test.go
```

## Domain / Package Checklist Results

### `atlas-maps/mist` (domain package — `model.go` present; in-memory registry, no GORM persistence)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` exists with `Build()` validation | **FAIL (Minor)** | `Builder`/`NewBuilder`/`Build()` live in `mist/model.go:231-377`; `Build()` (`model.go:349-377`) performs zero invariant checks (no duration>0, no rect-shape check). The only validation for a player-cast mist lives in the caller (`atlas-channel/skill/handler/poisonmist/poisonmist.go:111-126`), so any other producer of `CreateCommandBody` (e.g. a future caller, or a malformed Kafka message) sails through `Builder.Build()` unchecked. |
| DOM-02/DOM-03 | `ToEntity()` / `Make(Entity)` | N/A | Package has no `entity.go` — `Mist` is an in-memory, tenant-scoped registry value (`mist/registry.go`), never persisted to GORM. No file-responsibilities exception is needed because there is no DB layer to check against; the domain simply isn't GORM-backed (same shape as `atlas-channel/session`, `atlas-channel/macro`, etc.). |
| DOM-04/05 | `Transform`/`TransformSlice` | N/A | Package has no `rest.go` — mist has no REST surface, only Kafka in/out. |
| DOM-06 | Processor accepts `logrus.FieldLogger` | PASS | `mist/processor.go:39` — `func NewProcessor(l logrus.FieldLogger, ctx context.Context, p producer.Provider) Processor`. |
| DOM-07–DOM-19 | resource.go / handler checks | N/A | No `resource.go` in this package (no HTTP handlers). |
| DOM-20 | Table-driven tests | **FAIL (Minor)** | `mist/processor_test.go` and `mist/model_test.go` use discrete `TestXxx` functions, not `tests := []struct{...}{}` + `t.Run`. Contrast with `libs/atlas-packet/field/clientbound/affected_area_test.go` (same branch), which does use the table pattern correctly (lines 142-179, 260-283). |
| DOM-21 | No atlas-constants duplication | PASS | Uses `world.Id`, `channel.Id`, `_map.Id`, `field.Model` from `libs/atlas-constants` throughout `mist/model.go:8-11`; no local re-declaration. |
| DOM-23 | Kafka topic naming | PASS | `COMMAND_TOPIC_MIST` / `EVENT_TOPIC_MIST` present in `deploy/k8s/base/env-configmap.yaml:47,134` with `KEY: "KEY"` shape (pre-existing, unaffected by this diff); `COMMAND_TOPIC_MONSTER` (`deploy/k8s/base/env-configmap.yaml:48`) and `COMMAND_TOPIC_CHARACTER_BUFF` (`:20`) likewise present. The new local constants (`tasks/mist_tick.go:97,102`) are passed as `token` into `producer.Provider`, which resolves them via `topic.EnvProvider` inside `libs/atlas-kafka/producer/manager.go:67` — not bypassed. |
| DOM-25 | No client wire byte hardcoding | PASS | `mist/model.go:155-170` (`ElemAttr`/`SkillDelay`) and `kafka/consumer/mist/consumer.go:87` (`mistPhase`) are explicitly documented as unreachable/never-read client fields fixed at 0 (IDB-cited), not lookup-switch codes. `poisonmist.go:171` forwards the raw wire `SourceSkillId` — explicitly authorized as a deliberate, documented exception (task prompt; also see inline comment `poisonmist.go:166-170`). |
| DOM-26 | Goroutines via `routine.Go` | PASS | `tasks/mist_tick.go:299,333` — both `runOnce`'s per-tenant fan-out and `processTenant`'s per-mist fan-out use `routine.Go(r.l, ctx, ...)`. `tools/goroutine-guard.sh` exits 0. |
| FILE-05 | Builder in `builder.go` | **FAIL (Important)** | No `builder.go` file exists in the package (`ls services/atlas-maps/atlas.com/maps/mist/*.go` → `model.go, model_test.go, processor.go, processor_test.go, producer.go, producer_test.go, registry.go, registry_test.go`). `Builder`, `NewBuilder`, and every `Set*`/`Build()` method (`mist/model.go:231-377`) are defined inside `model.go` instead. This branch actively extends the collapsed file with new builder methods `SetKinds` (`model.go:319-325`) and `SetRender` (`model.go:327-333`), so the violation is live in the touched file, not merely inherited untouched. |

### `atlas-maps/monster` (support package — REST client to atlas-monsters; `GetInMapRect` added this branch)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01/02/03 | Processor/RestModel/requests placement | PASS | `Processor`+`ProcessorImpl` in `monster/processor.go`; `RestModel`+`Extract` in `monster/rest.go`; `getBaseRequest`/`inMapRectUrl`/`requestCreate` in `monster/requests.go`. Clean three-file split, no `<pkg>.go` catch-all. |
| EXT-01 | JSON:API relationship interfaces | **FAIL (Important)** | `monster/rest.go` (whole file) defines `RestModel` with `GetID()`/`SetID()`/`GetName()` but no `SetToOneReferenceID` / `SetToManyReferenceIDs`. `GetInMapRect` (new this branch, `monster/processor.go:60-62`) is a new call site into this same decode path. Per `libs/atlas-rest/CLAUDE.md` (cited in the guideline), api2go errors on ANY response carrying a `relationships` block if these methods are absent — task-037 precedent for this exact failure mode. |
| EXT-02 | httptest-backed integration test | PASS | `monster/processor_rect_test.go` — `httptest.NewServer` serving a paginated JSON:API fixture (`rectDoc`, lines 24-36), asserting `GetInMapRect` drains all pages and decodes `X`/`Y`/`Id` correctly (lines 41-84). |
| EXT-03 | 404 vs. other errors distinguished | PASS | `GetInMapRect` (`monster/processor.go:60-62`) bubbles the `requests.DrainProvider` error unmodified — no blanket "not found" remapping that would mask a transport/5xx failure. |
| EXT-04 | No hardcoded service URL | PASS | `monster/requests.go:16` — `requests.RootUrl("MONSTERS")`. |

### `atlas-channel/mist` (support package — command-emitting client to atlas-maps)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor in `processor.go` | PASS | `mist/processor.go:13-32` — `Processor` interface + `ProcessorImpl` + `NewProcessor`, nothing elsewhere. |
| DOM-06 | FieldLogger param | PASS | `mist/processor.go:22` — `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`. |
| DOM-24 | Kafka producer stubbed in tests | PASS | `mist/producer_test.go` calls `CreateCommandProvider(body)()` directly (a pure `model.Provider`, not routed through `producer.ProviderImpl`/`Emit`) — no live-producer path exercised, no stub needed. |

### `atlas-channel/skill/handler/poisonmist` (new package this branch)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-13 | No cross-domain logic in handler | PASS | `Apply` (`poisonmist.go:87-185`) only validates its own inputs and calls `loadCaster`/`emitCreate` seams; no direct provider or cross-service DB access. |
| DOM-24 | Kafka producer stubbed | PASS | `poisonmist_test.go:50-67` (`harness`) replaces the package-level `emitCreate` var with a recording stub for every test — the real `mist.NewProcessor(...).Create` / Kafka producer path is never exercised in tests. |
| DOM-20 | Table-driven tests | **FAIL (Minor)** | `poisonmist_test.go` is nine discrete `TestApply_*` functions, each hand-duplicating the `run(t, stubEffect(...))` call, rather than a single `tests := []struct{...}{}` + `t.Run` table. Every case does share the same `run`/`stubEffect` harness, so the gap is only against the DOM-20 pattern, not test isolation. |
| DOM-25 | No wire-byte hardcoding | PASS (documented exception) | `poisonmist.go:166-171` forwards `info.SkillId()` verbatim as `SourceSkillId` — explicitly authorized in this audit's scope as a deliberate, documented choice (client compares against its own WZ), not a lookup-switch code. |

### `atlas-channel/kafka/consumer/mist` (Kafka consumer — MIST_CREATED/MIST_DESTROYED → AffectedArea packets)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| Multitenancy | Tenant-scoped consumption | PASS | `consumer.go:29` — `consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser)`; handlers gate on `sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId)` (`consumer.go:94,120`). |
| DOM-24 | Producer stubbed | N/A | Handlers broadcast via `session.Announce`/`wp`, not a Kafka producer; `consumer_test.go` swaps `affectedAreaCreatedBroadcaster`/`affectedAreaRemovedBroadcaster` package vars for recording stubs (lines 24-42). |

### `atlas-data/skill` reader + `effect` model (WZ ingest — `dot`/`dotInterval`/`dotTime`/`lt`/`rb` fields)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| Unit conversion | Single conversion point, documented | PASS | `services/atlas-data/atlas.com/data/skill/reader.go` (+14 lines) converts WZ seconds→ms exactly once at read time, with an inline comment pointing at the task-054 precedent; `effect.Model.DotInterval()`/`DotTime()` doc comments (`atlas-channel/.../effect/model.go:149-159`) restate the millisecond contract for downstream callers — no re-scaling found in `poisonmist.go`. |
| FILE placement | `ModelBuilder` in `entity.go`/model construction file | PASS | `atlas-data/skill/effect/model.go` (`ModelBuilder`) is this package's builder-equivalent for a WZ-sourced (non-GORM) domain and was not restructured by this branch; new `Set/Get Dot/DotInterval/DotTime` accessors follow the existing file's established shape. |

### `atlas-maps/tasks` (`MistTick` — concurrency review, per prompt's explicit focus)

| Area | Finding | Evidence |
|------|---------|----------|
| Tenant-context propagation into workers | PASS | `processTenant` derives `tctx := tenant.WithContext(ctx, t)` (`mist_tick.go:323`) from the tick's own root `ctx` (carrying the OTel span from `Run`, `mist_tick.go:284-288`), then passes `tctx` into `routine.Go`; each worker's `gctx` (`mist_tick.go:333`) is exactly that tenant-scoped context, used unmodified for `Destroy` (`:349`) and `monstersInRect` (`:410`, which is REST — see NFR-3 comment at `:270-275`). |
| Registry locking under concurrent access | PASS | `mist.Registry` (`registry.go:32-35`) guards all reads/writes with `sync.RWMutex`; `AllByTenant` (read-lock, `:125-137`), `Destroy`/`Remove` and `UpdateLastTick` (write-lock, `:86-104`, `:139-153`) are called concurrently from per-mist workers per the doc comment at `mist_tick.go:313-321`, which is accurate. |
| WaitGroup / semaphore correctness under panic | PASS | `routine.Go` (`libs/atlas-routine/routine.go:14-24`) runs `fn(ctx)` under a `defer recover()`; `libs/atlas-routine/routine_test.go` (`TestGoFnDefersRunBeforeRecoverLog`) proves the caller's own `defer`s (here, `defer wg.Done()` and `defer func(){ <-sem }()`, `mist_tick.go:334-335`) run before the panic is swallowed — a panicking `tickOneMist` cannot leak a `WaitGroup` count or a semaphore slot. |
| Per-mist failure isolation / no silent swallow | PASS | `tickMonsters` logs at Error and returns on a rect-lookup failure (`mist_tick.go:411-414`), isolated per mist (proven by `TestMistTick_MonsterTarget_LookupFailureIsolatedPerMist`, `mist_tick_monster_test.go:287-307`); `tickCharacters`' per-character `Debugf`-and-`continue` on position-fetch failure (`mist_tick.go:384-387`) is unchanged, pre-existing logic explicitly extracted "verbatim" (doc comment `mist_tick.go:372-375`) — not introduced by this branch, so not counted as a new DOM-28 finding here. |
| goroutine-guard letter + spirit | PASS | Both fan-outs go through `routine.Go`; no bare `go` statements introduced; `tools/goroutine-guard.sh` exits 0 repo-wide. |

### Cross-service contract mirror (`kafka/message/mist/kafka.go` × 3, `COMMAND_TOPIC_MONSTER` body)

| Check | Status | Evidence |
|-------|--------|----------|
| `CreateCommandBody` byte-for-byte across atlas-maps/atlas-channel/atlas-monsters | PASS | All three `kafka.go` files (`atlas-maps/.../kafka/message/mist/kafka.go:46-70`, `atlas-channel/.../kafka/message/mist/kafka.go:55-79`, `atlas-monsters/.../kafka/message/mist/kafka.go:44-68`) declare identical field names/types/json tags, including the new `TargetKind`/`EffectKind` pair. |
| `applyStatusBody` key set matches `COMMAND_TOPIC_MONSTER`'s existing shared-handler key set | PASS | `mist_tick.go:191-199` doc comment cites the exact atlas-channel/atlas-monsters line numbers it was verified against; `TestMistTick_MonsterTarget_BodyKeySetMatchesChannel` (`mist_tick_monster_test.go:200-230`) pins the seven keys mechanically. |
| `atlas-monsters` AREA_POISON producer sets `TargetKind`/`EffectKind` explicitly | PASS | `monster/processor.go` diff (+5 lines) sets `TargetKind: mistKafka.TargetKindCharacter, EffectKind: mistKafka.EffectKindDisease` explicitly rather than relying on the empty-string default; covered by `TestBuildMistCreateBody` and `TestExecuteMist_ProducesMistCreateCommand` assertions added in `processor_test.go`. |

## Security Review

Not applicable — no auth/token/session-management code in the diff.

## Summary

### Blocking (must fix)
- **FILE-05 (Important)** — `services/atlas-maps/atlas.com/maps/mist/model.go:231-377`: `Builder`/`NewBuilder`/`Build()` (including this branch's new `SetKinds`/`SetRender` methods) live in `model.go` with no `builder.go` file in the package.
- **EXT-01 (Important)** — `services/atlas-maps/atlas.com/maps/monster/rest.go`: `RestModel` (now exercised by the new `GetInMapRect` call site) is missing `SetToOneReferenceID`/`SetToManyReferenceIDs`, the api2go relationship-interface methods required even when the current response has no `relationships` block.

### Non-Blocking (should fix)
- **DOM-01 (Minor)** — `services/atlas-maps/atlas.com/maps/mist/model.go:349-377`: `Builder.Build()` performs no invariant validation; all mist-shape validation lives solely in the calling handler.
- **DOM-20 (Minor)** — Several new test files (`mist/processor_test.go`, `mist/model_test.go`, `tasks/mist_tick_monster_test.go`, `skill/handler/poisonmist/poisonmist_test.go`) use discrete `TestXxx` functions rather than the table-driven `tests := []struct{...}{}` + `t.Run` pattern the guideline prefers; `libs/atlas-packet/field/clientbound/affected_area_test.go` in the same branch shows the pattern is known and used elsewhere.
