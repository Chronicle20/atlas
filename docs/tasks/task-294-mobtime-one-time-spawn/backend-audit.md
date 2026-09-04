# Backend Audit — atlas-maps (task-294-mobtime-one-time-spawn)

- **Service Path:** services/atlas-maps
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-09-03
- **Scope:** Changed Go packages between `a915de988..4457c3547` (25 changed files, 6 packages) in `services/atlas-maps/atlas.com/maps`
- **Build:** PASS
- **Tests:** all packages `ok` (module-local `go test ./... -count=1`)
- **Overall:** NEEDS-WORK

## Build & Test Results

```
$ go build ./...
(no output — success)

$ go test ./... -count=1
ok  	atlas-maps	...
ok  	atlas-maps/map	2.218s
ok  	atlas-maps/map/monster	5.764s
... (all packages ok, none failed)
```

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (DOM-01..05,11,16) | Fired | `map/monster` and `data/map/monster` both have `model.go` |
| FILE placement (FILE-01..06) | Fired | Every changed Go package runs this family unconditionally |
| SUB (SUB-01..04) | N/A | No changed package has `resource.go` without `model.go`; `map`(`_map`) is sub-domain-shaped but its `resource.go` was not changed by this diff and was read only for corroborating classification |
| REST (DOM-06..09,12..15,17..19,32) | Fired (narrow) | `map/monster/processor.go`, `map/processor.go`, `data/map/monster/processor.go` exist — only DOM-06 (processor ctor) is in scope; no `resource.go` in any changed package |
| Constants reuse (DOM-21) | Fired | New consts/types added: `Classified`, `metaFieldSeeded`/`metaFieldOneTimeFired`, `CommandTypeDestroyField`, `FieldCommand[E]`, `DestroyFieldBody` |
| Testing (DOM-10,20,24,33) | Fired | Diff touches multiple `_test.go`; `data/map/monster.Processor` interface changed (2 methods removed) |
| Cache (DOM-29) | Fired (narrow) | `map/monster` package holds shared registry state reached via `GetRegistry()`/`sync.Once` (registry-shaped, not literally `cache.go`) |
| Messaging (DOM-30) | Fired | `map/producer.go` adds `destroyFieldCommandProvider`; emitted via `mb.Put` inside `map/processor.go` `Exit` |
| Multi-tenancy (DOM-31) | Fired | `data/map/monster/rest.go` present; `map/monster/registry.go` and `map/processor.go` read/pass `tenant.Model` |
| Deploy & topics (DOM-22,23) | N/A | No `libs/atlas-*` module added; `COMMAND_TOPIC_MONSTER` already exists in `deploy/k8s/base/env-configmap.yaml:68` and both overlays — not a new/renamed topic var |
| Runtime safety (DOM-26) | Fired | Non-test Go files changed; verified with `tools/goroutine-guard.sh` |
| Channel wire values (DOM-25) | N/A | Diff does not touch `services/atlas-channel` or `libs/atlas-packet`; no client-interpreted byte added |
| Resilience (DOM-27,28) | N/A | No DB-backed handler branch changed; no `model.Decorator` touched |
| External clients (EXT-01..04) | Fired | `data/map/monster` calls atlas-data via `requests.RootUrlFor` + `requests.DrainProvider`/`PagedGetRequest` |
| Scaffolding (SCAFFOLD-01..09) | N/A | No new service directory, no channel Writer/Handler, no `routes.conf` change |
| Security (SEC-01..04) | N/A | atlas-maps does not handle auth/tokens/redirects/secrets |
| patterns-provider.md (foundational) | N/A | No provider composition defined/changed in diff |
| patterns-functional.md (foundational) | N/A | No curried-constructor/decorator/combinator changes in diff beyond existing curried `Processor.Exit(mb)(...)` shape, unchanged pattern |

## Checklist Results

### `data/map/monster` (domain — has `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` with `NewBuilder()`/`Build()` | FAIL | `services/atlas-maps/atlas.com/maps/data/map/monster/model.go:1` defines `SpawnPoint` as a domain `Model`; no `builder.go` in the package (`ls` confirms only classify.go, classify_test.go, model.go, processor.go, processor_drain_test.go, requests.go, rest.go). Pre-existing gap; diff does not add one while adding `Classify`/`Hide`. |
| DOM-02/03 | `Model.ToEntity()` / `Make(Entity)` in `entity.go` | N/A | No `entity.go` in package — not a GORM-persisted domain (REST-mirror only) |
| DOM-04 | `Transform(Model) (RestModel, error)` in `rest.go` | N/A | Package direction is inbound (`Extract`, not `Transform`) — `rest.go:37` defines `Extract`, not a `Transform`; DOM-04's own trigger ("package has `rest.go`") did fire, but the rule requires a `Transform` symbol which this inbound-only client package has no reason to define. No finding — see note below. |
| DOM-05 | `TransformSlice` used by list handlers | N/A | No handlers in this package (no `resource.go`) |
| DOM-06 | Processor ctor takes `logrus.FieldLogger` | PASS | `services/atlas-maps/atlas.com/maps/data/map/monster/processor.go:24` — `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor` |
| DOM-11 | Providers lazy via `database.Query`/`SliceQuery` | N/A | No `provider.go`; this is a Redis/HTTP-backed package, not GORM |
| DOM-16 | `administrator.go` for write domains | N/A | Package performs no writes (read-only mirror of atlas-data) |
| DOM-20 | Table-driven tests | PASS | `classify_test.go:76` (`TestClassify`) and `:5` (`TestExtractCarriesHide`) both use `tests := []struct{...}` + `t.Run` |
| DOM-21 | No redeclared shared constant/type | PASS | `Classify`/`Classified` are new, no `libs/atlas-constants` collision (`grep -rn "Classified" libs/atlas-constants/` — no match) |
| DOM-24 | producertest stub / no-op producer for emit-reaching tests | N/A | Package emits no Kafka messages, tests reach no emit path |
| DOM-33 | Mocks updated for interface change | PASS | `Processor` interface dropped `SpawnableSpawnPointProvider`/`GetSpawnableSpawnPoints` (`processor.go:13-16` diff); `grep -rn "SpawnableSpawnPointProvider\|GetSpawnableSpawnPoints"` across the service returns zero hits — no stale implementer, `go build ./...` green |
| EXT-01 | RestModel implements `SetToOneReferenceID`/`SetToManyReferenceIDs` | FAIL | `grep -n "SetToOneReferenceID\|SetToManyReferenceIDs" data/map/monster/rest.go` — zero matches. Package calls another atlas service (atlas-data) via `requests.DrainProvider`/`PagedGetRequest`; pre-existing gap in an in-scope, diff-touched file (`rest.go:1-40`; diff touched `rest.go:52` this same change) |
| EXT-02 | httptest fixture asserts populated struct | PASS | `processor_drain_test.go:44-79` (`TestSpawnPointProviderDrainsBeyondOnePage`) serves a two-page JSON:API fixture via `httptest.NewServer` and asserts 120 populated `SpawnPoint`s |
| EXT-03 | Only genuine 404 maps to "not found" | PASS | No `ErrNotFound`/error-swallowing logic exists anywhere in the package (`grep -rn "ErrNotFound\|errors.Is"` — zero matches); all errors from `DrainProvider` bubble up unmodified in `processor.go:37-41` |
| EXT-04 | URL via `requests.RootUrl(<DOMAIN>)` | PASS | `requests.go:16-18` — `getBaseRequest` returns `requests.RootUrlFor(ctx, "DATA")`, the context-aware `RootUrl` variant, not hardcoded DNS |

### `map/monster` (domain — has `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` present | FAIL | `map/monster/model.go:1` defines `CooldownSpawnPoint`; `ls map/monster/` shows only model.go, processor.go, registry.go + tests — no `builder.go`. Pre-existing gap (unchanged before/after this diff), but package is squarely in scope (registry.go/processor.go both heavily rewritten). |
| DOM-02/03 | entity.go Make/ToEntity | N/A | No `entity.go` — Redis-backed, not GORM |
| DOM-06 | Processor ctor `logrus.FieldLogger` | PASS | `map/monster/processor.go:56` — `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor` |
| DOM-11 | Provider lazy evaluation | N/A | No `provider.go` file |
| DOM-16 | `administrator.go` for write domain | FAIL | Package performs writes (`InitializeForMap` registry.go:304, `ReserveEligibleSpawnPoints` :386, `ClaimOneTimeSpawnPoints` :436, `RearmOneTime` :475, `SetSpawnPointsForMap` :529, `FlushTenant` :520) but has no `administrator.go`; writes are inlined into `registry.go` rather than an administrator file called by the processor |
| DOM-20 | Table-driven tests | FAIL | 19 of 20 new top-level `func Test...` added by this diff use bespoke `t.Run` subtests with per-case inline setup instead of a `tests := []struct{...}` table. Only one new test (`TestInitializeForMap_PartitionsByMobTimeAndHide`, `registry_test.go:171-172`, `cases := []struct{...}`) is table-driven. Representative non-table-driven additions: `registry_test.go:432` (`TestClaimOneTimeSpawnPoints`), `registry_test.go:676` (`TestRearmOneTime`), `registry_test.go:620` (`TestClaimOneTimeSpawnPoints_ConcurrentFiresExactlyOnce`), `registry_test.go:835` (`TestRearmOneTime_ConcurrentTrueExactlyOnce`), `processor_test.go:1177` (`TestSpawnMonsters_OneTimeBatch`), `processor_test.go:1301/1328/1400/1443/1484`. `grep -n ":= \[\]struct" registry_test.go processor_test.go` returns only the one hit at `registry_test.go:172`. |
| DOM-24 | producertest / no-op producer | N/A | `map/monster` tests emit no Kafka messages directly (`grep -n "AndEmit\|message.Emit\|producertest" map/monster/*_test.go` — zero matches); `SpawnMonsters` only calls `mp.CreateMonster` (REST client, not Kafka) |
| DOM-26 | goroutines via `routine.Go` | PASS | `processor.go:112`, `:180` both use `routine.Go(p.l, p.ctx, func(_ context.Context) {...})`; confirmed with `tools/goroutine-guard.sh` (exit 0 across all 93 modules) |
| DOM-29 | Singleton cache/registry via `GetCache()`-equivalent accessor | PASS | `registry.go:62-65` (`registryOnce sync.Once`), `:105-108` (`InitRegistry` guarded by `registryOnce.Do`), `:110-112` (`GetRegistry()` accessor); no cached state held on `ProcessorImpl` (`processor.go:44-51` struct has no cache field, calls `GetRegistry()` per-call inside `SpawnMonsters`) |
| DOM-30 | DB-writing ops emit via `AndEmit`+`message.Buffer` | PASS | Not applicable in this package (no DB writes; Redis-backed) — no direct emit call sites present in `map/monster` |
| DOM-31 | Tenant travels only via context | PASS | `registry.go` methods take `mapKey character.MapKey` (which itself carries `Tenant tenant.Model`), never a REST/Kafka payload field; `processor.go:78-81` builds `mapKey` from `tenant.Model` already resident on `ProcessorImpl` (`p.t = tenant.MustFromContext(ctx)` at `processor.go:60`) |
| DOM-33 | Mocks updated for interface change | PASS | `Processor` interface unchanged in this package; consumer of the changed `data/map/monster.Processor` interface (`mockDataProcessor` in `processor_test.go:735-748` diff) was updated in the same diff to drop `SpawnableSpawnPointProvider`/`GetSpawnableSpawnPoints` |
| FILE-01 | Processor interface/ctor/methods in `processor.go` | PASS | `processor.go:41` (`type Processor interface`), `:56` (`NewProcessor`), all `ProcessorImpl` methods in `processor.go` |
| FILE-05 | Writes in `administrator.go`, readers in `provider.go` | FAIL | Neither file exists; `registry.go` bundles both: writes at `registry.go:304` (`InitializeForMap`), `:386` (`ReserveEligibleSpawnPoints`), `:436` (`ClaimOneTimeSpawnPoints`), `:475` (`RearmOneTime`), `:529` (`SetSpawnPointsForMap`), `:520` (`FlushTenant`) alongside reads at `:361` (`Count`), `:373` (`CountOneTime`), `:496` (`GetSpawnPointsForMap`) |
| FILE-06 | No single file carrying ≥2 responsibilities | FAIL | `registry.go` (19.4 KB, ~540 lines) carries both the administrator-shaped write responsibility and the provider-shaped read responsibility described in FILE-05, plus the Lua-script-backed persistence layer — all in one file. Predates this diff but the diff adds 3 new write operations (`ClaimOneTimeSpawnPoints`, `RearmOneTime`, part of `InitializeForMap`'s classification) and 1 new read operation (`CountOneTime`) to the same collapsed file rather than splitting it. |

### `map` (`_map`, sub-domain-shaped — has `resource.go`, no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor ctor `logrus.FieldLogger` | PASS | `map/processor.go:53` — `func NewProcessor(l logrus.FieldLogger, ctx context.Context, p producer.Provider, db *gorm.DB) Processor` |
| DOM-26 | goroutines via `routine.Go` | PASS | `map/processor.go:91`, `:94` both wrapped in `routine.Go(p.l, p.ctx, func(_ context.Context) {...})` |
| DOM-30 | Write ops emit via `AndEmit`+`message.Buffer` | PASS | New `destroyFieldCommandProvider` emit is issued via `mb.Put(monsterKafka.EnvCommandTopic, destroyFieldCommandProvider(f))` at `map/processor.go:131`, inside `Exit(mb *message.Buffer) func(...)`, itself wrapped by `ExitAndEmit` → `message.Emit(p.p)(...)` at `map/processor.go:140-142` — buffered, not a bare `producer.ProviderImpl` call |
| DOM-31 | Tenant only via context | PASS | `map/processor.go:118` — `tenant.MustFromContext(p.ctx)`; `FieldCommand` (`kafka/message/monster/kafka.go:53-60`) carries only `worldId`/`channelId`/`mapId`/`instance`/`type`/`body`, no tenant field. Confirmed by `map/producer_test.go:39-52` pinning the exact key set (`worldId, channelId, mapId, instance, type, body`) |
| DOM-20 | Table-driven tests | FAIL | New `TestProcessorImpl_Exit_RearmsAndDestroysOnEmpty` (`map/processor_test.go:722`), `TestProcessorImpl_Exit_RearmIsPerFieldKey` (`:870`), `TestProcessorImpl_Exit_LogsRearm` (`:920`) all use ad hoc `t.Run` subtests with distinct inline setup, not a `tests := []struct{...}` table. `grep -n ":= \[\]struct" map/processor_test.go` — zero matches. |
| DOM-24 | producertest / injected no-op producer | PASS | `map/processor_test.go` injects a per-test fake producer via `mockProducerProvider`/`createTestProcessor` (`processor_test.go:120-166`) — satisfies the "inject a no-op producer per test" branch of DOM-24 |
| FILE-01 | Processor interface/methods in `processor.go` | PASS | `map/processor.go:31-41` interface, `:53` ctor, methods through `:186` |
| FILE-06 | No catch-all file | N/A | `map/producer.go` holds only `model.Provider[[]kafka.Message]` constructors (producer.go's documented responsibility); no second responsibility bundled in |

### `kafka/message/monster` (support — no `model.go`, no `resource.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-06 | No catch-all file | PASS | `kafka.go` holds only topic-token consts, command-type consts, and envelope/body structs — single responsibility (Kafka message-shape definitions), consistent with every sibling `kafka/message/*` package in the service |
| DOM-21 | No redeclared shared constant | PASS | `EnvCommandTopic`/`CommandTypeDestroyField`/`FieldCommand`/`DestroyFieldBody` are new to this package; `COMMAND_TOPIC_MONSTER` string value already exists service-wide (`tasks/mist_tick.go:104`) as a distinct local const — not a `libs/atlas-constants` redeclaration, and DOM-21's trigger is scoped to `libs/atlas-constants` collisions specifically |
| DOM-23 | Topic env var listed in configmap, both overlays, no literal `env:` | N/A | `COMMAND_TOPIC_MONSTER` is not new/renamed — already present at `deploy/k8s/base/env-configmap.yaml:68`, `deploy/k8s/overlays/main/kustomization.yaml:98`, `deploy/k8s/overlays/pr/kustomization.yaml:215`, `deploy/k8s/overlays/pr-sparse/kustomization.yaml:378` |

## Cross-service seam verification (DESTROY_FIELD)

- Consumer already exists in atlas-monsters and predates this diff: `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer.go:88,356-357` registers and dispatches `handleDestroyFieldCommand` on `CommandTypeDestroyField`.
- Envelope shape (`FieldCommand[E]`) matches field-for-field between `services/atlas-maps/.../kafka/message/monster/kafka.go:53-60` and `services/atlas-monsters/.../kafka/consumer/monster/kafka.go:252-259` (`worldId`, `channelId`, `mapId`, `instance`, `type`, `body` — no `transactionId` on either side).
- `DestroyFieldBody{}` (maps) matches `destroyFieldCommandBody struct{}` (monsters) at `.../kafka/consumer/monster/kafka.go:103`.
- Pinned by test: `map/producer_test.go:23-70` (`TestDestroyFieldCommandProvider_MatchesConsumerEnvelope`) asserts the exact JSON key set and values emitted.

## Prior-fix verification (per task instructions)

- **2c3cb80cd (tenant redis-key-helper violation):** Resolved. `registry.go:47-99` (`newRegistry`) builds `hashes`/`oneTime`/`meta` exclusively via `atlasredis.NewTenantKeyedHash[character.MapKey]`; `grep -n "client\.\(HGet\|HSet\|HDel\|HExists\|HLen\|HGetAll\)\b" registry.go registry_test.go` returns zero matches — no raw keyed redis client calls remain.
- **16fc28cde (RearmOneTime atomicity):** Resolved. `registry.go:275-289` defines `rearmOneTimeScript` (`redis.call('HDEL', KEYS[1], ARGV[1])`); `RearmOneTime` (`registry.go:475-484`) runs it as a single atomic Lua round trip and returns `deleted > 0` — no read-then-write `Exists`+`Del` pair remains.

## Not evaluable from the diff

- None. All applicable checklist items were settled from the changed files plus targeted symbol lookups (atlas-monsters consumer envelope, atlas-constants grep, deploy configmap grep, goroutine-guard run).

## Summary

### Blocking (must fix)

- DOM-20 (Testing): 19 of 20 new top-level test functions added across `map/monster/registry_test.go`, `map/monster/processor_test.go`, and `map/processor_test.go` are not table-driven (`t.Run` subtests with bespoke inline setup instead of `tests := []struct{...}`). Representative: `map/monster/registry_test.go:432,676`; `map/monster/processor_test.go:1177,1301,1328,1400,1443,1484`; `map/processor_test.go:722,870,920`.
- FILE-05 / FILE-06 (File placement): `map/monster/registry.go` bundles administrator-shaped writes (`InitializeForMap`, `ReserveEligibleSpawnPoints`, `ClaimOneTimeSpawnPoints`, `RearmOneTime`, `SetSpawnPointsForMap`, `FlushTenant`) and provider-shaped reads (`Count`, `CountOneTime`, `GetSpawnPointsForMap`) in one file with no `administrator.go`/`provider.go` split; this diff adds 4 more methods to the same collapsed file.
- DOM-16: `map/monster` package performs writes with no `administrator.go` — writes are inlined into `registry.go`.
- DOM-01: `map/monster/model.go` and `data/map/monster/model.go` both define a domain `Model` with no corresponding `builder.go`/`NewBuilder()`/`Build()` in either package.
- EXT-01: `data/map/monster/rest.go`'s `RestModel` (the atlas-data client model) is missing `SetToOneReferenceID`/`SetToManyReferenceIDs`.

### Non-Blocking (should fix)

- None beyond the blocking items above — all found gaps are structural/Important per guideline weight, not stylistic.

### Notes for the reviewer

- DOM-01, DOM-16, FILE-05/06, and EXT-01 above predate this diff's first commit (`b7cdb2716`) — they are not regressions this diff introduced. They are reported per the audit's Mindset rule ("prevalence/history does not exempt a deviation") because the diff substantially extends the same files (`registry.go` +4 methods, `rest.go`/`model.go` +1 field each) without correcting the pre-existing structural gap. DOM-20 is the one genuinely new violation: 19 non-table-driven tests were added by this diff itself.
