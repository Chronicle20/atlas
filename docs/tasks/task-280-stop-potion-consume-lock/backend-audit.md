# Backend Audit — atlas-consumables / atlas-channel (task-280)

- **Service Path:** `services/atlas-consumables/atlas.com/consumables`, `services/atlas-channel/atlas.com/channel`
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-28
- **Scope:** diff `bda6566f3..6e44e93b9` — 5 changed non-test Go files across 5 packages
- **Build:** PASS
- **Tests:** 102 passed, 0 failed (changed packages); `go test ./... -count=1` clean in both modules
- **Overall:** NEEDS-WORK

## Build & Test Results

```
services/atlas-channel/atlas.com/channel:     go build ./...  -> exit 0
services/atlas-channel/atlas.com/channel:     go test ./... -count=1 -> no FAIL lines
services/atlas-consumables/atlas.com/consumables: go build ./... -> exit 0
services/atlas-consumables/atlas.com/consumables: go test ./... -count=1 -> no FAIL lines
go test ./consumable/... ./character/buff/... -v -> 100 "--- PASS", 0 "--- FAIL"
go test ./kafka/consumer/consumable/... -v    ->   2 "--- PASS", 0 "--- FAIL"
tools/goroutine-guard.sh                       -> exit 0 (91 modules scanned)
```

`tools/verify.sh` / docker bake deliberately not re-run (already exited 0 at this HEAD per the dispatch brief).

## Package classification

| Package | model.go | entity.go | rest.go | provider.go | processor.go | resource.go | Class |
|---|---|---|---|---|---|---|---|
| `atlas-channel/kafka/consumer/consumable` | no | no | no | no | no | no | support |
| `atlas-channel/kafka/message/consumable` | no | no | no | no | no | no | support |
| `atlas-consumables/character/buff` | yes | no | yes | no | yes | no | domain |
| `atlas-consumables/consumable` | no | no | no | no | yes | no | support |
| `atlas-consumables/kafka/message/consumable` | no | no | no | no | no | no | support |

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (DOM-01..05, 11, 16) | FIRED | `character/buff/model.go` and `character/buff/rest.go` both present in a changed package |
| FILE placement (FILE-01..06) | FIRED | five changed Go packages; no exemptions |
| SUB (SUB-01..04) | N/A | no changed package has `resource.go` (`ls` of all five packages) |
| REST (DOM-06..09, 12..15, 17..19, 32) | FIRED | `character/buff/processor.go` + `rest.go`, `consumable/processor.go` |
| Constants reuse (DOM-21) | FIRED | diff declares `type errorAction` (consumer.go:99), a const block (consumer.go:101-108), `ErrPotionLocked` (processor.go:67), and two `ErrorTypePotionLocked` consts |
| Testing (DOM-10, 20, 24, 33) | FIRED | diff touches four `_test.go` files |
| Cache (DOM-29) | N/A | no `cache.go` in any changed package; `ProcessorImpl` fields (processor.go:91-98) are all processor deps, no cached state |
| Messaging (DOM-30) | FIRED | `consumable/processor.go:411` calls `producer.ProviderImpl` |
| Multi-tenancy (DOM-31) | FIRED | `character/buff/rest.go` present; changed code passes `p.ctx` to the producer |
| Migration hygiene (DOM-34, 35) | N/A | diff moves no symbol between a service and `libs/atlas-*` (`git diff --stat` shows no `libs/` path) |
| Deploy & topics (DOM-22, 23) | N/A | diff adds no `libs/` module and no `*_TOPIC_*` env var (`git diff … \| grep TOPIC` matches only a comment at processor_potion_lock_test.go:184) |
| Runtime safety (DOM-26) | FIRED | non-test Go files changed |
| Channel wire values (DOM-25) | FIRED | diff touches `services/atlas-channel` |
| Resilience (DOM-27, 28) | FIRED | `resolvePotionLocked` (processor.go:200) is a new remote-fetch fallback path |
| External clients (EXT-01..04) | FIRED | changed package `character/buff` composes its URL via `requests.RootUrlFor` (requests.go:15) and fetches via `requests.DrainProvider` (processor.go:67) |
| Scaffolding (SCAFFOLD-01..09) | N/A | diff adds no `services/atlas-<svc>/` directory, registers no new channel Writer/Handler (consumer.go:143 reuses the existing `statpkt.StatChangedWriter`), and does not touch `deploy/shared/routes.conf` |
| Security (SEC-01..04) | N/A | neither service handles tokens, auth, redirects, or secrets on the changed surface — no `jwt`, `ParseUnverified`, or redirect handler in any changed file |
| patterns-provider.md (foundational) | N/A | changed code defines no provider; `ErrorEventProvider` is unchanged (producer.go:18) |
| patterns-functional.md (foundational) | N/A | changed code defines no curried constructor, decorator, or model combinator; `IsPotionLocked` is a plain slice predicate (model.go:87) |

## Checklist Results

### atlas-channel/kafka/consumer/consumable (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | `Processor` interface/constructor/`ProcessorImpl` methods in `processor.go` | N/A | grep for `type Processor interface`/`ProcessorImpl`/`NewProcessor(` in consumer.go returns zero matches — package defines no processor |
| FILE-02 | `RestModel`/`Transform`/JSON:API methods in `rest.go` | N/A | grep for `type RestModel`/`func Transform(`/`func Extract(` in consumer.go returns zero matches |
| FILE-03 | Cross-service request functions in `requests.go` | N/A | grep for `requests.RootUrl(`/`GetRequest[`/`PostRequest[`/`getBaseRequest(` in consumer.go returns zero matches |
| FILE-04 | Entity/`Migration`/`TableName` in `entity.go` | N/A | grep for `type entity struct`/`func Migration(`/`TableName()` returns zero matches |
| FILE-05 | Builder/Model/writes/readers placed per the file table | PASS | no `Builder`, domain `Model`, `Create*/Update*/Delete*` write, or `database.Query`/`SliceQuery` reader in consumer.go (FILE-05's four verification greps all empty). The added `errorAction` enum (consumer.go:99-109) is a private consumer-routing enum, not the "domain-specific enums" `state.go` owns (file-responsibilities.md:117-119) |
| FILE-06 | No file carrying ≥2 responsibilities | PASS | package non-test files: `consumer.go` only; it carries Kafka consumer/handler registration, none of the responsibilities in the file table |
| DOM-20 | Tests are table-driven | WARN | consumer_test.go:15 `TestConsumableErrorAction` is table-driven; consumer_test.go:41 `TestPotionLockedWireValue` is a bare single assertion with no `tests := []struct{}` + `t.Run` |
| DOM-24 | Emit paths install `producertest` | N/A | consumer_test.go contains no `AndEmit(`/`message.Emit(`/`producer.Produce(`, and calls only the pure `consumableErrorAction` (consumer_test.go:32) — no consumer entry point is invoked |
| DOM-25 | Client-interpreted wire values resolved from tenant writer options | PASS | the added routing carries no client byte: consumer.go:112-127 switches on the semantic string `POTION_LOCKED`, and consumer.go:143 emits `statpkt.NewStatChanged(make([]statpkt.Update, 0), true)` — no integer wire literal introduced anywhere in the diff |
| DOM-26 | Goroutines via `routine.Go` | PASS | `tools/goroutine-guard.sh` exit 0; no `go ` statement in consumer.go |
| DOM-33 | Interface change updates every mock | N/A | diff adds/removes/re-signs no method on any `Processor`/`Provider`/`Administrator` interface in this package |

### atlas-channel/kafka/message/consumable (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..04 | Processor / RestModel / requests / entity placement | N/A | all four verification greps return zero matches in kafka.go — the file holds Kafka contract structs and constants only |
| FILE-05 | Builder/Model/writes/readers placed per the file table | PASS | no Builder, domain Model, write, or query reader in kafka.go |
| FILE-06 | No file carrying ≥2 responsibilities | PASS | package non-test files: `kafka.go` only; it is not `<pkgname>.go` (package is `consumable`) and carries only the message contract |
| DOM-21 | No redeclaration of something already in `libs/atlas-constants/` | PASS | `grep -rn "POTION_LOCKED" libs/` returns zero matches; the per-service Kafka contract constant at kafka.go:97 is the shape documented in cross-service-implementation.md:55-70 ("Add Kafka message types if needed" per affected service) |
| DOM-26 | Goroutines via `routine.Go` | PASS | `tools/goroutine-guard.sh` exit 0 |

### atlas-consumables/character/buff (domain)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` with `NewBuilder()`, fluent setters, validating `Build()` | **FAIL** | package has `model.go:10` (`type Model struct`) but no `builder.go` — `ls services/atlas-consumables/atlas.com/consumables/character/buff/` lists model.go, processor.go, producer.go, requests.go, rest.go only. Construction goes through the flat `NewBuff(...)` (model.go:51) with no validating `Build()`. Pre-existing; the diff added `IsPotionLocked` to this package but did not introduce the gap |
| DOM-02 | `Model.ToEntity()` in `entity.go` | N/A | no `entity.go` in the package (`ls`) |
| DOM-03 | `Make(Entity)` in `entity.go` | N/A | no `entity.go` in the package (`ls`) |
| DOM-04 | `Transform(Model) (RestModel, error)` in `rest.go` | **FAIL** | `rest.go` exists (RestModel at rest.go:10) but grep for `func Transform(` in the package returns zero matches; only `Extract` is defined (rest.go:34). No documented exception for read-only client packages exists in file-responsibilities.md:102-114 or cross-service-implementation.md. Pre-existing |
| DOM-05 | `TransformSlice` in `rest.go`, used by list handlers | **FAIL** | grep for `func TransformSlice(` in the package returns zero matches; rest.go:34 defines only `Extract`. Pre-existing |
| DOM-11 | Providers evaluate lazily | N/A | no `provider.go` in the package (`ls`) |
| DOM-16 | `administrator.go` holds writes | N/A | package performs no create/update/delete — processor.go:40-52 emits Kafka commands, processor.go:62 is a read; no `db.` call in the package |
| FILE-01 | Processor symbols in `processor.go` | PASS | `type Processor interface` processor.go:18, `type ProcessorImpl` processor.go:25, `func NewProcessor(` processor.go:30, and all four methods processor.go:40,46,50,62 — all in processor.go |
| FILE-02 | RestModel/Transform/Extract/JSON:API in `rest.go` | PASS | `type RestModel` rest.go:10, `GetName()` rest.go:21, `GetID()` rest.go:25, `SetID()` rest.go:29, `Extract` rest.go:34 — all in rest.go |
| FILE-03 | Cross-service request functions in `requests.go` | PASS | `requests.RootUrlFor` requests.go:15 and `getBaseRequest` requests.go:14 both in requests.go; processor.go:67 calls only `requests.DrainProvider`, which is not one of FILE-03's four greps |
| FILE-04 | Entity/Migration/TableName in `entity.go` | N/A | zero matches for `type entity struct`/`func Migration(`/`TableName()` in the package |
| FILE-05 | Builder/Model/writes/readers placed per the file table | PASS (with DOM-01 above) | domain `Model` in model.go:10; no Builder exists at all (graded by DOM-01); no writes; no `database.Query`/`SliceQuery` reader |
| FILE-06 | No file carrying ≥2 responsibilities | PASS | non-test files model.go / processor.go / producer.go / requests.go / rest.go each hold exactly one responsibility; there is no `buff.go` |
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | processor.go:30 `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor` |
| DOM-07,08,09,12..15,17,32 | `resource.go`-scoped REST checks | N/A | package has no `resource.go` (`ls`) and registers no HTTP route |
| DOM-18 | RestModel implements JSON:API interface | PASS | `GetName()` rest.go:21, `GetID()` rest.go:25, `SetID()` rest.go:29 |
| DOM-19 | Request models flat | PASS | rest.go:10-19 declares flat scalar/slice fields with `json:` tags only; no nested `Data`/`Type`/`Attributes` struct and no `jsonapi:` tag in the file |
| DOM-20 | Tests are table-driven | PASS | model_test.go `TestIsPotionLocked` uses `tests := []struct{...}` + `t.Run` across seven cases (diff hunk model_test.go:+90..+157) |
| DOM-24 | Emit paths install `producertest` | N/A | the only changed test file, model_test.go, exercises the pure `IsPotionLocked` predicate; it contains no `AndEmit(`/`message.Emit(`/`producer.Produce(` and calls no consumer/saga entry point |
| DOM-26 | Goroutines via `routine.Go` | PASS | `tools/goroutine-guard.sh` exit 0; no `go ` statement in model.go |
| DOM-31 | Tenant/trace in context only | PASS | rest.go:10-19 carries no `TenantId`/`tenant_id` field; requests.go:22-27 builds the URL from `characterId` only; tenant reaches the client via `p.ctx` (processor.go:63) |
| DOM-33 | Interface change updates every mock | N/A | the diff adds no method to `buff.Processor` (processor.go:18-23 unchanged); `mock/processor.go` untouched |
| EXT-01 | Target REST model implements `SetToOneReferenceID` / `SetToManyReferenceIDs` | **FAIL** | `grep -rn "SetToOneReferenceID\|SetToManyReferenceIDs" services/atlas-consumables/atlas.com/consumables/character/buff/` returns zero matches; rest.go:10-32 defines only `GetName`/`GetID`/`SetID`. Pre-existing |
| EXT-02 | `httptest`-backed integration test with a JSON:API fixture | PASS | processor_test.go:35-45 stands up an `httptest.NewServer` serving the `buffsDoc` fixture (processor_test.go:21) and asserts two populated `buff.Model`s with correct `SourceId()` mapping (processor_test.go:55-70). Note: the fixture carries no `relationships` block (`grep -n relationships` in the package returns zero matches), so it does not exercise the EXT-01 failure mode |
| EXT-03 | Only genuine 404s map to "not found" | PASS | processor.go:68 `if errors.Is(err, requests.ErrNotFound)` returns the empty slice; every other error is returned unchanged at processor.go:71, and processor_notfound_test.go:22-48 pins the 404 behaviour |
| EXT-04 | URL via `requests.RootUrl(<DOMAIN>)`, not hardcoded DNS | PASS | requests.go:15 `return requests.RootUrlFor(ctx, "BUFFS")`; no DNS literal in the package |

### atlas-consumables/consumable (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor symbols in `processor.go` or a `processor_<group>.go` split | **FAIL** | `vega.go:55` `func (p *ProcessorImpl) VegaScrollError(...)` and `vega.go:98` `func (p *ProcessorImpl) RequestVegaScroll(...)` are `ProcessorImpl` methods in a bare topic-named file — explicitly named as a FAIL in file-responsibilities.md:210 ("or a bare topic-named file like `custody.go` / `register.go`"). `processor_catch.go:53,131` is the permitted `processor_<group>.go` split. Interface (processor.go:72), `ProcessorImpl` (processor.go:90), and constructor (processor.go:100) are correctly placed. Pre-existing; unchanged in this diff |
| FILE-02 | RestModel/Transform/Extract/JSON:API in `rest.go` | N/A | zero matches for `type RestModel`/`func Transform(`/`func Extract(` across the package's non-test files |
| FILE-03 | Cross-service request functions in `requests.go` | N/A | zero matches for `requests.RootUrl(`/`GetRequest[`/`PostRequest[`/`getBaseRequest(` in the package |
| FILE-04 | Entity/Migration/TableName in `entity.go` | N/A | zero matches for `type entity struct`/`func Migration(`/`TableName()` in the package |
| FILE-05 | Builder/Model/writes/readers placed per the file table | PASS | no `type Builder`/`NewBuilder(`, no domain `Model` struct, no `database.Query`/`SliceQuery` reader in the package (all four FILE-05 greps empty) |
| FILE-06 | No file carrying ≥2 responsibilities | PASS | there is no `consumable.go`; all 14 `model.Provider[[]kafka.Message]` producer functions are in producer.go:18-165 and none appear in processor.go, so processor.go carries the processor responsibility alone |
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | processor.go:100 `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`; the new `bp` dependency is wired at processor.go:108 from that same `l` |
| DOM-07,08,09,12..15,17,18,19,32 | `resource.go`/`rest.go`-scoped REST checks | N/A | package has no `resource.go` and no `rest.go` (`ls`); registers no HTTP route |
| DOM-20 | Tests are table-driven | WARN | table-driven: processor_potion_lock_test.go:39, :98, :194. Not table-driven: processor_potion_lock_test.go:127 `TestRequestItemConsume_UnlockedInScopeReserves`, :158 `TestRequestItemConsume_BuffReadErrorFailsOpen`, processor_test.go:828 `TestConsumeErrorType_PotionLocked`, :836 `TestErrorEventProviderPotionLocked` — each a single straight-line case with no `tests := []struct{}` + `t.Run`. Graded WARN, not FAIL, on testing-guide.md:18 ("Prefer table-driven tests") |
| DOM-21 | No redeclaration of something already in `libs/atlas-constants/` | PASS | the STOP_PORTION classification is consumed from the shared library, not redeclared: model.go:93 uses `charconst.TemporaryStatTypeStopPortion`, defined at libs/atlas-constants/character/temporary_stat.go:81. `ErrPotionLocked` (processor.go:67) is a service-local sentinel with no libs equivalent (`grep -rn "POTION_LOCKED" libs/` → zero matches) |
| DOM-24 | Emit paths install `producertest` | PASS | processor_potion_lock_test.go:39 reaches `producer.ProviderImpl` via `rejectPotionLocked` (processor.go:411); testmain_test.go:16-18 installs the singleton stub from the shared package (`producertest.InstallCapturing()`), with no `t.Cleanup(producer.ResetInstance)` anywhere in the package |
| DOM-25 | Client-interpreted wire values | PASS | the domain service emits the semantic key `POTION_LOCKED` (processor.go:501 → kafka.go:127), not a client byte — exactly the layering DOM-25 point 3 requires; no integer literal is added to any event body in the diff |
| DOM-26 | Goroutines via `routine.Go` | PASS | `tools/goroutine-guard.sh` exit 0; no `go ` statement added in processor.go |
| DOM-27 | Handler error branches use `server.WriteErrorResponse` | N/A | rule requires a changed handler writing `http.StatusInternalServerError` in a service that calls `database.Connect`; `grep -rn "database.Connect" services/atlas-consumables --include='*.go'` returns zero matches — the service is not DB-backed, and the package has no handler |
| DOM-28 | Fallible enrichment degrades loudly (`degrade.Observe`) | **FAIL** | processor.go:200-207 `resolvePotionLocked` fetches remote data through `bp.GetByCharacterId` and, on error, returns the degraded answer `false` (potion use unlocked) with only `l.WithError(err).Warnf(...)` at processor.go:203 — no `degrade.Observe(...)`, so `atlas_enrichment_degraded_total` never increments. patterns-resilience.md:127 requires "Warn log **plus** an `atlas_enrichment_degraded_total` increment" and states the bare degrade "is a finding regardless of justification". The package already has the correct shape available and in use at processor.go:1382 (`degrade.Observe(p.l, "consumable.reward.name", …)`), and the file already imports `libs/atlas-rest/degrade` (processor.go:40). Introduced by this diff. (The identically-shaped `resolveZombified` at processor.go:185 is pre-existing and carries the same defect — prevalence does not exempt the new site.) |
| DOM-29 | Caches are singletons via `GetCache()` | N/A | no `cache.go` in the package; `ProcessorImpl` fields processor.go:91-98 are `l`, `ctx`, and five processor dependencies — no cached state |
| DOM-30 | DB write + events emit via `AndEmit` + `message.Buffer` | PASS | processor.go:411 is a direct `producer.ProviderImpl` call, which patterns-kafka.md exception 2 ("Operations over non-DB state") permits: the service performs no DB write on any path — `grep -rn "database.Connect" services/atlas-consumables --include='*.go'` returns zero matches, and `RequestItemConsume` (processor.go:324-341) reaches the rejection before any reservation or write |
| DOM-31 | Tenant/trace in context only | PASS | processor.go:411 passes tenancy only as `p.ctx` into `producer.ProviderImpl(p.l)(p.ctx)`; the emitted body carries `ts.Id(characterId)` (a character id) and the error string — `ErrorEventProvider` (producer.go:18) takes no tenant argument |
| DOM-33 | Interface change updates every mock | N/A | the diff adds no method to the `Processor` interface (processor.go:72-89); `rejectPotionLocked` (processor.go:409) is unexported and not on the interface, and `resolvePotionLocked` (processor.go:200) is a package function |

### atlas-consumables/kafka/message/consumable (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..04 | Processor / RestModel / requests / entity placement | N/A | all four verification greps return zero matches in kafka.go — the file holds the Kafka contract only |
| FILE-05 | Builder/Model/writes/readers placed per the file table | PASS | no Builder, domain Model, write, or query reader in kafka.go |
| FILE-06 | No file carrying ≥2 responsibilities | PASS | package non-test files: `kafka.go` only; not `<pkgname>.go`, one responsibility |
| DOM-21 | No redeclaration of something already in `libs/atlas-constants/` | PASS | `grep -rn "POTION_LOCKED" libs/` → zero matches; kafka.go:127 sits alongside the existing `ErrorTypeConsumeFailed` (kafka.go:122) in the same contract block, per cross-service-implementation.md:55-70 |
| DOM-26 | Goroutines via `routine.Go` | PASS | `tools/goroutine-guard.sh` exit 0 |

## Security Review

SEC-01..04 not evaluated: trigger did not fire. Neither changed service handles authentication, authorization, tokens, redirects, or secrets on this surface — no `jwt`, `ParseUnverified`, redirect handler, or credential literal appears in any of the five changed non-test files.

## Not evaluable from the diff

- none — every rule whose `Applies when` fired was settled from the changed files plus targeted symbol lookups (`vega.go` / `processor_catch.go` method placement, `buff/rest.go`, `buff/requests.go`, `producer.go:18`, `testmain_test.go`, `libs/atlas-constants/character/temporary_stat.go`).

## Summary

### Blocking (must fix)

- **DOM-28** — `services/atlas-consumables/atlas.com/consumables/consumable/processor.go:200-207`: `resolvePotionLocked` silently degrades a failed remote buff read to "unlocked" with a Warn log only; patterns-resilience.md:127 requires `degrade.Observe(...)` so `atlas_enrichment_degraded_total` increments. **Introduced by this diff**; the package already uses the correct shape at processor.go:1382.
- **FILE-01** — `services/atlas-consumables/atlas.com/consumables/consumable/vega.go:55,98`: `ProcessorImpl` methods live in a bare topic-named file instead of `processor.go` / `processor_<group>.go`. Pre-existing, unchanged file in a changed package.
- **DOM-01** — `services/atlas-consumables/atlas.com/consumables/character/buff/`: no `builder.go` for a package with `model.go:10`. Pre-existing.
- **DOM-04** — `services/atlas-consumables/atlas.com/consumables/character/buff/rest.go`: no `func Transform(`. Pre-existing.
- **DOM-05** — `services/atlas-consumables/atlas.com/consumables/character/buff/rest.go`: no `func TransformSlice(`. Pre-existing.
- **EXT-01** — `services/atlas-consumables/atlas.com/consumables/character/buff/rest.go:10-32`: `RestModel` implements neither `SetToOneReferenceID` nor `SetToManyReferenceIDs`; any upstream response carrying a `relationships` block will fail to unmarshal. Pre-existing.

### Non-Blocking (should fix)

- **DOM-20** — five added tests are single-case rather than table-driven: `consumer_test.go:41`, `processor_potion_lock_test.go:127`, `processor_potion_lock_test.go:158`, `processor_test.go:828`, `processor_test.go:836`.
