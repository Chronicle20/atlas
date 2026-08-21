# Backend Audit — task-255-auto-aggro-mobs

- **Service Path(s):** services/atlas-channel, services/atlas-monsters, services/atlas-configurations (seed-data only, no Go), libs/atlas-packet
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-21
- **Merge base / HEAD:** d17404dbc / b72b83c62
- **Build:** PASS (all three Go modules)
- **Tests:** PASS — atlas-channel, atlas-monsters, libs/atlas-packet all green (`-count=1`), no failures
- **Overall:** PASS

## Build & Test Results

```
cd services/atlas-channel/atlas.com/channel && go build ./...   -> exit 0, no output
cd services/atlas-channel/atlas.com/channel && go test ./... -count=1  -> all `ok`, no FAIL
cd services/atlas-monsters/atlas.com/monsters && go build ./... -> exit 0, no output
cd services/atlas-monsters/atlas.com/monsters && go test ./... -count=1 -> all `ok`, no FAIL
cd libs/atlas-packet && go build ./... -> exit 0
cd libs/atlas-packet && go test ./... -count=1 -> all `ok`, no FAIL
tools/goroutine-guard.sh -> exit 0 ("goroutineguard: 89 module(s), 8 parallel")
```

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (DOM-01..05,11,16) | Yes | `monster/model.go`, `monster/builder.go` (monsters), `monster/information/model.go`, `monster/information/rest.go` changed |
| FILE placement (FILE-01..06) | Yes | Every changed Go package (all packages, no exemption) |
| SUB (SUB-01..04) | No | No changed package has `resource.go` without `model.go` |
| REST (DOM-06..09,12..15,17..19,32) | Yes | `processor.go` changed in both `channel/monster` and `monsters/monster`; `information/rest.go` changed |
| Constants reuse (DOM-21) | Yes | New const blocks: `AutoAggroProximityThreshold` etc. (channel/monster/auto_aggro_gate.go), `AggroIdleThresholdMs`/`AutoAggroLeaseTtlMs` (monsters/monster/aggro.go), `CommandTypeSetAggro` (both kafka.go files) |
| Testing (DOM-10,20,24,33) | Yes | New/changed `_test.go` in every touched package; `Processor` interface changed (added `SetAggro`) in both services |
| Cache (DOM-29) | Yes | `AutoAggroGate`/`LiveMirror` hold cached state via `GetAutoAggroGate()`/`GetLiveMirror()` singletons |
| Messaging (DOM-30) | Yes | New `SetAggroCommandProvider`/`producer.ProviderImpl` call, new `p.emit(...)` call in `monsters/monster/processor.go` SetAggro |
| Multi-tenancy (DOM-31) | Yes | `AutoAggroGate`/`LiveMirror` are tenant-scoped; `SetAggro` reads `p.t` |
| Migration hygiene (DOM-34,35) | No | No diff moves symbols between a service and `libs/atlas-*` |
| Deploy & topics (DOM-22,23) | No | No new `libs/atlas-*` module; SET_AGGRO reuses existing `COMMAND_TOPIC_MONSTER` env var — confirmed present in `deploy/compose/.env.example:46` and `deploy/k8s/overlays/main/kustomization.yaml:99` pre-existing, untouched |
| Runtime safety (DOM-26) | Yes | Every changed non-test Go file; `tools/goroutine-guard.sh` exit 0 |
| Channel wire values (DOM-25) | Yes (opened, N/A on rule level) | Diff touches `services/atlas-channel` and `libs/atlas-packet`; see per-rule finding below |
| Resilience (DOM-27,28) | No | No `database.Connect`-backed handler writes 500 in changed files; no `model.Decorator` changed |
| External clients (EXT-01..04) | Opened but N/A — see rule table | `monster/information` package calls `requests.RootUrlFor("DATA")`/`requests.GetRequest`, but those call sites (`requests.go`, `cache.go`, `processor.go`) are untouched by this diff; only `model.go`/`builder.go`/`rest.go` field-plumbing changed |
| Scaffolding (SCAFFOLD-01..09) | Yes (SCAFFOLD-07 only) | Diff registers new atlas-channel handler `AutoAggroHandleFunc` in `main.go:912`; no new service directory |
| Security (SEC-01..04) | Opened (task-facts flag) but N/A on every rule | No token/JWT/redirect/secret handling touched in any changed file |

## Checklist Results

### libs/atlas-packet/monster/serverbound (packet codec, support package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File responsibilities | PASS | `auto_aggro.go` holds only the codec struct/Encode/Decode — no catch-all mixing (libs/atlas-packet/monster/serverbound/auto_aggro.go:41-71) |
| DOM-20 | Table-driven tests | N/A | `auto_aggro_test.go` is a single scenario + a version-matrix `pt.Variants` loop, not a `tests := []struct` table — but this exactly matches the sibling packet-test idiom (`pt.RoundTrip` loop is the table). Not flagged as FAIL since DOM-20's own check ("tests := []struct{...}" + t.Run) is a REST/domain-logic idiom; packet codec tests use the shared `pt.RoundTrip` harness instead. Recorded here as an observation, not scored against DOM-20 since no domain branching logic exists to enumerate. |
| DOM-26 | No bare `go` | PASS | No goroutine in this file; `tools/goroutine-guard.sh` exit 0 |

### services/atlas-channel/atlas.com/channel/monster (domain package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor ctor takes `logrus.FieldLogger` | PASS | `monster/processor.go:43` `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor` |
| DOM-33 | Interface change updates every mock | PASS | `Processor.SetAggro` added at `monster/processor.go:35`; `ProcessorMock.SetAggro` added at `monster/mock/processor.go` (git diff, +8 lines) in the same commit range |
| DOM-30 | AndEmit/message.Buffer for DB-backed writes | PASS (documented exception) | `SetAggro`/every sibling command (`ForceControl`, `ClearAggro`, etc.) in `monster/processor.go` call `producer.ProviderImpl` directly with no DB write on any path — this package holds no GORM-backed state; matches the documented "Operations over non-DB state" exception in `patterns-kafka.md` (cites `atlas-chairs` precedent) |
| DOM-29 | Cache reached via `GetCache()` singleton | PASS | `monster/auto_aggro_gate.go:56` `func GetAutoAggroGate() *AutoAggroGate` (sync.Once singleton, not constructed in a processor); same pattern for `GetLiveMirror()` at `monster/live_mirror.go:51` |
| DOM-31 | Tenant travels in context only | PASS | `AutoAggroGate.Admit`/`LiveMirror` methods take `tenant.Model` as a parameter (context-derived at call sites via `tenant.MustFromContext(ctx)`, e.g. `socket/handler/auto_aggro.go:51,62`), never as a REST-model/body field — `SetAggroCommandBody` (kafka/message/monster/kafka.go:133) carries only `CharacterId`, no tenant field |
| DOM-26 | No bare `go` outside guard marker | PASS | `monster/auto_aggro_gate.go:60` carries `//goroutine-guard:allow` with justification; `monster/live_mirror.go:55` likewise; `tools/goroutine-guard.sh` exit 0 |
| DOM-21 | No redeclared constant | PASS | Grepped `libs/atlas-constants/` for `Aggro`/`FirstAttack`/`SetAggro` — no hits; `AutoAggroProximityThreshold` etc. are genuinely new |
| FILE-01..06 | File responsibilities | PASS | `auto_aggro_gate.go` holds only the gate type + `Admit`/`EvictTenant`/`SweepStale`; `producer.go` holds only `*CommandProvider` functions; no catch-all file added |
| DOM-20 | Table-driven tests | PASS | `auto_aggro_gate_test.go:21` `tests := []struct{...}` + `t.Run`; `auto_aggro_test.go` (socket/handler) likewise at line 97 |
| DOM-24 | Producer stub / no-op injected in tests reaching emit | PASS | `socket/handler/auto_aggro_test.go:85` overrides package-level `autoAggroEmitFn` var with a recording no-op — the handler test never reaches a real `producer.ProviderImpl` call |

### services/atlas-channel/atlas.com/channel/socket/handler (support package, `resource.go`-equivalent: handler dispatch)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-13 | No cross-domain orchestration in handler | PASS | `AutoAggroHandleFunc` (`socket/handler/auto_aggro.go:35-69`) does only cheap local checks (character present, distance, mirror lookup, same-field, rate gate) then a single `autoAggroEmitFn` call; every authoritative decision is deferred to atlas-monsters (comment at line 32-34) |
| DOM-12 | No `os.Getenv()` in handler | PASS | No `os.Getenv` in `auto_aggro.go` |
| FILE-01..06 | File responsibilities | PASS | `auto_aggro.go` is handler-dispatch only, no processor/rest/entity logic embedded |

### services/atlas-channel/atlas.com/channel/kafka/consumer/monster (event-consumer package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-31 | Tenant from context only | PASS | Every handler resolves tenant via `tenant.MustFromContext(ctx)` (e.g. `consumer.go:143,197`); no tenant field added to any event body in this diff |
| FILE-01..06 | File responsibilities | PASS | `consumer.go` holds only Kafka handler functions and their seam vars; unchanged shape, additive lines only (`UpdateControl` calls, `monsterGetByIdFn`/`announceFn` indirection at `consumer.go:397,403`) |

### services/atlas-monsters/atlas.com/monsters/monster (domain package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` has `NewBuilder`, fluent setters, validating `Build()` | PASS | `builder.go` `SetAggroRefreshedMs` added following existing fluent pattern (git diff +9/-0) |
| DOM-06 | Processor ctor takes `logrus.FieldLogger` | PASS | Pre-existing `NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor` (processor.go), unchanged signature |
| DOM-33 | Interface change updates every mock | PASS (no mock exists) | `Processor.SetAggro` added at `processor.go:67` in the interface; no `monster/mock` directory exists in atlas-monsters (`find ... -iname "*mock*"` returns none under `monster/`), so there is no mock to update — confirmed by the same `find` returning only `information/mock`, `consumable/mock`, `drop/mock`, `mobskill/mock`, none of which implement `monster.Processor` |
| DOM-30 | AndEmit/message.Buffer atomicity | PASS (documented exception) | `SetAggro` (processor.go:1878-1979) writes through `GetMonsterRegistry().SetAggro(...)` (a Redis-backed atomic `Update`, not GORM) then calls `p.emit(EnvEventTopicMonsterStatus, ...)` directly at line 1955 — matches every sibling command (`ForceControl`, damage/kill paths) in the same file, all of which use the same non-DB registry + direct emit shape; documented exception in `patterns-kafka.md` ("Operations over non-DB state") applies |
| DOM-21 | No redeclared constant | PASS | `AggroIdleThresholdMs`, `AggroDecayMultiplier`, `AggroDecayFloor`, `AutoAggroLeaseTtlMs` (monster/aggro.go) — grepped `libs/atlas-constants/` for equivalents, none found |
| FILE-01..06 | File responsibilities | PASS | `aggro.go` holds only decay/lease constants + `IsAggroIdle`; `aggro_task.go` holds only the `MonsterAggroDecayTask` type and its `Run`/`SleepTime` — neither collapses ≥2 catch-all responsibilities |
| DOM-20 | Table-driven tests | PASS | `set_aggro_test.go:20` `TestSetAggro_Gates` uses `tests := []tc{...}` + `t.Run`; `aggro_task_test.go` and `registry_test.go` likewise (grep confirms `tests :=` pattern in both) |
| DOM-31 | Tenant from context only | PASS | `SetAggro(uniqueId, characterId uint32)` reads tenant via `p.t` (processor field populated from context in `NewProcessor`), never a parameter/body field; `setAggroCommandBody` (kafka.go:171) carries only `CharacterId` |

### services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster (event-consumer package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File responsibilities | PASS | `handleSetAggroCommand` added alongside every sibling `handle*Command` function in `consumer.go`; `setAggroCommandBody` added alongside sibling command bodies in `kafka.go` |
| DOM-20 | Table-driven tests | N/A | `kafka_test.go` additions (`TestSetAggroCommandUnmarshal`, `TestSetAggroCommandTypeConstant`) are single-scenario unmarshal/constant assertions, matching every sibling `TestXCommandBody_Decode` in the same file — no branching logic to enumerate as a table |

### services/atlas-monsters/atlas.com/monsters/monster/information (domain package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-02/03 | `ToEntity`/`Make`/`Extract` mapping complete | PASS | `Extract` (rest.go:104) now maps `firstAttack: rm.FirstAttack` — the `RestModel.FirstAttack` field already existed (`rest.go:35`, untouched), only the domain-model mapping was missing and is now added |
| DOM-01 | Builder fluent setter | PASS | `SetFirstAttack` added to `ModelBuilder` (builder.go) following the existing fluent pattern |
| DOM-20 | Table-driven tests | PASS | `rest_test.go` `TestExtractFirstAttack` uses `tests := []struct{...}` + `t.Run` |
| EXT-01..04 | Cross-service client conventions | N/A | The package does call `requests.RootUrlFor(ctx, "DATA")` (requests.go:15) and `requests.GetRequest[RestModel]` (cache.go:163), but neither of those files, nor `processor.go`, is touched by this diff — only `model.go`/`builder.go`/`rest.go`/`rest_test.go` (field plumbing) changed. No new or modified cross-service call site to grade. |

## Security Review

SEC-01..04 all opened per `tools/task-facts.sh` flag but every rule's own trigger is negative for this diff:

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SEC-01 | Verified JWT parsing | N/A | No token/JWT code in any changed file — grep for `token\|jwt\|redirect\|secret` in `auto_aggro_gate.go` and `socket/handler/auto_aggro.go` returns no hits |
| SEC-02 | Revocation reads validated claims | N/A | Same — no claims/token handling touched |
| SEC-03 | Redirect validation | N/A | No callback/redirect handler touched |
| SEC-04 | No hardcoded secrets | N/A | No secret-bearing code touched |

Untrusted-input handling specific to this feature (not a numbered SEC rule, but the task's stated focus) — traced by hand:

- The wire fields (`mobId`, `distance`) are bounded, unsigned 32-bit reads with no length-prefixed/variable-size decode (`libs/atlas-packet/monster/serverbound/auto_aggro.go:67-70`) — no buffer-overrun surface.
- `distance` (client-supplied) is used only as a numeric comparison against `AutoAggroProximityThreshold` (`socket/handler/auto_aggro.go:46`) — never used as an index, allocation size, or format-string argument.
- `mobId` (client-supplied) is used only as a map lookup key (`autoAggroMirrorLookupFn`, `GetAutoAggroGate().Admit`) and forwarded as a Kafka command field — never trusted as authoritative; atlas-monsters re-validates existence/liveness/aggressiveness/field-membership independently (`monster/processor.go:1878-1979`), so a forged `mobId` for a monster the claimant cannot see is rejected server-side by the field-membership gate (processor.go:1904-1921).
- The channel-side rate gate (`AutoAggroGate.Admit`) is a defense against a Kafka-storm DoS from a modified/scripted client sending AUTO_AGGRO far above the stock 1/sec rate; it is explicitly documented as non-authoritative (`auto_aggro_gate.go:41-43`) and atlas-monsters applies its own independent gates, so a bypass of the channel-side throttle (e.g., a hostile channel pod) degrades to atlas-monsters' own per-claim processing cost, not an unbounded control grant.
- SET_AGGRO's arbitration is fail-closed on every lookup failure (monster not found, template fetch error, field-list error all `return nil` — drop, no error propagated to the client) per `processor.go:1888,1908,1913,1934` — no case forwards a client-shaped error back to the emitting session (matches the PRD's FR-3.4 "no client-visible failure path").

## Not evaluable from the diff

- EXT-02 (httptest-backed integration test for the cross-service client) — the `monster/information` package's REST-fetch integration test coverage (`requests.go`/`cache.go`) is untouched by this diff and was not read in full; whether an `httptest` fixture already exists for the `DATA` client would require reading `cache_test.go` end-to-end, which is outside the changed-file surface.
- SCAFFOLD-09 (`tools/service-registration-guard.sh`) — this diff does not add a new service, so the rule's own trigger is negative; not run since it would report on unrelated repo state.
- packet-audit registry/status.json cross-checks (whether all ten `docs/packets/registry/*.yaml` and `docs/packets/audits/*/MonsterAutoAggro.*` entries are internally consistent with the opcode table) — out of scope for the backend-dev-guidelines rule set (no DOM/FILE/SUB/EXT/SCAFFOLD/SEC rule covers packet-audit bookkeeping); flagged only as an observation, not a finding.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- None identified beyond the observational notes above (DOM-20 packet-codec test shape, EXT-02 untouched-file gap) which are not rule violations.
