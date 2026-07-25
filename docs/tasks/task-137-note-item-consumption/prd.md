# Note Item Consumption & Memo Packet Verification — Product Requirements Document

Version: v2
Status: Draft
Created: 2026-07-09
Revised: 2026-07-25 (main-sync: scope expanded from 5 to 9 versions)
---

> **Scope revision (v2).** When this PRD was written the coverage matrix tracked
> five versions (gms_v83/v84/v87/v95, jms_v185). Main has since added four legacy
> columns — **gms_v48, gms_v61, gms_v72, gms_v79** — which are in scope for this
> task. All four were IDA-verified to use the same USE_CASH_ITEM note-send path
> (findings in `legacy-verify/`); the feature and every goal below extend to all
> nine versions. The additional matrix promotions are the four legacy
> `NoteOperationDiscard` cells (now ❌/🟡 on main), verified from the per-version
> `CMemoListDlg::SetRet` bodies. See design.md §1 for the consolidated evidence.

## 1. Overview

The Notes/memos feature (atlas-notes + `NoteOperationHandle` in atlas-channel) implements send, discard, and display, and the v83 clientbound `MEMO_RESULT` packet is byte-verified. However, the send path has an unverified — and, per code inspection, real — economy gap: sending a note is free. In the reference client, sending a note to an offline character consumes a Note cash item (item 5090000, classification 509; Cosmic consumes it in `UseCashItemHandler` case 509). In Atlas today:

- The memo-operation SEND arm (`services/atlas-channel/atlas.com/channel/socket/handler/note_operation.go:43`) calls `np.SendNote(...)` with **no Note-item ownership check and no consumption**.
- The cash-item-use handler maps Note items to `CashSlotItemType(21)` (`character_cash_item_use.go:251`) but then **falls through to a warn-log** (`character_cash_item_use.go:110`) — the UseCashItem note-send path is entirely unimplemented.

Which of these two serverbound paths the real client actually uses for note send — per version — is unverified; the coverage matrix's serverbound `NOTE_ACTION` fnames (`CMemoListDlg::SetRet`, `CWvsContext::OnMemoNotify_Receive`) are discard/receive-ack flavored, hinting that send may travel via UseCashItem instead. This task verifies the true path(s) in IDA (resolved: USE_CASH_ITEM in all nine versions), implements item consumption so notes can never be created for free, and closes the unverified packet-matrix cells: clientbound `MEMO_RESULT` on **v84** and **JMS185**, plus serverbound `NoteOperationDiscard` on **JMS185** and the four legacy versions **v48/v61/v72/v79** (all ❌/🟡 in `docs/packets/audits/STATUS.md`; line refs shifted with the 9-column matrix — regenerate).

## 2. Goals

Primary goals:
- No free notes: every player-initiated note send consumes exactly one Note item (5090000-family, classification 509), or is rejected with a client-visible error.
- IDA-ground the per-version client send path (UseCashItem vs memo-op SEND) for **all nine versions** (gms_v48/v61/v72/v79/v83/v84/v87/v95, jms_v185), and implement the path(s) each client actually uses. (Done at design time — USE_CASH_ITEM in all nine; see design §1.1.)
- Promote `MEMO_RESULT` (CWvsContext::OnMemoResult → `note/clientbound/NoteDisplay`) matrix cells to ✅ for gms_v84 and jms_v185. (The four legacy MEMO_RESULT cells are already ✅ on main.)
- Promote `note/serverbound/NoteOperationDiscard` to ✅ for **jms_v185 and the four legacy versions gms_v48/v61/v72/v79** (same family; per-version bodies verified — design §1.5).

Non-goals:
- Gift memos (notes attached to cash-shop gifts; `CCashShop::OnCashItemResLoadGiftDone`) — excluded per scope decision.
- atlas-notes service internals (storage, display, discard flow already work).
- Notes UI (atlas-ui) changes.
- Memo flag semantics beyond what send already uses (current `flag=1` unchanged unless IDA verification of the send body proves it wrong).

## 3. User Stories

- As a player, I want sending a note to consume one Note item from my cash inventory so that the item has its intended purpose and economy.
- As a player without a Note item, I want a clear client error when I attempt to send a note so that the failure isn't silent.
- As a player on a v84 or JMS185 tenant, I want received notes to display correctly on login so that the memo feature works on my version.
- As an operator, I want the packet coverage matrix to reflect verified reality for the note family so that ❌ cells mean "missing," not "unchecked."

## 4. Functional Requirements

### 4.1 Send-path verification (design-phase gate)

- FR-1: For each version (gms_v48, gms_v61, gms_v72, gms_v79, gms_v83, gms_v84, gms_v87, gms_v95, jms_v185), determine from IDA which serverbound packet(s) the client emits for a player-initiated note send: the UseCashItem opcode carrying item 5090000 (with recipient/message in the arm tail), the memo-operation opcode SEND arm, or both. Cosmic behavior is reference-only; the client binary is the source of truth (Verification Over Memory). **Resolved:** USE_CASH_ITEM in all nine versions (design §1.1 + `legacy-verify/`).
- FR-2: Document the verified read order for the note arm of UseCashItem (if that is a real path) per version, following `docs/packets/audits/VERIFYING_A_PACKET.md`.

### 4.2 Item consumption on send

- FR-3: Implement the IDA-confirmed send path(s). If UseCashItem-509 is a real client path, the cash-item-use handler's note arm (`CashSlotItemType(21)`) must decode the arm tail and perform consume-then-send; the current warn-log fall-through must be removed for this type.
- FR-4: The memo-operation SEND arm must gate on Note-item ownership regardless of whether it is the primary client path: verify the sender holds a Note item (classification 509) in the cash inventory; consume one on successful send; reject otherwise. Notes must not be creatable for free via either opcode.
- FR-5: Consumption and note creation must be coupled such that neither a free note (note created, item kept) nor a lost item (item destroyed, note not created) is a steady-state outcome. The existing consume-then-act saga precedent in the same handler (field-effect: `DestroyAsset` step + effect step, `character_cash_item_use.go:60–105`) is the candidate mechanism; the exact composition (saga step types, compensation) is a design decision.
- FR-6: On send with no Note item in inventory, the server must send the client an error via the existing note error plumbing (`notecb.NoteSendErrorBody`; receiver-unknown case already exists at `note_operation.go:39`). The specific error code must be IDA-verified against the client's OnMemoResult error arms — do not guess a byte.
- FR-7: Receiver-unknown behavior (existing) is retained; the item must NOT be consumed when the send is rejected pre-flight (unknown receiver, missing item, malformed request).

### 4.3 Packet-matrix promotion

- FR-8: Verify `MEMO_RESULT` × gms_v84 (opcode 0x029 currently listed) per the packet-verifier flow: decompile the client read order, write the byte-fixture test with a `packet-audit:verify` marker, pin the evidence record, regenerate the matrix. Note the known v84 clientbound opcode-table shift above ~0x3D (bug memory) does not affect 0x029, but the opcode must still be confirmed, not assumed.
- FR-9: Verify `MEMO_RESULT` × jms_v185 (opcode 0x026 currently listed) the same way. Use the `*_U_DEVM` build (retail dump is SMC) and pass `--audit-dir docs/packets/audits/jms_v185` explicitly (known tooling gotcha).
- FR-10: Verify `note/serverbound/NoteOperationDiscard` × jms_v185 (serverbound marker + evidence + REPORT via root `-ida-source`).
- FR-11: Regenerated `STATUS.md` must show ✅ for all three cells with no regressions elsewhere (matrix `--check` clean).

### 4.4 Configuration wiring

- FR-12: If implementing the UseCashItem note arm requires new reader options, operations-table keys, or handler routing, update ALL version seed templates AND document the live-tenant PATCH (known bug pattern: seed templates apply only at tenant creation; existing tenants silently drop unrouted opcodes).
- FR-13: Any client-interpreted byte (error codes, sub-op modes) must resolve from tenant configuration, never hard-coded (DOM-25; the msgType hardcoding incident is the precedent).

## 5. API Surface

No new or modified REST endpoints. Changes are socket-handler and (potentially) saga/Kafka-internal:

- Serverbound: existing UseCashItem opcode gains a functioning note arm (no new opcode); memo-operation SEND arm gains validation.
- Clientbound: existing `MEMO_RESULT` writer (`note/clientbound/NoteDisplay` + `NoteSendError` bodies) — no structural change expected; error-code resolution may move to config per FR-13.
- Kafka/saga: possible new saga composition (consume Note item → create note). Whether this requires a new saga step type in atlas-saga or reuses `DestroyAsset` + an existing note command is a design decision.

Error cases (client-visible):
- Receiver unknown → existing `NoteSendErrorReceiverUnknown`.
- Sender has no Note item → IDA-verified error code (FR-6).

## 6. Data Model

No schema changes. atlas-notes' note storage is unchanged. Item consumption uses the existing cash-inventory asset model (destroy quantity 1). Multi-tenancy is inherited from existing context plumbing (`tenant.MustFromContext`).

## 7. Service Impact

- **atlas-channel** — primary. `character_cash_item_use.go` note arm implementation; `note_operation.go` SEND-arm ownership gate + consumption + error response; possible new saga composition.
- **libs/atlas-packet** — arm-tail codec for the UseCashItem note body if that path is real (thin wrapper if a shared codec exists — check before duplicating, per the matrix-❌-can-mean-shared-codec gotcha); byte fixtures for MEMO_RESULT v84/jms and NoteOperationDiscard jms.
- **atlas-saga** — expected no code change (`DestroyAsset` exists); only if design requires a new step type.
- **atlas-notes** — expected unchanged.
- **Seed templates / tenant config** — only if FR-12 triggers.

## 8. Non-Functional Requirements

- **Atomicity:** consume+send must have no steady-state where the player loses an item without a note being created, or vice versa (FR-5).
- **Multi-tenancy:** all behavior version-resolved per tenant; no version-conditional literals in handlers where config resolution is the pattern.
- **Grounding:** every packet byte, opcode, and error code cited from IDA or checked-in exports; no values from Cosmic or memory land in code or fixtures.
- **Observability:** rejected sends (missing item, unknown receiver) logged at warn with character id and reason; no silent drops.
- **Anti-cheat:** a client that emits memo-op SEND without owning a Note item gets an error, not a free note.

## 9. Open Questions

- **Q1 (design):** Which serverbound path does each version's client actually use for note send? Resolved by FR-1 IDA work; determines whether the UseCashItem arm, the SEND-arm gate, or both are the player-reachable path per version.
- **Q2 (design):** Exact consume+send composition — saga (`DestroyAsset` + note-create step, with compensation) vs. direct processor call with explicit rollback. Field-effect saga is the precedent; atlas-saga may or may not already have a usable note step.
- **Q3 (design):** The IDA-verified error code for "no Note item" in the client's OnMemoResult handling, and whether existing `NoteSendError*` codes in `libs/atlas-packet/note/clientbound` already cover it.

## 10. Acceptance Criteria

- [ ] Per-version send path documented with IDA evidence (fnames + read order) for all nine versions: v48, v61, v72, v79, v83, v84, v87, v95, jms185.
- [ ] Sending a note consumes exactly one Note item (classification 509) on every player-reachable path; verified by handler/processor tests.
- [ ] Send attempt with no Note item is rejected and the client receives the IDA-verified error packet; no note is created; no item is consumed.
- [ ] Receiver-unknown rejection consumes no item.
- [ ] `MEMO_RESULT` × gms_v84 cell ✅: byte-fixture test with `packet-audit:verify` marker, pinned evidence, regenerated matrix.
- [ ] `MEMO_RESULT` × jms_v185 cell ✅: same artifacts.
- [ ] `note/serverbound/NoteOperationDiscard` × jms_v185 cell ✅: marker + evidence + REPORT.
- [ ] `note/serverbound/NoteOperationDiscard` × gms_v48, gms_v61, gms_v72, gms_v79 cells ✅: marker + evidence + REPORT per cell (per-version bodies, design §1.5).
- [ ] Nine-version templates carry correct MEMO_RESULT `operations`/`errors` tables, including the shifted v48/v61 mode table (SEND_ERROR=4); a v48/v61 tenant's SEND_ERROR path is confirmed to unlock the client.
- [ ] `packet-audit` matrix/fname-doc checks exit 0; no cell regressions.
- [ ] Seed templates updated for any new routing/options in all versions, and live-tenant PATCH documented (or explicit statement that no config change was needed).
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module; `docker buildx bake atlas-channel` (and any other touched service) clean; `tools/redis-key-guard.sh` clean.
- [ ] Gift-memo flow untouched and explicitly out of scope in design.md.
