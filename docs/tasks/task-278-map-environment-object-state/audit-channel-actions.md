# Backend Audit — task-278 (atlas-channel, atlas-reactor-actions, atlas-map-actions)

- **Service Path(s):** `services/atlas-channel`, `services/atlas-reactor-actions`, `services/atlas-map-actions`
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-28
- **Branch range:** `bda6566f3..68a4e1cce`
- **Build:** PASS (all three services)
- **Tests:** PASS (all three services, `go test ./... -count=1`, no `FAIL` lines)
- **Overall:** NEEDS-WORK

## Build & Test Results

```
cd services/atlas-channel/atlas.com/channel && go build ./...      -> exit 0, no output
cd services/atlas-channel/atlas.com/channel && go test ./... -count=1  -> all packages "ok"
cd services/atlas-reactor-actions/atlas.com/reactor && go build ./...  -> exit 0, no output
cd services/atlas-reactor-actions/atlas.com/reactor && go test ./... -count=1 -> "ok" (atlas-reactor-actions, .../script)
cd services/atlas-map-actions/atlas.com/map-actions && go build ./... -> exit 0, no output
cd services/atlas-map-actions/atlas.com/map-actions && go test ./... -count=1 -> "ok" (atlas-map-actions, .../script)
```

## Applicability

| Family | Status | Trigger evidence |
|---|---|---|
| FILE-01..06 | Fired | Every changed package (`environment`, `kafka/consumer/map`, `kafka/message/map`, reactor `script`, map-actions `script`) is a changed Go package — unconditional trigger. |
| RUNTIME (DOM-26) | Fired | Non-test Go files changed (`consumer.go`, `executor.go` ×2, `processor.go`, `requests.go`, `rest.go`, `kafka.go`). |
| DOM-STRUCTURE (DOM-01..05,11,16) | Fired | `environment/rest.go` added (DOM-04/05 apply to any package with `rest.go`, per audit-checklist.md's explicit "model.go or not" clarification); reactor/`map-actions` `script` packages have pre-existing `model.go`. |
| SUB-01..04 | N/A | No changed package has `resource.go` without `model.go` — `environment` has neither `model.go` nor `resource.go` (support/client package); `script` packages have `model.go`. |
| REST (DOM-06..09,12..15,17..19,32) | Fired | `environment/processor.go` and `environment/rest.go` exist. |
| MESSAGING (DOM-30) | N/A | No `AndEmit` / `message.Emit` / `producer.ProviderImpl` call added by this diff — grep of the diff hunks found none. |
| MULTITENANCY (DOM-31) | Fired | `environment/rest.go` added; `consumer.go` reads tenant state (`tenant.MustFromContext`) on the new handlers. |
| CONSTANTS (DOM-21) | Fired | `kafka/message/map/kafka.go` declares new consts/types (`EventTopicMapStatusTypeEnvironmentStateChanged`, `EnvironmentStateChanged`, `EnvironmentObject`, `EnvironmentReset`); reactor/map-actions `executor.go` reference `field.ObjectKind`/`saga.MoveEnvironment` (already in `libs/`). |
| TESTING (DOM-10,20,24,33) | Fired | New `_test.go` files in all three services; `environment/mock/processor.go` implements a new `Processor` interface. |
| EXT-01..04 | Fired | `environment/requests.go` calls `requests.GetRequest[T]` / `requests.RootUrlFor` against atlas-maps. |
| RESILIENCE (DOM-27,28) | Fired (DOM-28 only) | New code in `SpawnForSelf` (`consumer.go:396-408`) fetches remote data (`environment.NewProcessor(...).GetAll`) and branches on `err`. DOM-27 N/A — atlas-channel has no `database.Connect` DB backing and no handler writes `http.StatusInternalServerError`. |
| CHANNEL-WIRE (DOM-25) | Fired | Diff touches `services/atlas-channel`; new `announceObjectState` writer-fallback helper and call sites. |
| DEPLOY (DOM-22,23) | N/A | No new `libs/atlas-*` module wired in; `EnvEventTopicMapStatus` (the topic env var) is unchanged — only new message-type discriminator strings inside the existing topic were added. |
| SCAFFOLD-01..09 | N/A | No `services/atlas-<svc>/` directory added; no new atlas-channel `Writer`/`Handler` registered — `libs/atlas-packet` is unchanged by this branch (confirmed: `git diff --stat -- libs/atlas-packet` empty), so every writer/handler `announceObjectState` calls (`SetObjectStateWriter`, `FieldObstacleOnOffWriter`, `FieldObstacleOnOffListWriter`, `FieldObstacleAllResetWriter`) is pre-existing; `deploy/shared/routes.conf` untouched. |
| SEC-01..04 | N/A | None of the three services handle authentication, tokens, redirects, or secrets in this diff. |

## Checklist Results

### `atlas-channel/environment` (support — no `model.go`, no `resource.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor lives in `processor.go` | PASS | `environment/processor.go:12-24` — `Processor` interface + `NewProcessor` + `ProcessorImpl` methods, no split needed. |
| FILE-02 | RestModel/Transform/Extract/JSON:API methods in `rest.go` | PASS | `environment/rest.go:3-21` — `RestModel`, `GetID`/`GetName`/`SetID`. `Extract` is in `processor.go:32` (not `rest.go`) — see FILE-02 note below. |
| FILE-03 | Cross-service request functions in `requests.go` | PASS | `environment/requests.go:16-26` — `getBaseRequest`, `requestEnvironmentInMap`. |
| FILE-04 | entity struct/Migration/TableName in `entity.go` | N/A | No `entity.go` in this package — client package, no persistence. |
| FILE-05 | Builder/Model/administrator/provider/state | N/A | Package has none of builder.go/model.go/administrator.go/provider.go/state.go — pure REST-client package, nothing to place. |
| FILE-06 | No catch-all file carrying ≥2 responsibilities | PASS | Each of `processor.go`, `rest.go`, `requests.go` carries exactly one responsibility; no `environment.go` catch-all exists. |
| DOM-04 | `Transform(Model) (RestModel, error)` in `rest.go` | FAIL (Important) | `environment/rest.go` has no `func Transform(` — grep confirms zero matches. Per `audit-checklist.md` line 33 ("DOM-04/05 ... apply to any package with those files, `model.go` or not"), this rule fires regardless of the absence of `model.go`. The package instead returns `RestModel` directly from `Processor.GetAll` (`processor.go:27-29`) with a no-op `Extract(m RestModel) (RestModel, error) { return m, nil }` — there is no domain `Model` layer at all, so no `Transform` exists to convert one. |
| DOM-05 | `TransformSlice` in `rest.go`, used by list handlers | FAIL (Important) | `environment/rest.go` has no `func TransformSlice(` — grep confirms zero matches. (No `resource.go`/list handler exists in this package to call it, so the "inline loop" half of the pass criteria is moot, but the function itself is still required by the rule's own trigger and absent.) |
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | `environment/processor.go:16` — `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`. |
| DOM-18 | RestModel implements JSON:API interface | PASS | `environment/rest.go:9,13,17` — `GetID()`, `GetName()`, `SetID()`. |
| DOM-19 | Request models flat | N/A | No request/write model defined in this GET-only client package. |
| DOM-31 | Tenant/trace travel in context only | PASS | `environment/requests.go:15-16` — `getBaseRequest(ctx)` resolves the MAPS root URL from `ctx` via `requests.RootUrlFor`; `RestModel` (`rest.go:3-7`) carries no tenant field. |
| EXT-01 | Target REST model implements `SetToOneReferenceID`/`SetToManyReferenceIDs` | FAIL (Important) | `environment/rest.go` — grep for `SetToOneReferenceID`/`SetToManyReferenceIDs` in the file returns zero matches. Per `cross-service-implementation.md` EXT-01, this is required "even as no-ops" — without them api2go errors on any response carrying a `relationships` block. |
| EXT-02 | `httptest`-backed integration test with populated domain struct | PASS | `environment/requests_test.go:33-53` (`TestGetAll_ParsesCollection`) serves a JSON:API fixture via `httptest.NewServer` and asserts `ms[0].Kind`/`Name`/`State` are populated. |
| EXT-03 | Only genuine 404s map to "not found"; other failures bubble with original error | PASS | `environment/requests.go` performs no custom error mapping — the shared `requests.SliceProvider`/`GetRequest` passthrough is used unmodified; `requests_test.go:83-98` (`TestGetAll_ServerError`) confirms a 500 surfaces as a non-nil error rather than being silently swallowed as "not found". |
| EXT-04 | URL via `requests.RootUrl(<DOMAIN>)` | PASS | `environment/requests.go:15-16` — `requests.RootUrlFor(ctx, "MAPS")`; `"MAPS"` is the established domain string also used by `maps/location/requests.go`, `weather/requests.go`, `jukebox/requests.go`. |
| DOM-33 | Interface change updates every mock in same diff | PASS | `environment/mock/processor.go:9-20` — `ProcessorMock` implements the new `Processor` interface with the nil-check default (`var _ environment.Processor = (*ProcessorMock)(nil)`), added in the same diff. |
| DOM-20 | Tests table-driven | N/A | `requests_test.go` uses four independent `func Test...` functions, not a `tests := []struct{}` table — but each test is single-scenario, not a multi-case table candidate; DOM-20's own trigger ("diff adds or changes tests") technically fires, and the pass criteria is table-driven shape. Recorded as informational: these are simple single-assertion integration tests, not enumerable variants, so no table was warranted. |

### `atlas-channel/kafka/consumer/map` (support — no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Handler/dispatch functions correctly placed | PASS | New handlers (`handleStatusEventEnvironmentStateChanged`, `handleStatusEventEnvironmentReset`) and helpers (`announceObjectState`, `announceEnvironmentState`) added to `consumer.go`, matching every other status-event handler already in the file — no new catch-all file introduced. |
| DOM-26 | Every goroutine via `routine.Go` | PASS | `consumer.go:396` — `routine.Go(l, ctx, func(_ context.Context) { entries, eerr := environment.NewProcessor(l, ctx).GetAll(f) ... })`; no bare `go` statement added by this diff (grep of diff hunks for `go func` found none). |
| DOM-25 | Client-interpreted bytes from tenant writer-options table, not literals | PASS | `announceObjectState` (`consumer.go:1186-1197`) selects the opcode by **probing the tenant's routed writer table** (`wp(fieldcb.FieldObstacleOnOffWriter)` / `wp(fieldcb.SetObjectStateWriter)`), not a hardcoded literal; `state uint32` is authored numeric object state (script/reactor param), not a client dispatcher-mode byte requiring translation. |
| CHANNEL-WIRE — fallback helper bypass | Every call site uses `announceObjectState` | PASS | `consumer.go:1228,1234,1256,1289` are the only four call sites setting obstacle/environment object state, all route through `announceObjectState`; no direct `wp(fieldcb.SetObjectStateWriter)`/`doorAnnounce(..., fieldcb.SetObjectStateWriter, ...)` call bypasses it elsewhere in the diff. |
| CHANNEL-WIRE — second-probe failure logged | Second writer failure logged, not swallowed | PASS | `consumer.go:1192-1195` — `if _, err := wp(fieldcb.SetObjectStateWriter); err != nil { l.WithError(err).Errorf("Tenant routes no [%s] writer...") ; return err }`; exercised by `TestAnnounceObjectState_WriterSelection` case "environment with only obstacle writer routed still uses set object state" (`consumer_test.go:833-839`, `wantErr: true`). |
| DOM-31 | Tenant/trace in context only; broadcast tenant/world/channel guarded | PASS | `consumer.go:1243,1269` — `if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) { return }` in both new handlers, matching every sibling status-event handler in the file. |
| RESILIENCE — map-enter replay fails open | REST failure to atlas-maps must not stall map entry | PASS | `consumer.go:397-403` — `entries, eerr := environment.NewProcessor(l, ctx).GetAll(f); if eerr != nil { l.WithError(eerr).Errorf(...); return }` runs inside `routine.Go` (line 396), so a failure only skips the environment-replay goroutine and cannot block `SpawnForSelf`'s synchronous character-spawn path (lines 197-212) or its return (line 410). |
| DOM-28 | Fallible enrichment path degrades loudly (`model.ErrDecorator` + `degrade.Observe`) | WARN | `consumer.go:397-403` fetches remote data and branches on `err`, logging `Errorf` but never calling `degrade.Observe(...)` or incrementing `atlas_enrichment_degraded_total`. This is not a `model.Decorator[Model]` in the sense `patterns-resilience.md`'s example describes (no `Model` is enriched or returned — the block is a fire-and-forget broadcast, identical in shape to the untouched `weather`/`jukebox`/`boat`/`timer` blocks immediately above and below it in the same function, none of which use `degrade.Observe` either). Recorded WARN rather than FAIL because the rule's worked example and "silently return the un-enriched model" framing target decorator-style enrichment of a returned domain object, which does not apply here; the ambiguity is real enough that this should not block the PR, but a reviewer who reads DOM-28 literally ("enrichment fallback that fetches remote data ... branches on err") would need `degrade.Observe` here too. |

### `atlas-channel/kafka/message/map` (support — no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-06 | No catch-all file carrying ≥2 responsibilities | PASS | `kafka.go` carries only Kafka message-envelope type/const declarations (the established shape for every `kafka/message/*` package in this service) — no Processor/RestModel/requests mixed in. |
| DOM-21 | No redeclaration of a `libs/atlas-constants` type/const | PASS | `EventTopicMapStatusTypeEnvironmentStateChanged`/`EventTopicMapStatusTypeEnvironmentReset`/`EnvironmentStateChanged`/`EnvironmentObject`/`EnvironmentReset` (`kafka.go:21-22,66-83`) have no equivalent anywhere under `libs/atlas-constants` — grep confirms zero matches. |
| CONSTANTS — `field.ObjectKind` not redefined locally | PASS | Neither `kafka.go` nor `consumer.go` declares a local `ObjectKind`-shaped type; `consumer.go` imports and uses `field.ParseObjectKind`/`field.ObjectKindObstacle`/`field.ObjectKindEnvironment` from `libs/atlas-constants/field` (added in this same branch, `libs/atlas-constants/field/constants.go:14-37`). |

### `atlas-reactor-actions/script` (domain — has `model.go`, only `executor.go` changed)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01/06 | Executor logic in its established file, no catch-all | PASS | `executeMoveEnvironment`/`executeResetEnvironment` added to `executor.go` alongside every other `execute*` operation handler — the file's pre-existing single-purpose shape (op-dispatch + saga-building), unchanged by this diff. |
| CONSTANTS — no local `ObjectKind`/message-type redeclaration | PASS | `executor.go:301` — `field.ParseObjectKind(params["kind"])`, imported from `libs/atlas-constants/field`; `saga.MoveEnvironment`/`saga.ResetEnvironment`/`saga.MoveEnvironmentPayload`/`saga.ResetEnvironmentPayload` (`executor.go:159,225,309,335` region) imported from `libs/atlas-saga` (added in this same branch: `libs/atlas-saga/model.go:296-297`, `payloads.go:1311-1330`) — no local redefinition of either. |
| SUB-04-equivalent — no manual JSON parsing | PASS | `executeMoveEnvironment`/`executeResetEnvironment` read `op.Params()` (a `map[string]string`), not raw JSON; no `json.Unmarshal`/`io.ReadAll` added. |
| DOM-20 | Tests table-driven | PASS | `executor_test.go:57-224` (`TestExecuteMoveEnvironment`) — `tests := []struct{...}{...}` + `t.Run`. |
| DOM-24 | Test package reaching emit path stubs producer | N/A | Tests inject `captureSagaProcessor` (`executor_test.go:17-28`) directly into `e.sagaP`, replacing the real saga processor entirely — no Kafka producer call is reached from the test. |
| Boundary-row assertion depth (informational, non-blocking per task brief) | `value` boundary rows assert message content | INFO (pre-accepted, not re-litigated) | `executor_test.go:135-148` — "non-numeric value errors" / "negative value errors" / "overflow value errors" rows have no `wantErr` string, so the generic `else { if err == nil { t.Fatalf(...) } }` branch (`executor_test.go:186-190`) only asserts `err != nil`, not that the message mentions "value". Per the task brief this was already accepted at per-task review and is not re-litigated as blocking here. |

### `atlas-map-actions/script` (domain — has `model.go`, only `executor.go` changed)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01/06 | Executor logic in its established file, no catch-all | PASS | `executeMoveEnvironment`/`executeResetEnvironment` added to `executor.go` alongside every other `execute*` operation handler, mirroring `atlas-reactor-actions/script/executor.go`. |
| CONSTANTS — no local redeclaration | PASS | `executor.go:274` — `field.ParseObjectKind(params["kind"])`; saga action/payload types imported from `libs/atlas-saga`, same as `atlas-reactor-actions`. |
| SUB-04-equivalent — no manual JSON parsing | PASS | Reads `op.Params()` only; no `json.Unmarshal`/`io.ReadAll` added. |
| DOM-20 | Tests table-driven | PASS | `executor_test.go:52-` (`TestExecuteMoveEnvironment`) — `tests := []struct{...}{...}` + `t.Run`, structurally identical to the reactor-actions test. |
| DOM-24 | Test package reaching emit path stubs producer | N/A | Same `captureSagaProcessor` in-package double pattern (`executor_test.go:17-28`) — real saga/producer path never reached. |
| Boundary-row assertion depth (informational, non-blocking per task brief) | Same shape as reactor-actions | INFO (pre-accepted, not re-litigated) | `executor_test.go` (map-actions) mirrors the reactor-actions boundary rows byte-for-byte in structure; same pre-accepted, non-blocking disposition. |

## Security Review

SEC-* family did not fire — none of the three services in scope handle authentication, authorization, tokens, redirects, or secrets in this diff (N/A, per Applicability table above).

## Not evaluable from the diff

- DOM-01/02/03/11/16 for `atlas-reactor-actions/script` and `atlas-map-actions/script`: these packages have pre-existing `model.go`, so the DOM-structure family's own applies-when ("package has model.go") technically fires, but `builder.go`, `entity.go`, and `provider.go` were not touched by this diff. Grading their structural correctness would require reading files the diff does not call into and whose correctness this change does not depend on; not evaluated to stay inside the change's review surface. Would need: a full read of `builder.go`/`entity.go`/`provider.go` in both `script` packages, out of scope for a change confined to `executor.go`.
- DOM-27 in atlas-channel: the family's own trigger ("changed handler writes `http.StatusInternalServerError` and the service calls `database.Connect`") did not fire for this diff's changed files, and confirming atlas-channel has no DB backing at all (vs. simply not using it in this diff) would require surveying `main.go` beyond what this diff touches. Disposed as N/A above on the visible trigger, but a from-scratch confirmation that atlas-channel is DB-less service-wide was not performed.

## Summary

### Blocking (must fix)
- DOM-04: `atlas-channel/environment/rest.go` has no `Transform(Model) (RestModel, error)` — the package returns `RestModel` directly with a no-op `Extract`, skipping the domain-model layer the rule requires.
- DOM-05: `atlas-channel/environment/rest.go` has no `TransformSlice` function.
- EXT-01: `atlas-channel/environment/rest.go`'s `RestModel` does not implement `SetToOneReferenceID`/`SetToManyReferenceIDs`, even as no-ops — a response carrying a `relationships` block from atlas-maps would error in api2go unmarshal.

### Non-Blocking (should fix)
- DOM-28 (WARN): `consumer.go:397-403`'s new environment-replay fetch inside `SpawnForSelf` logs on failure but does not call `degrade.Observe(...)` / increment `atlas_enrichment_degraded_total`, unlike the pattern doc's decorator example — flagged as ambiguous-but-worth-fixing rather than blocking, given it mirrors untouched sibling blocks in the same function.
- INFO (pre-accepted, not counted toward blocking): boundary-row (`non-numeric`/`negative`/`overflow` `value`) test assertions in both `atlas-reactor-actions/script/executor_test.go` and `atlas-map-actions/script/executor_test.go` check only `err != nil`, not message content — already accepted at per-task review per the task brief.
