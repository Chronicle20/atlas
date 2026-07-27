# Backend Audit — task-137-note-item-consumption (Go changes)

- **Scope:** whole-branch diff `b64ee7936..619bb66ea` (55 Go/tooling files), per audit request — note-feature codec, saga, atlas-notes, saga-orchestrator, atlas-channel.
- **Guidelines Source:** `.claude/skills/backend-dev-guidelines/resources/*`
- **Date:** 2026-07-25
- **Build:** PASS (spot-verified: `go build ./...` clean in `libs/atlas-packet`, `libs/atlas-saga`, `services/atlas-notes/atlas.com/notes`, `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`, `services/atlas-channel/atlas.com/channel`)
- **Tests:** not re-run in full (Task 18 gate already recorded all 5 modules green with `-race`); no reason found to distrust that record
- **Overall:** **PASS** — zero Critical, zero Important, one Minor (pre-vetted by the requester) findings.

## Mindset note

Every item below was graded against the specific guideline text (file-responsibilities.md table, DOM-25/anti-patterns.md client-wire-value rule, DOM-24 kafka-producertest rule), not against "this is how the rest of the service already does it." Where a pattern recurs across sibling files (e.g. `producer.ProviderImpl(...)` called directly from a Kafka consumer error branch), I checked whether the guideline documents an actual DB-transaction/buffer requirement for that specific call site before deciding it's not a violation — see the DOM-notes section.

## Files reviewed

- `libs/atlas-packet/cash/serverbound/item_use.go`, `item_use_note.go` (+ tests)
- `libs/atlas-packet/note/clientbound/operation_body.go` (+ tests), `libs/atlas-packet/note/serverbound/operation_discard.go` (+ tests)
- `libs/atlas-saga/model.go`, `payloads.go`, `unmarshal.go` (+ test)
- `services/atlas-notes/atlas.com/notes/note/{processor,producer,resource,mock/processor}.go` (+ tests), `kafka/consumer/note/consumer.go`, `kafka/message/note/kafka.go`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/note/{processor,producer,mock/processor}.go` (+ test), `kafka/consumer/note/consumer.go` (+ test + testmain), `kafka/message/note/kafka.go`, `main.go`, `saga/{model,handler,event_acceptance,producer,compensator}.go` (+ tests)
- `services/atlas-channel/atlas.com/channel/compartment/model.go` (+ test), `saga/model.go`, `socket/handler/{note_send,note_operation,character_cash_item_use}.go` (+ tests), `kafka/message/saga/kafka.go`, `kafka/consumer/saga/consumer.go` (+ test), `note/{processor,producer}.go` (dead-code removal)
- `tools/packet-audit/internal/matrix/build.go` (tooling, DOM rules applied loosely per instructions)

## Domain / Sub-domain / Support Checklist Results

None of the touched packages introduced a new `model.go`-bearing domain package — all changes are additions to existing domain packages (`note` in atlas-notes, already has model.go — DOM checklist below), a new Kafka-only support package (`note` in saga-orchestrator — File Responsibilities Checklist), and packet-codec / saga-model library additions (no REST/DB layer, DOM checklist N/A).

### `services/atlas-notes/.../note` (existing domain package, model.go present)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor accepts `FieldLogger` | PASS | `note/processor.go` unchanged constructor signature; not touched by this diff, still `logrus.FieldLogger` |
| DOM-08 | POST uses `RegisterInputHandler` | PASS (pre-existing, unaffected) | `note/resource.go` — `CreateNoteHandler` still registered via `RegisterInputHandler[RestModel]` (unchanged in diff) |
| DOM-09 | Transform errors handled | PASS | no `rest.go` Transform call sites touched by this diff |
| DOM-15 | No direct entity creation in handlers | PASS | `note/resource.go:152` calls `NewProcessor(...).CreateAndEmit(uuid.Nil, ...)`, no `db.Create` |
| DOM-17 | Error → HTTP status mapping | PASS | `note/resource.go:153` `server.WriteErrorResponse(d.Logger())(w)(err)` on failure |
| DOM-24 | Kafka producer stubbed in tests that emit | PASS | `note/testmain_test.go:11` `producertest.InstallNoop()`, no per-test `t.Cleanup(producer.ResetInstance)` found |
| DOM-27 | Transient DB errors → 503 | PASS (pre-existing) | `main.go:64-65` registers `server.RegisterTransientErrorClassifier` composing `database.IsTransientConnectionError` |

`Create`/`CreateAndEmit` signature change (adding `transactionId uuid.UUID` as the leading curried argument) is threaded consistently through `note/processor.go:23,78`, `note/mock/processor.go:9-19,29-34`, all call sites in `processor_test.go`, `processor_fame_award_test.go`, `resource.go:152`, and `kafka/consumer/note/consumer.go:51` — no stale call site found (`go build` confirms).

### `services/atlas-saga-orchestrator/.../note` (new support package — Kafka command dispatch, no model.go/DB)

File Responsibilities Checklist:

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor interface + impl in `processor.go` | PASS | `note/processor.go:14-38` — `Processor` interface and `ProcessorImpl` both declared there, no other file has a `Processor`/`ProcessorImpl` symbol |
| FILE-03 | N/A — no cross-service REST call | N/A | package only produces Kafka commands (`note/producer.go`), no `requests.GetRequest`/`PostRequest` |
| FILE-06 | No package-named catch-all file | PASS | package has exactly `processor.go`, `producer.go`, `mock/processor.go`, `producer_test.go` — each single-purpose, no `note.go` bundling multiple responsibilities |

`note.CreateNote` action wiring is complete end to end: `libs/atlas-saga/model.go:184-185` (Action const) → `payloads.go:962-967` (`CreateNotePayload`) → `unmarshal.go:588-593` (switch case) → `saga-orchestrator/saga/model.go:191,303` (re-export) → `saga/model.go:1494-1497` (local unmarshal switch case) → `saga/handler.go:938-939,3145-3167` (`GetHandler` dispatch + `handleCreateNote`) → `saga/event_acceptance.go:100-101,149-150,370-373` (EventKind + acceptance table + outcome table) → `saga/compensator.go` (`compensateNoteSend`, `DispatchNoteSendRollbacks`). Verified with `git diff` line-by-line; no dangling reference.

### `services/atlas-channel` socket/handler + saga/compartment (packet-dispatch layer, not a REST domain package — File Responsibilities table's resource.go/rest.go split doesn't literally apply; graded on DOM-25 and dead-code hygiene instead)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | Use atlas-constants types, no reinvention | PASS | `note_operation.go:47` `inventory.TypeValueCash`; `:53` `item.ClassificationNote`; `compartment/model.go:83` `item.GetClassification(item.Id(...))` — all resolve to existing `libs/atlas-constants` symbols (verified via grep against `libs/atlas-constants/inventory/constants.go:16` and `item` package) |
| DOM-25 | Client wire values config-resolved | PASS | `character_cash_item_use.go:47` replaces a raw `t.MajorVersion() >= 87` literal with `cashsb.UpdateTimeFirst(t)` (the shared codec's Region/MajorVersion gate — a wire-LAYOUT decision, not a client-interpreted mode/error byte, per the task's documented exception); `note_operation.go` NOTE_ACTION SEND error path resolves `notecb.NoteSendErrorNoNoteItem`/`NoteSendErrorReceiverOnline`/`NoteSendErrorReceiverUnknown` — all semantic keys resolved by `notecb.NoteSendErrorBody` → `atlas_packet.ResolveCode(l, options, "operations"/"errors", key)` (`libs/atlas-packet/note/clientbound/operation_body.go:46-54`), never a raw byte in channel code |
| — | Dead code cleanup after re-architecture | PASS | `note/processor.go` (channel) — `SendNote` method removed along with `note/producer.go`'s now-unused `CreateCommandProvider`, since the direct client→atlas-notes command path was replaced by the destroy-first saga (`git diff` confirms both deletions, no remaining call sites — `go build` clean) |

`compartment.Model.FindFirstByClassification` (new method, `compartment/model.go:75-82`) is placed correctly — it's an addition to the existing domain package's `model.go`, matching file-responsibilities.md's "`model.go`: immutable domain objects... accessor methods."

### Kafka producer stubbing (DOM-24) — full sweep of new/touched test packages that emit

| Package | Emits? | Stub mechanism | Verdict |
|---|---|---|---|
| `atlas-notes/note` (`testmain_test.go`) | yes (`CreateAndEmit` → outbox) | `producertest.InstallNoop()` in `TestMain`, no `t.Cleanup(ResetInstance)` | PASS |
| `atlas-saga-orchestrator/kafka/consumer/note` (`testmain_test.go:11`) | yes (transitively, via `saga.NewProcessor(...).StepCompleted`) | `producertest.InstallNoop()` in `TestMain` | PASS |
| `atlas-saga-orchestrator/saga` (`testmain_test.go`, pre-existing, unmodified by this diff) | yes (`compensateNoteSend`→`EmitSagaFailedByIds`, `handleCreateNote`→`noteP.CreateNote`) | `producertest.InstallNoop()` in `TestMain` (pre-existing, still installed — confirmed present, not removed by this diff) | PASS |
| `atlas-channel/kafka/consumer/saga` (`consumer_test.go` — `TestExtractResultCharacterId`) | no (pure function test, no processor/producer touched) | N/A | PASS (no stub needed) |
| `atlas-channel/socket/handler` (`note_send_test.go` — `TestBuildNoteSendSaga`) | no (tests `buildNoteSendSaga` pure struct builder only, never calls `saga.NewProcessor(...).Create`) | N/A | PASS (no stub needed) |
| `libs/atlas-packet/note/*`, `libs/atlas-packet/cash/serverbound` (codec tests) | no (Encode/Decode only, no Kafka) | N/A | PASS |

No test found that calls an emit path without a stub installed.

## Notes (not findings)

1. **`producer.ProviderImpl(...)` called directly (not via `AndEmit`+`message.Buffer`) for CREATE_FAILED notification** — `services/atlas-notes/atlas.com/notes/kafka/consumer/note/consumer.go:54` and the mirrored `saga-orchestrator/note/processor.go:38`. Read literally, ai-guidance.md Core Rule 4 ("Only emit messages via `AndEmit` + buffer") would flag this. However: (a) this fires only on the branch where `CreateAndEmit`'s DB transaction has already failed — there is no live transaction/buffer to attach to; (b) the identical idiom (`producer.ProviderImpl(l)(ctx)(topic)(...)` called directly from a consumer's error branch, no buffer) is the established pattern for post-failure status notification elsewhere in the fleet (`services/atlas-cashshop/atlas.com/cashshop/kafka/consumer/cashshop/consumer.go:95,104,113`; `services/atlas-messengers/atlas.com/messengers/messenger/processor.go` — dozens of call sites). Graded as **not a violation**: the buffer requirement exists to keep a DB write and its Kafka side-effect atomic; there is no DB write on this branch to keep atomic with. Recorded here for visibility, not blocking.
2. **`transactionId,omitempty` on `uuid.UUID` fields** (`atlas-notes/kafka/message/note/kafka.go:19,52`; `saga-orchestrator/kafka/message/note/kafka.go:16,29`) — `omitempty` is a no-op on an array-kind field (`uuid.UUID` is `[16]byte`), so the field is always serialized. Per the task's own pre-vetted note, this is intentional field-for-field wire-contract mirroring across atlas-notes/orchestrator/channel; Minor at most, not filed as a blocking finding.
3. `tools/packet-audit/internal/matrix/build.go` — reviewed as requested; it's a scoped, narrowly-documented allowlist fix (`legacyConsumedSiblingWriters`) to the grader, not service code. No DOM/file-responsibility concerns apply (it's not a domain package); the change is well-commented and internally consistent with the diff's own doc comments.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- None (see Notes above for two pre-existing-pattern observations, neither rises to a fix-required finding).
