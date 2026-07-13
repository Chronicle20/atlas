# Vega's Spell — Design

Task: task-130-vegas-spell
Status: Approved PRD → this document covers architecture, alternatives, and tradeoffs.
Inputs: `docs/tasks/task-130-vegas-spell/prd.md`, code exploration of atlas-channel,
atlas-consumables, atlas-inventory, libs/atlas-packet, libs/atlas-constants, seed templates,
the Cosmic reference checkout (repo-external), and **live IDA verification of the v83 and v95
clients** (both IDBs loaded during this design session; addresses cited inline).

---

## 1. Summary

Vega's Spell 10 (5610000) / 60 (5610001) are implemented as:

1. A new serverbound sub-body codec `ItemUseVegaScroll` in `libs/atlas-packet/cash/serverbound/`
   — **six int32s**: equip-tab marker (=1), equip slot, scroll-tab marker (=2), scroll (use) slot,
   a constant flag (=1), and a **trailing updateTime present on every version** (IDA-verified on
   v83 and v95 — see §2.1; this is two more fields than the Cosmic hypothesis in PRD FR-1.2).
2. A new arm in atlas-channel's `CharacterCashItemUseHandleFunc` matching the category-561
   `CashSlotItemType` (68 pre-v95 GMS and JMS, 71 GMS ≥ 95), which validates the tab markers and
   forwards a new Kafka command to atlas-consumables.
3. A new `REQUEST_VEGA_SCROLL` command in atlas-consumables (distinct from `REQUEST_SCROLL` —
   §3.1) that validates everything up front, then reserves the vega cash item and the scroll via
   **two chained single-item reservations** (cash first, then use — §3.2), and on the second
   confirmation runs `ConsumeVegaScroll`: the existing scroll application machinery with
   `successProb` overridden to 30/90, `whiteScroll=false`, `legendarySpirit=false`, consuming
   both reserved items. This deletes the `// TODO consume vega scroll` at
   `services/atlas-consumables/atlas.com/consumables/consumable/processor.go:641`.
4. A new single-byte `VegaScroll` clientbound writer in `libs/atlas-packet/cash/clientbound/`
   whose mode byte is resolved from the tenant `operations` table under **outcome-keyed** names
   (`START_SUCCESS`, `START_FAILURE`, `RESULT_SUCCESS`, `RESULT_FAILURE`, `INVALID`) — required
   because the mode values shifted +4 between v83 and v95 **and** v95 selects its result popup
   from the START byte, not the result byte (§2.2, §2.3).
5. A new `VEGA_SCROLL` consumable status event (and a `VEGA_INVALID` error type) consumed by
   atlas-channel to send, immediately and in order: VEGA start(outcome) → VEGA result(outcome)
   to the user session, the map-broadcast `ItemUpgrade` (`legendarySpirit=false`,
   `whiteScroll=false`), and enable-actions. No server-side delay (owner decision).

No REST changes. No database changes. v92 is parked (§2.6). v84's registry opcode is flagged
as suspect and must be re-verified before wiring (§2.5).

---

## 2. Findings that refine the PRD (IDA-verified this session)

Each item below supersedes the corresponding PRD hypothesis and is a deliberate design input.
v83 evidence is from the MapleStory_dump.exe IDB; v95 from GMS_v95.0_U_DEVM.exe.

### 2.1 Serverbound body is 6 int32s, not 4 — and the "1"/"2" are inventory tab indices

The vega request is not sent by `CWvsContext::SendConsumeCashItemUseRequest` directly. On both
v83 and v95, the case-68/case-71 arm **constructs the CUIVega dialog and hands it the partially
built USE_CASH_ITEM packet** (v83 `sub_82B89C` via the case-68 arm of 0xa0a63f; v95
`CUIVega::CUIVega` via the case-71 arm of 0x9eb3e0 — both receive `nItemID % 10` as the variant).
The dialog appends the sub-body and sends when the user clicks Start:

- v83 `sub_82CBE2` (CUIVega send, LABEL_28): `Encode4(this[366..369])`, `Encode4(this[38]=1)`,
  `Encode4(get_update_time())`, then `SendPacket`. The drop handler `sub_82CE58` proves the
  field semantics: dropping an equip (itemId/1000000==1) sets `this[366]=1, this[367]=slot`;
  dropping anything else sets `this[368]=2, this[369]=slot`.
- v95 `CUIVega::OnButtonClicked` (0x7bf4a0) names the same fields:
  `Encode4(m_nEquipItemTI)` (=1), `Encode4(m_nEquipSlotPosition)`, `Encode4(m_nScrollItemTI)`
  (=2), `Encode4(m_nScrollSlotPosition)`, `Encode4(m_nWhiteScrollUse)` (set to constant 1
  immediately before), `Encode4(get_update_time())`.

So the wire sub-body on **both** v83 and v95 is:

| # | type | field | value |
|---|---|---|---|
| 1 | int32 | equip tab index | 1 (equip inventory) |
| 2 | int32 | equip slot position | from the dialog drop |
| 3 | int32 | scroll tab index | 2 (use inventory) |
| 4 | int32 | scroll slot position | from the dialog drop |
| 5 | int32 | flag | constant 1 (v95 IDB names it `m_nWhiteScrollUse` but always writes 1; semantics unconfirmed — read and ignored) |
| 6 | int32 | updateTime | trailing on **all** versions, independent of the prefix's `updateTimeFirst` (v95 carries updateTime in the prefix **and** here — both verified) |

Cosmic reads only fields 1–4 and never notices 5–6 (Java servers ignore trailing bytes). The
Atlas codec must read all six. The v95 outer prefix was also re-verified:
`COutPacket(85=0x55); Encode4(updateTime); Encode2(slot); Encode4(itemId)` — consistent with the
existing `ItemUse` codec's `updateTimeFirst` handling and the registry's v95 USE_CASH_ITEM 0x55.

### 2.2 VEGA_SCROLL mode bytes are version-shifted (+4 from v83 to v95)

`CUIVega::OnVegaResult` reads a **single mode byte** on both versions, but the accepted values
differ:

| arm | v83 (0x82d8d5) | v95 (0x7bf7b0) |
|---|---|---|
| start animation (twinkle sound, gauge) | 0x40 or 0x45 | 0x44 or 0x49 |
| result (latched, displayed after the animation) | 0x41 or 0x43 | 0x45 or 0x47 |
| anything else | "This item cannot be used." notice + dialog closes | same (string 0x1A6C) |

This is the operations-table gotcha in packet form: PRD FR-5.1's `0x40/0x41/0x43` are **v83-only
values**, and hard-coding them would break v95+ exactly like the missing per-version operations
tables did for dispatcher packets. Config resolution (FR-5.3) is not just policy here — it is
load-bearing.

Also corrected: Cosmic's comment "0x39/0x45 crash the client" is wrong for this client — v83
routes 0x45 into the *start* arm and 0x39 into the safe notice arm. There is no crash arm in
either version; the else-arm is always the notice.

### 2.3 On v95 the START byte carries the outcome — writer keys must be outcome-keyed

v95 `CUIVega::Draw` (0x7c1dd0) selects the result popup from `m_nRet1` — the **start** byte:
68 (0x44) → `CUIVegaResultPopup(type 1)` (template string-pool 5417), 73 (0x49) → popup type 2
(template 5418); the result byte (`m_nRet2`) is only validated to be in {0x45, 0x47}. On v83
there is no popup — the dialog renders `EffectSuccess`/`EffectFail` (verified nodes in the v83
`UIWindow.img/VegaSpell` tree) from the result byte, and the start byte is fixed.

Therefore the writer exposes outcome-keyed operations, resolved per tenant:

| key | v83 (verified) | v95 (arm-verified; success/fail pairing to confirm at implementation) |
|---|---|---|
| `START_SUCCESS` | 0x40 | 0x44 |
| `START_FAILURE` | 0x40 | 0x49 |
| `RESULT_SUCCESS` | 0x41 | 0x45 |
| `RESULT_FAILURE` | 0x43 | 0x47 |
| `INVALID` | 0x42 | 0x42 |

The server always resolves the outcome before sending (immediate resolution), so it can select
the start byte by outcome; on v83 both start keys collapse to 0x40 harmlessly. The one open
sub-question — whether v95 popup type 1 = success (start 0x44) or the reverse — is pinned at
implementation by reading the v95 string-pool templates 5417/5418 (or a live check); the byte
fixtures then lock it in. `INVALID` = 0x42 is safe on both versions by the else-arm analysis
above, and it is *required* (not optional): after sending, the client sets its exclusive-request
state and disables the dialog (`m_bRequestSent`); a rejection that sent nothing would leave the
dialog wedged. 0x42 shows the notice and closes it (`SetRet(1)`).

### 2.4 Clientbound opcodes: v83 and v95 confirmed; v84 suspect; v87/jms unverified; v92 absent

`CUIVega::OnPacket` dispatches on the opcode directly: v83 `a2 == 0x166` (0x82d8bf), v95
`nType == 0x1AD` (0x7c0680) — both **match the registry**. Remaining versions:

| version | registry opcode | provenance | status |
|---|---|---|---|
| gms_v83 | 0x166 (358) | csv-import | **IDA-confirmed this session** |
| gms_v84 | 0x166 (358) | csv-import | **suspect** — see §2.5 |
| gms_v87 | 0x17B (379) | csv-import | verify against v87 IDB at implementation |
| gms_v92 | — (no registry file; CSV column says 0x1A0) | — | parked — §2.6 |
| gms_v95 | 0x1AD (429) | csv-import | **IDA-confirmed this session** |
| jms_v185 | 0x183 (387) | csv-import | verify against jms IDB at implementation |

The v87/jms/v84 IDBs are not currently loaded (instance set rotates); if one cannot be loaded at
implementation time, that version's wiring is **escalated to the owner, never guessed** —
unresolved-fname/opcode escalation rule.

### 2.5 v84's 0x166 is almost certainly a stale CSV carryover

The v84 clientbound opcode table is shifted +2→+10 versus v83 above ~0x3D (known bug memory),
and the task-100 reshift is documented to have corrected only `ida-discovered` rows, leaving
`csv-import` rows unshifted. VEGA_SCROLL v84 is `csv-import` at the *identical* v83 value, far
above the shift threshold. Treat 358 as wrong until `discover-ops`/IDA proves otherwise; do not
register the v84 writer until verified. (The v84 seed template ships the writer entry only after
verification — an unverified wrong opcode is worse than absence, since a stray packet at another
handler's opcode can crash the client.)

### 2.6 v92 is parked (matches task-126 §2.3 and the v92 mount-food precedent)

v92 has no IDB, no registry file, and — decisive — no serverbound USE_CASH_ITEM opcode known and
no `CharacterCashItemUseHandle` wired in its template, so the client's vega request could not
even reach the handler. The entire cash-item-use feature is inert on v92 today. PRD FR-6.1 lists
v92, but its own verification bar (IDA-verified opcodes) is unsatisfiable there. **Decision:**
park v92 with an explicit note; the CSV's 0x1A0 is recorded in this doc for the day a v92 IDB
exists.

### 2.7 The existing scroll path cannot address the vega target set (PRD open question 4)

`RequestScroll`/`ConsumeScroll` resolve the equip via `slot.GetSlotByPosition` +
`c.Equipment().Get(s.Type)` — **equipped items (negative positions) only**. The vega dialog's
equips come from the **equip inventory (positive slots)**: the v83 drop handler stores the
drag-source position, and Cosmic reads `getInventory(InventoryType.EQUIP)` (its `eSlot < 0`
branch is dead code — negative slots live in a different Cosmic inventory object and return
null). **Decision:** the vega path gets a dual-sign resolver — negative → equipped
(`Equipment().Get`), positive → `c.Inventory().Equipable().FindBySlot` (accessor exists,
`services/atlas-consumables/atlas.com/consumables/inventory/model.go:15`) — used identically in
validation and consumption. The normal scroll path is left untouched (PRD non-goal), but the
gap is noted in §9 for the owner.

### 2.8 Discovered pre-existing bug: batched reservations only process the first item

`services/atlas-inventory/atlas.com/inventory/compartment/processor.go:738` — `RequestReserve`
`return`s from inside its loop on the **first** reservation request, so a multi-item reserves
list never reserves items 2..n and never emits their RESERVED events. Consequence today: the
white-scroll flow in `RequestScroll` (which batches `[whiteScroll, scroll]`) reserves only the
white scroll, the once-listener waits for the scroll's RESERVED event that never comes, and the
whole scroll silently no-ops (the white-scroll reservation expires after its 30s TTL). This is a
pre-existing bug on main, independent of this task.

**Design consequence for vega:** never batch — issue one single-item `RequestReserve` per
compartment (§3.2), which sidesteps the bug entirely. **The bug itself is flagged to the owner
here** (fixing it changes the semantics of a shared inventory-service path and the white-scroll
repro deserves its own verification; it is one `return` misplacement plus tests, but it is not
vega scope per PRD non-goals). If the owner prefers it fixed inside task-130, it slots in as an
isolated plan task with its own tests.

### 2.9 Reservation failures emit no event; reservations self-expire after 30s

The inventory-side `RequestReserve` emits `RESERVED` on success and *nothing* on failure (log
only); `AddReservation` sets a 30-second TTL. So a chained flow that stalls mid-way self-heals:
the earlier reservation expires and the player keeps everything. This bounds the worst case of
§3.2 without new machinery (matching the existing `RequestScroll` behavior, whose once-listener
also dangles if the reservation never confirms).

---

## 3. Approaches considered

### 3.1 Delivery mechanism (PRD open question 1)

**A. Extend `RequestScrollBody` with optional vega fields.** One command type; the consumer
branches on presence of `vegaSlot`. Backward compatible, but overloads one contract with two
behaviors, makes the required/optional matrix ambiguous for future producers, and threads vega
concerns through the normal path the PRD says not to touch.

**B. New `REQUEST_VEGA_SCROLL` command on the existing `COMMAND_TOPIC_CONSUMABLE`. (Chosen.)**
Distinct typed contract (the topic already carries four command types — precedent), zero risk to
existing `RequestScrollBody` producers, and the consumer arm maps 1:1 onto a new processor
method. Costs one more command constant and mirror struct in the channel copy.

**C. Saga (`CASH_ITEM_USE`-initiated, like task-126's point-reset).** Rejected: the scroll
pipeline is reservation-based inside one service, not a cross-service saga; expressing "reserve
2, roll, mutate equip, consume 2" as saga steps would either wrap the whole thing in one opaque
step (saga adds nothing) or force the equip mutation/curse/roll logic out of
`ConsumeScroll`-shaped code into orchestrator vocabulary (large duplication for no atomicity
gain — the reservation flow already provides the all-or-nothing property the PRD asks for).

### 3.2 Cross-compartment atomicity (PRD FR-3.3)

The vega item is CASH, the scroll is USE; `RequestReserve` is per-compartment.

**A. Extend the compartment RESERVE command to span compartments.** Authoritative, but changes
the atlas-inventory domain model (reservations are keyed by compartment) for one caller —
largest blast radius, and still wouldn't be transactional across the two compartment locks.

**B. No reservation for the vega item — validate, apply scroll, then destroy the cash item
(Cosmic parity).** Simplest, but opens a race window where the vega item can be moved/used
between validation and destroy: the destroy fails after the boosted scroll already applied — a
free-boost exploit for a hacked client (the legit client is excl-request-blocked). Violates
FR-3.1's "reserved together".

**C. Chained single-item reservations. (Chosen.)** Reserve the vega (CASH) first; its RESERVED
confirmation (once-listener keyed by `transactionId + vegaItemId`) triggers the scroll (USE)
reservation; the scroll's RESERVED confirmation (keyed by `transactionId + scrollItemId`)
triggers `ConsumeVegaScroll`. Item-id keying is collision-free (561xxxx vs 20xxxxx). Properties:
- Both items are reserved before anything is consumed; consumption commits both reservations
  (`ConsumeItem` USE then CASH) only after the scroll is actually applied — FR-3.1/3.2 hold.
- Any synchronous failure cancels the already-made reservations explicitly
  (`CancelItemReservation` per compartment + the error event); an *asynchronous* stall (reserve
  rejected inventory-side → no event) leaves the earlier reservation to its 30s TTL (§2.9) —
  the player never loses items, worst case the vega is locked for 30s. Same failure envelope as
  the existing scroll flow.
- One extra Kafka round-trip of latency versus a batch — irrelevant for a user-paced dialog
  action, and batching is broken anyway (§2.8).

### 3.3 Writer key shape

Raw mode passthrough (`NewVegaScroll(mode byte)`) was rejected: every call site would need
version knowledge, which is exactly what the operations table exists to encapsulate, and the
v95 start-byte-carries-outcome discovery (§2.3) makes "the mode" a per-version *function of the
outcome*, not a constant. Outcome-keyed discrete constructors (`NewVegaScrollStart(success)`,
`NewVegaScrollResult(success)`, `NewVegaScrollInvalid()`) with `WithResolvedCode("operations",
key)` follow the task-096 discrete-per-mode rule and the config-drive-all-modes policy
(including the v83-stable INVALID byte — uniformity over "version-stable", owner-established).

---

## 4. Component design

### 4.1 libs/atlas-constants

- `item/constants.go` (or the ids file per lib layout): `VegasSpell10 Id = 5610000`,
  `VegasSpell60 Id = 5610001`, `ClassificationVegasSpell Classification = 561` (the channel
  handler currently compares raw `561`; the new constant replaces that literal in the vega arm),
  and `func IsVegasSpell(id Id) bool`. Re-check the lib index at implementation per DOM-21
  (verified absent as of this design).
- Vega policy helper (rate pairing) lives in atlas-consumables, not the constants lib — it is
  server policy, not a wire/domain identity: `vegaRates(id) (required uint32, boosted uint32)` →
  (10, 30) / (60, 90).

### 4.2 libs/atlas-packet — serverbound `ItemUseVegaScroll`

New `cash/serverbound/item_use_vega_scroll.go`, following the `ItemUseFieldEffect` sub-body
pattern:

```go
type ItemUseVegaScroll struct {
    equipTab   uint32 // wire =1 (equip inventory) — validated, see below
    equipSlot  int32  // positive = equip inventory; sign passed through to the service
    scrollTab  uint32 // wire =2 (use inventory)
    scrollSlot int32
    flag       uint32 // constant 1 on v83+v95; semantics unconfirmed (v95 IDB: m_nWhiteScrollUse); read, logged, ignored
    updateTime uint32 // trailing on ALL versions (§2.1) — unconditional read, no updateTimeFirst gate
}
```

Decode reads all six int32s unconditionally. No constructor flag is needed (unlike
`ItemUsePointReset`) because the trailing updateTime is present regardless of version — the
prefix-side `updateTimeFirst` distinction is already handled by the outer `ItemUse` codec.

Verification discipline (per `docs/packets/audits/VERIFYING_A_PACKET.md`): byte-fixture tests
with `packet-audit:verify` markers per wired version (v83 fixture derivable now from §2.1;
v87/jms/v84 after their IDB pass), serverbound evidence + REPORT under the shared
`CWvsContext::SendConsumeCashItemUseRequest` fname — **coordinate with task-126**, which is
producing the audit for the same fname's point-reset arm; whichever lands second splices its arm
into the existing report rather than overwriting (export splice rule).

### 4.3 libs/atlas-packet — clientbound `VegaScroll` writer

New `cash/clientbound/vega_scroll.go`: `VegaScrollWriter = "VegaScroll"`; body = one byte.
Discrete outcome-keyed constructors per §3.3; the operation body resolves the byte via
`WithResolvedCode("operations", <KEY>)` at encode time (storage `StoreAssets` is the reference
implementation of this pattern). Keys and per-version values per §2.3's table.

Round-trip tests plus exact-byte fixtures per version with `packet-audit:verify` markers;
clientbound audit REPORT under `docs/packets/audits/<version>/VegaScroll.*`; registry rows
promoted from `csv-import` with evidence for v83/v95 (already derived — §2.2/§2.4), v87/v84/jms
after their IDB pass; matrix regenerated.

### 4.4 atlas-channel — handler arm

In `CharacterCashItemUseHandleFunc` (`socket/handler/character_cash_item_use.go`), above the
fall-through warn:

1. Match `it == CashSlotItemType(68) || it == CashSlotItemType(71)` (named constants
   `CashSlotItemTypeVegasSpellPre95` / `CashSlotItemTypeVegasSpell95`). These values are
   server-internal dispatch bookkeeping mirroring the client's `get_consume_cash_item_type` —
   they never appear on the wire, so JMS falling into the pre-95 else-branch (68) is correct by
   construction.
2. Decode `ItemUseVegaScroll`. If `equipTab != 1 || scrollTab != 2` → warn log +
   enable-actions (impossible from a legit client; the dialog fills both before Start enables).
3. The outer prefix already resolved and template-checked the cash item at `source`
   (`GetItemInSlot` + template mismatch guard — existing code). Guard `IsVegasSpell(itemId)`
   (defense against a non-vega 561 id) → warn + enable-actions.
4. Emit the command:
   `consumable.NewProcessor(l, ctx).RequestVegaScrollUse(s.Field(), characterId, vegaSlot=source,
   vegaItemId=itemId, scrollSlot, equipSlot)` → `CommandRequestVegaScroll` on
   `COMMAND_TOPIC_CONSUMABLE` (channel-local mirror structs in `kafka/message/consumable/` and
   provider in `consumable/producer.go`, exactly parallel to `RequestScrollCommandProvider`).

The `// TODO for v83 there is a trailing updateTime` at line 108 is thereby resolved for this
arm (the sub-codec consumes it); the TODO itself stays for the remaining un-migrated arms (it is
task-126's §4.3 concern too — do not double-remove).

`cash_shop_entry.go:29` ("block when performing vega scrolling") stays untouched (PRD non-goal).

### 4.5 atlas-consumables — kafka contract

`kafka/message/consumable/kafka.go` additions:

```go
CommandRequestVegaScroll = "REQUEST_VEGA_SCROLL"

type RequestVegaScrollBody struct {
    VegaSlot   slot.Position `json:"vegaSlot"`   // cash compartment
    VegaItemId item.Id       `json:"vegaItemId"` // re-validated against slot contents
    ScrollSlot slot.Position `json:"scrollSlot"` // use compartment
    EquipSlot  slot.Position `json:"equipSlot"`  // sign convention per §2.7
}

EventTypeVegaScroll = "VEGA_SCROLL"
type VegaScrollBody struct {
    Success bool `json:"success"`
    Cursed  bool `json:"cursed"`
}

ErrorTypeVegaInvalid = "VEGA_INVALID" // on the existing ErrorBody
```

`RequestScrollBody` and its producers are untouched. Consumer registration in the existing
consumable command consumer, new arm dispatching to the processor.

### 4.6 atlas-consumables — processor

**`RequestVegaScroll(characterId, vegaSlot, vegaItemId, scrollSlot, equipSlot)`** — validate
everything before reserving anything (FR-2; every rejection here consumes nothing and emits
`ErrorTypeVegaInvalid`):

1. Load character with `InventoryDecorator`.
2. Vega: `c.Inventory().Cash().FindBySlot(vegaSlot)` exists, template equals `vegaItemId`, and
   `IsVegasSpell` (FR-2.1).
3. Scroll: `c.Inventory().Consumable().FindBySlot(scrollSlot)` exists (FR-2.2).
4. Rate gate (FR-2.3): `ci := cdp.GetById(scroll.TemplateId())`;
   `ci.SuccessRate() == required` where `required, boosted := vegaRates(vegaItemId)`. Exact
   match only — clean-slate/spike/cold-protection scrolls fail naturally (no 10/60 rates in the
   v83-era data; no special-casing, per PRD).
5. Equip: dual-sign resolver (§2.7) finds the target; `ValidateScrollUse(scroll, equip)` (FR-2.4
   — keeps the ≥1 upgrade-slot rule and the existing scroll-class semantics).
6. Chain per §3.2: register once-listener B (txn + scroll templateId → `ConsumeVegaScroll`),
   register once-listener A (txn + vega templateId → issue the scroll reservation
   `cpp.RequestReserve(txn, charId, TypeValueUse, [scroll])`), then kick off the chain with
   `cpp.RequestReserve(txn, charId, TypeValueCash, [vega])`. Synchronous producer errors →
   `VegaConsumeError` (below).

**`ConsumeVegaScroll(txn, characterId, vegaItem, scrollItem, equipSlot)`** (the once-handler,
mirroring `ConsumeScroll`'s shape):

1. Re-load character; re-resolve equip (dual-sign); re-run the rate gate and
   `ValidateScrollUse` (state may have moved between request and reservation confirmations).
   Failure → `VegaConsumeError`: `CancelItemReservation(USE, scrollSlot)` +
   `CancelItemReservation(CASH, vegaSlot)` + `ErrorTypeVegaInvalid` event.
2. `successProb := boosted` (30/90) — **this line replaces the TODO at processor.go:641** in the
   extracted core (below). Roll/curse/stat-change/slot-decrement/level-increment logic is the
   existing machinery, unchanged (including the pre-existing `roll <= prob` comparator —
   deliberately inherited for parity with the normal path, per the no-math-changes non-goal).
   `whiteScroll=false`, `legendarySpirit=false` throughout (FR-4.2).
3. Consume both: `cpp.ConsumeItem(charId, TypeValueUse, txn, scrollSlot)` then
   `cpp.ConsumeItem(charId, TypeValueCash, txn, vegaSlot)`. Curse → `DestroyItem` on the equip's
   compartment (equip inventory for positive slots; equipped for negative — same call, type
   `TypeValueEquip`, as today).
4. Emit `EventTypeVegaScroll{Success, Cursed}` (instead of `PassScroll`/`FailScroll` — the
   normal SCROLL event would trigger the plain broadcast-only consumer and lose the VEGA dialog
   packets).

**Refactor:** extract the shared middle of `ConsumeScroll` (equip re-validation → data lookup →
roll → change-set assembly → `ChangeStat` → scroll consume → curse destroy) into an internal
helper parameterized by `successProb` override, equip resolver, extra consume steps, and result
emitter. `ConsumeScroll` keeps byte-identical behavior (its tests prove it); `ConsumeVegaScroll`
is the second caller. No exported-surface change.

Observability: debug logs on vega use (ids/slots), rate-gate rejection (expected vs actual
rate), boost application (existing "Rolled/Needed" line now shows 30/90), and every
cancellation path — consistent with existing scroll logging (NFR).

### 4.7 atlas-channel — status event consumer

In `kafka/consumer/consumable/consumer.go`:

- New handler for `Event[VegaScrollBody]` (`EventTypeVegaScroll`): to the user's session, in
  order — `VegaScrollWriter` start(outcome), `VegaScrollWriter` result(outcome) (back-to-back is
  safe: both clients latch the result and animate on their own clock — §2.2/§2.3); then
  map-broadcast `CharacterItemUpgradeWriter` via `NewItemUpgrade(charId, success, cursed,
  legendarySpirit=false, whiteScroll=false)` (existing writer; its enchant fields stay zero —
  the enchant variant is for a different feature); then enable-actions (empty
  `StatChangedWriter` announce, the pairing the error consumer already uses). Inventory-modify
  packets arrive through the existing compartment/asset event consumers, as with normal scrolls.
- Extend `handleErrorConsumableEvent`: `ErrorTypeVegaInvalid` → `VegaScrollWriter` invalid
  (0x42 — closes the dialog with the client's own "cannot be used" notice, §2.3) followed by
  the existing empty-StatChanged enable-actions. This resolves PRD open question 3: **send
  0x42**, because silence would wedge the dialog (client-verified).

Writer registration: `VegaScrollWriter` added to `produceWriters()` in `main.go`.

### 4.8 Seed templates + live config

Per version, add to `services/atlas-configurations/seed-data/templates/`:

- `writers[]`: `{"opCode": "<VEGA_SCROLL per §2.4>", "writer": "VegaScroll", "options":
  {"operations": {"START_SUCCESS": …, "START_FAILURE": …, "RESULT_SUCCESS": …,
  "RESULT_FAILURE": …, "INVALID": …}}}` — gms_83 (0x166, values §2.3), gms_95 (0x1AD, values
  §2.3), gms_87/jms_185 after IDA verification, gms_84 only after §2.5 is resolved, gms_92
  parked.
- `handlers[]`: `CharacterCashItemUseHandle` (+ `LoggedInValidator` — silently-dropped-handler
  gotcha) is already wired in gms_83/gms_84 only. **Task-126 is adding it for gms_87 (0x52),
  gms_95 (0x55), jms_185 (0x47).** Coordination: whichever task lands second rebases onto the
  identical wiring; task-130 carries the same three entries in its plan as a
  skip-if-already-present step so it does not depend on task-126's merge order.
- Rollout note for plan.md: live tenants do not re-seed — PATCH the live configs (writer +
  operations + handler where missing) and restart channels (handlers/writers don't hot-reload).

---

## 5. Data flow

### Success

```
client (CUIVega dialog) ──USE_CASH_ITEM + 6-int sub-body──► atlas-channel vega arm
  decode/guards ► REQUEST_VEGA_SCROLL ──► atlas-consumables
    validate (vega, scroll, rate==10/60, equip via dual-sign resolver, ValidateScrollUse)
    ► reserve CASH[vega] ─RESERVED─► reserve USE[scroll] ─RESERVED─► ConsumeVegaScroll
        re-validate ► roll @ 30/90 ► ChangeStat(equip) ► ConsumeItem(USE) ► ConsumeItem(CASH)
        ► emit VEGA_SCROLL{success, cursed}
             └─► atlas-channel: VEGA start(outcome) + result(outcome) → session
                              ItemUpgrade(legendary=false, white=false) → map broadcast
                              enable-actions → session
        (equip stat + inventory packets flow via existing equipable/compartment consumers)
```

### Failure paths

| Where | What happens | Player sees | Items |
|---|---|---|---|
| channel guards (bad markers / non-vega id / missing cash item) | warn log, no command | enable-actions (dialog closes via its own cancel) | untouched |
| service validation (missing vega/scroll/equip, rate mismatch, 0 slots) | no reservations made; `VEGA_INVALID` event | "This item cannot be used." + dialog closes + enable-actions | untouched |
| reservation stalls (item raced away; inventory emits nothing) | chain never fires; reservations TTL-expire (30s) | dialog until closed/cancelled; no packets | untouched (vega briefly locked) |
| `ConsumeVegaScroll` re-validation fails (race) | both reservations cancelled; `VEGA_INVALID` event | same as service validation | untouched |
| scroll fails the roll (normal outcome) | items consumed, slot decremented (unless curse rules say otherwise), curse possible | VEGA start+fail, map fail effect, inventory updates, enable-actions | vega+scroll consumed; equip per scroll rules |

---

## 6. Version support and verification plan

| version | serverbound layout | VEGA_SCROLL opcode | modes | disposition |
|---|---|---|---|---|
| gms_v83 | **verified** (§2.1) | **verified** 0x166 | **verified** (§2.3) | wire fully at implementation with fixtures |
| gms_v84 | presumed ≡ v83 (task-083 structural parity) — confirm | **suspect 0x166** (§2.5) | presumed ≡ v83 — confirm | wire only after v84 IDB verification; else escalate |
| gms_v87 | verify (IDB when loaded) | 0x17B csv — verify | verify | wire after verification; escalate if no IDB |
| gms_v92 | unknown | unknown (CSV 0x1A0) | unknown | **parked** (§2.6) |
| gms_v95 | **verified** (§2.1) | **verified** 0x1AD | arm-verified; success/fail popup pairing to pin (§2.3) | wire fully; pin pairing via string-pool 5417/5418 |
| jms_v185 | verify | 0x183 csv — verify | verify | wire after verification; escalate if no IDB |

Every wired version ships: writer + operations template entries, serverbound arm evidence under
the shared USE_CASH_ITEM audit, clientbound `VegaScroll` REPORT + byte fixtures with
`packet-audit:verify` markers, registry promotion, matrix regeneration.

---

## 7. Resolved PRD open questions

1. **Delivery mechanism** — new `REQUEST_VEGA_SCROLL` command (§3.1); existing
   `RequestScrollBody` producers untouched.
2. **Wire layouts/opcodes** — serverbound body pinned for v83+v95 during design (§2.1: 6 int32s,
   trailing updateTime everywhere); clientbound opcode+modes pinned for v83+v95 (§2.2–§2.4);
   v84/v87/jms at implementation against their IDBs, escalate if unavailable; v92 parked.
3. **FR-2.3 rejection surface** — `VEGA_SCROLL` INVALID (0x42) + enable-actions; required to
   unwedge the dialog, client-verified safe on v83 and v95 (§4.7).
4. **Equip addressing** — vega targets are equip-inventory items (positive slots), which the
   existing path cannot address; dual-sign resolver on the vega path only (§2.7); normal-path
   gap documented, not fixed here.

---

## 8. Testing

- **libs/atlas-packet**: `ItemUseVegaScroll` round-trip + exact-byte decode fixture from §2.1's
  IDA-derived layout (v83 now; per-version fixtures as IDBs are verified); `VegaScroll` writer
  round-trip + per-version exact-byte fixtures asserting each operations key resolves to §2.3's
  table; `packet-audit` matrix/fname-doc checks clean.
- **atlas-consumables**: table-driven `RequestVegaScroll` validation matrix (missing
  vega/scroll/equip; wrong compartment contents; rate mismatch 10-vs-60 both directions;
  0-upgrade-slot equip; positive- and negative-slot equips), chain orchestration (listener A
  triggers USE reserve; listener B triggers consume; itemId-keyed validators don't cross-fire),
  `ConsumeVegaScroll` (30/90 override visible in the roll threshold, whiteScroll/legendarySpirit
  false, both `ConsumeItem` calls, curse destroy, cancellation of both reservations on
  re-validation failure), and a regression test that `ConsumeScroll`'s behavior is unchanged by
  the core extraction. Builder-pattern setup only (no `*_testhelpers.go`).
- **atlas-channel**: vega arm decode + guard matrix (marker mismatch, non-vega id) + command
  emission; `VEGA_SCROLL` event consumer packet order (start → result → broadcast → enable);
  `VEGA_INVALID` error arm.
- **Verification bar** (PRD §8): `go test -race`, `go vet`, `go build` per changed module;
  `docker buildx bake` for atlas-channel + atlas-consumables **and all Go services** (both
  shared libs change: atlas-packet, atlas-constants); `tools/redis-key-guard.sh`; template
  `--check` tooling; code review before PR.

---

## 9. Deviations & known limitations

- **v92 parked** (§2.6) — the item stays inert there, as the whole cash-item-use path already is.
- **v84 wiring gated on opcode re-verification** (§2.5) — shipping 0x166 unverified risks a
  client crash via opcode collision; absence is the safer failure (item no-ops with the
  fall-through warn).
- **v95 success/fail popup pairing** (start 0x44 vs 0x49) is the one mode-table cell not fully
  pinned in design; it is a two-value swap locked by fixture at implementation.
- **Chained-reservation stall window**: if the second reservation is rejected inventory-side, no
  failure event exists (§2.9); the player sees no result until the dialog is closed and the vega
  unlock takes ≤30s. Accepted — identical envelope to the existing scroll flow's silent-failure
  mode; fixing it properly means reservation-failure events in atlas-inventory (out of scope).
- **Pre-existing findings handed to the owner**: the batched-reservation first-item bug that
  breaks white-scroll scrolling today (§2.8), and the normal scroll path's inability to address
  equip-inventory (positive-slot) targets (§2.7). Neither is fixed by this task per PRD
  non-goals; both are one-paragraph repros above if the owner wants them scheduled (or folded
  into this task's plan on request).
- The `flag` int32 (wire field 5) is read and ignored; if some version's client conditionally
  sets it ≠1 (e.g. an actual white-scroll toggle in later builds), the per-version IDA pass will
  surface it — the codec logs the value so live traffic would reveal a surprise cheaply.
