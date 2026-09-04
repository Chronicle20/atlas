# Backend Audit — task-281-map-back-effects

- **Service Path:** services/atlas-channel, services/atlas-configurations, services/atlas-maps, services/atlas-messages, services/atlas-saga-orchestrator, libs/atlas-packet, libs/atlas-saga
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-28
- **Build:** PASS
- **Tests:** all packages `ok` / `no test files`, 0 failed
- **Overall:** NEEDS-WORK

## Build & Test Results

`go build ./...` and `go test ./... -count=1` run module-local (per session
constraint, `tools/verify.sh`/`tools/lint.sh` NOT run) for each changed module:

- `services/atlas-channel/atlas.com/channel` — build clean, tests all `ok`.
- `services/atlas-maps/atlas.com/maps` — build clean, tests all `ok` (incl.
  `atlas-maps/map/backeffect` 0.036s, `atlas-maps/kafka/consumer/map` 0.033s).
- `services/atlas-messages/atlas.com/messages` — build clean, tests all `ok`.
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator` — build
  clean, tests all `ok` (incl. `map_command` 0.023s, `saga` 0.581s).
- `services/atlas-configurations/atlas.com/configurations` — build clean,
  tests all `ok` (incl. `socket` corpus test, updated count 3405).
- `libs/atlas-packet` — build clean, tests all `ok`.
- `libs/atlas-saga` — build clean, tests all `ok`.

No build or test failures anywhere in scope.

## Applicability

| Family | Fired? | Evidence |
|---|---|---|
| FILE-01..06 | Fired | Every changed Go package audited (no exemptions). |
| DOM structure (rest.go/processor.go/provider.go) | Fired | `map/backeffect/rest.go`, `resource.go`, `processor.go` (atlas-maps); `backeffect/rest.go`, `processor.go` (atlas-channel). |
| SUB-01..04 | Fired | `services/atlas-maps/.../map/backeffect` has `resource.go`, no `model.go`. |
| REST (DOM-06..09,12..15,17..19,32) | Fired | `map/backeffect/resource.go` registers a GET route. |
| Constants reuse (DOM-21) | Fired | New types/consts: `BackEffectEntry`, `FieldKey`, `SetBackEffectCommandBody`, `BackEffectSet`, saga `Action`/`Payload` consts, packet `BackEffectShow`/`BackEffectHide`. |
| Testing (DOM-10,20,24,33) | Fired | Multiple new/changed `_test.go` files; `map_command.Processor` and `saga.Handler` interfaces gained methods. |
| Cache (DOM-29) | Fired | `map/backeffect/registry.go` holds package-level singleton state. |
| Messaging (DOM-30) | Fired | `map/backeffect/producer.go`, `map_command/producer.go`, `command/map/back_effect.go` all emit via `producer.ProviderImpl`. |
| Multi-tenancy (DOM-31) | Fired | `map/backeffect/rest.go` + `processor.go` reads `tenant.MustFromContext`. |
| Migration hygiene (DOM-34/35) | N/A | No symbol relocation between a service and a `libs/atlas-*` module in this diff — all `libs/atlas-saga`/`libs/atlas-packet` additions are net-new code, not moves. |
| Deploy & topics (DOM-22/23) | N/A | No new `libs/atlas-*` module; `SET_BACK_EFFECT`/`CLEAR_BACK_EFFECT` are new `Type` string values inside the pre-existing `COMMAND_TOPIC_MAP`/`EVENT_TOPIC_MAP_STATUS` envelopes, not new topic env vars. |
| Runtime safety (DOM-26) | Fired (trivial) | Every non-test Go file in scope changed; no new bare `go` statement found — new goroutine use (`consumer.go:396`) is via `routine.Go(l, ctx, ...)`. |
| Channel wire values (DOM-25) | Fired | Diff touches `services/atlas-channel` and `libs/atlas-packet`; also a domain service (`atlas-messages`, `atlas-maps`) emits a client-interpreted byte — see finding below. |
| Resilience (DOM-27/28) | N/A | No changed handler writes `http.StatusInternalServerError`; no `model.Decorator` enrichment path changed (the `model.Decorator[consumer.Config]` generic in `InitConsumers` is unchanged consumer-config plumbing, not an enrichment decorator). |
| External clients (EXT-01..04) | Fired | `atlas-channel/backeffect` calls atlas-maps via `requests.GetRequest[T]`. |
| Scaffolding (SCAFFOLD-01..09) | Fired (SCAFFOLD-07 only) | New `SetBackEffect`/`ClearBackEffect` clientbound writers registered in `atlas-channel/main.go`; no new service added (01-06/08/09 N/A). |
| Security (SEC-01..04) | N/A | No changed service in scope handles auth/tokens/redirects/secrets. |

## Checklist Results

### `libs/atlas-packet/field/clientbound` (support — packet codecs)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File placement | PASS | `set_back_effect.go` / `clear_back_effect.go` each hold exactly one codec type + its own Encode/Decode; no bundling. |
| DOM-25 | Wire byte source | PASS (partial) | `BackEffectShow`/`BackEffectHide` byte consts live inside `libs/atlas-packet/field/clientbound/set_back_effect.go:20-23` — the codec-internals exemption in the pattern doc's pass criteria ("No client wire code appears as a Go literal outside `libs/atlas-packet` codec internals") applies to *this* file. See the atlas-messages/atlas-maps/atlas-channel finding below for the cross-service half of DOM-25 that fails. |
| DOM-20 | Table-driven tests | N/A | `set_back_effect_test.go` / `clear_back_effect_test.go` follow the packet byte-fixture pin playbook (`packet-audit:verify` golden + `test.RoundTrip` over `test.Variants` with `t.Run`), the documented exception to DOM-20's generic shape. |

### `services/atlas-maps/.../map/backeffect` (sub-domain: `resource.go`, no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-05 | Model/writes/reads placement | **FAIL** | `map/backeffect/registry.go:15-76` bundles the domain struct (`BackEffectEntry`, `FieldKey`), the writer functions (`Set`, `Clear`), and the reader function (`Get`) all in one file, instead of `model.go` / `administrator.go` / `provider.go`. |
| FILE-06 | No catch-all file with ≥2 responsibilities | **FAIL** | Same evidence as FILE-05 — `registry.go` carries domain-Model + write + read responsibilities in a single file. Structural violation → Important, not softened by there being no DB behind it. |
| DOM-29 | Cache/registry is an application-scoped singleton | PASS | `registry.go:26-42` — package-level `var registry *Registry` + `sync.Once` + `sync.RWMutex`-guarded `Set`/`Get`/`Clear`, reached via `getRegistry()`; `ProcessorImpl` (`processor.go:18-21`) holds no cached state itself. Placement is graded separately under FILE-05/06 above (per patterns-cache.md's own note that DOM-29 grades scope, not file). |
| SUB-01 | Business logic in processor, not handler | PASS | `resource.go:37` calls `NewProcessor(...).GetActive(f)`; no registry access from the handler directly. |
| SUB-02 | No `db.Create`/`db.Save` in resource.go | PASS | No DB calls anywhere in this package (in-memory registry only). |
| SUB-03 | POST via RegisterInputHandler[T] | N/A | Package registers only a GET route (`resource.go:26`). |
| SUB-04 | No manual JSON parsing in resource.go | PASS | No `json.Unmarshal`/`NewDecoder`/`io.ReadAll` in `resource.go`. |
| DOM-04 | `Transform(Model) (RestModel, error)` in rest.go | PASS | `rest.go:26-33` `func Transform(e BackEffectEntry) (RestModel, error)`. |
| DOM-05 | `TransformSlice` used by list handlers; no inline loop in resource.go | **FAIL** | `rest.go` defines no `TransformSlice`; `resource.go:38-47` hand-loops over `entries`, calling `Transform` per element inline. |
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | `processor.go:23` `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`. |
| DOM-07 | Handler passes `d.Logger()` into NewProcessor | PASS | `resource.go:37` `NewProcessor(d.Logger(), d.Context())`. |
| DOM-08 | POST/PATCH via RegisterInputHandler[T] | N/A | No POST/PATCH route registered. |
| DOM-09 | Every `Transform(` call site checks its error | PASS | `resource.go:40-45` — `rm, err := Transform(e)`; error checked and mapped to a 500 response. |
| DOM-12 | No `os.Getenv` in resource.go | PASS | No matches. |
| DOM-13 | No cross-domain orchestration in handler | PASS | Handler calls only its own `backeffect` processor. |
| DOM-14 | Handler calls processor methods only | PASS | `resource.go:37` calls `Processor.GetActive`, never a `provider` function. |
| DOM-15 | No `db.Create`/`Save`/`Delete` in resource.go | PASS | No matches. |
| DOM-17 | Domain errors mapped to correct HTTP status | N/A | No validation/not-found/conflict path exists — GET always returns 200 (possibly empty list); confirmed by `resource_test.go:99-121` `TestGetBackEffectsInMap_EmptyIsTwoHundred`. |
| DOM-18 | RestModel implements JSON:API interface | PASS | `rest.go:13-24` `GetID`/`GetName`/`SetID`. |
| DOM-19 | Flat request models | N/A | No request model — GET-only endpoint. |
| DOM-31 | Tenant travels via context only | PASS | `processor.go:30,37,45` — `tenant.MustFromContext(p.ctx)`; `RestModel` (`rest.go:5-11`) carries no tenant field; handler path/query carries only `worldId`/`channelId`/`mapId`/`instanceId` (resource identifiers, not tenant). |
| DOM-32 | Routes via `server.RegisterHandler`/`RegisterInputHandler[T]` | PASS | `resource.go:26` uses `rest.RegisterHandler`, which is `var RegisterHandler = server.RegisterHandler` (`rest/handler.go:29`) — direct delegation confirmed. |
| DOM-30 | DB write + events via `AndEmit`/`message.Buffer` | PASS (documented exception) | `kafka/consumer/map/consumer.go:135-140,152-159` call `producer.ProviderImpl(...)` directly on the success path, but the operation is over in-memory registry state with no DB write on any path — the "Operations over non-DB state" exception in patterns-kafka.md's DOM-30 section applies verbatim (cites `atlas-chairs` as the same shape). |
| DOM-24 | Producer stub installed for tests reaching an emit path | PASS | `kafka/consumer/map/testmain_test.go:10-13` `TestMain` calls `producertest.InstallNoop()` (pre-existing file, covers the new back-effect command tests in the same package). |
| DOM-20 | Table-driven tests | WARN | `map/backeffect/registry_test.go` (4 funcs) and `map/backeffect/resource_test.go` (2 funcs) are each independent `Test...` functions, not `tests := []struct{...}` + `t.Run`. Same shape issue in `kafka/consumer/map/consumer_test.go:108-249` (`TestHandleSetBackEffectCommand_*`, `TestHandleClearBackEffectCommand_*`). Non-blocking — no packet-fixture playbook applies to these. |

### `services/atlas-maps/.../kafka/consumer/map` and `kafka/message/map` (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File placement | PASS | `command.go`/`kafka.go` hold only constants + message/command structs; `consumer.go` holds only handler functions — no bundling. |
| DOM-25 | Domain service emits semantic key, not client byte | **FAIL** | `kafka/message/map/command.go:41-46` `SetBackEffectCommandBody.Effect uint8` and `kafka/message/map/kafka.go:65-70` `BackEffectSet.Effect uint8` both carry the raw client wire byte (0=show/1=hide) end-to-end rather than a semantic key. See consolidated finding below. |
| DOM-31 | Tenant via context only | PASS | `consumer.go:135,152` derive tenant via `ctx` passed to `backeffect.NewProcessor`; no tenant field on `Command`/`StatusEvent` structs. |

### `services/atlas-channel/.../backeffect` (support — REST client)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File placement | PASS | `processor.go` (Processor only), `rest.go` (RestModel/Extract/JSON:API methods only), `requests.go` (request functions only) — no catch-all file. |
| EXT-01 | Target RestModel implements SetToOne/SetToMany | PASS | `rest.go:31-32` — no-op implementations present. |
| EXT-02 | httptest-backed integration test with populated domain struct | PASS | `services/atlas-channel/.../kafka/consumer/map/consumer_test.go:970-1008` `TestAnnounceActiveBackEffects_ReplaysWithZeroDuration` spins an `httptest.Server` returning a JSON:API `backEffect` fixture, points `MAPS_SERVICE_URL` at it, and asserts the decoded wire packets reflect the parsed `RestModel` values — a real HTTP round trip, not a `FakeClient`/mock. |
| EXT-03 | Only genuine 404s map to "not found"; other failures bubble | PASS | `processor.go:29` returns whatever `requests.SliceProvider`/`GetRequest` yields verbatim — no special-cased error mapping introduced. |
| EXT-04 | URL via `requests.RootUrl`/`RootUrlFor` | PASS | `requests.go:17` `requests.RootUrlFor(ctx, "MAPS")`. |
| DOM-31 | Tenant via context only | PASS | Tenant reaches atlas-maps via the standard kafka/HTTP tenant-header decorators; no tenant field on `backeffect.RestModel` (`rest.go:6-12`). |

### `services/atlas-channel/.../kafka/consumer/map`, `kafka/message/map`, `socket/writer` (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File placement | PASS | New handler functions added to the existing `consumer.go` alongside its siblings (weather/jukebox), same shape; `socket/writer/set_back_effect.go` / `clear_back_effect.go` each hold one thin wrapper. |
| DOM-26 | Goroutines via `routine.Go` | PASS | `consumer.go:396` `routine.Go(l, ctx, func(_ context.Context) { announceActiveBackEffects(...) })`. |
| DOM-25 | Wire byte handling at the channel boundary | PASS (channel side) | `consumer.go:1169-1183` — `byte(e.Body.Effect)` cast happens right before `fieldcb.NewSetBackEffect(...)`, i.e. at the codec boundary, which is where DOM-25 expects resolution to happen; the fault is upstream (see consolidated finding) where the byte is minted, not here. |
| DOM-20 | Table-driven tests | WARN | `TestHandleStatusEventBackEffectSet_*`, `TestHandleStatusEventBackEffectClear_*`, `TestAnnounceActiveBackEffects_*` (`consumer_test.go:837-1046`) are each standalone `Test...` functions, not table-driven. Non-blocking. |

### `services/atlas-messages/.../command/map` (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File placement | PASS | `back_effect.go` mirrors the sibling `weather.go`'s shape (producer-function + provider-builder pair); consistent with the package's existing per-feature-file convention, no bundling of unrelated responsibilities. |
| DOM-25 | Domain service parses and forwards raw client code | **FAIL** | `command/map/back_effect.go:39,71` — `effect, err := strconv.ParseUint(match[2], 10, 8)` parses the GM's literal `0`/`1` digit straight out of chat text and stores it unchanged as `mapKafka.SetBackEffectCommandBody.Effect` — a domain (non-channel) service originating and forwarding the raw client wire code rather than a semantic key. Root of the consolidated finding below. |
| DOM-20 | Table-driven tests | PASS | `back_effect_test.go:18-79` uses `testCases := []struct{...}` + `t.Run` for both the set and clear producers. |
| DOM-24 | Producer stub for emit-reaching tests | N/A | Tests only exercise the regex-matching/gating half of the producer (`producer(ctx)(f, char, tc.message)`); they never invoke the returned `Executor`, so no emit path is transitively reached. |

### `services/atlas-saga-orchestrator/.../map_command` (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File placement | PASS | `processor.go` (Processor interface/impl only), `producer.go` (provider functions only) — no bundling. |
| DOM-33 | Every mock of a changed `Processor` interface updated | PASS | `map_command.Processor` gained `SetBackEffect`/`ClearBackEffect` (`processor.go:15-19`); `grep -rn "map_command.Processor" .` finds only the field declaration in `saga/handler.go:240` — no mock implementation of this interface exists in the module, so there is nothing to update. |
| DOM-25 | Domain service forwards raw client byte | **FAIL** | `map_command/producer.go:50-67` `SetBackEffectCommandProvider(..., effect uint8, ...)` forwards the same raw byte unchanged into the outbound `Command`. Part of the consolidated finding below. |
| DOM-30 | DB write + events atomic via AndEmit | PASS (documented exception) | `processor.go:44-46` calls `producer.ProviderImpl` directly; this saga step performs no DB write (it only emits a command), matching the "operations over non-DB state" exception. |
| DOM-20 | Table-driven tests | WARN | `producer_test.go` — `TestSetBackEffectCommandProvider`, `TestClearBackEffectCommandProvider` are standalone functions, not table-driven. Non-blocking. |

### `services/atlas-saga-orchestrator/.../saga` (domain, pre-existing `Handler`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-20 | Table-driven tests | PASS | `saga/handler_test.go` — 13 `t.Run` / 13-entry table for the new `handleSetBackEffect`/`handleClearBackEffect` cases. |
| DOM-33 | Mocks updated for interface change | N/A | `Handler` (`saga/handler.go:188`) gained two new unexported methods (`handleSetBackEffect`, `handleClearBackEffect`), but `Handler` is not a `Processor`/`Provider`/`Administrator` interface (DOM-33's own trigger names those three families specifically), and no mock of `Handler` exists in the module regardless. |

### `libs/atlas-saga` (shared library)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | No redeclaration of an existing shared constant/type | PASS | `grep -rl "BackEffect" libs/atlas-constants/` finds nothing; no pre-existing show/hide enum to reuse. |
| DOM-22 | New lib wired into root Dockerfile/go.work | N/A | `libs/atlas-saga` and `libs/atlas-packet` are pre-existing modules; this diff adds code to them, not a new module. |
| DOM-20 | Table-driven tests | PASS | `unmarshal_test.go` uses the file's established table-driven harness elsewhere; the two new tests (`TestUnmarshalSetBackEffectStep`, `TestUnmarshalClearBackEffectStep`, lines 1667-1691) are standalone functions, matching this file's one-function-per-action convention for every other of its ~100+ actions — flagged as WARN below for literal completeness. |

## Consolidated cross-service finding — DOM-25

The `Effect` value (0 = show, 1 = hide — the CLIENT's `CMapLoadable::OnSetBackEffect`
wire byte, per `libs/atlas-packet/field/clientbound/set_back_effect.go:20-23`)
is parsed directly out of GM chat text and threaded, unchanged, through every
domain hop as a raw `uint8`, not a semantic key resolved at the channel
boundary:

- `services/atlas-messages/atlas.com/messages/command/map/back_effect.go:39` —
  `effect, err := strconv.ParseUint(match[2], 10, 8)` (the GM typed `0` or `1`
  literally).
- same file, line 71 — `Effect: effect` placed straight into
  `mapKafka.SetBackEffectCommandBody`.
- `services/atlas-maps/atlas.com/maps/kafka/message/map/command.go:41-46` —
  `SetBackEffectCommandBody.Effect uint8` (Kafka **command** wire shape).
- `services/atlas-maps/atlas.com/maps/kafka/message/map/kafka.go:65-70` —
  `BackEffectSet.Effect uint8` (Kafka **status event** wire shape, produced by
  `map/backeffect/producer.go:24`).
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/map_command/producer.go:50-67`
  — `SetBackEffectCommandProvider(..., effect uint8, ...)` forwards the same
  byte from a saga step.
- `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:1169-1183`
  — `handleStatusEventBackEffectSet` casts `e.Body.Effect` straight to `byte`
  and hands it to `fieldcb.NewSetBackEffect`.

Per the DOM-25 verification procedure (anti-patterns.md): *"domain
(non-channel) services emit SEMANTIC keys ... A `byte` field carrying a client
code in a Kafka event produced by a domain service is a finding."* Both
`atlas-messages` (the command producer) and `atlas-maps`
(`SetBackEffectCommandBody`/`BackEffectSet`) are domain services carrying the
literal client code end-to-end, not a semantic key such as
`BackEffectVisibility{Show,Hide}` resolved to a wire byte only at the
`atlas-channel` boundary. The fact that the numeric encoding happens to be
version-stable across every supported client build does **not** exempt it —
the task-103 uniformity ruling cited in the same pattern-doc section.

## Not evaluable from the diff

- none — every applicable checklist item was settled from the changed files,
  targeted symbol lookups (`grep -rn "map_command.Processor"`,
  `grep -rl "BackEffect" libs/atlas-constants/`), and the module-local
  build/test runs.

## Summary

### Blocking (must fix)

- **DOM-25** — `services/atlas-messages/.../command/map/back_effect.go:39,71`,
  `services/atlas-maps/.../kafka/message/map/command.go:41-46`,
  `services/atlas-maps/.../kafka/message/map/kafka.go:65-70`,
  `services/atlas-saga-orchestrator/.../map_command/producer.go:50-67`: the
  client's raw `Effect` wire byte (0/1) is parsed from GM chat text and carried
  unchanged through the Kafka command and status-event bodies of two domain
  services instead of a semantic key resolved to a wire byte only at
  `atlas-channel`.
- **FILE-05 / FILE-06** — `services/atlas-maps/.../map/backeffect/registry.go:15-76`:
  the domain struct (`BackEffectEntry`, `FieldKey`), the writer functions
  (`Set`, `Clear`), and the reader function (`Get`) are all bundled into one
  file instead of `model.go` / `administrator.go` / `provider.go`.
- **DOM-05** — `services/atlas-maps/.../map/backeffect/resource.go:38-47` /
  `rest.go`: no `TransformSlice` is defined; the list handler hand-loops
  `Transform` calls inline instead.

### Non-Blocking (should fix)

- **DOM-20** — not table-driven: `services/atlas-maps/.../map/backeffect/registry_test.go`,
  `services/atlas-maps/.../map/backeffect/resource_test.go`,
  `services/atlas-maps/.../kafka/consumer/map/consumer_test.go:108-249`,
  `services/atlas-channel/.../kafka/consumer/map/consumer_test.go:837-1046`,
  `services/atlas-saga-orchestrator/.../map_command/producer_test.go`,
  `libs/atlas-saga/unmarshal_test.go:1667-1691`.
