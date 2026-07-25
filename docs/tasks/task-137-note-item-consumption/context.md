# Task 137 — Context for Execution

Companion to `plan.md`. Everything here was verified during design/planning (IDA reads, code reads at the cited lines); nothing is inherited from Cosmic or memory.

## What this task is

Sending a note is free today — the memo-op SEND arm calls `SendNote` with no item check, and the USE_CASH_ITEM note arm (the path real clients actually use, **all nine versions**) is an unimplemented warn-log fall-through. This task implements consume-gated note sending via a new `note_send` saga (destroy-first), closes the cheat path on the memo-op arm, populates the note writer/handler config across all nine templates (the cash handler is **already routed** on main — see plan Task 14), and promotes the note-family matrix cells: MEMO_RESULT × v84, MEMO_RESULT × jms185, and NoteOperationDiscard × jms185 **plus the four legacy versions v48/v61/v72/v79**.

> **Scope (main-sync):** main added four legacy version columns (v48/v61/v72/v79) after this task was planned; all four were IDA-verified to have the cash-item note-send arm (findings in `legacy-verify/`). The saga/service core is version-agnostic and unchanged; the nine-version deltas are the codec-gate, the writer config (incl. the v48/v61 shifted MEMO_RESULT mode table), and the discard-cell promotions.

## Key verified facts (design.md §1 — do not re-derive)

- **Send path = USE_CASH_ITEM everywhere (all nine versions).** `CWvsContext::SendConsumeCashItemUseRequest`; the Note item (classification 509) resolves to the send-memo arm. Opcodes: v48 0x3E, v61 0x49, v72 0x4E, v79 0x4D, v83 0x4F, v84 0x4F, v87 0x52, v95 0x55, jms185 0x47. Arm body is uniform: `toName` string, `message` string. updateTime is TRAILING on all GMS ≤ v84 (v48/v61/v72/v79/v83/v84), LEADING on v87/v95/jms — the existing codec gate `>=95` is off by one (fix to `(GMS && >=87) || JMS`, which classifies all nine correctly). The client-internal cash-slot type (v48→19, v61→20, v72+→21) is a dispatch index, NOT a wire field — irrelevant to Atlas's codec. Per-version fn addrs + evidence in design §1.1 and `legacy-verify/`.
- **NOTE_ACTION SEND (mode 0) is never a player path** — where a mode-0 writer exists it is ONLY the cash-shop gift flow (`CCashShop::OnCashItemResLoadGiftDone`, server-side unimplemented); on v48 and v79 there is NO mode-0 writer at all. Player traffic on that arm = tampered client in every version. Gate it with the same consume flow; when gifting lands later it must NOT go through the consume gate.
- **OnMemoResult — TWO mode tables.** Standard (v72/v79/v83 0xa2508b/v84 0xa70785/v87 0xabccc2/v95 0x9f9da0/jms 0xb0c6d0): SHOW=3, SEND_SUCCESS=4, SEND_ERROR=5, REFRESH=7. **Shifted (v48 0x71d8e2, v61 0x8468be): SHOW=2, SEND_SUCCESS=3, SEND_ERROR=4, no REFRESH.** SEND_ERROR sub-codes 0/1/2 show dialogs in every version; any other sub-code = silent no-op. **No "no Note item" arm exists** — `NO_NOTE_ITEM` is a chosen out-of-range sub-code 3 (config-resolved, silent unlock). The mode byte itself is config-resolved per version (Task 14) — a hard-coded 5 wedges v48/v61.
- **Exclusive-request lock:** SEND_ERROR (mode 5) clears it before decoding the code; SEND_SUCCESS (mode 4) does NOT — the success-path unlock rides on the inventory-operation packet from the destroyed item. Verified server-side: `handleAssetDeletedEvent` announces `NewChangeBatch(false, RemoveEntry)` (`services/atlas-channel/.../kafka/consumer/asset/consumer.go:421`) and the writer emits `WriteBool(!silent)` → leading byte 1 (`libs/atlas-packet/inventory/clientbound/change.go`). So: every accepted send eventually unlocks via inventory op + mode 4; every rejection must send mode 5.

## Bugs found during planning (fixed by the plan)

1. **`NoteSendErrorBody` sends the wrong mode** — `libs/atlas-packet/note/clientbound/operation_body.go:38` resolves the mode from `NoteOperationSendSuccess` (4) instead of `NoteOperationSendError` (5). Receiver-unknown errors currently go out as a success mode. Plan Task 3.
2. **gms_95 `NoteOperationHandle` historically had no `validator`** (`BuildHandlerMap` silently drops validator-less entries) — re-inspect on main; fix if still present. Plan Task 14.
3. **`NoteOperationHandle` `options.operations` missing/incomplete on several versions** — `isNoteOperation` error-logs and returns false for every arm, so note ops are dead there. Plan Task 14 sets the per-version serverbound operations table. (NOTE: the *handler routing* itself already exists on all nine templates as of main — only the `options` need populating; do not re-add routes.)
4. **`NoteOperation` writer `errors`/`operations` tables missing/incomplete on several versions.** Plan Task 14 sets them per version — including the **shifted v48/v61 clientbound mode table** (SEND_ERROR=4, not 5). A standard `SEND_ERROR:5` on v48/v61 wedges the client.

## Architecture decisions (design §3, Q2 resolved = Approach A)

- Saga type `note_send`: Step 1 `DestroyAsset{sender, templateId, qty 1}`, Step 2 `CreateNote{SenderId, ReceiverId, Message, Flag:1}` (new action). Destroy-first is the FR-5 invariant: no free note under any interleaving; a failed step 2 re-awards via compensation.
- Orchestrator `CreateNote` handler emits the note CREATE command **carrying the saga transaction id**; atlas-notes emits `CREATED`/`CREATE_FAILED` with the txn id echoed; a new orchestrator note-status consumer matches by txn id → `StepCompleted(txn, true/false)`. `AcceptEvent` + `acceptanceTable` entry `CreateNote: {note.created, note.create_failed}` are mandatory (default-deny otherwise).
- Client feedback via the channel saga consumer: COMPLETED + `SagaType=="note_send"` → SEND_SUCCESS to the sender (sender id rides in `Body.Results["characterId"]`, populated by the orchestrator like the CharacterCreation precedent); FAILED → SEND_ERROR `NO_NOTE_ITEM` (silent unlock; server warn-logs).
- Compensation mirrors pet evolution: dispatch half `DispatchNoteSendRollbacks` (completed DestroyAsset → `RequestCreateItem`) + terminal half with `TryTransition(Compensating→Failed)` and one `EmitSagaFailedByIds` carrying the SENDER id.
- Pre-flight rejections (receiver unknown, receiver online same-channel, no item) happen in the channel handler before saga creation — nothing consumed (FR-7). Receiver-online check is same-channel-scope only (no cross-channel session lookup exists in atlas-channel; documented limitation, harmless false-negative).
- The channel's `note.SendNote` / `CreateCommandProvider` are removed — the CREATE command is orchestrator-owned now. DISCARD/display paths unchanged.
- No new REST endpoints; REST note creation in atlas-notes passes `uuid.Nil` txn id.

## Key files & line anchors

| What | Where |
|---|---|
| Cash-item-use handler (note arm goes here; type map at 251) | `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go` |
| Memo-op handler (SEND gate at 32–48) | `services/atlas-channel/atlas.com/channel/socket/handler/note_operation.go` |
| USE_CASH_ITEM prefix codec (gate at 38/50) | `libs/atlas-packet/cash/serverbound/item_use.go` |
| Arm codec pattern to mirror | `libs/atlas-packet/cash/serverbound/item_use_chalkboard.go` |
| Note clientbound bodies (mode bug at 38) | `libs/atlas-packet/note/clientbound/operation_body.go` |
| Saga lib type/action/payload/unmarshal | `libs/atlas-saga/{model,payloads,unmarshal}.go` |
| Orchestrator action dispatch (`GetHandler` at 704) | `services/atlas-saga-orchestrator/.../saga/handler.go` |
| Orchestrator re-exports + step unmarshal (~1382) | `services/atlas-saga-orchestrator/.../saga/model.go` |
| Event acceptance tables | `services/atlas-saga-orchestrator/.../saga/event_acceptance.go` |
| Status-consumer pattern to mirror (txn matching) | `services/atlas-saga-orchestrator/.../kafka/consumer/pet/consumer.go` (+ its `consumer_test.go`) |
| Compensator (pet reverse-walk precedent at 1093–1180) | `services/atlas-saga-orchestrator/.../saga/compensator.go` |
| Completed-event Results precedent | `services/atlas-saga-orchestrator/.../saga/producer.go:17-54` |
| atlas-notes create path (error swallowed at consumer:50) | `services/atlas-notes/atlas.com/notes/{note/processor.go,kafka/consumer/note/consumer.go}` |
| Channel saga consumer (storage-failure precedent) | `services/atlas-channel/atlas.com/channel/kafka/consumer/saga/consumer.go` |
| Channel saga msg (empty completed body) | `services/atlas-channel/atlas.com/channel/kafka/message/saga/kafka.go` |
| Compartment helper site (`FindFirstByItemId` at 56) | `services/atlas-channel/atlas.com/channel/compartment/model.go` |
| Seed templates | `services/atlas-configurations/seed-data/templates/template_{gms_48,gms_61,gms_72,gms_79,gms_83,gms_84,gms_87,gms_95,jms_185}_1.json` |
| Legacy IDA evidence | `docs/tasks/task-137-note-item-consumption/legacy-verify/{v48,v61,v72,v79}.md` |
| Matrix rows | `docs/packets/audits/STATUS.md` MEMO_RESULT + NoteOperation{,Discard,Send} rows (regenerate line refs; they shifted with the 9-column matrix) |
| Verification playbook | `docs/packets/audits/VERIFYING_A_PACKET.md` |

## Kafka / infra notes

- Topics `COMMAND_TOPIC_NOTE` and `EVENT_TOPIC_NOTE_STATUS` already exist in the shared `atlas-env` configmap (`deploy/k8s/base/env-configmap.yaml:52,129`); the orchestrator consumes env via `envFrom: atlas-env` — **no k8s manifest change needed**.
- Orchestrator's own `kafka/message/note` package mirrors atlas-notes' JSON field-for-field (no cross-service Go import). `Command.WorldId/ChannelId` are unused by the CREATE handler — zero values from the orchestrator are fine.
- The channel `StatusEventCompletedBody` change (empty struct → SagaType/Results) is additive JSON decoding — old events decode fine.

## Matrix promotion particulars (execute-phase gotchas)

- **v84:** `OnMemoResult` @ 0xa70785 (old audit's "function not found" is stale). Opcode 0x029 is below the v84 shift boundary (~0x3D) but must be confirmed from the dispatch table, not assumed.
- **jms185:** instance `MapleStory_dump_SCY.exe` port **13344** (retail dump is SMC — never use it). Always pass `--audit-dir docs/packets/audits/jms_v185` (the default dir name is wrong for jms and silently reports 0/0/0/0). `OnMemoResult` @ 0xb0c6d0, opcode 0x026 to confirm.
- **jms discard:** `CMemoListDlg::SetRet` @ 0x6c2d43 is 0x33d bytes vs ~0x26b GMS — derive the write order from scratch; budget for a version-gated codec delta. Serverbound cell = marker + evidence + REPORT via root `-ida-source`; export splices are surgical, never overwrite.
- **legacy discard (v48/v61/v72/v79):** SetRet addrs v48 0x534dc4, v61 0x5ad50c, v72 0x5fb443, v79 0x619fb7. Uniform header `mode=1, totalCount u8, specialCount u8, emptySlots u8`; per-entry special (gift/reward) flag = **2** on v48/v61, **3** on v72/v79, with an extra int32 (reward/itemId/mesos). v48 ALSO emits a follow-up 0x66 packet for flag==1 notes — resolve during verification. Full shapes in `legacy-verify/` and design §1.5. At least two distinct legacy shapes → don't assume one fixture covers all four.
- Confirm the IDB instance/binary NAME before any read (list + match; instance set rotates — CLAUDE.md IDA rule).

## Test-environment notes

- `pt.Variants` (libs/atlas-packet/test/context.go) includes GMS v28/83/84/86/87/95, JMS v185 — round-trip tests iterate it; the v84 entries were appended, positional refs stay valid. It does NOT include v48/61/72/79, and it should NOT be extended for this task (other packets' gates depend on it positionally). The `ItemUseNote` codec has no per-version-number branch (only the trailing/leading gate), so v28/v83 (trailing) + v87 (leading) already exercise both code paths the four legacy versions use — no extra variants needed.
- Channel handler tests are lightweight (decode-pinning + symbol checks — see `mount_food_test.go`); behavioral saga assertions live in the pure `buildNoteSendSaga` unit test.
- Orchestrator consumer tests manipulate the real saga cache (`saga.GetCache().Put`) — pattern in `kafka/consumer/pet/consumer_test.go`; Kafka emission is best-effort in tests.
- atlas-notes `Create`/`CreateAndEmit` signature change touches `note/mock/processor.go`, `note/resource.go` (uuid.Nil), and existing tests.

## Module verification list (Task 18)

`libs/atlas-packet`, `libs/atlas-saga`, `services/atlas-notes/atlas.com/notes`, `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`, `services/atlas-channel/atlas.com/channel`; bake `atlas-channel atlas-saga-orchestrator atlas-notes`; from repo root `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `tools/lint.sh --check` (all added to main's gate since this plan; no GOWORK=off prefix); packet-audit `--check`s exit 0.
