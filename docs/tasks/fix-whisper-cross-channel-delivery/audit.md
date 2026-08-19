# Backend Audit — atlas-channel (fix-whisper-cross-channel-delivery)

- **Service Path:** services/atlas-channel/atlas.com/channel
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-19
- **Audit Range:** 748510af2..489bb3327
- **Scope:** `kafka/consumer/message/consumer.go`, `kafka/consumer/message/consumer_test.go` (new)
- **Build:** PASS
- **Tests:** package `atlas-channel/kafka/consumer/message` — `ok` (1.831s), all tests passed
- **Overall:** NEEDS-WORK

## Build & Test Results

```
$ go build ./...
(no output — success)

$ go test ./kafka/consumer/message/... -count=1
ok  	atlas-channel/kafka/consumer/message	1.831s
```

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (DOM-01..05,11,16) | No | Package `kafka/consumer/message` has no `model.go`, `entity.go`, `rest.go`, or `provider.go` (`ls` of the package dir shows only `consumer.go`/`consumer_test.go`) |
| FILE placement (FILE-01..06) | Yes | Runs on every changed Go package unconditionally |
| SUB (SUB-01..04) | No | No `resource.go` in the package |
| REST (DOM-06..09,12..15,17..19,32) | No | No `resource.go`/`rest.go`/`processor.go`; no HTTP route registration in the diff |
| Constants reuse (DOM-21) | No | Diff declares no new type, named const block, or numeric-literal classification (`whisperDeliveryPlan`/`presentRecipients` add functions only) |
| Testing (DOM-10,20,24,33) | Yes | Diff adds `consumer_test.go` |
| Cache (DOM-29) | No | No `cache.go`; no cached state held by any struct in the diff |
| Messaging (DOM-30) | No | `consumer.go` calls no `AndEmit`/`message.Emit`/`producer.ProviderImpl`; it only reads (`character.NewProcessor(...).GetById()`) and writes to sessions (`session.Announce`) |
| Multi-tenancy (DOM-31) | Yes | `consumer.go` reads `tenant.MustFromContext(ctx)` |
| Migration hygiene (DOM-34,35) | No | No symbol moved to/from `libs/atlas-*` |
| Deploy & topics (DOM-22,23) | No | No topic env var added or renamed |
| Runtime safety (DOM-26) | Yes | `consumer.go` (non-test) changed |
| Channel wire values (DOM-25) | Yes | Diff touches `services/atlas-channel` |
| Resilience (DOM-27,28) | No | `consumer.go` is a Kafka consumer handler, not a DB-backed REST handler; no `model.Decorator` changed |
| External clients (EXT-01..04) | No | `consumer.go` itself calls no `requests.RootUrl`/`GetRequest[T]`/`PostRequest[T]` — it calls `character.NewProcessor(...).GetById()`, whose own package (`character/`) is unchanged and out of scope |
| Scaffolding (SCAFFOLD-01..09) | No | No new service directory, writer/handler registration, or `routes.conf` change |
| Security (SEC-01..04) | No | atlas-channel's chat consumer handles no tokens, auth, redirects, or secrets |

## Checklist Results

### kafka/consumer/message (support package — no `model.go`, no `resource.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor interface/constructor/methods live in `processor.go` | N/A | No `Processor`/`ProcessorImpl`/`NewProcessor` declared in this package's files |
| FILE-02 | `RestModel`/`Transform`/`Extract`/JSON:API methods live in `rest.go` | N/A | No `RestModel`/`Transform`/`Extract` declared |
| FILE-03 | Cross-service request functions live in `requests.go` | N/A | No `requests.RootUrl`/`GetRequest`/`PostRequest`/`getBaseRequest` in this package |
| FILE-04 | Entity struct/`Migration`/`TableName` live in `entity.go` | N/A | No entity struct declared |
| FILE-05 | Builder/`Model`/writes/readers placed per file table | N/A | No `Builder`, domain `Model`, `Create*`/`Update*`/`Delete*`, or `database.Query`/`SliceQuery` in this package |
| FILE-06 | No package-named catch-all file carrying ≥2 responsibilities | PASS | `consumer.go` carries none of the FILE-01..05 responsibilities (Processor/RestModel/requests/entity/builder) — it is a single-purpose Kafka-consumer-handler file (`kafka/consumer/message/consumer.go:1-312`), matching the established `kafka/consumer/<domain>/consumer.go` shape used elsewhere in the service |
| DOM-10 | Test DB setup calls `database.RegisterTenantCallbacks` | N/A | `consumer_test.go` opens no GORM DB |
| DOM-20 | Tests are table-driven (`tests := []struct{...}` + `t.Run`) | **FAIL** | `kafka/consumer/message/consumer_test.go` — no `t.Run` and no `tests := []struct{...}` anywhere in the file (`grep -n "t.Run\|tests := \[\]struct" consumer_test.go` → no matches). 17 separate `Test...` functions are used instead, several structurally identical across chat types (e.g. `TestHandleMultiChat_RecipientChannel_Delivers` at line 314, `TestHandleMessengerChat_RecipientChannel_Delivers` at line 391, `TestHandlePinkChat_RecipientChannel_Delivers` at line 462 differ only in the handler/event constructor called) — exactly the shape DOM-20 is written to collapse into one table |
| DOM-24 | Test packages reaching an emit path install `producertest`/inject a no-op producer | N/A | Test entry points (`handleWhisperChat`, `handleMultiChat`, `handleMessengerChat`, `handlePinkChat`) reach only `character.NewProcessor(...).GetById()` (HTTP GET, not Kafka) and `session.Announce`/`session.NewProcessor(...).IfPresentByCharacterId`/`GetByCharacterId` (socket write, not Kafka). `session.NewProcessor` does store a `producer.Provider` closure (`session/processor.go:76`) but it is only invoked from `SessionCreated` (`session/processor.go:349`) and `Destroy` (`session/processor.go:426`), neither of which any test path here reaches — `pipedSession`'s `Create(...)` call (`consumer_test.go:121`) does not call `SessionCreated` (`session/processor.go:364-378`) |
| DOM-33 | Every mock of a changed `Processor`/`Provider`/`Administrator` interface updated in the same diff | N/A | No interface method added/removed/re-signed in this diff — `handleWhisperChat` etc. are free functions, not interface methods |
| DOM-26 | Every goroutine spawned via `routine.Go(l, ctx, fn)` | PASS | `grep -n "^\s*go \|go func" kafka/consumer/message/consumer.go` → no matches; no goroutine is spawned in the changed non-test file |
| DOM-31 | Tenant/trace identifiers travel in context only | PASS | `consumer.go:89,122,199,244,264,284` all read tenant via `tenant.MustFromContext(ctx)`; `whisperDeliveryPlan(t tenant.Model, ...)` (`consumer.go:186`) receives it as an ordinary internal function parameter derived from context, not as a REST-model field, request body, or public path/query parameter |
| DOM-25 | Client-interpreted wire bytes resolved from a tenant writer-options table, never a Go literal | **FAIL** | `consumer.go:210`: `fieldcb.NewWhisperSendResult(0x0A, tc.Name(), true)` and `consumer.go:228`: `fieldcb.NewWhisperReceive(0x12, c.Name(), byte(e.ChannelId), c.Gm(), e.Message)` pass raw Go byte literals (`0x0A`, `0x12`) directly as sub-op-code arguments instead of resolving them via `WithResolvedCode(...)`/a tenant writer-options table. These two literals are unchanged in *value* from the pre-fix code (they were already present before this branch, at `748510af2`) but this diff substantially restructures the surrounding `handleWhisperChat` body (moving each literal into a new conditional branch), so they are part of the reviewed surface; the violation is real regardless of which commit first introduced it |

## Security Review

Not applicable — SEC-* trigger did not fire (no token/auth/redirect/secret handling in `kafka/consumer/message`).

## Not evaluable from the diff

- DOM-25 (rollout/versioning half of the check): whether `0x0A`/`0x12` are meant to be seeded per-tenant-version in `services/atlas-configurations/seed-data/templates/` cannot be settled from this diff alone — that would require reading the fieldcb `WhisperSendResult`/`WhisperReceive` codec and the seed-template directory, both outside the two changed files and pre-dating this fix. Recorded as a FAIL on the literal-in-code test (step 1 of the verification procedure); step 2/4 (table-seeding verification) is not evaluated.

## Summary

### Blocking (must fix)
- DOM-20: `kafka/consumer/message/consumer_test.go` does not use the table-driven `tests := []struct{...}` + `t.Run` pattern — 17 ad hoc `Test...` functions, several near-duplicates across chat types, should collapse into table-driven subtests.
- DOM-25: `kafka/consumer/message/consumer.go:210` and `:228` pass raw client-interpreted sub-op-code literals (`0x0A`, `0x12`) into `fieldcb.NewWhisperSendResult`/`NewWhisperReceive` instead of resolving them from a tenant writer-options table.

### Non-Blocking (should fix)
- None identified.

## DOM-20 remediation

`kafka/consumer/message/consumer_test.go` no longer uses ad hoc `Test...` functions for the
scenarios that repeat across chat types. The 13 `Test...` functions that existed (whisper: 4,
multi: 3, messenger: 3, pink: 3) collapse to:

- `TestHandleChat_RecipientChannel_Delivers` — table-driven (`tests := []struct{...}` +
  `t.Run`) over `{whisper, multi, messenger, pink}`, sharing the generic helper
  `runRecipientChannelDelivers[B any]`. Replaces `TestHandleWhisperChat_RecipientChannel_DeliversReceiveOnly`,
  `TestHandleMultiChat_RecipientChannel_Delivers`, `TestHandleMessengerChat_RecipientChannel_Delivers`,
  `TestHandlePinkChat_RecipientChannel_Delivers`.
- `TestHandleChat_TwoRecipientsTwoChannels_EachDeliversOwnOnly` — table-driven over
  `{multi, messenger, pink}` (whisper has no two-recipient equivalent), sharing
  `runTwoRecipientsTwoChannelsEachDeliversOwnOnly[B any]`. Replaces
  `TestHandleMultiChat_TwoRecipientsTwoChannels_EachDeliversOwnOnly`,
  `TestHandleMessengerChat_TwoRecipientsTwoChannels_EachDeliversOwnOnly`,
  `TestHandlePinkChat_TwoRecipientsTwoChannels_EachDeliversOwnOnly`.
- `TestHandleChat_DifferentWorld_NoAnnounce` — table-driven over `{whisper, multi, messenger,
  pink}`, sharing `runDifferentWorldNoAnnounce[B any]`. Replaces
  `TestHandleWhisperChat_DifferentWorld_NoAnnounce`, `TestHandleMultiChat_DifferentWorld_NoAnnounce`,
  `TestHandleMessengerChat_DifferentWorld_NoAnnounce`, `TestHandlePinkChat_DifferentWorld_NoAnnounce`.

The two genuinely whisper-only scenarios (`TestHandleWhisperChat_SenderChannel_DeliversResultOnly`,
`TestHandleWhisperChat_SameChannel_DeliversBothOnce`) have no analogue in the other chat types
(multi/messenger/pink never emit a sender confirmation), so they remain standalone `Test...`
functions rather than being forced into a table with only one row.

The shared row logic is a Go-generic helper (`chatHandler[B any]`, instantiated explicitly at
each call site, e.g. `runRecipientChannelDelivers[message3.WhisperChatBody](...)`) rather than a
plain `any`-typed closure, so each row still resolves to the concrete `handleXxxChat` function and
its matching event-body type at compile time. All 13 original scenarios are still asserted; total
test count went from 13 top-level `Test...` functions (0 subtests) to 5 top-level `Test...`
functions (11 `t.Run` subtests + 2 standalone), verified with
`go test ./kafka/consumer/message/... -v`.
