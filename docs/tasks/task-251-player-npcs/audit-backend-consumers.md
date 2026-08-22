# Backend Audit — task-251-player-npcs (backend-consumer modules)

- **Service Path(s):** services/atlas-saga-orchestrator, services/atlas-channel, services/atlas-query-aggregator, services/atlas-tenants, services/atlas-messages, services/atlas-data, services/atlas-configurations, libs/atlas-saga, libs/atlas-packet, libs/atlas-object-id, libs/atlas-constants
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-22
- **Build:** PASS (all 11 modules)
- **Tests:** all packages `ok` or `no test files`; zero failures
- **Overall:** NEEDS-WORK
- **Explicitly out of scope:** `services/atlas-player-npcs` (audited separately)

## Build & Test Results

```
cd <module> && go build ./...   → clean, no output, for all 11 modules
cd <module> && go test ./... -count=1 → all `ok` or `[no test files]`, zero FAIL, for all 11 modules
```

Verbatim per-module test summaries are in the session transcript; no failing
package in any of the 11 modules.

`tools/goroutine-guard.sh` → exit 0 (DOM-26, repo-wide, machine-checked).

## Applicability

| Family | Fired? | Trigger observed |
|---|---|---|
| DOM structure (01-05,11,16) | Yes | `model.go` in atlas-channel/playernpc; `rest.go`/`provider.go` in atlas-tenants/configuration, atlas-data/map, atlas-data/npc |
| FILE placement (01-06) | Yes | every changed Go package, unconditional |
| SUB sub-domain (01-04) | No | no changed package has `resource.go` without `model.go` — atlas-tenants/configuration has both `resource.go` and (implicitly, via `Model`) domain state managed through `processor.go`/`entity.go`, i.e. it is a domain package, not sub-domain; no bare action-event package touched |
| REST (06-09,12-15,17-19,32) | Yes | `resource.go`/`rest.go`/`processor.go` in atlas-tenants/configuration, atlas-data/map, atlas-channel/playernpc, saga-orchestrator/playernpc |
| Constants reuse (DOM-21) | Yes | new types/consts: `playernpc.EligibilityModel`, `saga.DeployPlayerNpc`, `job.MaxLevelFor`, `objectid.PlayerNpcObjectIdBase` |
| Testing (10,20,24,33) | Yes | every module's diff touches `_test.go`; saga `Handler` interface gains a method; several test packages reach `AndEmit`/producer paths transitively |
| Cache (DOM-29) | No | no changed package has `cache.go`, no processor holds cached state (saga's existing `saga.Cache`/`GetCache()` singleton is read, not redefined) |
| Messaging (DOM-30) | Yes | `producer.go` touched in saga-orchestrator (`DeployPlayerNpcCommandProvider`); `AndEmit`/`message.Buffer` pattern used in atlas-tenants/configuration's new Player NPC config handlers |
| Multi-tenancy (DOM-31) | Yes | `rest.go` changed in atlas-tenants/configuration, atlas-data/map, atlas-channel/playernpc, atlas-query-aggregator/playernpc; tenant context read throughout consumers |
| Migration hygiene (34,35) | No | no symbol moved between a service and a `libs/atlas-*` module in this diff — new libs code is net-new, not extracted |
| Deploy & topics (22,23) | Partial | no new `libs/atlas-*` module added (DOM-22 N/A); `COMMAND_TOPIC_PLAYER_NPC`/`EVENT_TOPIC_PLAYER_NPC_STATUS` are consumed here but were not newly introduced by this diff's in-scope modules — confirmed already present in `deploy/k8s/base/env-configmap.yaml` and both `main`/`pr` overlays (pre-existing from the topic's own shipping task) |
| Runtime safety (DOM-26) | Yes | non-test Go files changed throughout; `routine.Go` used correctly in atlas-channel; `tools/goroutine-guard.sh` exit 0 |
| Channel wire values (DOM-25) | Yes | diff touches `services/atlas-channel` and `libs/atlas-packet` (new `ImitatedNpcData`/`Remove` writers) |
| Resilience (27,28) | Yes | atlas-tenants/configuration and atlas-data/map are DB-backed services with new handlers |
| External clients (EXT-01..04) | Yes | atlas-channel/playernpc, atlas-query-aggregator/playernpc, saga-orchestrator/playernpc all call another atlas service |
| Scaffolding (01-09) | Partial | no new service added (01-03,06,08,09 N/A); SCAFFOLD-07 fires — new atlas-channel writers `ImitatedNPCData`/`RemoveNPC` registered |
| Security (SEC-01..04) | No | none of the in-scope modules handle auth tokens, login, or secrets in this diff |

## Checklist Results

### services/atlas-saga-orchestrator — `saga` (domain), `playernpc` (domain, read client), `kafka/consumer/playernpc` (support), `kafka/message/playernpc` (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | `playernpc/processor.go:16` `NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor` |
| DOM-33 | Interface change updates every mock | PASS | `saga.Handler` gained `WithPlayerNpcLocationProcessor`/`handleDeployPlayerNpc`; sole implementation `HandlerImpl` updated in the same diff (`saga/handler.go:101,742`); `playernpc.Processor` gained by this diff, its only mock `playernpc/mock/processor.go` added in the same diff with a matching signature (`playernpc/mock/processor.go:20`) |
| DOM-24 | producertest stub installed where an emit path is reached | PASS | `kafka/consumer/playernpc/testmain_test.go:11` `producertest.InstallNoop()`; `saga/handler_test.go` new subtest uses `producertest.InstallCapturing()` (line ~1045) |
| DOM-20 | Table-driven tests | WARN | `saga/handler_test.go:1007` `TestDeployPlayerNpcAction` uses ad-hoc `t.Run` blocks, not a `tests := []struct{...}` table; same for `kafka/consumer/playernpc/consumer_test.go`'s four `Test...` functions |
| EXT-01 | Target RestModel implements SetToOneReferenceID/SetToManyReferenceIDs | PASS | `playernpc/requests.go:39-41` (`LocationRestModel`) |
| EXT-02 | httptest-backed integration test | N/A | package has no dedicated `requests_test.go`/httptest coverage, but `LocationRestModel` is exercised indirectly only through `handler_test.go`'s fake `playernpcmock.ProcessorMock` — **see Not evaluable** below; the interface boundary is mocked, not the wire decode, so strictly this is the same EXT-02 gap noted for query-aggregator (recorded there as the primary finding; not double-counted here since the deploy_player_npc test suite mocks at the `Processor` interface layer by design, consistent with `handler_test.go`'s existing pattern for every other injected processor) |
| EXT-03 | Only genuine 404s map to "not found" | PASS | `playernpc/processor.go:30-33` returns the raw `err` from `requestLocationByCharacterId` unmodified — no not-found translation exists to mis-classify a transport/5xx error |
| EXT-04 | URL composed via `requests.RootUrlFor` | PASS | `playernpc/requests.go:31` `requests.RootUrlFor(ctx, "MAPS")` |
| DOM-31 | Tenant/trace only via context | PASS | `playernpc/requests.go` takes `ctx` and never a tenant field on `LocationRestModel` |
| FILE-01 | Processor in `processor.go` | PASS | `playernpc/processor.go` |
| FILE-02 | RestModel/Extract in `rest.go`... wait, restmodel is in `requests.go` here | WARN | `LocationRestModel` and its JSON:API methods live in `requests.go` (`playernpc/requests.go:29-41`), not a separate `rest.go` — this package has no `rest.go` at all, so FILE-02's placement rule is not met by file name, though the content itself is otherwise compliant (JSON:API methods present) |
| FILE-06 | No catch-all file | WARN | `requests.go` in this package carries both FILE-02's responsibility (RestModel + JSON:API methods) and FILE-03's (request functions) — two of the six responsibilities in one file |
| DOM-25 | Client wire values from writer-options table | N/A | this package never encodes a client packet |

### services/atlas-channel — `playernpc` (domain, read client), `kafka/consumer/playernpc` (support), `kafka/consumer/map` player_npc.go (support, existing package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` has `NewBuilder()`, fluent setters, validating `Build()` | **FAIL** | `playernpc/builder.go:1225-1252` `Build()` unconditionally assembles `Model{}` from builder fields with no invariant check and no error return — no validation of any kind |
| DOM-04 | `Transform(Model) (RestModel, error)` in `rest.go` | N/A | this package is a pure REST-client (never serves its own `player-npcs` resource); only the reverse direction (`Extract`, `rest.go:65`) applies. FILE-02 (file-responsibilities.md:212) explicitly accepts `Extract(` as satisfying evidence for a REST-client package's `rest.go`; the sibling `kite/` package (pre-existing, same shape) follows the identical pattern |
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | `playernpc/processor.go:20` |
| DOM-18 | RestModel implements JSON:API interface | PASS | `playernpc/rest.go:69-88` `GetName`/`GetID`/`SetID` |
| DOM-19 | Request models flat | PASS | `RestModel` (`playernpc/rest.go:31-56`) is flat |
| DOM-25 | Client wire values via writer-options table, no literal opcodes | PASS | `libs/atlas-packet/npc/clientbound/imitated_npc_data.go:14` and `remove.go:14` expose semantic `Operation()` names (`NpcImitatedDataWriter`/`NpcRemoveWriter`), never a literal opcode; opcode resolution happens through the tenant socket writer table (`services/atlas-configurations` seed templates) |
| DOM-31 | Tenant/trace only via context | PASS | `playernpc/requests.go:15-17` `requests.RootUrlFor(ctx, "PLAYER_NPCS")`; no tenant field on `RestModel` |
| FILE-01/02/03/05/06 | File placement | PASS | `builder.go`/`model.go`/`processor.go`/`requests.go`/`rest.go` each carry exactly one responsibility |
| EXT-01 | Target RestModel implements SetToOneReferenceID/SetToManyReferenceIDs | PASS | `playernpc/rest.go:79-85` |
| EXT-02 | httptest-backed integration test, populated struct assertion | PASS | `playernpc/rest_test.go:1751-1823` `TestForEachInMap_RequestsByMapAndWorld` serves a representative JSON:API fixture via `httptest.NewServer` and asserts a populated `[]Model` |
| EXT-03 | Only genuine 404s map to "not found" | PASS | no not-found translation exists in `processor.go`/`requests.go` — errors propagate unmodified |
| EXT-04 | URL via `requests.RootUrlFor` | PASS | `playernpc/requests.go:15-17` |
| DOM-26 | Goroutines via `routine.Go` | PASS | `kafka/consumer/map/consumer.go:9-13` `routine.Go(l, ctx, func(_ context.Context) {...})` |
| SCAFFOLD-07 | New writers seeded in every targeted tenant template | PASS | `services/atlas-configurations/atlas.com/configurations/seed_template_writers_test.go` asserts the exact opcode (or deliberate absence) per template and passes; `git diff` confirms `ImitatedNPCData`/`RemoveNPC` added to gms_61/72/79/83/84/87/92/95 and jms_185, `RemoveNPC`-only on gms_48, neither on gms_12 — all three gaps are evidenced deliberate exclusions (test comment: "or absent where no opcode is evidenced"), not omissions |
| DOM-20 | Table-driven tests | WARN | `kafka/consumer/playernpc/consumer_test.go:810` `TestPlayerNpcStatusConsumer` and `kafka/consumer/map/player_npc_test.go:216` `TestSpawnPlayerNpcForSession` use ad-hoc `t.Run` blocks rather than a `tests := []struct{...}` table |
| DOM-24 | producertest stub where an emit path is reached | N/A | these consumers only broadcast to sessions (`session.Announce`) and read HTTP; they never emit a Kafka message, so no stub is required |

### services/atlas-query-aggregator — `playernpc` (support, read client), `validation` (domain, existing)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-05 | Domain `Model` lives in `model.go` | **FAIL** | `playernpc/processor.go:19-25` defines `EligibilityModel` (a domain type, with constructors `NewEligibilityModel`/`NewUnavailableEligibility`) inside `processor.go`; the package has no `model.go` at all |
| EXT-01 | Target RestModel implements SetToOneReferenceID/SetToManyReferenceIDs | N/A | `EligibilityRestModel` (`playernpc/rest.go:9-12`) is deliberately not JSON:API — the endpoint returns a plain body (`rest.go` doc comment) and is decoded with `encoding/json`, not `jsonapi.Unmarshal`, so the JSON:API reference-ID interface has no application here |
| EXT-02 | httptest-backed integration test, populated struct assertion | **FAIL** | package `playernpc/` contains only `processor.go`, `requests.go`, `rest.go` — no `_test.go` file exists anywhere in the package; `requestEligibility`'s HTTP call and `json.Unmarshal` path is never exercised by a test. The only coverage is `validation/model_test.go`'s `fakePlayerNpcProcessor`, which satisfies the `playernpc.Processor` *interface* directly and never touches the real HTTP/decode path — precisely the "FakeClient mocks alone do NOT satisfy this" case the rule calls out |
| EXT-03 | Only genuine 404s map to "not found" | PASS | `playernpc/requests.go:149-151` treats any non-200 status as a generic wrapped error; no not-found translation exists to misclassify a 5xx/transport failure |
| EXT-04 | URL via `requests.RootUrlFor` | PASS | `playernpc/requests.go:107-109` |
| DOM-33 | Interface change updates every mock | PASS | `validation.ValidationContext`/`ValidationContextBuilder` are not interfaces (no mock needed); `playernpc.Processor` is new in this diff, and its only consumer test double, `fakePlayerNpcProcessor` (`validation/model_test.go:452-463`), implements it correctly in the same diff |
| DOM-28 | Fallible enrichment degrades loudly (`ErrDecorator`/`degrade.Observe`) | WARN | `validation/context.go:243-260` `GetPlayerNpcEligibility` degrades to a fail-closed `EligibilityModel` on error but only logs via `ctx.l.Warnf` — it does not use `model.ErrDecorator`/`degrade.Observe`. This mirrors every other `GetX`/`WithX` method already in `ValidationContext` (e.g. `GetTransportState`), which use the same bare-log-and-degrade shape; DOM-28's own trigger is "a `model.Decorator` or an enrichment fallback that fetches remote data" — `ValidationContext`'s condition evaluators are not `model.Decorator` implementations, so DOM-28 likely does not literally apply, but the fetch-remote-and-degrade shape is exactly what DOM-28 describes. Recorded as WARN pending the "Not evaluable" note below |
| DOM-31 | Tenant/trace only via context | N/A | package has no `rest.go` in the JSON:API-handler sense (`EligibilityRestModel` is a plain decode target); `characterId`/`mapId`/`worldId` travel as explicit function parameters sourced from the caller's already-resolved `character.Model`, not a public request surface |

### services/atlas-tenants — `configuration` (domain, existing package extended)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | `configuration/processor.go:250` (pre-existing, unchanged signature) |
| DOM-08 | POST/PATCH via `RegisterInputHandler[T]` | PASS | `configuration/resource.go:822,833-834` `registerPlayerNpcConfigInputHandler := rest.RegisterInputHandler[PlayerNpcConfigRestModel](l)(si)` |
| DOM-17 | Domain errors map to correct HTTP status | PASS | `configuration/resource.go:689-691` (404 on `gorm.ErrRecordNotFound`), `:768-771` (404 on update-not-found), `:721` (400 on extract/validation failure) |
| DOM-18 | RestModel implements JSON:API interface | PASS | `configuration/rest.go:866-877` `GetID`/`SetID`/`GetName` |
| DOM-19 | Request models flat | PASS | `PlayerNpcConfigRestModel` (`configuration/rest.go:855-864`) is flat |
| DOM-27 | Transient DB errors → 503 via `server.WriteErrorResponse` | PASS | `configuration/resource.go:694,701,727,735,741,773,779,785,805` all route non-404 errors through `server.WriteErrorResponse` |
| DOM-30 | DB write + emit via `AndEmit`/`message.Buffer` | PASS | `configuration/processor.go:538-555` `CreatePlayerNpcConfigAndEmit` uses `message.EmitWithResult[...](outbox.EmitProvider(...))`, and the inner `CreatePlayerNpcConfig(mb)` calls `mb.Put(EventTopicConfigurationStatus, ...)` (line 510/528) — no direct `producer.ProviderImpl` call from a write success path |
| DOM-31 | Tenant/trace only via context | **FAIL** | `configuration/resource.go:683,716,757,801` — `GetPlayerNpcConfigHandler`/`CreatePlayerNpcConfigHandler`/`UpdatePlayerNpcConfigHandler`/`DeletePlayerNpcConfigHandler` all resolve the target tenant via `rest.ParseTenantId(...)` reading the `{tenantId}` path segment (`resource.go:832-835`'s route registration), not `tenant.MustFromContext(ctx)`. This is the exact shape `patterns-multitenancy-context.md`'s DOM-31 verification procedure calls a FAIL ("Grep... for route patterns... that carry a tenant ({tenantId}...)... A tenant field on a public REST model, request body, path, or query parameter is a FAIL"). The new code is not a new pattern — it precisely mirrors every pre-existing sibling handler in this same file (rankings, kite-configs, etc.), all of which share the identical `{tenantId}` path shape; per this audit's mindset, prevalence across the file does not exempt the new code from the same grading |
| DOM-33 | Interface change updates every mock | PASS | `configuration.Processor` gains 8 new methods; its only mock, `configuration/mock/processor.go`, implements all 8 in the same diff (lines 96-188) with the documented nil-check default pattern |
| DOM-20 | Table-driven tests | PASS | `configuration/player_npc_config_handler_test.go` is a single linear scenario (create→get→update→get→delete), matching the sibling `rankings_handler_test.go`'s own non-table shape it explicitly mirrors — not evaluated as a table-driven violation since it is a wire round-trip integration test, not a matrix of input/output cases |
| FILE-01..06 | File placement | PASS | new code lands in the package's existing `kafka.go`/`mock/processor.go`/`processor.go`/`provider.go`/`resource.go`/`rest.go` split, each carrying exactly one responsibility, consistent with every pre-existing sibling config (rankings, kite-configs) |

### services/atlas-messages — `command/playernpc` (support), `kafka/consumer/playernpc` (support), `kafka/message/playernpc` (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01/03/06 | File placement | PASS | `commands.go` (command-executor logic, matching the existing `command/monster` shape), `consumer.go`, `kafka.go` each carry one responsibility |
| DOM-26 | Goroutines via `routine.Go` | N/A | no goroutine spawned in this diff's messages files |
| DOM-24 | producertest stub where an emit path is reached | N/A | `commands_test.go`/`consumer_test.go` inject fake `commandPublisher`/`pinkTextSender` closures and never call the real `producer.ProviderImpl`/`message.NewProcessor(...).IssuePinkText` — the real emit path is never reached by these tests, so no stub is required |
| DOM-20 | Table-driven tests | WARN | `commands_test.go:229` `TestPlayerNpcCommands` uses ad-hoc `t.Run` blocks; `commands_test.go:417` `TestRemoveCommandProducer_Gate` and `kafka/consumer/playernpc/consumer_test.go:727` `TestFailureCodesProduceDistinctSentences` **do** use the `tests := []struct{...}` table pattern correctly |
| DOM-25 | Client wire values via writer-options table | N/A | this diff's messages files never encode a client packet (pink text is a string payload on an existing writer, not new wire-value logic) |

### services/atlas-data — `map` (domain, existing), `npc` (domain, existing)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-08 | POST route via `RegisterInputHandler[T]` | PASS | `map/resource.go:9` `rest.RegisterInputHandler[GroundRequestRestModel](l)(si)("get_map_ground", ...)` |
| DOM-17 | Domain errors map to correct HTTP status | PASS | `map/resource.go:23` 400 on empty points (`server.WriteBadRequest`); `:31` `server.WriteErrorResponse` on lookup failure |
| DOM-18 | RestModel implements JSON:API interface | PASS | `map/rest.go:198-238` `GroundRequestRestModel`/`GroundResultRestModel` both implement `GetName`/`GetID`/`SetID` |
| DOM-19 | Request models flat | PASS | `GroundRequestRestModel`/`GroundPointRestModel` (`map/rest.go:188-196`) are flat |
| FILE-02 | RestModel/JSON:API methods in `rest.go` | PASS | `map/rest.go:188-238` |
| DOM-27 | Transient DB errors → 503 | PASS | `map/resource.go:31` `server.WriteErrorResponse` |
| DOM-31 | Tenant/trace only via context | PASS | no tenant field added to any new RestModel; `mapId` is a path segment (pre-existing pattern for this resource-id, not a tenant identifier — DOM-31 governs tenant/trace only) |
| DOM-20 | Table-driven tests | WARN | `map/resource_ground_test.go:114` `TestHandleGetMapGroundRequest` and `npc/reader_test.go:286` `TestNpcReadImitateFlag` both use ad-hoc `t.Run` blocks, not a `tests := []struct{...}` table |

### libs/atlas-saga, libs/atlas-packet, libs/atlas-object-id, libs/atlas-constants (support/leaf)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | No redeclaration of an existing shared constant/type | PASS | `saga.DeployPlayerNpc`/`DeployPlayerNpcPayload`, `job.MaxLevelFor`, `objectid.PlayerNpcObjectIdBase`/`PlayerNpcObjectIdFor` are all genuinely new concepts, not duplicates of an existing constant elsewhere — `job.MaxLevelFor`'s own doc comment (`libs/atlas-constants/job/max_level.go:2-5`) explicitly cites and does not duplicate `atlas-character`'s existing `MaxLevel = 200` |
| DOM-25 | Client wire values via writer-options table | PASS | `libs/atlas-packet/npc/clientbound/imitated_npc_data.go`/`remove.go` — see atlas-channel section above |
| DOM-20 | Table-driven tests | PASS | `libs/atlas-constants/job/max_level_test.go:6` uses the `tests := []struct{...}` + `t.Run` table; `libs/atlas-object-id/reserved_test.go` uses discrete `t.Run` blocks without a struct table (WARN, minor — five independent single-assertion cases, not a data matrix) |
| DOM-22 | New `libs/atlas-*` module wired into Dockerfile/go.work | N/A | no new `libs/` module added — all four are additions to already-registered modules |

## Not evaluable from the diff

- EXT-02 for `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/playernpc` (the atlas-maps location client): whether the mocked-interface test coverage in `saga/handler_test.go` is an acceptable substitute for a dedicated httptest-backed decode test would require reading how the sibling `foothold`/`saved_location` packages in the same service are tested, to confirm whether an interface-level mock is this service's established substitute for a per-package httptest fixture — out of the diff's own files.
- DOM-28 applicability to `atlas-query-aggregator/validation/context.go`'s `GetPlayerNpcEligibility`: settling whether `ValidationContext`'s `GetX` methods are meant to be graded as `model.Decorator` enrichment paths under DOM-28, or are structurally exempt as condition-evaluator helpers, would require reading `patterns-resilience.md`'s full DOM-28 definition against every sibling `GetX` method already in this file (pre-existing, out of this diff's scope) to establish the service's own precedent.
- Whether `atlas-player-npcs`' actual `CommandDeployBody`/`CommandRemoveBody`/`StatusEvent` wire shapes exactly match the three independent leaf copies added here (`atlas-saga-orchestrator`, `atlas-channel`, `atlas-messages`) is explicitly out of scope — `services/atlas-player-npcs` is excluded from this audit and owned by a separate reviewer.
- Full line-by-line cross-check of every opcode in `seed_template_writers_test.go`'s table against each of the 10 `docs/packets/registry/<version>.yaml` files was not performed — the test itself passing, plus spot-checks confirming `docs/packets/registry/gms_v48.yaml` exists and `docs/packets/registry/gms_v12.yaml` does not, was treated as sufficient corroboration for SCAFFOLD-07.

## Summary

### Blocking (must fix)

- DOM-01: `services/atlas-channel/atlas.com/channel/playernpc/builder.go:1225-1252` — `Build()` performs no invariant validation and returns no error.
- FILE-05: `services/atlas-query-aggregator/atlas.com/query-aggregator/playernpc/processor.go:19-25` — domain type `EligibilityModel` is defined in `processor.go` instead of a `model.go`.
- EXT-02: `services/atlas-query-aggregator/atlas.com/query-aggregator/playernpc/` — no test file at all in the package; the real HTTP/JSON decode path (`requestEligibility`) is never exercised, only a hand-rolled interface-level fake in a different package's test file.
- DOM-31: `services/atlas-tenants/atlas.com/tenants/configuration/resource.go:683,716,757,801` — the four new Player NPC config handlers resolve tenant identity from the `{tenantId}` URL path via `rest.ParseTenantId`, not `tenant.MustFromContext(ctx)`; this matches every pre-existing sibling handler in the same file, but is graded on its own merits per this audit's mindset.

### Non-Blocking (should fix)

- FILE-02/FILE-06: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/playernpc/requests.go` — `LocationRestModel` and its JSON:API methods live in `requests.go`, not a `rest.go`; the file carries two of the six FILE-* responsibilities.
- DOM-20 (table-driven tests): multiple new test functions across atlas-saga-orchestrator, atlas-channel, atlas-messages, and atlas-data use ad-hoc `t.Run` blocks instead of the documented `tests := []struct{...}` table shape — see per-package rows above for exact function names/lines.
- DOM-28 (loud degradation): `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/context.go:243-260` `GetPlayerNpcEligibility` degrades to fail-closed on error with only a `Warnf` log, not `model.ErrDecorator`/`degrade.Observe` — flagged as WARN pending the applicability question recorded under Not Evaluable.
