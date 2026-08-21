# Backend Audit — task-253-self-destructing-mobs

- **Service Path:** services/atlas-channel, services/atlas-data, services/atlas-monsters, services/atlas-monster-death, libs/atlas-packet
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-21
- **Build:** PASS
- **Tests:** Not re-run in full (task instruction: branch is green under flagless `tools/verify.sh`, exit 0 with bake); `go build ./...` re-verified clean for `atlas-monsters` module during this review.
- **Overall:** NEEDS-WORK

## Build & Test Results

`tools/verify.sh` (flagless, includes bake and `-race`) reported exit 0 per task
instructions and is not re-litigated here. This review additionally ran
`cd services/atlas-monsters/atlas.com/monsters && go build ./...` — clean, no
output.

## Applicability

Diff range `d17404dbc..77fc9bc64`, 54 files across atlas-channel, atlas-data,
atlas-monsters, atlas-monster-death, and libs/atlas-packet.

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (DOM-01..05,11,16) | Yes | `information/model.go`, `information/rest.go`, `information/builder.go`, `monster/model.go` changed |
| FILE placement (FILE-01..06) | Yes | every changed Go package, no exemptions |
| SUB (SUB-01..04) | Yes | `services/atlas-data/atlas.com/data/monster` has `resource.go`, no `model.go` — sub-domain classification; only `reader.go`/`reader_test.go` changed in that package |
| REST (DOM-06..09,12..15,17..19,32) | Yes | `information/rest.go` changed, `information/processor.go` present in changed package |
| Constants reuse (DOM-21) | Yes | new types `SelfDestructTrigger`, `SelfDestruction`, `DeathType*`/`DestroyType*` const blocks declared |
| Testing (DOM-10,20,24,33) | Yes | many new `_test.go` files, `monster.Processor` and `information` interfaces referenced by mocks |
| Cache (DOM-29) | Yes | `information` package (has `cache.go`, unmodified) is a changed package |
| Messaging (DOM-30) | Yes | `producer.go` files changed / `AndEmit`-style emit paths touched in `processor.go` |
| Multi-tenancy (DOM-31) | Yes | `rest.go` files changed; tenant read via `tenant.MustFromContext` in new registry/timer code |
| Migration hygiene (DOM-34,35) | No | no symbols moved between service and `libs/atlas-*` |
| Deploy & topics (DOM-22,23) | No | no new `libs/atlas-*` module, no new/renamed Kafka topic env var (SELF_DESTRUCT reuses existing `COMMAND_TOPIC_MONSTER` / `EVENT_TOPIC_MONSTER_STATUS`) |
| Runtime safety (DOM-26) | Yes | non-test Go files changed across all four services |
| Channel wire values (DOM-25) | Yes | diff touches `services/atlas-channel` and `libs/atlas-packet`; atlas-monsters (domain) emits a client dead-type byte |
| Resilience (DOM-27,28) | No | no changed HTTP handler writes `http.StatusInternalServerError`; no `model.Decorator`/enrichment path changed |
| External clients (EXT-01..04) | No | no `requests.go` changed, no new `requests.*Request[T]` call sites in the diff |
| Scaffolding (SCAFFOLD-01..09) | No | no new service, no new channel Writer/Handler constant, `deploy/shared/routes.conf` untouched |
| Security (SEC-01..04) | No | no auth/token/redirect/secret code touched |
| patterns-provider.md (foundational) | No | no new provider composition introduced |
| patterns-functional.md (foundational) | No | no new curried constructor/decorator pattern introduced beyond existing style already used pre-diff |

## Checklist Results

### services/atlas-monsters/atlas.com/monsters/monster (domain — has `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` has `NewBuilder`/fluent setters/validating `Build()` | PASS | `builder.go` unchanged by this diff, pre-existing compliant builder still used by new self-destruct tests (`self_destruct_test.go`) |
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | `services/atlas-monsters/atlas.com/monsters/monster/processor.go:135` `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor` |
| DOM-25 | Client-interpreted wire values resolved from tenant table, domain service emits semantic keys | **FAIL** | `services/atlas-monsters/atlas.com/monsters/monster/kafka.go:63-64` defines `DeathTypeUnset`/`DeathTypeFadeOut` as raw bytes explicitly documented (line 56-62) as mirroring `libs/atlas-packet` `DestroyType` (`CMob::m_nDeadType`); `services/atlas-monsters/atlas.com/monsters/monster/kafka.go:104-105,148` add `DeathType byte` fields to `statusEventDestroyedBody`/`statusEventKilledBody`; `services/atlas-monsters/atlas.com/monsters/monster/processor.go:600,687,796,1413` populate that field with `sd.Action()` (the raw WZ `selfDestruction.action` byte) or the `DeathTypeFadeOut`/`DeathTypeUnset` literals. This is the exact anti-pattern in `anti-patterns.md:136-166` — "A `byte` field carrying a client code in a Kafka event produced by a domain service is a finding" (verification point 3, `anti-patterns.md:236-238`). Compare the correct sibling pattern in the same file: `StatusEventCatchFailedBody.Cause string` (`services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go:294-298`), whose own comment states "the wire-reason mapping is resolved in atlas-channel, never emitted by atlas-monsters (DOM-25)" — the DeathType field does the opposite of what CatchFailed does two structs above it. |
| DOM-26 | Every goroutine via `routine.Go` | PASS | `tools/goroutine-guard.sh` exits 0 (ran during this review); only bare `go func()` in changed non-test surface is none — the sole hit is `self_destruct_registry_test.go:119`, a `_test.go` file, exempt from the trigger |
| DOM-30 | DB-writing operations emit via `AndEmit`/`message.Buffer`, not a bare `producer.ProviderImpl` from the success path | N/A | `SelfDestruct`/`finalizeKill` write to the Redis-backed `Registry`, not a GORM database; the emit-after-mutation shape (`p.emit(...)` after `GetMonsterRegistry().SelfDestruct(...)`) is the pre-existing pattern this file already used for every other kill path (`damageCore`, `Kill`) before this diff, unchanged by task-253 |
| DOM-31 | Tenant/trace travel in context only | PASS | `services/atlas-monsters/atlas.com/monsters/monster/self_destruct_timer_registry.go:73` `Register(ctx context.Context, t tenant.Model, ...)` — tenant passed as a typed context/param, never on a REST model or request body |
| FILE-01 | Processor lives in `processor.go` | PASS | `SelfDestruct`/`selfDestructFrom`/`finalizeKill`/`monsterInformation` all added to `services/atlas-monsters/atlas.com/monsters/monster/processor.go` |
| FILE-05 | Builder/Model/writes/readers placed per file table | PASS | `SelfDestruct` write lives in `registry.go:494-524` (registry already owns every other atomic transition — `ApplyDamage`, `ApplyRecovery` — in this file); `SelfDestructTimerEntry`/`SelfDestructTimerRegistry` in their own `self_destruct_timer_registry.go`; `SelfDestructTimerTask` in its own `self_destruct_timer_task.go` |
| FILE-06 | No catch-all file bundling ≥2 responsibilities | PASS | new files (`self_destruct_timer_registry.go`, `self_destruct_timer_task.go`) each carry a single responsibility (registry, sweep task) — no new file mixes Processor + RestModel + requests |
| DOM-33 | Interface changes update every mock | PASS | `Processor.SelfDestruct` added at `processor.go:68`; grep confirms no external mock of `monster.Processor` exists in this service (only `atlas-channel`'s separate `monster.Processor` interface has one, and its mock was updated — see below) |

### services/atlas-monsters/atlas.com/monsters/monster/information (domain — has `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` fluent + validating `Build()` | PASS | `information/builder.go:59-64` adds `SetSelfDestruction`; `Build()` at `information/builder.go:74-84` still assembles the immutable `Model` |
| DOM-04 | `Transform(Model) (RestModel, error)` in `rest.go` | PASS | pre-existing `Transform` in `information/rest.go` untouched, still compiles with the new field (verified via `go build`) |
| DOM-06 | Processor ctor takes `logrus.FieldLogger` | PASS | `information/processor.go:22` `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor` (unchanged file, still compliant) |
| DOM-18/19 | RestModel implements JSON:API methods; request models flat | PASS | `information/rest.go:76,80` `GetName()`/`GetID()` unaffected by the new `SelfDestruction selfDestruction` field |
| DOM-21 | No redeclaration of a `libs/atlas-constants` type | PASS | `find libs/atlas-constants -iname "*monster*" -o -iname "*destruct*"` returns only `libs/atlas-constants/monster` (unrelated); no `SelfDestruction`/`DeathType`/`DestroyType` equivalent exists there |
| DOM-29 | Caches are singletons via `GetCache()` | PASS | `information/cache.go` untouched by this diff; no new cache construction site added |
| FILE-05 | Model/Builder placed per file table | PASS | `information/model.go:1-54` adds `SelfDestruction` struct + `Model.SelfDestruction()` accessor; `information/builder.go` mirrors it |

### services/atlas-data/atlas.com/data/monster (sub-domain — has `resource.go`, no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SUB-01..04 | Business logic, writes, POST registration, JSON parsing in `resource.go` | N/A | `resource.go` was not touched by this diff; only `reader.go`/`reader_test.go` (WZ XML parsing) changed, which SUB-01..04 does not govern |
| FILE-05 | Readers placed per file table | Not evaluable from the diff | `reader.go` (not `provider.go`) is atlas-data's established WZ-parsing file across the whole service; whether that pre-existing service-wide naming choice is itself FILE-05-compliant predates this diff (only 2 lines changed) and would require surveying atlas-data's full convention beyond the review surface |

### services/atlas-channel/atlas.com/channel/monster (domain — has `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor ctor takes `logrus.FieldLogger` | PASS | `services/atlas-channel/atlas.com/channel/monster/processor.go` — `SelfDestruct` method added to the existing `ProcessorImpl`, constructor unchanged |
| DOM-33 | Interface change updates every mock | PASS | `Processor.SelfDestruct` added at `processor.go:33`; `mock/processor.go:26,138-143` adds `SelfDestructFunc` field and `SelfDestruct` method with the nil-check default |
| DOM-25 | Client wire codes not hardcoded outside packet internals | PASS (this package) | `SelfDestructCommandProvider` (`producer.go:191-208`) carries only `CharacterId uint32`, no client byte; the DOM-25 violation lives downstream in the consumer (see atlas-channel kafka/consumer/monster below) |

### services/atlas-channel/atlas.com/channel/kafka/consumer/monster (support — no `model.go`, no `resource.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-25 | Client wire codes resolved from a tenant writer-options table | **FAIL** | `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go:216-221` `destroyTypeFor(deathType byte) monsterpkt.DestroyType { ... return monsterpkt.DestroyType(deathType) }` — a raw cast of the byte the domain service emitted, with no `WithResolvedCode(...)`/tenant table lookup. Compare the established correct pattern elsewhere in the same service: `services/atlas-channel/atlas.com/channel/socket/writer/claim_result.go:24,30` and `sue_character_result.go:21` both call `atlas_packet.WithResolvedCode("operations", ...)`. This site does not. |
| DOM-26 | Goroutines via `routine.Go` | PASS | `tools/goroutine-guard.sh` exit 0; no bare `go` statement added in this file |

### libs/atlas-packet/monster/clientbound (codec internals)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-25 | No client wire code as a Go literal outside `libs/atlas-packet` codec internals | PASS | `DestroyType`/`DestroyTypeBomb`/etc. constants and `hasSwallowCharacterId` version-gating (`destroy.go:14-53`) live entirely inside `libs/atlas-packet` — the codec-internal exemption in `anti-patterns.md:242-244` covers this file itself. The FAIL is the two upstream sites (atlas-monsters domain service, atlas-channel consumer) that hand this codec a raw byte instead of a semantic key. |

## Security Review

SEC-* family did not fire — no auth/token/redirect/secret-handling code in the diff. N/A.

## Not evaluable from the diff

- FILE-05 (atlas-data `reader.go` vs `provider.go` naming): whether atlas-data's service-wide WZ-parsing file convention satisfies FILE-05's reader-placement wording predates this diff; would need to read atlas-data's full file layout beyond the two changed files (`reader.go`, `reader_test.go`) to settle.
- DOM-20 (table-driven tests) for single-scenario test functions in `self_destruct_test.go` (`TestSelfDestructAttributesToDamageLeader`, `TestSelfDestructNoDamageEntriesReportsNoKiller`, `TestSelfDestructIsIdempotent`) and `self_destruct_dot_test.go`: these are single-case tests with no natural table, and the guideline's pass criterion ("tests use the table-driven pattern") does not state whether a single-scenario test is exempt — not settled either way without a broader read of how DOM-20 has been graded on similar single-case tests elsewhere in the codebase, which is outside this diff's surface.

## Summary

### Blocking (must fix)
- DOM-25: `services/atlas-monsters/atlas.com/monsters/monster/kafka.go:63-64,104-105,148` and `services/atlas-monsters/atlas.com/monsters/monster/processor.go:600,687,796,1413` — atlas-monsters (a domain, non-channel service) emits `DeathType byte`, a raw client dead-type/`DestroyType` wire code, directly on `KILLED`/`DESTROYED` Kafka events instead of a semantic key.
- DOM-25: `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go:216-221` — `destroyTypeFor` casts that raw byte straight into `monsterpkt.DestroyType` with no tenant writer-options table resolution (`WithResolvedCode`), unlike the established pattern in `claim_result.go`/`sue_character_result.go` in the same service.

### Non-Blocking (should fix)
- None identified beyond the items listed under "Not evaluable from the diff."
