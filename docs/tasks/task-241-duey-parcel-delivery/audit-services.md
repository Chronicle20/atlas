# Backend Audit — task-241-duey-parcel-delivery (existing services)

- **Service Paths:** `services/atlas-channel`, `services/atlas-character`, `services/atlas-configurations`, `services/atlas-npc-conversations`, `services/atlas-saga-orchestrator`
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-19
- **Build:** PASS
- **Tests:** all packages touched by the diff PASS (`go test ./...` scoped to changed packages, per service)
- **Overall:** NEEDS-WORK

Scope note: this audit covers ONLY the changed packages in the five services
listed above (diff base `d9ec287b8`). `services/atlas-parcel/` is reviewed
separately (`audit-parcel.md`) and is not duplicated here, except where this
branch's saga-orchestrator/npc-conversations changes construct a payload that
crosses into atlas-parcel's contract by reference.

## Build & Test Results

```
services/atlas-channel/atlas.com/channel:            go build ./... -> clean
                                                       go test ./kafka/consumer/parcel/... ./kafka/message/parcel/... \
                                                         ./parcel/... ./saga/... ./socket/handler/... -> ok (all packages)
services/atlas-character/atlas.com/character:        go build ./... -> clean
                                                       go test ./pending_change/... -count=1 -> ok (222.661s)
services/atlas-npc-conversations/atlas.com/npc:       go build ./... -> clean
                                                       go test ./conversation/... ./saga/... -> ok (all packages)
services/atlas-saga-orchestrator/atlas.com/saga-orchestrator: go build ./... -> clean
                                                       go test ./kafka/... ./parcel/... ./saga/... -> ok (all packages)
tools/goroutine-guard.sh (repo-wide)                 -> exit 0
```

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (DOM-01..05,11,16) | Yes | `services/atlas-channel/atlas.com/channel/parcel` has `model.go`; `saga-orchestrator/parcel` has no `model.go` (N/A internally) |
| FILE placement (FILE-01..06) | Yes | Every changed Go package — unconditional |
| SUB sub-domain (SUB-01..04) | No | No changed package has `resource.go` with no `model.go` |
| REST (DOM-06..09,12..15,17..19,32) | Yes | `saga-orchestrator/parcel` has `processor.go`, `rest.go`; `pending_change` has `processor.go` |
| Constants reuse (DOM-21) | Yes | New types: `RejectReason`, `DueyActionMode`, `EventKind` additions, `gateDeps` |
| Testing (DOM-10,20,24,33) | Yes | Diff touches many `_test.go`, adds `parcel.Processor`, extends `saga.Handler`/mocks |
| Cache (DOM-29) | No | No `cache.go` touched, no cached struct state added |
| Messaging (DOM-30) | Yes | `parcel/producer.go`, `AndEmit` usage in `saga-orchestrator/parcel/processor.go` |
| Multi-tenancy (DOM-31) | Yes | `rest.go` files added/touched; all reviewed models carry no tenant/trace field |
| Migration hygiene (DOM-34,35) | No | No symbol moved between a service and `libs/atlas-*` |
| Deploy & topics (DOM-22,23) | Yes | New topic env vars `COMMAND_TOPIC_PARCEL`, `COMMAND_TOPIC_PARCEL_CUSTODY`, `EVENT_TOPIC_PARCEL_CUSTODY_STATUS`, `EVENT_TOPIC_PARCEL_STATUS` |
| Runtime safety (DOM-26) | Yes | Non-test Go files changed; `goroutine-guard.sh` exit 0 |
| Channel wire values (DOM-25) | Yes | Diff touches `services/atlas-channel` |
| Resilience (DOM-27,28) | No | No `model.Decorator` / enrichment path changed |
| External clients (EXT-01..04) | Yes | `channel/parcel/requests.go`, `saga-orchestrator/parcel/requests.go`, `character/pending_change/requests.go` (gate 12) |
| Scaffolding (SCAFFOLD-01..09) | No | No `services/atlas-<svc>/` directory added in this scope; no new channel Writer/Handler registered by these five services (only new dispatcher modes resolved through existing config table) |
| Security (SEC-01..04) | No | None of these five services' changed code handles auth tokens/redirects/secrets |

## Checklist Results

### atlas-channel/parcel (support / read-only atlas-parcel client — has `model.go`, no `rest.go`, no `builder.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` exists (package has `model.go`) | FAIL | No `builder.go` in `services/atlas-channel/atlas.com/channel/parcel/` (`ls` confirms only `fee.go`, `fee_test.go`, `model.go`, `model_test.go`, `processor.go`, `requests.go`, `validation.go`, `validation_test.go`). |
| DOM-02/03 | `entity.go` `ToEntity`/`Make` | N/A | No `entity.go` in package. |
| DOM-04/05 | `rest.go` `Transform`/`TransformSlice` | N/A | No `rest.go` in package — trigger absent. |
| DOM-06 | Processor ctor takes `logrus.FieldLogger` | PASS | `parcel/processor.go:188` `func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor`. |
| DOM-11 | provider.go lazy evaluation | N/A | No `provider.go`. |
| DOM-16 | administrator.go for writes | N/A | No local DB writes (remote-only client). |
| FILE-01 | Processor in `processor.go` | PASS | `Processor`/`NewProcessor`/methods all in `parcel/processor.go:183-245`. |
| FILE-02 | `RestModel`/`Extract`/JSON:API methods in `rest.go` | FAIL (Important) | No `rest.go` exists; `RestModel` struct, `GetName`/`GetID`/`SetID`/`SetToOneReferenceID`/`SetToManyReferenceIDs`, and `Extract(` all live in `parcel/processor.go:22-179` instead. |
| FILE-03 | requests in `requests.go` | PASS | `parcel/requests.go` — `getBaseRequest`, `requestForRecipient`, `requestById`, `discardRequest`, `notifyRequest`. |
| FILE-04 | entity struct/Migration/TableName | N/A | No `entity.go`, package has no local persistence entity. |
| FILE-05 | Builder / domain `Model` / writes / readers placement | FAIL | Domain `Model` struct is defined in `parcel/processor.go:91-112`, not `model.go` (which holds only `ToPacket`/`WireId`). |
| FILE-06 | No single file carrying ≥2 responsibilities | FAIL (Important) | `parcel/processor.go` carries Processor (FILE-01) + RestModel/Extract (FILE-02) + domain Model (FILE-05) in one file — a collapsed-file violation of the same shape the pattern doc calls out (task-102 `wallet.go`). |
| DOM-25 | Client-interpreted wire bytes from config table | PASS | `duey_action.go:65-86` `isDueyAction` resolves `DueyActionMode` from `readerOptions["operations"]`, never a literal. |
| EXT-01 | Target RestModel implements ref-ID stubs | PASS | `parcel/processor.go:86,88` (`RestModel`); `parcel/requests.go:74,76` (`discardRestModel`); `:105,107` (`notifyRestModel`). |
| EXT-02 | httptest-backed integration test | FAIL | No `httptest` anywhere in `services/atlas-channel/atlas.com/channel/parcel/` (`grep -rln httptest .` → no matches); `model_test.go`/`fee_test.go`/`validation_test.go` construct `RestModel` literals directly, never round-tripping through an actual HTTP response. |
| EXT-03 | Only genuine 404s map to "not found" | PASS | No `errors.Is(err, requests.ErrNotFound)` anywhere in the package — every remote error bubbles raw (`parcel/processor.go:194,219,231,244`), so nothing is misclassified. |
| EXT-04 | URL via `requests.RootUrl`/`RootUrlFor` | PASS | `parcel/requests.go:29` `requests.RootUrlFor(ctx, "PARCEL")`. |
| DOM-30 | AndEmit + Buffer atomicity | N/A | Package emits no Kafka messages (Processor is a pure REST client). |

### atlas-channel/kafka/consumer/parcel (support — consumer package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | Consumer-package shape (InitConsumers/InitHandlers/handler funcs) | N/A | No `Processor interface`/`RestModel`/`entity struct`/`Builder` declared in `consumer.go` — matches the repo-wide consumer-package convention, not the domain-package file table. |
| DOM-25 | Client wire bytes from config | PASS | `handleShowParcelCommand`/`showParcel` (`consumer.go:77-161`) build packet bodies from `dueyparcel.Model` fields, no client dispatcher-byte literal introduced. |
| DOM-26 | goroutine safety | PASS | `tools/goroutine-guard.sh` exit 0; no bare `go` in non-test files of this package. |
| DOM-31 | tenant/trace in context only | PASS | `handleShowParcelCommand`/`handleParcelArrivedEvent` read `tenant.MustFromContext(ctx)`; no tenant field on `ShowParcelCommand`/`StatusEvent`. |

### atlas-channel/socket/handler (duey_action*.go, character_cash_item_use_duey.go — existing large support package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-25 | Client wire bytes from config | PASS | `duey_action.go:65-86` (`isDueyAction`); `CashSlotItemType(31/32)` (`character_cash_item_use.go:1422-1431`) is an INBOUND classification never passed into a `*Body(...)` call (`grep` for `CashSlotItemType.*Body(` → no matches) — extends the pre-existing `GetCashSlotItemType` table shape unchanged, not new outbound wire-byte hardcoding. |
| DOM-13/14 | No cross-domain orchestration / provider calls in handlers | PASS | `sendParcel`/`receiveParcel`/`discardParcel` call only injected `deps.*` functions that themselves call `Processor` methods (`dueyparcel.NewProcessor(...).CountPending`, etc.) — no direct provider access. |
| DOM-26 | goroutine safety | PASS | `tools/goroutine-guard.sh` exit 0; the only new bare `go func()` is in a `_test.go` (`duey_action_receive_test.go` region, diff line ~353), out of DOM-26's non-test trigger. |

### atlas-character/pending_change (domain package, gate 12 addition)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| EXT-01 | Ref-ID stubs on target RestModel | WARN | New `parcelStatusRestModel` (`pending_change/rest.go:218-225`) has no `SetToOneReferenceID`/`SetToManyReferenceIDs`, unlike sibling `worldRestModel`/`familyMemberRestModel`/`partyRestModel` in the same file. Low risk per the doc comment ("No relationships block", cross-checked against `services/atlas-parcel/atlas.com/parcel/parcel/resource.go:55` — `handleGetParcelStatus` always returns 200, never carries a relationships block) but the rule text requires the stubs unconditionally ("even as no-ops"); not upgraded to blocking because the endpoint is confirmed relationship-free. |
| EXT-02 | httptest-backed integration test | PASS | `pending_change/requests_test.go:90-149` builds `parcelStatusDoc` fixtures and serves them via `httptest.NewServer`, asserting `parcelPending(...)` returns the populated/negative/absent cases. |
| EXT-03 | Only genuine 404s → not found | PASS | `parcelPending` (`requests.go:207-214`) has no special-case at all; verified atlas-parcel's `handleGetParcelStatus` (`services/atlas-parcel/.../resource.go:273-290`) never 404s this route, so no misclassification is possible. |
| EXT-04 | URL via `requests.RootUrl` | PASS | `pending_change/requests.go:201` `requests.RootUrl("PARCEL")`. |
| DOM-21 | No redeclaration of shared constants | PASS | `world.Id` reused throughout `gateDeps`; no numeric-literal classification introduced besides the pre-existing `guildMasterTitle`/`worldStatusFull` pattern. |
| DOM-23 | Topic env vars in configmap/overlays | N/A | This gate is a synchronous REST call, not a Kafka topic. |

### atlas-npc-conversations/conversation (operation_executor.go — `open_duey` operation)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | Reuse shared saga action/payload types | PASS | `saga.ShowParcel`/`saga.ShowParcelPayload` re-exported from `sharedsaga` (`saga/model.go:42,125`), not redeclared. |
| **Cross-service seam** | Saga step this operation constructs must deserialize at the orchestrator | **FAIL (blocking)** | See saga-orchestrator finding below — `case "open_duey"` (`conversation/operation_executor.go:2130-2151`) emits a `saga.Step{Action: saga.ShowParcel, Payload: saga.ShowParcelPayload{...}}`; the orchestrator that receives this JSON has no `case ShowParcel:` in its `Step[T].UnmarshalJSON`, so every `open_duey` saga this operation builds will fail orchestrator-side with `"unknown action: show_parcel"` before `handleShowParcel` ever runs. |

### atlas-saga-orchestrator/saga (model.go, handler.go, event_acceptance.go, compensator.go, processor.go)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| **Cross-service seam / functional correctness** | `Step[T].UnmarshalJSON` covers every declared `Action`, including the new parcel actions | **FAIL (blocking)** | `saga/model.go:1557-1587` adds `case TransferToParcel:`, `case AcceptToParcel:`, `case ReleaseFromParcel:`, `case WithdrawFromParcel:` but has **no `case ShowParcel:`**, even though `ShowParcel`/`ShowParcelPayload` are declared and re-exported (`saga/model.go:210,351`) and registered in `acceptanceTable` (`saga/event_acceptance.go:264`) and in the handler dispatch table (`saga/handler.go:947-948`, `handleShowParcel` at `:2618-2639`, type-asserting `st.Payload().(ShowParcelPayload)`). Any inbound saga carrying a `ShowParcel` step — produced by `atlas-channel/socket/handler/character_cash_item_use_duey.go:90-110` (classification 533 Quick Delivery Ticket) and by `atlas-npc-conversations/conversation/operation_executor.go`'s `open_duey` case — hits the `default: return fmt.Errorf("unknown action: %s", s.action)` branch (`saga/model.go:1800-1801`) and fails to unmarshal, breaking the entire saga before `handleShowParcel` is reached. No test in this diff (`saga/parcel_expansion_test.go`, `saga/parcel_compensation_test.go`, `saga/mts_expansion_test.go`) round-trips a `ShowParcel` step through `Step[T]`'s `MarshalJSON`/`UnmarshalJSON`, which is exactly the test that would have caught this. |
| DOM-30 | AndEmit + Buffer atomicity | PASS | Every `*AndEmit` method in `saga-orchestrator/parcel/processor.go:110-168` wraps `message.Emit(p.p)(func(mb *message.Buffer) error {...})`. |
| DOM-33 | Interface change updates every mock | PASS | `parcel.Processor` gained no new methods beyond what `parcelTestMock` implements — all 8 methods present (`saga/parcel_compensation_test.go:53-99`). |
| DOM-23 | Topic env vars wired | PASS | `COMMAND_TOPIC_PARCEL`, `COMMAND_TOPIC_PARCEL_CUSTODY`, `EVENT_TOPIC_PARCEL_CUSTODY_STATUS`, `EVENT_TOPIC_PARCEL_STATUS` all present as `KEY: "KEY"` in `deploy/k8s/base/env-configmap.yaml:61-62,155-156` and re-listed in both `deploy/k8s/overlays/pr/kustomization.yaml:224-225,316-317` and `deploy/k8s/overlays/main/kustomization.yaml:100-101,192-193`. |
| DOM-26 | goroutine safety | PASS | `tools/goroutine-guard.sh` exit 0. |

### atlas-saga-orchestrator/parcel (support — command-dispatch + REST client)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor in `processor.go` | PASS | `parcel/processor.go:77-168`. |
| FILE-02 | RestModel/JSON:API methods in `rest.go` | PASS | `parcel/rest.go:50-94`. |
| FILE-03 | requests in `requests.go` | PASS | `parcel/requests.go`. |
| FILE-06 | No collapsed catch-all file | PASS | Responsibilities cleanly split across `processor.go`/`producer.go`/`requests.go`/`rest.go` — contrast with `atlas-channel/parcel` above. |
| EXT-01 | Ref-ID stubs | PASS | `parcel/rest.go:92,94`. |
| EXT-02 | httptest-backed integration test | FAIL | `ls services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/parcel/` → `processor.go, processor_test.go, producer.go, requests.go, rest.go`; `grep -rln httptest .` → no matches. `RequestParcel` (called from `saga/processor.go:2309`, `expandWithdrawFromParcel`) is untested against a live-shaped JSON:API fixture. |
| EXT-03 | Only genuine 404 → not found | PASS | No `errors.Is(err, requests.ErrNotFound)` in the package — raw error bubble, no misclassification. |
| EXT-04 | URL via `requests.RootUrlFor` | PASS | `parcel/requests.go:20`. |
| DOM-21 | Shared constants reused | PASS | `world.Id`, `channel.Model` reused (`parcel/processor.go:11`, `producer.go:10`), not redeclared. |

### atlas-saga-orchestrator/kafka/consumer/parcel/custody, kafka/message/parcel/custody, kafka/message/parcel (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | Consumer/message-package shape | N/A | No domain-file responsibilities (Processor/RestModel/entity/Builder) declared here — mirrors the pre-existing MTS custody twin exactly (`kafka/message/mts/custody/kafka.go`). |
| DOM-30 | Custody status events routed through `AcceptEvent`/`StepCompleted` | PASS | `kafka/consumer/parcel/custody/consumer.go:48-97`. |

## Security Review

Not applicable — none of the five services' changed code in this diff handles authentication, authorization, tokens, redirects, or secrets (SEC-* trigger did not fire).

## Not evaluable from the diff

- EXT-02 risk assessment for `services/atlas-channel/atlas.com/channel/parcel` and `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/parcel`: whether atlas-parcel's actual `GET /parcels/{id}` / `GET /parcels?filter[...]` responses carry a `relationships` block (which would make the missing `httptest` coverage and the `EXT-01`-adjacent gap in `atlas-channel/parcel` an active bug rather than a coverage gap) — would require reading `services/atlas-parcel/atlas.com/parcel/parcel/rest.go`'s `Transform` output shape, which is the sibling reviewer's surface (`audit-parcel.md`), not re-verified here.
- Whether `services/atlas-configurations/seed-data/templates/*.json` changes (gms_72/79/83/84/87/92, jms_185) correctly seed the `DUEY_ACTION` operation codes `isDueyAction` resolves at runtime — the diff is JSON-only (not a changed Go package under this audit's scope), and cross-checking each template's `operations` block against `duey_action.go`'s `DueyActionMode` keys would require re-deriving the full opcode table from `docs/packets/dispatchers/duey_action.yaml`, which is packet-audit territory already covered by the branch's own packet-verification commits (see `git log` — "verify PARCEL and DUEY_ACTION on gms_v92").
- Whether the `saga.ShowParcel` `UnmarshalJSON` gap (blocking finding above) is also present in any other consumer of the shared `libs/atlas-saga` `Step[T]` type — only `atlas-saga-orchestrator`'s own local `Step[T]` (which re-implements its own `UnmarshalJSON`, distinct from the shared library's) was checked; `libs/atlas-saga` itself is outside this audit's five-service scope.

## Summary

### Blocking (must fix)

- **`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go:1557-1587`** — `Step[T].UnmarshalJSON` is missing `case ShowParcel:`, even though `ShowParcel`/`ShowParcelPayload` are declared, registered in `event_acceptance.go:264`, and dispatched by `handler.go:947-948,2618-2639`. Every saga containing a `ShowParcel` step — produced by `atlas-channel/socket/handler/character_cash_item_use_duey.go:90-110` (classification 533 Quick Delivery Ticket) and `atlas-npc-conversations/conversation/operation_executor.go:2130-2151` (`open_duey`) — fails orchestrator-side unmarshal with `"unknown action: show_parcel"` before ever reaching `handleShowParcel`. No test in the diff exercises the `Step[T]` JSON round-trip for this action.
- **`services/atlas-channel/atlas.com/channel/parcel/processor.go:22-179`** (FILE-02, FILE-05, FILE-06) — `RestModel`, `Extract`, and the domain `Model` struct all live in `processor.go` with no `rest.go` in the package; a collapsed-file violation of the same shape the pattern doc names explicitly (task-102 `wallet.go`).
- **`services/atlas-channel/atlas.com/channel/parcel/`** (DOM-01) — package has `model.go` with no `builder.go`.

### Non-Blocking (should fix)

- **EXT-02** — `services/atlas-channel/atlas.com/channel/parcel/` and `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/parcel/` both call another atlas service via `requests.GetRequest[T]`/`PatchRequest[T]` with no `httptest`-backed integration test asserting a populated domain struct from a representative JSON:API fixture.
- **EXT-01** — `services/atlas-character/atlas.com/character/pending_change/rest.go:218-225` (`parcelStatusRestModel`) has no `SetToOneReferenceID`/`SetToManyReferenceIDs` no-op stubs; low risk since the upstream route is confirmed relationship-free, but the rule requires them unconditionally.
