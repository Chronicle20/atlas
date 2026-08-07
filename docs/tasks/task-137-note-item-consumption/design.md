# Note Item Consumption & Memo Packet Verification — Design

Task: task-137-note-item-consumption
Status: Proposed (revised for 9-version scope)
PRD: `docs/tasks/task-137-note-item-consumption/prd.md`

> **Scope revision (main-sync).** This design was originally written against the
> five versions in the coverage matrix at plan time (gms_v83, gms_v84, gms_v87,
> gms_v95, jms_v185). Main has since added four legacy version columns —
> **gms_v48, gms_v61, gms_v72, gms_v79** — which are in scope for this task. All
> four were IDA-verified in a dedicated pass; the per-version findings live in
> `docs/tasks/task-137-note-item-consumption/legacy-verify/{v48,v61,v72,v79}.md`
> and every claim below is cited to an address there. The feature extends to all
> nine versions; the divergences the legacy pass surfaced (a shifted MEMO_RESULT
> mode table on v48/v61, and per-version note-discard bodies) are folded in
> below.

---

## 1. Verified ground truth (FR-1 / FR-2 — IDA verification)

All nine client binaries were read. Every claim below is IDA-verified at the
cited address; nothing is inherited from Cosmic or memory.

### 1.1 The player note-send path is USE_CASH_ITEM in ALL nine versions (Q1 resolved)

`CWvsContext::SendConsumeCashItemUseRequest` builds the USE_CASH_ITEM packet and
switches on the item's cash-slot type. The Note cash item (classification 509 /
item 5090000) resolves to a "send memo" arm that opens a recipient+message
compose modal, then encodes the two strings. This is the ONLY player-reachable
note-send path in every version (the NOTE_ACTION SEND arm is never a legitimate
player path — §1.2).

| Version | IDB fn addr | Opcode | updateTime | cash-type (client-internal) |
|---|---|---|---|---|
| gms_v48 | 0x70e495 | 0x3E | **trailing** | 19 (WZ/`LOTTERYITEM`-driven) |
| gms_v61 | 0x832a5d | 0x49 | **trailing** | 20 |
| gms_v72 | 0x904fe2 | 0x4E | **trailing** | 21 |
| gms_v79 | 0x95634a | 0x4D | **trailing** | 21 |
| gms_v83 | 0xa0a63f | 0x4F | **trailing** | 21 |
| gms_v84 | 0xa54a2f | 0x4F | **trailing** | 21 |
| gms_v87 | 0xa9fef9 | 0x52 | **leading** | 21 |
| gms_v95 | 0x9eb3e0 | 0x55 | **leading** | 21 |
| jms_v185 | 0xaef2f5 | 0x47 | **leading** | 21 |

Full wire format (serverbound USE_CASH_ITEM, note arm) — **identical field order
in all nine versions**, differing only in opcode and updateTime position:

```
opcode
[updateTime int32]        // GMS v87+, JMS185: leading
slot      int16           // Encode2(nPOS)
itemId    int32           // Encode4(nItemID)
toName    string          // EncodeStr  (recipient, from compose modal)
message   string          // EncodeStr  (body,      from compose modal)
[updateTime int32]        // GMS v48/v61/v72/v79/v83/v84: trailing
```

**The client-internal cash-slot type value (19/20/21) is a dispatch index, NOT a
wire field.** It never appears on the serverbound packet, so its per-version
divergence (v48→19, v61→20, v72+→21) is irrelevant to Atlas's codec. Atlas
decodes the arm purely by opcode; the only wire-relevant differences across the
nine versions are the opcode (already per-template) and the updateTime position.

**Codec gate:** `libs/atlas-packet/cash/serverbound/item_use.go` currently gates
the leading updateTime on `Region=="GMS" && MajorVersion>=95`. The correct gate
is `(GMS && MajorVersion>=87) || JMS ⇒ leading, else trailing`. This single
predicate classifies **all nine versions correctly**: the four legacy GMS
versions (v48/61/72/79) and v83/v84 are all `< 87` ⇒ trailing (IDA-confirmed
trailing in every legacy IDB); v87/v95 and JMS ⇒ leading. No per-version special
case is needed. (This is another instance of the known "`>83` vs `>=87`"
off-by-one family; the current `>=95` is wrong for v87.)

### 1.2 The NOTE_ACTION "SEND" arm is never a legitimate player path

The NOTE_ACTION opcode's serverbound writers were enumerated per version by
scanning every `COutPacket(<note-opcode>)` construction site. The mode-0
("SEND") arm, where it exists, is written **only** by the cash-shop gift flow
(`CCashShop::OnCashItemResLoadGiftDone`) — never by a player note compose:

| Version | note-op | writers present |
|---|---|---|
| gms_v48 | 0x65 | SetRet=discard(mode 1) only (+ a follow-up 0x66 for flag-1 notes). **No mode-0 writer.** |
| gms_v61 | 0x77 | SetRet=discard(1), gift SEND(0). No request(2). |
| gms_v72 | 0x81 | SetRet=discard(1), request(2), gift SEND(0). |
| gms_v79 | 0x80 | SetRet=discard(1), request(2). **Gift fn is a pure reader — no mode-0 writer.** |
| gms_v83–jms_v185 | (per template) | SetRet=discard(1), request(2), gift SEND(0). |

**Consequence (uniform across all nine):** no legitimate player-initiated send
ever arrives on the NOTE_ACTION SEND arm. Traffic there is either the
(out-of-scope, server-side-unimplemented) cash-shop gift flow or a tampered
client. The current free-of-charge `SendNote` at
`services/atlas-channel/atlas.com/channel/socket/handler/note_operation.go:43`
is reachable only by cheaters today — and it creates free notes.

### 1.3 MEMO_RESULT client arms and error codes (Q3 resolved) — TWO mode tables

`CWvsContext::OnMemoResult` decodes a leading mode byte and dispatches. The mode
numbering is **NOT uniform**: v48 and v61 use a compressed/shifted table; v72 and
everything newer use the standard table. This must be config-resolved per tenant
(§5.3), never hard-coded.

| Version(s) | SHOW/LOAD | SEND_SUCCESS | SEND_ERROR | REFRESH | OnMemoResult addr |
|---|---|---|---|---|---|
| gms_v48 | 2 | 3 | **4** | — | 0x71d8e2 (`mode = Decode1()-2`) |
| gms_v61 | 2 | 3 | **4** | — | 0x8468be |
| gms_v72 | 3 | 4 | **5** | 7 | 0x91d23d |
| gms_v79 | 3 | 4 | **5** | 7 | 0x96f185 |
| gms_v83 | 3 | 4 | **5** | 7 | 0xa2508b |
| gms_v84 | 3 | 4 | **5** | 7 | 0xa70785 |
| gms_v87 | 3 | 4 | **5** | 7 | 0xabccc2 |
| gms_v95 | 3 | 4 | **5** | 7 | 0x9f9da0 |
| jms_v185 | 3 | 4 | **5** | 7 | 0xb0c6d0 |

- **SEND_ERROR sub-codes are 0/1/2 in every version:** **0** = "the other
  character is online now, use whisper"; **1** = "check the name of the receiving
  character"; **2** = "the receiver's inbox is full". Any sub-code **≥3 is a
  silent no-op** (no dialog) in every version.
- **There is NO "no Note item" error arm in any version.** FR-6's "IDA-verified
  error code" for the missing-item case does not exist client-side; §5.3 resolves
  this with the always-safe out-of-range sub-code.
- The existing tenant writer tables encode the standard table
  (`operations {SHOW:3, SEND_SUCCESS:4, SEND_ERROR:5, REFRESH:7}`,
  `errors {RECEIVER_ONLINE:0, RECEIVER_UNKNOWN:1, RECEIVER_INBOX_FULL:2}`), which
  is correct for v72–jms. **gms_v48 and gms_v61 need their own operations table**
  (`SHOW:2, SEND_SUCCESS:3, SEND_ERROR:4`, no REFRESH) — §5.1.

### 1.4 Exclusive-request lock: the response contract (uniform)

`SendConsumeCashItemUseRequest` sets `SetExclRequestSent(1)`. While set, the
client refuses further item-use/transfer requests — a server that never responds
wedges the client. Verified unlock paths (v83 addresses cited in the original
pass; legacy pass confirmed the same structural contract, e.g. v79 SEND_ERROR
clears the excl flag at 0x96f1c7 before decoding the error code):

- **MEMO_RESULT SEND_ERROR arm** clears the excl flag *before* decoding the error
  sub-code — so an error response with ANY sub-code (including out-of-range)
  unlocks the client.
- **OnInventoryOperation / OnStatChanged first byte** — the standard
  "enableActions" unlock that rides on the inventory-change packet emitted when
  the item is destroyed.
- **SEND_SUCCESS does NOT clear the flag**; on the success path the unlock comes
  from the inventory operation of the consumed item.

**Contract for the server:** every accepted send must eventually produce an
inventory-operation packet (with the excl byte set) + SEND_SUCCESS; every
rejected send must produce SEND_ERROR.

### 1.5 Note-discard (NOTE_ACTION SetRet, mode 1) bodies — per version

`CMemoListDlg::SetRet` (mode 1) is the serverbound note-list confirm/discard the
client sends; it is `note/serverbound/NoteOperationDiscard` in the matrix. The
body is variable-length and count-prefixed, with a common header and a
per-entry loop that special-cases "gift/reward" memos:

```
mode           u8   = 1
totalCount     u8   // memos in the list
specialCount   u8   // memos whose flag == <special> (see table)
emptySlots     u8   // free ETC/type-4 inventory slots
per memo entry:
  if flag == <special> AND a free slot remains:
      SN     int32
      flag   u8
      extra  int32   // reward/itemId/mesos claimed from the gift memo
      (emptySlots--)
  else if flag == <special> AND no free slot:
      (entry skipped; client shows an "inbox full" notice)
  else (normal):
      SN     int32
      flag   u8
```

| Version | note-op | special flag | extra field | SetRet addr | notes |
|---|---|---|---|---|---|
| gms_v48 | 0x65 | **2** | reward int32 | 0x534dc4 | ALSO emits a follow-up **0x66** packet for flag==1 notes (0x535001) |
| gms_v61 | 0x77 | **2** | itemId int32 | 0x5ad50c | — |
| gms_v72 | 0x81 | **3** | mesos int32 | 0x5fb443 | — |
| gms_v79 | 0x80 | **3** | value int32 | 0x619f32 | function-entry address (0x619fb7 is the internal `COutPacket` construction call-site within this same function, not the entry — matches the other rows' convention and the committed marker) |

The four legacy `NoteOperationDiscard` cells (all ❌ on main) are promoted in this
task using these shapes (§6.3). jms_v185's discard body (0x33d bytes vs GMS's
~0x26b) is derived from scratch during verification, as originally planned.

---

## 2. Scope decisions confirmed by the evidence

1. **Implement the USE_CASH_ITEM note arm as the primary (and only legitimate)
   player send path** on every version (FR-3). The `CharacterCashItemUseHandle`
   route already exists in **all nine** seed templates (main wired it during the
   version bring-ups; the original design's "routed only on gms_83/84, must add
   to 87/95/jms" premise is now stale — see §5.1). The remaining work is the
   handler-code note arm plus the writer config, not the handler routing.
2. **Gate the NOTE_ACTION SEND arm with the same ownership+consumption flow**
   (FR-4) on every version. Since its only legitimate writer is the unimplemented
   gift flow (and it has NO mode-0 writer at all on v48/v79), uniform gating
   cannot break any player-reachable behavior and it closes the cheat path. When
   cash-shop gifting is implemented later it must NOT reuse this arm's
   consume-gated path (the gift note is paid for by the gift purchase); that
   future task gets a pointer here.
3. **Gift memos remain out of scope** (PRD non-goal). The `OperationSend` codec
   keeps decoding only `toName`+`message`; the trailing gift fields are left
   unread — the handler rejects the arm before they matter. `note/serverbound/
   NoteOperationSend` (❌ on all nine, the gift-send codec) is NOT a task target.
4. **Flag stays `1`** on send (PRD non-goal): nothing in any version's send body
   contradicts the current `SendNote(..., flag=1)`.

---

## 3. Atomicity architecture (Q2 / FR-5) — version-agnostic

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
*Cons:* "item destroyed, note never created" is a real steady state if
atlas-notes fails — violates FR-5; and nothing tells the client. Rejected.

**C. Note-first, destroy-after.**
*Cons:* the free-note steady state is farmable (destroy fails after the note
exists). "No free notes" is the primary goal; rejected.

### Decision: Approach A

Saga type `note_send` (new `SagaType` constant), initiated by the channel
handler, steps in this order:

```
Step 1  DestroyAsset  {CharacterId: sender, TemplateId: <the 509 item>, Quantity: 1, RemoveAll: false}
Step 2  CreateNote    {SenderId, ReceiverId, Message, Flag: 1}   // new action
```

- **New saga action `CreateNote`** in `libs/atlas-saga` + orchestrator handler
  that emits the note CREATE command **carrying the saga transaction id**.
- **atlas-notes threads the transaction id through**: optional `TransactionId`
  field on `note.Command`/`StatusEvent` (additive, zero-value = absent), and a
  new `CREATE_FAILED` status event emitted when `Create` errors. The
  orchestrator's new note status consumer matches `CREATED`/`CREATE_FAILED` by
  transaction id → `StepCompleted(txnId, true/false)`.
- **Compensation:** add a `note_send` branch to `CompensateFailedStep` — if
  Step 2 fails after Step 1 completed, re-award via the existing
  `RequestCreateItem` path (mirrors the pet-evolution/gachapon re-award
  precedent, `compensator.go:1336-1344`).
- **Client feedback** in the channel saga consumer:
  - `handleCompletedEvent` + `SagaType == note_send` → announce
    `NoteSendSuccessBody` (MEMO_RESULT SEND_SUCCESS mode — **version-resolved**,
    3 on v48/v61, 4 on v72+). Excl lock released by the destroy's inventory op
    (§1.4).
  - `handleFailedEvent` + `SagaType == note_send` → announce
    `NoteSendErrorBody` (SEND_ERROR mode — version-resolved, 4 on v48/v61, 5 on
    v72+; code per §5.3) — unlocks the client and surfaces the failure.

Pre-flight rejections (receiver unknown, no item, malformed) happen in the
channel handler **before the saga is created**, so no item is consumed on any
rejected send (FR-7). This machinery is entirely version-agnostic; only the
client-facing MEMO_RESULT mode/error bytes are version-resolved from config.

---

## 4. Handler design (atlas-channel) — version-agnostic

### 4.1 USE_CASH_ITEM note arm (`character_cash_item_use.go`)

New `CashSlotItemTypeNote = CashSlotItemType(21)` named constant (replacing the
bare `21` at line 251-253) and a dispatch arm:

1. Decode `cashsb.NewItemUseNote(updateTimeFirst)` — arm codec (§6.1).
2. The existing slot/template validation already proves the sender owns the item
   in the claimed slot; additionally require
   `item.GetClassification(itemId) == item.ClassificationNote`.
3. Resolve receiver: `character.GetByName(toName)` → miss ⇒ announce
   `NoteSendErrorBody(RECEIVER_UNKNOWN)`, return (no saga).
4. Receiver-online check ⇒ `NoteSendErrorBody(RECEIVER_ONLINE)` (code 0,
   IDA-verified). *Optional-but-recommended; drop and document if no cheap
   cross-world online lookup exists.*
5. Create the `note_send` saga (§3) with the slot's `TemplateId`.

> **Note on the cash-slot type value.** Atlas keys the note arm on
> `item.ClassificationNote` (509), NOT on the client-internal cash-slot type
> (which is 19/20/21 depending on version and never appears on the wire). The
> `CashSlotItemType(21)` constant is Atlas's own inventory-type tag, unrelated to
> the client's per-version dispatch index.

The stale `// TODO for v83 there is a trailing updateTime` comment (line 108) is
resolved by the arm codec and must be removed.

### 4.2 NOTE_ACTION SEND arm (`note_operation.go`)

Replace the direct `np.SendNote(...)` with the same flow, minus the slot:

1. Receiver resolution (existing, kept) → `RECEIVER_UNKNOWN` on miss.
2. Ownership scan: fetch the cash compartment and find the first asset with
   `item.GetClassification(TemplateId) == item.ClassificationNote`. No such
   asset ⇒ announce `NoteSendErrorBody(NO_NOTE_ITEM)` (§5.3), warn-log with
   character id and reason, return. (No session destroy: unlike the discard
   flag-tamper case this packet is also emitted by the future gift flow.)
3. Create the same `note_send` saga using the found asset's template id.

A small compartment helper (`FindFirstByClassification`) is added next to the
existing `FindFirstByItemId`.

### 4.3 Success/failure announcements

Handled by the saga status consumer branches (§3). No packet is sent inline on
the accept path — the handler's only inline announcements are the pre-flight
errors.

---

## 5. Configuration & wire values (FR-12 / FR-13, DOM-25)

### 5.1 Template wiring — nine seed templates + live PATCH doc

**The `CharacterCashItemUseHandle` and `NoteOperationHandle` routes already exist
in all nine templates** (verified in the seed data: opcodes 0x3E/0x49/0x4E/0x4D/
0x4F/0x4F/0x52/0x55/0x47 for cash-item, per §1.1). The original design's
"add handler routing to 87/95/jms" step is obsolete. The remaining template work
is the **writer config** (operations mode table + errors table), which must be
correct per version — including the shifted v48/v61 table:

| Change | v48 | v61 | v72 | v79 | v83 | v84 | v87 | v95 | jms185 |
|---|---|---|---|---|---|---|---|---|---|
| `CharacterCashItemUseHandle` routed | ✅ 0x3E | ✅ 0x49 | ✅ 0x4E | ✅ 0x4D | ✅ 0x4F | ✅ 0x4F | ✅ 0x52 | ✅ 0x55 | ✅ 0x47 |
| `NoteOperationHandle` routed | ✅ 0x65 | ✅ 0x77 | ✅ 0x81 | ✅ 0x80 | ✅ | ✅ | ✅ | ✅ | ✅ |
| MEMO_RESULT writer `operations` | **{SHOW:2,SUCCESS:3,ERROR:4}** | **{SHOW:2,SUCCESS:3,ERROR:4}** | {SHOW:3,SUCCESS:4,ERROR:5,REFRESH:7} | same | same | same | same | same | same |
| MEMO_RESULT writer `errors` + `NO_NOTE_ITEM` | add key (val 3) | add key (3) | add key (3) | add key (3) | add key (3) | add key (3) | add key (3) | add key (3) | add key (3) |

- Every handler entry must have a `validator` (LoggedInValidator) — a
  validator-less entry is silently dropped (known bug: `BuildHandlerMap` skips
  it). The existing routes already carry validators; verify on any edit.
- The per-version SEND/DISCARD/REQUEST **serverbound** mode bytes for the
  `NoteOperationHandle` `operations` table were confirmed at the writer level in
  the legacy pass (SetRet=1, request=2, gift=0 where present); the execute
  phase's fixture work pins them per version.
- **Live-tenant PATCH must be documented** for every existing tenant of all nine
  versions (seed templates only apply at creation — known bug pattern), including
  a channel restart note. The v48/v61 tenants specifically need the shifted
  MEMO_RESULT operations table PATCHed, not just the new `NO_NOTE_ITEM` key.

### 5.2 New reader options

None required: the note arm decodes two strings + version-positioned updateTime,
driven by the tenant-derived `updateTimeFirst`, not by per-packet options. The
`ItemUse` codec's version gate moves from `>=95` to `(GMS>=87)||JMS` — a code
change validated by byte fixtures, not config.

### 5.3 The "no Note item" error byte (FR-6 resolution)

Verified reality: no client arm exists for this case in any version (§1.3), and a
legitimate client cannot reach it (the send dialog only opens by using an owned
item). Decision:

- New semantic key `NO_NOTE_ITEM` in the writer `errors` table, value **3** on
  all nine versions — deliberately outside the client's 0-2 dialog range.
  IDA-verified effect (every version): the SEND_ERROR arm clears the excl lock,
  then a sub-code ≥3 shows no dialog. This is not a guessed byte; it is a chosen
  out-of-range value with verified client semantics (silent unlock).
- **The SEND_ERROR *mode* byte is version-resolved** (4 on v48/v61, 5 on v72+ —
  §1.3) and comes from the `operations` table, so the emitted error packet uses
  the correct mode per tenant. The `NO_NOTE_ITEM` sub-code (3) is the same
  everywhere.
- The rejection is warn-logged server-side (observability NFR).
- PRD FR-6's intent is satisfied — the client receives a verified-safe error
  packet via existing plumbing — with the factual correction that no dedicated
  code exists to send.

---

## 6. Packet library changes (`libs/atlas-packet`)

### 6.1 New serverbound arm codec: `cash/serverbound/item_use_note.go`

`ItemUseNote{ toName, message string; updateTime uint32 }`, shape mirroring
`ItemUseChalkboard`: `toName = ReadAsciiString()`, `message = ReadAsciiString()`,
then trailing `updateTime` iff `!updateTimeFirst`. fname:
`CWvsContext::SendConsumeCashItemUseRequest#Note`. **Byte-fixture tests for all
nine versions** from the §1.1 write orders (six trailing: v48/v61/v72/v79/v83/v84;
three leading: v87/v95/jms185). This also gives the USE_CASH_ITEM matrix row its
first linked packet; full promotion of that row is not a task goal but the
fixtures land the evidence.

### 6.2 `item_use.go` gate fix

`updateTimeFirst`: `Region=="GMS" && MajorVersion>=95` → `(GMS && >=87) || JMS`.
IDA-verified against every version: trailing for v48/v61/v72/v79 (legacy pass),
v83, and v84 (0xa54a2f); leading for v87 (0xa9fef9), v95, JMS. The same flag
flows to all existing arms; the six trailing versions and the three leading
versions are all classified correctly by the single predicate, and nothing
regresses (v87/95/jms were previously unrouted for the note arm).

### 6.3 Matrix promotions (FR-8/9/10/11)

Standard packet-verifier flow per cell (decompile read order → byte-fixture with
`packet-audit:verify` marker → pin evidence → regenerate matrix):

**MEMO_RESULT (clientbound):**
- `MEMO_RESULT` × gms_v84 — `CWvsContext::OnMemoResult` @ 0xa70785 (the old
  audit's "function not found" is stale); arms identical to v83. Opcode 0x029
  confirmed from the v84 dispatch table (the known v84 shift starts above ~0x3D;
  0x029 is below it but is confirmed, not assumed).
- `MEMO_RESULT` × jms_v185 — `OnMemoResult` @ 0xb0c6d0, opcode 0x026 to confirm;
  instance `MapleStory_dump_SCY.exe`; pass `--audit-dir docs/packets/audits/
  jms_v185` explicitly (known tooling gotcha).
- The four legacy MEMO_RESULT cells (v48/v61/v72/v79) are **already ✅** on main —
  not targets. (Note for the execute phase: their ✅ reflects the SHOW/NoteDisplay
  list shape; the SEND_SUCCESS/SEND_ERROR modes those tenants use for the *send*
  feature are the shifted v48/v61 values in §1.3, delivered via config in §5.1 —
  no codec change, but the operations table must be right.)

**NoteOperationDiscard (serverbound) — jms + four legacy cells:**
- `NoteOperationDiscard` × jms_v185 — `CMemoListDlg::SetRet` @ 0x6c2d43; **0x33d
  bytes vs ~0x26b on GMS — derive the jms read order from scratch.**
- `NoteOperationDiscard` × gms_v48 (SetRet @ 0x534dc4), × gms_v61 (0x5ad50c),
  × gms_v72 (0x5fb443), × gms_v79 (0x619f32, function entry — not 0x619fb7,
  which is the internal `COutPacket` construction call-site inside the same
  function) — promote ❌→✅ using the per-version
  body shapes in §1.5 (uniform header + special-flag entries; special flag = 2
  on v48/v61, 3 on v72/v79; v48 also emits a follow-up 0x66). Each cell gets its
  own byte fixture derived from its cited SetRet; the shapes are close but the
  special-flag value and v48's 0x66 tail differ, so do not assume one shape
  covers all four.

**NoteOperationSend** stays ❌ on all nine (gift-send codec, out of scope).

`packet-audit` matrix/fname-doc/operations `--check` must exit 0 with no
regressions elsewhere.

---

## 7. Saga & service changes (summary of new surface) — version-agnostic

| Module | Change |
|---|---|
| `libs/atlas-saga` | `SagaType` `note_send`; `Action` `CreateNote`; `CreateNotePayload{SenderId, ReceiverId, Message, Flag}` |
| atlas-saga-orchestrator | `GetHandler` case for `CreateNote` → note CREATE command with txn id; note-status consumer (`CREATED`/`CREATE_FAILED` → StepCompleted); `CompensateFailedStep` branch for `note_send` (re-award destroyed item); emit Failed status event for `note_send` failures |
| atlas-notes | optional `TransactionId` on command/status messages (additive); emit `CREATE_FAILED` status event on create error |
| atlas-channel | two handler arms (§4); `FindFirstByClassification` compartment helper; saga consumer branches for `note_send` completed/failed. The note CREATE command is now emitted by the orchestrator (with txn id), so the SEND arm stops calling `note.SendNote`; the processor's `DiscardNotes`/`GetByCharacter` paths are unchanged |

`atlas-notes` storage/display/discard are untouched. atlas-saga-orchestrator's
existing `DestroyAsset` handler is reused as-is. None of this surface is
version-conditional — the nine-version fan-out is confined to the packet codec
(§6.1 fixtures), the codec gate (§6.2), and the writer config (§5.1).

## 8. Error handling matrix (version-agnostic behavior; mode/error bytes config-resolved)

| Case | Where caught | Client packet | Item consumed? | Note created? |
|---|---|---|---|---|
| Receiver name unknown | handler pre-flight | SEND_ERROR `RECEIVER_UNKNOWN` (sub 1) | no | no |
| Receiver online (if check lands) | handler pre-flight | SEND_ERROR `RECEIVER_ONLINE` (sub 0) | no | no |
| No 509 item owned (memo-op arm) / slot mismatch (cash arm) | handler pre-flight | SEND_ERROR `NO_NOTE_ITEM` (sub 3, silent unlock) | no | no |
| DestroyAsset fails (item vanished mid-flight) | saga step 1 | SEND_ERROR via `note_send` failed branch | no | no |
| Note create fails after destroy | saga step 2 + compensation | SEND_ERROR via failed branch | **re-awarded** | no |
| Success | saga completed | inventory op (excl unlock) + SEND_SUCCESS | yes, exactly 1 | yes |

The SEND_ERROR/SEND_SUCCESS **mode** byte is resolved per tenant (4/3 on v48/v61,
5/4 on v72+); the sub-codes (0/1/2/3) are uniform. No steady state leaves
item-without-note or note-without-item (FR-5); every path releases the client
excl lock (§1.4).

## 9. Testing strategy

- **Packet fixtures:** `ItemUseNote` × **9 versions** (six trailing, three
  leading); the MEMO_RESULT v84/jms promotions; the **five** `NoteOperationDiscard`
  promotions (jms + v48/v61/v72/v79) from §1.5; regression fixtures for the
  `item_use.go` gate change.
- **Handler tests (Builder pattern, no test-helper files):** both arms —
  pre-flight rejection paths (no saga created, correct error body with the
  version-resolved mode byte), accept path (saga created with destroy-first step
  order and correct payloads).
- **Orchestrator tests:** `CreateNote` handler emission; note-status event
  matching by txn id (incl. ignoring events without txn id); `note_send`
  compensation re-award on step-2 failure.
- **atlas-notes tests:** txn id round-trip; `CREATE_FAILED` emission.
- **Channel saga-consumer tests:** completed → SEND_SUCCESS announce (correct
  mode per version); failed → SEND_ERROR announce.
- Full verification gate per CLAUDE.md (test -race / vet / build / bake for
  atlas-channel, atlas-saga-orchestrator, atlas-notes; redis-key-guard;
  goroutine-guard; lint).

## 10. Risks & follow-ups

- **v48/v61 shifted mode table** is the highest-value legacy finding: a
  hard-coded SEND_ERROR=5 (the standard value) would silently wedge those clients
  (mode 5 is unhandled on v48/v61 → no unlock, no dialog). The operations-table
  config and the fixture assertions must use 4 there. Guard with a config test.
- **Excl unlock on success** relies on the destroy-driven inventory-operation
  packet setting its leading excl byte — the plan must verify Atlas's inventory
  operation writer does this for saga-driven destroys; fallback is announcing
  StatChanged-with-excl from the completed branch.
- **jms_v185 discard shape** may add fields (0x33d fn size); budget for a codec
  delta behind a version gate if the fixture derivation demands it.
- **v48 discard 0x66 tail** — v48's SetRet emits a second packet (0x66) for
  flag==1 notes; confirm whether Atlas's NoteOperationDiscard cell needs to model
  it or whether it is a separate op, during the v48 cell verification.
- **Gift flow future-proofing:** when cash-shop gifting lands, the NOTE_ACTION
  SEND arm's consume-gate must be bypassed for genuine gift sends — pointer
  recorded here intentionally.
- **v84 opcode-table caution** and the SMC/`_SCY` jms dump particulars are called
  out in §6.3 so the execute phase doesn't rediscover them.
