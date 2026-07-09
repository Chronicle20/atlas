# Note Item Consumption & Memo Packet Verification — Design

Task: task-137-note-item-consumption
Status: Proposed
PRD: `docs/tasks/task-137-note-item-consumption/prd.md`

---

## 1. Verified ground truth (FR-1 / FR-2 — design-phase IDA verification)

All five client binaries were read during this design phase. Every claim below is
IDA-verified at the cited address; nothing is inherited from Cosmic or memory.

### 1.1 The player note-send path is USE_CASH_ITEM in ALL five versions (Q1 resolved)

`CWvsContext::SendConsumeCashItemUseRequest` builds the USE_CASH_ITEM packet and
switches on `get_consume_cash_item_type(nItemID)`. Case `0x15` (21 — matching
Atlas `CashSlotItemType(21)` for classification 509) opens a send-memo modal
dialog (named `CUISendMemo` in v95), then encodes recipient and message:

| Version | IDB fn addr | Opcode | updateTime position | Note arm body |
|---|---|---|---|---|
| gms_v83 | 0xa0a63f | 0x4F | **trailing** (`Encode4(get_update_time())` at shared `LABEL_41` epilogue) | `EncodeStr(toName)`, `EncodeStr(message)` |
| gms_v84 | 0xa54a2f (named during this design) | 0x4F | **trailing** (same `LABEL_41` shape) | same |
| gms_v87 | 0xa9fef9 | 0x52 | **leading** (`Encode4(update_time)` immediately after ctor; no trailing encode at `LABEL_41`) | same |
| gms_v95 | 0x9eb3e0 | 0x55 | **leading** | same (`CUISendMemo::GetResult` → two `EncodeStr`) |
| jms_v185 | 0xaef2f5 | 0x47 | **leading** | same |

Full wire format (serverbound USE_CASH_ITEM, note arm):

```
opcode
[updateTime int32]        // GMS v87+, JMS185: leading
slot      int16           // Encode2(nPOS)
itemId    int32           // Encode4(nItemID)
toName    string          // EncodeStr
message   string          // EncodeStr
[updateTime int32]        // GMS v83/v84: trailing
```

**Codec off-by-one found:** `libs/atlas-packet/cash/serverbound/item_use.go`
gates the leading updateTime on `Region=="GMS" && MajorVersion>=95`. v87
(0xa9fef9) already encodes it leading. The gate must become `>=87` (JMS185 also
leads; the JMS gate must be handled too). This is another instance of the known
"`>83` vs `>=87`" off-by-one family. No live regression today because
`CharacterCashItemUseHandle` is only routed on gms_83/84 templates — but the
fix is a prerequisite for wiring v87/v95/jms.

### 1.2 The NOTE_ACTION "SEND" arm is written ONLY by the cash-shop gift flow

Every writer of the NOTE_ACTION opcode was enumerated per version by scanning
code xrefs to the `COutPacket` constructor for the pushed opcode immediate
(0x83 v83 / 0x87 v84 / 0x8B v87 / 0x9A v95 / 0x86 jms185). In **all five
versions** the writers are exactly:

1. `CMemoListDlg::SetRet` — mode 1 (DISCARD). v83 @ 0x64aa57: `Encode1(1)`,
   count, per-entry id/flag. (jms185 SetRet @ 0x6c2d43 is 0x33d bytes vs 0x26b
   GMS — expect a shape difference; the jms discard fixture must derive its own
   read order, not assume the GMS shape.)
2. `CWvsContext::OnMemoNotify_Receive` — mode 2 (REQUEST), single byte
   (v83 @ 0xa251ef: `Encode1(2)`).
3. `CCashShop::OnCashItemResLoadGiftDone` — mode **0 (SEND)**, and only this.
   v83 @ 0x47959e: `Encode1(0)`, `EncodeStr(toName)`, `EncodeStr(message)`,
   `Encode1(1)`, `Encode4(index)`, `EncodeBuffer(8 bytes)` — matching the jms
   audit evidence (`docs/packets/audits/jms_v185/NoteOperationSend.md`: byte,
   string, string, byte, int32, bytes). The client shows "THE NOTE HAS
   SUCCESSFULLY BEEN SENT" immediately after `SendPacket` without awaiting a
   response.

**Consequence:** no legitimate player-initiated send ever arrives on the
NOTE_ACTION SEND arm. Traffic on that arm is either the (out-of-scope,
server-side-unimplemented) cash-shop gift flow or a tampered client. The
current free-of-charge `SendNote` at
`services/atlas-channel/atlas.com/channel/socket/handler/note_operation.go:43`
is therefore reachable only by cheaters today — and it creates free notes.

### 1.3 MEMO_RESULT client arms and error codes (Q3 resolved)

`CWvsContext::OnMemoResult` (v83 0xa2508b, v84 0xa70785, v87 0xabccc2,
v95 0x9f9da0, jms185 0xb0c6d0) is structurally identical in all five versions:

- mode 3 = SHOW (note list), mode 4 = SEND_SUCCESS ("note has successfully
  been sent" notice), mode 5 = SEND_ERROR, mode 7 = REFRESH (re-invokes
  `OnMemoNotify_Receive`). Matches the existing tenant writer tables
  (`operations {SHOW:3, SEND_SUCCESS:4, SEND_ERROR:5, REFRESH:7}`).
- SEND_ERROR codes: **0** = "the other character is online now, please use the
  whisper function", **1** = "please check the name of the receiving
  character", **2** = "the receiver's inbox is full". **Any other code is a
  silent no-op** (no dialog). Matches `errors {RECEIVER_ONLINE:0,
  RECEIVER_UNKNOWN:1, RECEIVER_INBOX_FULL:2}`.
- **There is NO "no Note item" error arm in any version.** FR-6's "IDA-verified
  error code" for the missing-item case does not exist client-side; §5.3 below
  resolves this.

### 1.4 Exclusive-request lock: the response contract

`SendConsumeCashItemUseRequest` sets `CWvsContext::SetExclRequestSent(1)`
(v83: flag at `this+0x20A4`, checked by `CanSendExclRequest` @ 0x485bf7).
While set, the client refuses to send further item-use/transfer requests —
a server that never responds wedges the client. Verified unlock paths:

- **MEMO_RESULT mode 5 (SEND_ERROR)** clears the flag *before* decoding the
  error code (v83 0xa2508b: the mode-5 arm zeroes `+0x20A4` and stamps the
  timestamp first; same in v95 @ 0x9f9da0 case 5). So an error response with
  ANY code — including an out-of-range one — unlocks the client.
- **OnInventoryOperation / OnStatChanged first byte** (v83 0xa1ead9 / 0xa1fb52:
  `if (Decode1(iPacket)) { clear +0x20A4 }`) — the standard "enableActions"
  unlock that rides on the inventory-change packet emitted when the item is
  destroyed.
- MEMO_RESULT mode 4 (SEND_SUCCESS) does **not** clear the flag; on the success
  path the unlock comes from the inventory operation of the consumed item.

**Contract for the server:** every accepted send must eventually produce an
inventory-operation packet (with the excl byte set) + SEND_SUCCESS; every
rejected send must produce SEND_ERROR.

---

## 2. Scope decisions confirmed by the evidence

1. **Implement the USE_CASH_ITEM note arm as the primary (and only legitimate)
   player send path** on every version (FR-3). This requires wiring
   `CharacterCashItemUseHandle` into the gms_87/gms_95/jms_185 templates
   (opcodes 0x52/0x55/0x47, IDA-verified above) — it is currently routed only
   on gms_83/84.
2. **Gate the NOTE_ACTION SEND arm with the same ownership+consumption flow**
   (FR-4). Since its only legitimate writer is the unimplemented gift flow,
   uniform gating cannot break any player-reachable behavior, and it closes the
   cheat path. When cash-shop gifting is implemented later it must NOT reuse
   this arm's consume-gated path (the gift note is paid for by the gift
   purchase); that future task gets a pointer here.
3. **Gift memos remain out of scope** (PRD non-goal). The `OperationSend` codec
   keeps decoding only `toName`+`message`; the trailing gift fields
   (byte/int32/8-byte SN) are left unread — the handler rejects the arm before
   they matter. The existing NOTE_ACTION matrix row stays ✅ (not a target).
4. **Flag stays `1`** on send (PRD non-goal): the gift writer encodes flag 1 and
   the discard flow round-trips it; nothing in the send body contradicts the
   current `SendNote(..., flag=1)`.

---

## 3. Atomicity architecture (Q2 / FR-5)

### Constraints discovered in the current machinery

- `atlas-channel`'s note producer is fire-and-forget: `note.Command` with
  `Type=CREATE` on `COMMAND_TOPIC_NOTE`
  (`services/atlas-channel/atlas.com/channel/note/producer.go:12-26`), no
  transaction id.
- atlas-notes emits `EVENT_TOPIC_NOTE_STATUS` `CREATED` with **no transaction
  id and no failure event** (`services/atlas-notes/.../note/processor.go:55-88`)
  — a saga cannot match on note creation today.
- There is **no note saga Action** (`libs/atlas-saga/model.go:42-161`), and
  `DestroyAsset` has **no generic failed-step compensation** — if a later step
  fails after DestroyAsset completed, the orchestrator logs "No compensation
  logic available" and does not re-award
  (`services/atlas-saga-orchestrator/.../saga/compensator.go:279-333`).
- The channel's saga status consumer sends client feedback **only** for
  `SagaTypeStorageOperation` failures
  (`services/atlas-channel/.../kafka/consumer/saga/consumer.go:78-130`);
  completed events are a no-op.

### Considered approaches

**A. Saga: `DestroyAsset` → new `CreateNote` action, destroy-first (recommended).**
New saga action executed by the orchestrator; atlas-notes gains
transaction-correlated status events; a dedicated compensator re-awards the
item if note creation fails after the destroy. Client feedback via new
saga-type branches in the channel's saga status consumer.
*Pros:* no free note under any interleaving (destroy is confirmed before the
note exists); reuses the project's transaction mechanism and the existing
`DestroyAsset` handler; failure feedback is wired once, correctly.
*Cons:* touches four modules (channel, orchestrator, notes, shared saga lib);
needs the txn-id threading in atlas-notes.

**B. Destroy-first, fire-and-forget note command from the channel handler.**
Single-step `DestroyAsset` saga (or direct compartment command), then emit the
note CREATE command on saga completion.
*Pros:* smallest diff.
*Cons:* "item destroyed, note never created" is a real steady state if the
note command is emitted and atlas-notes fails (DB error, receiver deleted) —
violates FR-5's "no lost item"; and nothing tells the client. Also the channel
would have to consume saga-completed events to emit the note command — at which
point approach A's orchestrator step is the same work done in a worse place.

**C. Note-first, destroy-after.**
*Pros:* item can never be lost.
*Cons:* the free-note steady state is farmable: destroy fails whenever the item
vanished between the pre-flight check and the destroy (cash-shop locker moves,
double-send race). "No free notes" is the primary goal; rejected.

### Decision: Approach A

Saga type `note_send` (new `SagaType` constant), initiated by the channel
handler, steps in this order:

```
Step 1  DestroyAsset  {CharacterId: sender, TemplateId: <the 509 item>, Quantity: 1, RemoveAll: false}
Step 2  CreateNote    {SenderId, ReceiverId, Message, Flag: 1}   // new action
```

- **New saga action `CreateNote`** in `libs/atlas-saga` + orchestrator handler
  (`GetHandler` switch) that emits the note CREATE command **carrying the saga
  transaction id**.
- **atlas-notes threads the transaction id through**: optional `TransactionId`
  field on `note.Command`/`StatusEvent` (backward-compatible additive change,
  zero-value = absent), and a new `CREATE_FAILED` status event emitted when
  `Create` errors (today the error is silent). The orchestrator's new note
  status consumer matches `CREATED`/`CREATE_FAILED` by transaction id →
  `StepCompleted(txnId, true/false)`.
- **Compensation:** add a `note_send` branch to `CompensateFailedStep` — if
  Step 2 fails after Step 1 completed, re-award via the existing
  `RequestCreateItem` path (mirrors the pet-evolution/gachapon re-award
  precedent, `compensator.go:1336-1344`). `RemoveAll` is always false here, so
  the re-award is well-defined.
- **Client feedback** in the channel saga consumer:
  - `handleCompletedEvent` + `SagaType == note_send` → announce
    `NoteSendSuccessBody` (MEMO_RESULT mode 4). The excl lock is released by
    the inventory-operation packet from the destroy (§1.4); the plan must
    verify the destroy-driven inventory op sets the excl byte, since mode 4
    alone does not unlock.
  - `handleFailedEvent` + `SagaType == note_send` → announce
    `NoteSendErrorBody` (mode 5, code per §5.3) — unlocks the client and
    surfaces the failure.

Pre-flight rejections (receiver unknown, no item, malformed) happen in the
channel handler **before the saga is created**, so no item is consumed on any
rejected send (FR-7).

---

## 4. Handler design (atlas-channel)

### 4.1 USE_CASH_ITEM note arm (`character_cash_item_use.go`)

New `CashSlotItemTypeNote = CashSlotItemType(21)` named constant (replacing the
bare `21` at line 251-253) and a dispatch arm:

1. Decode `cashsb.NewItemUseNote(updateTimeFirst)` — new arm codec (§6.1).
2. The existing slot/template validation (lines 37-41) already proves the
   sender owns the item in the claimed slot; additionally require
   `item.GetClassification(itemId) == item.ClassificationNote`.
3. Resolve receiver: `character.GetByName(toName)` → miss ⇒ announce
   `NoteSendErrorBody(RECEIVER_UNKNOWN)`, return (no saga).
4. Receiver-online check ⇒ `NoteSendErrorBody(RECEIVER_ONLINE)` (code 0,
   IDA-verified; the reference client expects "use the whisper function" when
   the target is online). *Marked optional-but-recommended: it is the only
   remaining verified error arm and prevents notes invisible until relog. If
   the plan finds no cheap cross-world online lookup, drop it and document.*
5. Create the `note_send` saga (§3) with the slot's `TemplateId`.

The stale `// TODO for v83 there is a trailing updateTime` comment
(line 108) is resolved by the arm codec and must be removed.

### 4.2 NOTE_ACTION SEND arm (`note_operation.go`)

Replace the direct `np.SendNote(...)` with the same flow, minus the slot:

1. Receiver resolution (existing, kept) → `RECEIVER_UNKNOWN` on miss.
2. Ownership scan: fetch the cash compartment
   (`GetByType(charId, inventory.TypeValueCash)`) and find the first asset with
   `item.GetClassification(TemplateId) == item.ClassificationNote`. No such
   asset ⇒ announce `NoteSendErrorBody(NO_NOTE_ITEM)` (§5.3), warn-log with
   character id and reason, return. (No session destroy: unlike the discard
   flag-tamper case this packet is also emitted by the future gift flow.)
3. Create the same `note_send` saga using the found asset's template id.

A small compartment helper (`FindFirstByClassification`) is added next to the
existing `FindFirstByItemId`
(`services/atlas-channel/atlas.com/channel/compartment/model.go:56`).

### 4.3 Success/failure announcements

Handled by the saga status consumer branches (§3). No packet is sent inline on
the accept path — the handler's only inline announcements are the pre-flight
errors.

---

## 5. Configuration & wire values (FR-12 / FR-13, DOM-25)

### 5.1 Template wiring (all five seed templates + live PATCH doc)

| Change | gms_83 | gms_84 | gms_87 | gms_95 | jms_185 |
|---|---|---|---|---|---|
| `CharacterCashItemUseHandle` routed | ✅ 0x4F (exists) | ✅ 0x4F (exists) | **add 0x52** | **add 0x55** | **add 0x47** |
| `NoteOperationHandle` `options.operations {SEND,DISCARD,REQUEST}` | ✅ exists | ✅ exists | **add** | **add** | **add** |
| `NoteOperation` writer `options.errors` + `NO_NOTE_ITEM` | **add key** | **add key** | **add key** | **add key** | **add key** |

- The gms_87/95/jms operations-table mode bytes (SEND=0, DISCARD=1, REQUEST=2
  from the v83/v84 tables) were confirmed for jms/v87/v95 senders during this
  design only at the writer level (mode 0/1/2 immediates in the three writer
  fns); the execute phase's fixture work pins them per version.
- Every new handler entry gets a `validator` (LoggedInValidator) — a
  validator-less entry is silently dropped
  (known bug: `BuildHandlerMap` skips it).
- **Live-tenant PATCH must be documented** for every existing tenant of the
  five versions (seed templates only apply at creation — known bug pattern),
  including a channel restart note.

### 5.2 New reader options

None required: the note arm decodes two strings + version-positioned
updateTime, driven by the tenant-derived `updateTimeFirst`, not by
per-packet options. The `ItemUse` codec's version gate moves from `>=95` to
`>=87` for GMS (JMS always leading) — this is a code change validated by byte
fixtures, not config.

### 5.3 The "no Note item" error byte (FR-6 resolution)

Verified reality: no client arm exists for this case (§1.3), and a legitimate
client cannot reach it (the send dialog only opens by using an owned item).
Decision:

- New semantic key `NO_NOTE_ITEM` in the writer `errors` table, value **3** on
  all five versions — deliberately outside the client's 0-2 dialog range.
  IDA-verified effect: the mode-5 arm clears the excl lock, shows no dialog.
  This is not a guessed byte; it is a chosen out-of-range value with verified
  client semantics (silent unlock).
- The rejection is warn-logged server-side (observability NFR), so the case is
  visible to operators even though the (necessarily tampered) client shows
  nothing.
- PRD FR-6 wording ("the IDA-verified error code") is satisfied in its intent —
  the client receives a verified-safe error packet via the existing plumbing —
  with the factual correction that no dedicated code exists to send.

---

## 6. Packet library changes (`libs/atlas-packet`)

### 6.1 New serverbound arm codec: `cash/serverbound/item_use_note.go`

`ItemUseNote{ toName, message string; updateTime uint32 }`, shape mirroring
`ItemUseChalkboard`: `toName = ReadAsciiString()`, `message =
ReadAsciiString()`, then trailing `updateTime` iff `!updateTimeFirst`.
fname: `CWvsContext::SendConsumeCashItemUseRequest#Note`. Byte-fixture tests
for all five versions from the §1.1 write orders (this also gives the
USE_CASH_ITEM matrix row its first linked packet; full promotion of that row is
not a task goal but the fixtures land the evidence).

### 6.2 `item_use.go` gate fix

`updateTimeFirst`: `Region=="GMS" && MajorVersion>=95` → `(GMS && >=87) || JMS`
— IDA-verified against v84 (trailing, 0xa54a2f) and v87 (leading, 0xa9fef9).
The same flag flows to all existing arms; gms_83/84 behavior is unchanged, and
v87/95/jms were previously unrouted so nothing regresses.

### 6.3 Matrix promotions (FR-8/9/10/11)

Standard packet-verifier flow per cell (decompile read order → byte-fixture
with `packet-audit:verify` marker → pin evidence → regenerate matrix):

- `MEMO_RESULT` × gms_v84 — unblocked: `CWvsContext::OnMemoResult` is now
  resolved in the v84 IDB at **0xa70785** (the old audit's "function not found"
  is stale); arms verified identical to v83. Opcode 0x029 must be confirmed
  from the v84 dispatch table during verification (the known v84 shift starts
  above ~0x3D; 0x029 is below it but is confirmed, not assumed).
- `MEMO_RESULT` × jms_v185 — `OnMemoResult` @ 0xb0c6d0, opcode 0x026 to
  confirm; instance is `MapleStory_dump_SCY.exe` on port 13344; pass
  `--audit-dir docs/packets/audits/jms_v185` explicitly (known tooling gotcha —
  the default dir name is wrong for jms).
- `NoteOperationDiscard` × jms_v185 — `CMemoListDlg::SetRet` @ 0x6c2d43;
  **0x33d bytes vs 0x26b/0x273 on GMS — derive the jms read order from scratch,
  do not assume the GMS shape** (serverbound cell: marker + evidence + REPORT
  via root `-ida-source`).
- `packet-audit` matrix/fname-doc/operations `--check` must exit 0 with no
  regressions elsewhere.

---

## 7. Saga & service changes (summary of new surface)

| Module | Change |
|---|---|
| `libs/atlas-saga` | `SagaType` `note_send`; `Action` `CreateNote`; `CreateNotePayload{SenderId, ReceiverId, Message, Flag}` |
| atlas-saga-orchestrator | `GetHandler` case for `CreateNote` → note CREATE command with txn id; note-status consumer (`CREATED`/`CREATE_FAILED` → StepCompleted); `CompensateFailedStep` branch for `note_send` (re-award destroyed item); emit Failed status event for `note_send` failures |
| atlas-notes | optional `TransactionId` on command/status messages (additive); emit `CREATE_FAILED` status event on create error |
| atlas-channel | two handler arms (§4); `FindFirstByClassification` compartment helper; saga consumer branches for `note_send` completed/failed. The note CREATE command is now emitted by the orchestrator (with txn id), so the SEND arm stops calling `note.SendNote`; the processor's `DiscardNotes`/`GetByCharacter` paths are unchanged |

`atlas-notes` storage/display/discard are untouched. atlas-saga-orchestrator's
existing `DestroyAsset` handler is reused as-is.

## 8. Error handling matrix

| Case | Where caught | Client packet | Item consumed? | Note created? |
|---|---|---|---|---|
| Receiver name unknown | handler pre-flight | SEND_ERROR `RECEIVER_UNKNOWN` (1) | no | no |
| Receiver online (if check lands) | handler pre-flight | SEND_ERROR `RECEIVER_ONLINE` (0) | no | no |
| No 509 item owned (memo-op arm) / slot mismatch (cash arm) | handler pre-flight | SEND_ERROR `NO_NOTE_ITEM` (3, silent unlock) | no | no |
| DestroyAsset fails (item vanished mid-flight) | saga step 1 | SEND_ERROR via `note_send` failed branch | no | no |
| Note create fails after destroy | saga step 2 + compensation | SEND_ERROR via failed branch | **re-awarded** | no |
| Success | saga completed | inventory op (excl unlock) + SEND_SUCCESS (4) | yes, exactly 1 | yes |

No steady state leaves item-without-note or note-without-item (FR-5); every
path releases the client excl lock (§1.4).

## 9. Testing strategy

- **Packet fixtures:** `ItemUseNote` × 5 versions; the three promotion cells
  (§6.3); regression fixtures for the `item_use.go` gate change (v83/84 trailing
  vs v87/95/jms leading).
- **Handler tests (Builder pattern, no test-helper files):** both arms —
  pre-flight rejection paths (no saga created, correct error body), accept path
  (saga created with destroy-first step order and correct payloads).
- **Orchestrator tests:** `CreateNote` handler emission; note-status event
  matching by txn id (incl. ignoring events without txn id); `note_send`
  compensation re-award on step-2 failure.
- **atlas-notes tests:** txn id round-trip; `CREATE_FAILED` emission.
- **Channel saga-consumer tests:** completed → SEND_SUCCESS announce; failed →
  SEND_ERROR announce.
- Full verification gate per CLAUDE.md (test -race / vet / build / bake for
  atlas-channel, atlas-saga-orchestrator, atlas-notes; redis-key-guard).

## 10. Risks & follow-ups

- **Excl unlock on success** relies on the destroy-driven inventory-operation
  packet setting its leading excl byte — the plan must verify Atlas's inventory
  operation writer does this for saga-driven destroys before relying on it;
  fallback is announcing StatChanged-with-excl from the completed branch.
- **jms_v185 discard shape** may add fields (0x33d fn size); budget for a codec
  delta behind a version gate if the fixture derivation demands it.
- **Gift flow future-proofing:** when cash-shop gifting lands, the NOTE_ACTION
  SEND arm's consume-gate must be bypassed for genuine gift sends (item cost is
  in the purchase) — pointer recorded here intentionally.
- **v84 opcode-table caution** and the SMC/`_SCY` jms dump particulars are
  called out in §6.3 so the execute phase doesn't rediscover them.
