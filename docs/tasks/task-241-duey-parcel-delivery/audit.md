# Backend Audit — task-241-duey-parcel-delivery (parcel-sent notification)

- **Service Path:** services/atlas-channel, services/atlas-parcel
- **Commit Range:** 5b99d25a4..d9adf914d
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-21
- **Build:** PASS
- **Tests:** all packages `ok` in both modules, 0 failed
- **Overall:** NEEDS-WORK

## Scope note

The range requested contains three commits: `94cb58a6f` (feat: tell the sender
when a parcel_send completes), `1610d359c` (deploy: scope atlas-parcel's
topics/DB to the pr-sparse baseline), `d9adf914d` (docs only). The
`quickEnabled`→`receiveOnly` PARCEL[OPEN] field rename described in the
dispatch brief landed earlier, in base commit `5b99d25a4` itself, which is
**excluded** by the `A..B` range (exclusive on `A`) — `git log -p
5b99d25a4..d9adf914d -- libs/atlas-packet` shows zero hits for
`receiveOnly`/`quickEnabled`/`quickDeliveryEnabled`. The rename is therefore
out of this review's surface; only the PARCEL_SENT status-event feature and
the pr-sparse deploy-scoping fix are in range. `git diff --stat` confirms: 7
Go files, 1 docs file, 2 deploy YAML files.

## Build & Test Results

```
cd services/atlas-channel/atlas.com/channel && go build ./...   # exit 0, no output
cd services/atlas-channel/atlas.com/channel && go test ./... -count=1
  ok  atlas-channel/kafka/consumer/parcel   0.038s
  ... (all other packages ok)

cd services/atlas-parcel/atlas.com/parcel && go build ./...   # exit 0, no output
cd services/atlas-parcel/atlas.com/parcel && go test ./... -count=1
  ok  atlas-parcel/kafka/consumer/custody   0.028s
  ok  atlas-parcel/parcel                   0.095s
  (other packages: no test files)
```

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (DOM-01..05,11,16) | N/A | No changed file is `model.go`/`entity.go`/`rest.go`/`provider.go` (changed files are `kafka.go`, `producer.go`, `consumer.go`, `consumer_test.go`, `sent_test.go`) |
| FILE placement (FILE-01..06) | Fired | Every changed Go package runs FILE-* unconditionally |
| SUB (SUB-01..04) | N/A | No changed package has `resource.go` |
| REST (DOM-06..09,12..15,17..19,32) | N/A | No changed package has `resource.go`/`rest.go`/`processor.go`, no HTTP route registration touched |
| Constants reuse (DOM-21) | Fired | Diff adds `const StatusEventParcelSent` and `type StatusEventParcelSentBody struct{}` in both `kafka.go` files |
| Testing (DOM-10,20,24,33) | Fired | Diff adds `sent_test.go` and changes `consumer_test.go` |
| Cache (DOM-29) | N/A | No `cache.go`, no cached processor/struct state touched |
| Messaging (DOM-30) | Fired | `producer.go` changed (new `ParcelSentStatusEventProvider`); `custody/consumer.go` calls `mb.Put` |
| Multi-tenancy (DOM-31) | Fired | `handleParcelSentEvent` reads `tenant.MustFromContext(ctx)` |
| Migration hygiene (DOM-34,35) | N/A | No symbol moved between service and `libs/atlas-*` |
| Deploy & topics (DOM-22,23) | N/A | No new `libs/atlas-*` module; the four re-listed topic vars (`COMMAND_TOPIC_PARCEL*`, `EVENT_TOPIC_PARCEL*`) already existed pre-range — `1610d359c` only re-scopes them into the `pr-sparse` overlay generator, it neither adds nor renames a topic env var |
| Runtime safety (DOM-26) | Fired | Non-test `.go` files changed; `git diff ... \| grep '^\+.*go func\|^\+.*go \w'` — no hits |
| Channel wire values (DOM-25) | Fired | Diff touches `services/atlas-channel` consumer code that calls a clientbound writer |
| Resilience (DOM-27,28) | N/A | No HTTP handler writes `http.StatusInternalServerError`; no `model.Decorator` touched |
| External clients (EXT-01..04) | N/A | No `requests.*Request[T]` call added |
| Scaffolding (SCAFFOLD-01..09) | N/A | No new `services/atlas-<svc>/` dir; no `deploy/shared/routes.conf` change; the `ParcelWriter`/`ParcelSuccessfullySentBody` constants reused here are pre-existing (base commit `5b99d25a4`), not newly registered |
| Security (SEC-01..04) | N/A | No auth/token/redirect/secret code touched |

## Checklist Results

### kafka/message/parcel (support — atlas-channel and atlas-parcel, mirrored)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | No redeclaration of an existing `libs/atlas-constants/` type/const | PASS | `StatusEventParcelSent`/`StatusEventParcelSentBody` are new domain-specific event names; `grep -rn "PARCEL_SENT\|ParcelSent" libs/atlas-constants` returns nothing — no shared equivalent exists |
| FILE-06 | No package-named catch-all file carrying ≥2 of FILE-01..05's responsibilities | PASS | `kafka.go` in both modules carries only message-envelope constants/structs — none of Processor/RestModel/requests/Entity/Builder-Model-Administrator-Provider-state; unchanged shape from the pre-existing `StatusEventParcelArrived` entry |
| (cross-module mirror) | The two `kafka.go` files agree field-for-field | PASS | `services/atlas-channel/atlas.com/channel/kafka/message/parcel/kafka.go:44,74-77` and `services/atlas-parcel/atlas.com/parcel/kafka/message/parcel/kafka.go:16,42-45` both declare `const StatusEventParcelSent = "PARCEL_SENT"` and `type StatusEventParcelSentBody struct{}` — identical name, identical (empty) shape |

### kafka/producer/parcel (support — atlas-parcel)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-30 | Write-then-emit goes through `AndEmit`/`message.Buffer`, not a direct `producer.ProviderImpl(...)` call from the success path | PASS | `ParcelSentStatusEventProvider` (`services/atlas-parcel/atlas.com/parcel/kafka/producer/parcel/producer.go:33-40`) returns a `model.Provider[[]kafka.Message]`; it is invoked only via `mb.Put(...)` inside the existing `buffer.Emit(p)(...)` transaction in `custody/consumer.go:129-142`, the same pattern already used for `AcceptedStatusEventProvider` |
| FILE-01 | Processor lives in `processor.go`/`processor_<group>.go` | N/A | `producer.go` is a producer file, not a processor |

### kafka/consumer/custody (support — atlas-parcel)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-30 | Same AndEmit/Buffer discipline | PASS | `kafka/consumer/custody/consumer.go:129-142` — new `mb.Put(parcelmsg.EnvStatusEventTopic, parcelproducer.ParcelSentStatusEventProvider(b.CharacterId))` is the second `mb.Put` inside the same `buffer.Emit` closure that already emits the custody ack; both are atomic with the row create |
| (correctness) | `b.CharacterId` addresses the parcel's SENDER, not the recipient, matching the comment's claim | PASS | `parcel/processor_custody.go:88` — `AcceptCustody` calls `.SetSenderId(params.CharacterId)`, and `params.CharacterId` is populated from `b.CharacterId` at the call site (`kafka/consumer/custody/consumer.go:92`) — confirms the field is the sender's character id |
| DOM-24 | Test package reaching an emit path installs a no-op producer stub | PASS | `handleAcceptToParcel(pf providerFn)` (`kafka/consumer/custody/consumer.go:77`) takes the Kafka provider as a parameter; `consumer_test.go` injects `rp := &recordingProducer{}` (e.g. line 133) at every call site instead of `producer.ProviderImpl(...)`, so the new `mb.Put` the diff adds never reaches the real network producer — same injection style already covering the pre-existing custody-ack emit |
| DOM-20 | Table-driven `tests := []struct{...}` + `t.Run` | PASS | `consumer_test.go:124` `tests := []struct{ name string; run func(t *testing.T) }{...}`, driven at `consumer_test.go:385` `t.Run(tt.name, tt.run)`; the new assertions (`consumer_test.go:158-165`) were added inside an existing table entry, not as a new bare `t.Run` |
| DOM-26 | No bare `go` statement | PASS | `git diff 5b99d25a4..d9adf914d -- '*.go' \| grep '^\+.*go func\|^\+.*go \w'` — no hits in this file |

### kafka/consumer/parcel (support — atlas-channel)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| (registration) | New handler registered on the correct topic/handle list | PASS | `kafka/consumer/parcel/consumer.go:57-62` — `handleParcelSentEvent` is registered via `rf(t, ...)` reusing the `t` already resolved for `EnvStatusEventTopic` at line 47 (the same topic `handleParcelArrivedEvent` is on), and its `HandlerHandle` is appended to `handles` before return |
| DOM-25 | No client wire byte as a Go literal outside `libs/atlas-packet` codec internals; domain/channel code resolves via a semantic key through the writer-options table | PASS | `handleParcelSentEvent` (`kafka/consumer/parcel/consumer.go:214-231`) calls `parcelcb.ParcelSuccessfullySentBody()` — a semantic accessor, no `0x12`/mode-byte literal in the diff. The accessor and its `ParcelOperationSuccessfullySent` key predate this range (`libs/atlas-packet/parcel/clientbound/parcel_body.go:131-133`, landed in base commit `5b99d25a4`) — this diff reuses it, it does not introduce a new table or a new literal |
| DOM-31 | Tenant/trace identifiers travel in context only | PASS | `kafka/consumer/parcel/consumer.go:220-223` — `tenant.MustFromContext(ctx)` read from context; no tenant field added to any struct/body |
| DOM-24 | Emit-path stub required | N/A | `grep -n "AndEmit\|message.Emit\|producer.Produce\|producer.ProviderImpl" kafka/consumer/parcel/*.go` — no hits; `handleParcelSentEvent` writes to the socket via `session.Announce`, not Kafka |
| DOM-26 | No bare `go` statement | PASS | Same repo-wide grep as above — no hits in this file |
| **DOM-20** | Table-driven `tests := []struct{...}` + `t.Run` | **FAIL** | `sent_test.go:50-136` — `TestParcelSentEvent` uses four bare `t.Run("online sender", ...)` / `t.Run("offline sender", ...)` / `t.Run("wrong tenant", ...)` / `t.Run("wrong event type", ...)` blocks (lines 51, 76, 94, 116) with no `tests := []struct{...}` table backing them. No more-specific playbook governs a Kafka consumer-handler test file, so DOM-20's generic table-driven shape applies and is not met. |

## Not evaluable from the diff

- SCAFFOLD-07 (opcode-template seeding for new Writer/Handler constants): `ParcelWriter` and the `ParcelOperationSuccessfullySent` semantic key are reused, not newly registered, in this range — whether they are seeded in every targeted tenant's opcode template was settled by the prior (out-of-range) work and is not re-verifiable from this diff without reading `services/atlas-configurations/seed-data/templates/*`, which no file in this range touches.
- DOM-23's "re-listed in both overlays' generator" clause for the four parcel topic vars, beyond the `pr-sparse` overlay this diff touches: confirming the base overlay/configmap and the other (non-`pr-sparse`) overlay already carry these four keys would require reading `deploy/k8s/base/*` and any additional overlay, neither of which is in this diff's changed-file set.

## Summary

### Blocking (must fix)
- DOM-20: `services/atlas-channel/atlas.com/channel/kafka/consumer/parcel/sent_test.go:50-136` — `TestParcelSentEvent`'s four sub-cases are plain `t.Run(...)` calls, not the required `tests := []struct{...}` + `t.Run` table-driven form.

### Non-Blocking (should fix)
- none
