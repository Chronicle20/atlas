# Backend Audit — atlas-maps + atlas-saga-orchestrator (task-278 shard)

- **Service Paths:** `services/atlas-maps`, `services/atlas-saga-orchestrator`
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-28
- **Branch range:** `bda6566f3..68a4e1cce`
- **Build:** PASS
- **Tests:** all packages `ok` in both services (0 failures)
- **Overall:** NEEDS-WORK

## Build & Test Results

```
cd services/atlas-maps/atlas.com/maps && go build ./...           -> exit 0, no output
cd services/atlas-maps/atlas.com/maps && go test ./... -count=1   -> all `ok`, including
    atlas-maps/map/environment                    0.044s
    atlas-maps/kafka/consumer/map                 0.070s
    atlas-maps/kafka/message/map                  0.023s
    atlas-maps/map                                2.342s

cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./...          -> exit 0, no output
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./... -count=1  -> all `ok`, including
    atlas-saga-orchestrator/map_command           0.164s
    atlas-saga-orchestrator/saga                  0.776s
```

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| FILE (FILE-01..06) | Fired | Every changed Go package audited: `environment`, `map_command`, `map` (character-registry teardown), `kafka/consumer/map`, `kafka/consumer/character`. |
| RUNTIME / DOM-26 goroutines | Fired (any non-test .go changed) | No bare `go` statement found in any changed non-test file (`grep -n "\bgo func\|^\s*go "` empty in `map/environment/*.go`, `map_command/*.go`). N/A on the merits — no goroutine spawned. |
| DOM structure (DOM-01..05,11,16) | Fired | `map/environment/rest.go` exists (no `model.go`/`entity.go`/`provider.go` in this package). DOM-04/05/06/07 evaluated. |
| SUB (SUB-01..04) | Fired | `map/environment` has `resource.go`, no `model.go` — action/REST-hybrid sub-domain shape. |
| REST (DOM-06..09,12..15,17..19,32) | Fired | `map/environment` has `resource.go`, `rest.go`, `processor.go`, registers HTTP routes in `main.go`. |
| Constants reuse (DOM-21) | Fired | Diff uses `field.ObjectKind` / `field.ParseObjectKind` from `libs/atlas-constants/field`; verified no local redeclaration of the wire strings `ENVIRONMENT`/`OBSTACLE` in either service. |
| Testing (DOM-10,20,24,33) | Fired | New/changed test packages reach `producer.ProviderImpl`; `Handler`/`map_command.Processor` interfaces changed. |
| Cache (DOM-29) | Fired | `map/environment/registry.go` holds package-level singleton state reached via `getRegistry()` (`sync.Once`), same shape as a `GetCache()` singleton — not a `cache.go`, graded under FILE-* for placement, DOM-29 for scope. |
| Messaging (DOM-30) | Fired | `map/environment/resource.go`, `kafka/consumer/map/consumer.go`, `map_command/processor.go` call `producer.ProviderImpl` directly (not `AndEmit`). |
| Multi-tenancy (DOM-31) | Fired | `map/environment/rest.go` exists; `registry.go`/`processor.go` read `tenant.MustFromContext(ctx)`. |
| Migration hygiene (DOM-34/35) | N/A | No diff hunk moves/extracts symbols between a service and a `libs/atlas-*` module (the `libs/` changes in this branch are additive: new `field.ObjectKind` type and new `atlas-saga` action constants/payloads, not moved symbols). |
| Deploy & topics (DOM-22/23) | N/A | No new/renamed `COMMAND_TOPIC_*`/`EVENT_TOPIC_*` env var — `COMMAND_TOPIC_MAP` and `EVENT_TOPIC_MAP_STATUS` both pre-exist (`services/atlas-maps/atlas.com/maps/kafka/message/map/command.go:12`, `kafka.go:12`); no `libs/atlas-*` module added. |
| Channel wire values (DOM-25) | N/A | Diff does not touch `services/atlas-channel` or `libs/atlas-packet`; no domain service in this shard emits a client-interpreted opcode byte — `EnvironmentStateChanged`/`EnvironmentReset` carry the `field.ObjectKind` string, resolved client-side. |
| Resilience (DOM-27/28) | N/A on the merits (family opened, no violation) | No changed handler in `map/environment/resource.go` writes `http.StatusInternalServerError` directly (uses `server.WriteErrorResponse`, resource.go:54) — DOM-27 N/A. No `model.Decorator` or remote-data enrichment path changed — DOM-28 N/A. The `Exit()`/consumer teardown reads `p.cp.GetCharactersInMap` whose only implementation (`map/character/processor.go:39-42`) always returns a `nil` error (in-memory registry, no I/O), so the `err == nil` gate cannot silently swallow a real failure today. |
| External clients (EXT-01..04) | N/A | No `requests.RootUrl`/`requests.GetRequest[T]`/`requests.PostRequest[T]` call in either changed package. |
| Scaffolding (SCAFFOLD-01..09) | N/A | No `services/atlas-<svc>/` directory added, no channel Writer/Handler registered, `deploy/shared/routes.conf` untouched (`git diff --stat` for this range shows no scaffold-related path). |
| Security (SEC-01..04) | N/A | Neither `atlas-maps` nor `atlas-saga-orchestrator` handles auth tokens/secrets in the changed surface. |

## Checklist Results

### map/environment (atlas-maps) — sub-domain / REST hybrid (`resource.go` + `rest.go` + `processor.go`, no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor interface/constructor/methods in `processor.go` | PASS | `map/environment/processor.go:20-62` — interface, `NewProcessor`, and all `ProcessorImpl` methods. |
| FILE-02 | RestModel/Transform/JSON:API methods in `rest.go` | PASS | `map/environment/rest.go:5-32` — `RestModel`, `GetID`/`GetName`/`SetID`, `Transform`. |
| FILE-06 | No package-named catch-all bundling ≥2 responsibilities | PASS | Package files are `resource.go` (handlers), `rest.go` (model/transform), `processor.go` (business logic), `producer.go` (event providers), `registry.go` (state) — each single-purpose; no `environment.go` catch-all. |
| SUB-01 | Business logic not in handler | PASS | `resource.go` handlers delegate to `NewProcessor(...).Set/Reset/GetAll` (`resource.go:47,85,114`); logic lives in `processor.go`. |
| SUB-02 | No `db.Create`/`db.Save` in `resource.go` | PASS | No DB calls in `resource.go` — package has no `entity.go`/DB writes at all (in-memory registry). |
| SUB-03 | POST uses `RegisterInputHandler[T]` | PASS | `resource.go:32` — `rest.RegisterInputHandler[RestModel](l)(si)(setEnvironmentInMap, handleSetEnvironmentInMap)`. |
| SUB-04 | No manual JSON parsing in `resource.go` | PASS | No `json.Unmarshal`/`json.NewDecoder`/`io.ReadAll` in `resource.go`. |
| DOM-04 | `Transform(Model) (RestModel, error)` in `rest.go` | PASS | `rest.go:25-32`. |
| DOM-05 | `TransformSlice` defined and used by list handlers — no inline loop in `resource.go` | **FAIL** | `resource.go:49-58` — `handleGetEnvironmentInMap` builds the response with an inline `for _, e := range entries { rm, err := Transform(e); ... }` loop. `rest.go` defines no `TransformSlice` (`grep -rn "TransformSlice" map/environment/` → no matches). |
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | `processor.go:31` — `NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`. |
| DOM-07 | Handlers pass `d.Logger()`/`d.Context()` | PASS | `resource.go:47,85,114`. |
| DOM-09 | Every `Transform(` call site checks its error | PASS | `resource.go:51-56` checks `err` and writes an error response on failure. |
| DOM-12 | No `os.Getenv()` in handlers | PASS | No `os.Getenv` in `resource.go`. |
| DOM-13 | No cross-domain orchestration in handlers | PASS | Handlers only call the `environment` processor and one producer call each; no other domain touched from `resource.go`. |
| DOM-14 | Handlers call processor methods, never providers | PASS | `resource.go:47,85,114` call `Processor` methods; no provider call. |
| DOM-15 | No `db.Create`/`db.Save`/`db.Delete` in handlers | PASS | None present. |
| DOM-17 | Domain errors map to correct HTTP status | PASS | `resource.go:78-82` (bad kind → 400), `resource.go:86-89` (blank name → 400). |
| DOM-18 | RestModel implements JSON:API interface | PASS | `rest.go:12-23` — `GetID`, `GetName`, `SetID`. |
| DOM-19 | Request models flat | PASS | `RestModel{Id, Kind, Name, State}` — no nested `Data`/`Attributes`. |
| DOM-29 | Cache/registry is an application-scoped singleton via an accessor, never per-instance | PASS | `registry.go:27-38` — package-level `registry *Registry` behind `sync.Once`/`getRegistry()`; `processor.go` never constructs its own map, only calls `getRegistry()` (`processor.go:43,54,61`). |
| DOM-30 | DB writes emit via `AndEmit`+`message.Buffer`, not a direct `producer.ProviderImpl` on the success path | PASS (documented exception) | `resource.go:91,116` call `producer.ProviderImpl` directly on the success path, but the operation has no DB write — state lives only in the in-memory `Registry` (`registry.go`). This is the documented DOM-30 exception for "operations over non-DB state" (`.claude/skills/backend-dev-guidelines/resources/patterns-kafka.md:82-87`, citing the identical `GetRegistry().Set(...)` + direct-emit shape in `atlas-chairs`). |
| DOM-31 | Tenant/trace travel only via context | PASS | `RestModel` (`rest.go:5-10`) carries no tenant field; `resource.go` reads only `worldId`/`channelId`/`mapId`/`instanceId` path params (`resource.go:41-46` etc.) — no tenant path/query param; tenant resolved via `tenant.MustFromContext(p.ctx)` inside `processor.go:41,53,60`. Cross-tenant isolation asserted directly: `registry_test.go:87-102` (`TestRegistryTenantIsolation`) and `resource_test.go:130-152` (`TestGetEnvironmentInMap_TenantIsolation`). |
| DOM-32 | Routes register via `server.RegisterHandler`/`RegisterInputHandler[T]` | PASS | `resource.go:31-33` use `rest.RegisterHandler`/`rest.RegisterInputHandler[RestModel]`, which are direct aliases of `server.RegisterHandler`/`server.RegisterInputHandler[M]` (`services/atlas-maps/atlas.com/maps/rest/handler.go:28,31`). |
| DOM-20 | Tests are table-driven | WARN | Most new tests in this package are individually-named `func Test...` cases rather than `tests := []struct{...}{}` + `t.Run` tables (e.g. `registry_test.go:26-114`, `resource_test.go:1-233`). Each case is short and behavior-distinct, but the rule's letter is table-driven form; flagging as non-blocking since coverage itself is thorough and per-case assertions are clear. |
| DOM-24 | Test package reaching an emit path installs the `producertest` stub | PASS | `testmain_test.go:14-17` — `TestMain` calls `producertest.InstallNoop()`. |

### kafka/consumer/map (atlas-maps) — support package (consumer arms, no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Handler functions grouped appropriately | PASS | `consumer.go` holds `InitConsumers`/`InitHandlers`/`handle*Command` — single-purpose consumer file, no `processor.go`-shaped logic misplaced here. |
| DOM-26 | Goroutines via `routine.Go` | N/A | No `go` statement in `kafka/consumer/map/consumer.go` diff. |
| DOM-24 | Emit-reaching test package installs producer stub | PASS | `kafka/consumer/map/testmain_test.go:10-12` (pre-existing, unchanged this diff, still governs the new `TestHandleSetEnvironmentStateCommand_*`/`TestHandleResetEnvironmentCommand_*` cases). |
| DOM-31 (tenant via context) | PASS | `handleSetEnvironmentStateCommand`/`handleResetEnvironmentCommand` (`consumer.go:110-152`) resolve tenant only via `ctx` passed into `environment.NewProcessor(l, ctx)` — no tenant field read from the command body. |
| Messaging/DOM-30 (chained empty-field reset) | PASS (documented exception, same as above) | `consumer.go` (character consumer) calls `ep.Reset(mk.Field)` directly after an in-memory `GetCharactersInMap` check — no DB write in this path either; see `kafka/consumer/character/consumer.go:195-203`. |

### map (atlas-maps, `map/processor.go`) — domain package (has `model.go` elsewhere in package; this diff only touches `Exit`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-31 (tenant consistency across the teardown call) | PASS | `map/processor.go:106-118` (`Exit`) calls `p.cp.GetCharactersInMap(transactionId, f)` and, on empty, `environment.NewProcessor(p.l, p.ctx).Reset(f)` — both reads use the same `p.ctx`, so the emptiness check and the registry clear are scoped to the same tenant; no cross-tenant leak path. |
| DOM-30 | Direct producer call only where there is no DB write | PASS | `Exit` still returns `mb.Put(...)` through the existing `message.Buffer` for the `CHARACTER_EXIT` event (`map/processor.go:118`); the new environment-teardown call (`Reset`) does not itself emit — it only clears registry state, consistent with the DOM-30 documented exception. |

### map_command (atlas-saga-orchestrator) — support package (no `model.go`, no `resource.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor interface + methods in `processor.go` | PASS | `map_command/processor.go:15-50` — `Processor` interface, `NewProcessor`, and all `ProcessorImpl` methods including the new `SetEnvironmentState`/`ResetEnvironment`. |
| FILE-06 | No catch-all file | PASS | Package is `processor.go` + `producer.go`, each single-purpose. |
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | `processor.go:27` — `NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`. |
| DOM-21 | No redeclared wire constant | PASS | `SetEnvironmentStateCommandProvider`/`ResetEnvironmentCommandProvider` (`producer.go`) take `field.ObjectKind` and stringify it (`string(kind)`) rather than redefining `"ENVIRONMENT"`/`"OBSTACLE"` locally — confirmed no local `ObjectKind`-shaped const block in `map_command/`. |
| DOM-30 | Direct producer call only where there is no DB write | PASS (documented exception) | `processor.go:44-50` call `producer.ProviderImpl` directly — this package has no DB/model at all, it is a pure command-emission processor (same shape as the pre-existing `FieldEffectWeather`/`PlayJukebox` methods it sits beside). |
| DOM-33 | Interface change updates every mock in the same diff | PASS | `Processor` gained `SetEnvironmentState`/`ResetEnvironment` (`processor.go:18-19`); the test double `mapCommandProcessorMock` in `saga/handler_test.go:1709-1731` implements both new methods and is statically asserted (`var _ map_command.Processor = (*mapCommandProcessorMock)(nil)`, `handler_test.go:1731`). |

### saga (atlas-saga-orchestrator) — domain package (`model.go`, `handler.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-33 | `Handler` interface change updates its implementation in the same diff | PASS | `Handler` gained `WithMapCommandProcessor`, `handleMoveEnvironment`, `handleResetEnvironment` (`saga/handler.go:98,192-193`); `HandlerImpl` implements all three (`handler.go:822-826` for `WithMapCommandProcessor`, `handler.go:3787-3846` — diff hunk — for the two `handle*` methods) and `GetHandler` dispatches both new actions (`handler.go:1041-1044`). |
| `WithMapCommandProcessor` field-copy correctness | PASS | Uses the shallow-copy form `c := *h; c.mapCommandP = mapCommandP; return &c` (`handler.go:822-826`) — the form the file's own comment at `WithMtsProcessor` (`handler.go:531-539`) documents as the fix for the field-by-field `With*` methods silently nil-ing fields they forget to list. `WithMapCommandProcessor` does not exhibit that bug. |
| DOM-21 | No redeclared saga action/type constant | PASS | `MoveEnvironment`/`ResetEnvironment` action constants and `MoveEnvironmentPayload`/`ResetEnvironmentPayload` types are re-exported `= sharedsaga.X` aliases (`saga/model.go:258-259,400-401`), not locally redefined. |
| Resilience — fire-and-forget completion (informational, no rule ID) | Informational | `handleMoveEnvironment`/`handleResetEnvironment` (`handler.go:3787-3846`) self-complete the step via `NewProcessor(h.l, h.ctx).StepCompleted(s.TransactionId(), true)` after a successful produce, matching the pre-existing `handlePlayJukebox` pattern; on produce failure, the step is left Pending and the error is logged and returned (`h.logActionError`), so a broker outage stalls the step rather than silently completing it — this is the fail-safe behavior the task description asked to confirm. |
| **Known out-of-scope defect (not a task-278 finding)** | Informational | `WithNoteProcessor` (`saga/handler.go:813-820`) constructs a fresh `&HandlerImpl{l, ctx, t, noteP}` literal, dropping every other processor field (`charP`, `compP`, ..., `mapCommandP`, `partyP`, `pendingChangeP`) to their zero value — the same field-by-file bug `WithMtsProcessor`'s comment (`handler.go:531-539`) describes for `WithPartyProcessor`/`WithPendingChangeProcessor`, which were fixed with the shallow-copy form. `WithMapCommandProcessor` added by this diff correctly uses the shallow-copy form and does **not** have this defect. `WithNoteProcessor` was pre-existing before this branch and is out of scope per the task brief (already logged separately). |

### saga/event_acceptance.go (atlas-saga-orchestrator)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| Fire-and-forget action registered with empty acceptance list | PASS | `sharedsaga.MoveEnvironment: {}` and `sharedsaga.ResetEnvironment: {}` added (`event_acceptance.go:326-327`), consistent with other self-completing actions (`UiLock`, `PlayPortalSound`) in the same table; asserted by `TestEventAcceptance_EnvironmentActionsAreFireAndForget` (`saga/model_test.go`, diff). |

## Security Review

Not applicable — SEC-* trigger did not fire (neither service handles authentication, authorization, tokens, redirects, or secrets in the changed surface).

## Not evaluable from the diff

- none

## Summary

### Blocking (must fix)
- DOM-05 (`map/environment/resource.go:49-58`, `map/environment/rest.go`): `handleGetEnvironmentInMap` builds its response list with an inline `for` loop calling `Transform` per entry instead of a `TransformSlice` helper defined in `rest.go`. Add `TransformSlice([]ObjectEntry) ([]RestModel, error)` to `rest.go` and use it from the handler.

### Non-Blocking (should fix)
- DOM-20 (`map/environment/registry_test.go`, `map/environment/resource_test.go`, `map/environment/processor_test.go`): new tests are written as individually-named `func Test...` cases rather than the table-driven `tests := []struct{...}{} ` + `t.Run` form the rule calls for. Coverage itself is thorough; this is a style/form gap only.

### Informational (not task-278 findings)
- `WithNoteProcessor` (`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go:813-820`) drops every `HandlerImpl` field except `l`/`ctx`/`t`/`noteP` when building its returned `Handler`, unlike `WithPartyProcessor`/`WithMapCommandProcessor`, which use the shallow-copy fix. Independently confirmed present and already logged for its own task, per the brief — not raised here as a blocking finding.
