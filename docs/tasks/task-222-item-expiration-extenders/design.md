# Item-Expiration Extenders (Magical Sandglass) — Design

Task: task-222-item-expiration-extenders
Status: Draft for review
Created: 2026-08-13
Inputs: [`prd.md`](./prd.md)

---

## 0. Summary of what the client research changed

Every open question in PRD §9 is now answered from the clients. Three answers
change the requirements materially, and one requirement in the PRD is
contradicted by the client. Read this section before the rest.

| PRD open question | Answer | Impact |
|---|---|---|
| Q1 — `CashSlotItemType` for classification 550 | **61** on GMS < 95, **62** on GMS ≥ 95. Already implemented in Atlas as bare numeric literals. | Not a blocking unknown. Name the constants, add the version resolver. |
| Q2 — sub-body identical to `ItemUseSeal`? | **No.** It is a bare `int16` target slot — byte-identical to `ItemUseItemTag` (the client literally shares one jump-table arm between case 25 and case 61). | FR-3.1 changes: reuse `ItemUseItemTag`, not `ItemUseSeal`. FR-3.2(1) (inventory-type gate) has nothing to gate — no inventory type is on the wire. |
| Q3 — equipped (negative) slots accepted? | **Yes, and they are the *only* target.** The client's drop handler is `CDraggableItem::ModifyEquipItem`, which resolves the drop point to an equip position and negates it. | FR-3.3 resolves to "accept negative slots"; the compartment is always EQUIP. |
| Q4 — per-version presence | Present v72 → v95 and JMS v185. **Absent on v48 and v61** (no switch arm exists). | FR-6.1's "GMS v83+" hypothesis is wrong on the low end. |
| Q5 — reject locked targets? | **Yes** — the client rejects a sealed target before sending. | Confirms FR-3.2(5). |
| Q6 — 5500005 / 5500006 obtainable? | They can never be used: `addTime` (50 d / 99 d) always exceeds `maxDays` (30 d), and the client refuses to send in that case. | Consistent with their absent `String.wz` names. No special handling. |

**One new eligibility gate the PRD does not have:** the client reads
`info/notExtend` off the **target equip** and refuses with
"Time limit extension is not possible on this item" when it is non-zero.
Nothing in Atlas parses `notExtend` today.

**One PRD requirement contradicted by the client:** FR-3.4 says an over-cap
extension is *clamped and the item is still consumed*. The client never sends
that request at all — it shows a notice and returns. See §7 (Decision D1).

---

## 1. Evidence

All addresses are from the IDBs listed in `docs/packets/PROCESS.md`, read via
ida-pro-mcp on 2026-08-13. Function names in quotes are the symbols present in
the IDB, not guesses.

### 1.1 The classification → slot-item-type map

`get_cashslot_item_type(int nItemID)` — a pure `switch (nItemID / 10000)`.
Atlas's `GetCashSlotItemType`
(`services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:715`)
is a port of it. Case 550:

| Version | IDB | Function addr | `case 550` |
|---|---|---|---|
| GMS v72 | `GMS_v72.1_U_DEVM.exe` | `0x49fb33` | `61` |
| GMS v79 | `GMS_v79_1_DEVM.exe` | `0x47ec3e` | `61` |
| GMS v83 | `MapleStory_dump.exe` | `0x48645b` | `61` |
| GMS v87 | `GMSv87_4GB.exe` | `0x473d96` | `61` |
| GMS v95 | `GMS_v95.0_U_DEVM.exe` | `0x488c70` | `62` |
| JMS v185 | `MapleStory_dump_SCY.exe` | `0x49a1ee` | `39` |
| GMS v48 | `GMS_v48_1_DEVM.exe` | *(classifier unnamed)* | **no arm** |
| GMS v61 | `GMS_v61.1_U_DEVM.exe` | *(classifier unnamed)* | **no arm** |
| GMS v84 | `GMS_v84.1_U_DEVM.i64` | `SendConsumeCashItemUseRequest` @ `0xa54a2f` | `61` † |
| GMS v92 | `GMS_v92_1_DEVM.exe.i64` | `SendConsumeCashItemUseRequest` @ `0x9bfe10` | `61` † |

† v84/v92 are corroborated, not directly observed — see the paragraph below
the table. The v72–v87/v95/JMS rows above are a direct decompile of
`get_cashslot_item_type`'s `case 550` arm; v84/v92 are not.

v48 / v61 are settled without decompiling their classifier, from the dispatch
switch bound in `CWvsContext::SendConsumeCashItemUseRequest`:

- v48 `0x70e527`: `add eax, -12` / `cmp eax, 23h` → the switch covers types
  12–47. Type 61 is out of range.
- v61 `0x832aec`: `add eax, -12` / `cmp eax, 28h` → types 12–52. Out of range.

**GMS v84 and GMS v92 checked 2026-08-13 — a different, weaker evidentiary
tier than the rows above.** `get_cashslot_item_type` remains unnamed in both
IDBs, and unlike v48/v61 the `add eax, -12` / `cmp eax, <range>` bounds-check
idiom does not appear anywhere in either function (`insn_query` scanned both
in full for an `add`/`sub`/`lea` with a ±12 immediate: zero matches). What
was actually shown is (a) a case-61 arm exists and is reachable in
`SendConsumeCashItemUseRequest`'s own dispatch table (i.e. the family is
present, as expected inside the pre-95 band), and (b) that arm's instruction
shape — a single `Encode2` of the target-slot argument — is identical to
v83's confirmed case-61 arm (§1.2). That establishes reachability and
packet-shape consistency for value 61; it is **not** a direct observation
that the classifier maps item-id prefix 550 to 61 in these two versions,
because the classifier itself was never located. It is corroborating
evidence for the pre-existing assumption, at the tier the plan's Task 15
brief sanctioned as sufficient given the bounded risk (this confirms a
pre-existing mapping, not new code, and Task 14's resolver mirrors the
existing branch's condition exactly) — not a decompiled classifier result
like the rows above it.

(Session resolved via `idb_list` by filename, per project convention.)

- **v84** (`GMS_v84.1_U_DEVM.i64`, session `5881cf84`):
  `SendConsumeCashItemUseRequest` @ `0xa54a2f`. Dispatch jumps via
  `jmp ds:jpt_A54ADD[eax*4]` at `0xa54add`. The IDA-recognized case-61 arm is
  merged with case 25 — comment `jumptable 00A54ADD cases 25,61` — at
  `0xa56ecd`:
  ```
  0xa56ecd  push  [ebp+arg_8]      ; jumptable 00A54ADD cases 25,61
  0xa56ed0  lea   ecx, [ebp+var_38]
  0xa56ed3  call  COutPacket::Encode2
  0xa56ed8  jmp   loc_A54CE8       ; <send>
  ```
  Byte-for-byte the same shape as v83's `0xa0cae0` arm (§1.2): one
  `Encode2` of the `nEPOS` argument, nothing else. Shows the case-61 arm is
  reachable and shaped identically to v83's confirmed arm — treated as `61`
  on that basis.
- **v92** (`GMS_v92_1_DEVM.exe.i64`, session `acdfccff`):
  `SendConsumeCashItemUseRequest` @ `0x9bfe10`. Dispatch jumps via
  `jmp ds:jpt_9BFF39[eax*4]` at `0x9bff39`. Case 61 is a standalone arm here
  (not merged with 25) — comment `jumptable 009BFF39 case 61` — at
  `0x9bff40`:
  ```
  0x9bff40  mov   ecx, [esp+234h+arg_8]   ; jumptable 009BFF39 case 61
  0x9bff47  push  ecx
  0x9bff48  lea   ecx, [esp+238h+var_204]
  0x9bff4c  call  COutPacket::Encode2
  ```
  Same shape: one `Encode2` of the target-slot argument. Same basis: reachable
  arm, shape-matched to v83 — treated as `61`.

Both dispatch arms exist, are reachable, and are shape-identical to v83's
confirmed arm — consistent with `61` and with the pre-95 band every other
decompiled version in it agrees on. No stop condition triggered (a stop would
require finding the arm *absent* or shaped differently, which would mean
`GetCashSlotItemType`'s existing 550 branch needed a version arm — neither
happened). No version arm added to `GetCashSlotItemType`. The residual gap —
the classifier itself not being located — is accepted per the brief's
bounded-risk framing, not resolved.

**Value 61 is overloaded across the version boundary.** At GMS ≥ 95,
`61` is the *megaphone* arm (`case 507`, `otherCategory == 7` →
`get_cashslot_item_type` returns 61 at v95, 60 below). A bare
`CashSlotItemType(61)` comparison would therefore route v95 megaphones into
the sandglass arm. The version-resolving helper is not stylistic — it is
required for correctness. Same shape as `viciousHammerCashSlotItemType`
(`character_cash_item_use.go:708`) and the `sealTimed` switch at `:260`.

### 1.2 The sub-body

v83 `SendConsumeCashItemUseRequest` (`0xa0a63f`) header, then the switch:

```
COutPacket(0x4F)              ; opcode
Encode2(nPOS)                 ; source slot (the sandglass)
Encode4(nItemID)
switch (get_consume_cash_item_type(nItemID) - 12)
```

Jump-table `jpt_A0A6E6` @ `0xa0ead4`, entry `(61-12)=49` @ `0xa0eb98` →
**`0xA0CAE0`**, which IDA labels *"jumptable 00A0A6E6 cases 25,61"*:

```
0xa0cae0  push  [ebp+Unknown]      ; = nEPOS, the 3rd argument (target slot)
0xa0cae3  lea   ecx, [ebp+var_38]
0xa0cae6  call  COutPacket::Encode2
0xa0caeb  jmp   <send>
```

One `Encode2`. Nothing else. Entry 48 (case 60) resolves to `0xA0A6ED`
(IDA: "case 60") and entry 50 (case 62) to the default block — both consistent
with the table base, so index 49 is not an off-by-one.

Corroborated at v95, where the PDB gives the real signature:

```
CWvsContext::SendConsumeCashItemUseRequest(
    CWvsContext *this, int nPOS, int nItemID,
    unsigned __int16 nEPOS, ZXString<char> sDefaultValue)
```

`nEPOS` is a 16-bit equip position. No arm can encode it as anything but two
bytes.

This is exactly `libs/atlas-packet/cash/serverbound/item_use_item_tag.go`
(`int16 slot`, then the trailing `updateTime` when `!updateTimeFirst`). It is
**not** `item_use_seal.go` (`int32 inventoryType`, `int32 slot`).

### 1.3 The drop handler and its gates

`CDraggableItem::ModifyEquipItem` — v83 `0x4f4bb7`, JMS v185 `0x523074`. The
two decompile to the same structure. The classification-550 branch, in order:

```c
if ( TSecType<long>::GetData(a5, 0) / 10000 != 550 ) return 0;        // (a) classification
if ( CompareFileTime(target->dateExpire, PERMANENT) >= 0 ) return 0;  // (b) permanent target
if ( target->liCashItemSN ) return 0;                                 // (c) cash item
if ( target->vfunc1() ) return 0;                                     // (d) sealed / locked
info = lookup_sandglass_info(nItemID);                                // {addTime, maxDays}
if ( !info || !get_field() ) return 0;
if ( CItemInfo::IsNotExtendItem(target->nItemID) )                    // (e) info/notExtend
    { Notice(SP_5242 "Time Limit Extension is not possible on this item"); return 0; }

proposed = target->dateExpire + addTime * 10^7;                       // FILETIME 100ns units
if ( maxDays <= 0 || CompareFileTime(proposed, GetCorrectTime() + maxDays*86400*10^7) != 1 ) {
    ... YesNo(SP_4563 "When used on %s, the effective date will be extended by %s...")
    if answered yes: SendConsumeCashItemUseRequest(nPOS, nItemID, nEPOS, name);
} else {
    Notice(SP_4564 "You cannot extend the effective date beyond %d days");  // (f) NO PACKET SENT
    return 0;
}
```

Notes that matter:

- `addTime` is multiplied by `10^7`, i.e. **seconds**. `maxDays` is multiplied
  by `86400 * 10^7`, i.e. **days**. FR-1.1's units are confirmed.
- The cap is anchored to `GetCorrectTime()` — **now**, not the target's current
  expiration. FR-3.4's formula is confirmed on that point.
- Gate (d) is virtual method index 1 on `GW_ItemSlotBase`. The same call
  earlier in the function guards the string *"Sealed items cannot be sold,
  traded or dropped"*, so it is the sealed/locked predicate. Atlas's
  `asset.Model.Locked()` is the equivalent.
- Gate (e): `sub_5D586A` (v83 `0x5d586a`) calls
  `CItemInfo::GetItemInfo(nItemID)` and reads the property named by StringPool
  entry `SP_5109_NOTEXTEND` → the WZ node `info/notExtend`.
- Branch (f) is the contradiction with FR-3.4.
- The target is always fetched as `CharacterData::GetItem(charData, 1, a3)` —
  inventory type **1 (EQUIP)**, hard-coded — with `a3 = -hitTestResult`, i.e. a
  **negative** equipped position.

### 1.4 What is on the wire vs. what is derived

Atlas does not read the slot-item-type from the packet; it recomputes it from
`itemId` via `GetCashSlotItemType` (`character_cash_item_use.go:60`). So the
JMS divergence (39) does not affect dispatch — Atlas's classifier has no JMS
branch and will compute 61 for a JMS tenant, which the resolver will match.
What *must* agree with the client is the **sub-body layout**, and that is a
single `int16` on every version that has the arm.

---

## 2. Architecture

Five components, each with one job. Nothing new structurally — every seam
already exists and carries an analogous feature (Sealing Lock, Item Tag).

```
  client  ──CASH_ITEM_USE(0x4F/0x4D/0x4E…)──▶  atlas-channel
                                                 │  decode ItemUseItemTag → int16 nEPOS
                                                 │  GET /data/cash/{sandglassId}   → addTime, maxDays
                                                 │  GET character item in EQUIP slot nEPOS
                                                 │  GET /data/equipment/{targetId} → notExtend
                                                 │  eligibility gates + extension formula
                                                 ▼
                                    saga: ExpirationExtenderUse
                                      step 1  DestroyAsset (the sandglass)
                                      step 2  ExtendAssetExpiration (the target)
                                                 │
                       atlas-saga-orchestrator ───┤ COMMAND_TOPIC_COMPARTMENT
                                                 ▼
                                          atlas-inventory
                                            EXTEND_EXPIRATION {slot, expiration}
                                            re-validate cap, write expiration only
                                            emit asset UPDATED
                                                 │
                                                 ▼
                                    atlas-channel handleAssetUpdatedEvent
                                            INVENTORY_OPERATION add-entry → tooltip refresh
```

### 2.1 `libs/atlas-constants` — the classification

```go
ClassificationExpirationExtender = Classification(550)
```

Placed between `ClassificationRemoteStore` (547) and
`ClassificationViciousHammer` (557) in
`libs/atlas-constants/item/constants.go`, matching the file's ascending order.
`GetCashSlotItemType`'s existing bare `if category == 550` becomes
`if category == item.ClassificationExpirationExtender` (DOM-21).

### 2.2 `libs/atlas-packet` — no new codec

`ItemUseItemTag` is reused verbatim. The only change is its doc comment, which
currently claims exclusivity:

```go
// ItemUseItemTag is the type-25/61/62 sub-body of the cash ItemUse packet: a
// bare int16 target equip position (negative when equipped). The client shares
// one dispatch arm between the Item Tag (25) and Magical Sandglass (61 pre-v95,
// 62 at v95+) cases — gms_v83 SendConsumeCashItemUseRequest @0xA0CAE0, labelled
// "cases 25,61"; the v95 PDB types the argument as `unsigned __int16 nEPOS`.
```

The existing `item_use_item_tag_test.go` fixture already covers the layout;
one added case asserting a negative slot round-trips is the whole test delta.

**Rejected alternative:** a new `ItemUseExpirationExtender` struct with an
identical body. It would be a second source of truth for one wire layout, and
the guard we have against that kind of drift (`trade-contract-mirror-guard.sh`)
exists precisely because duplicated contracts silently diverge. The client
does not distinguish these two cases; neither should we.

### 2.3 `atlas-data` — two readers, three new fields

Two separate WZ readers are involved, which the PRD collapsed into one.

**Cash reader** (`services/atlas-data/atlas.com/data/cash/reader.go:78`,
alongside the existing `protectTime` parse):

```go
m.AddTime = uint32(i.GetIntegerWithDefault("addTime", 0))   // seconds
m.MaxDays = uint32(i.GetIntegerWithDefault("maxDays", 0))   // days
```

exposed on `cash.RestModel` (`data/cash/rest.go`) as `addTime,omitempty` /
`maxDays,omitempty`, mirrored on the channel consumer model
(`services/atlas-channel/atlas.com/channel/data/cash/rest.go`, which today
carries only `Id`, `StateChangeItem`, `BgmPath`, `ProtectTime`).

**Equipment reader** (`services/atlas-data/atlas.com/data/equipment/reader.go`,
in the `info.GetBool` block at `:112–115` next to `Cash`, `TimeLimited`,
`TradeBlock`):

```go
NotExtend: info.GetBool("notExtend", false),
```

exposed on `equipment.RestModel` and mirrored on
`services/atlas-channel/atlas.com/channel/data/equipment/`. Nothing in the
tree parses `notExtend` today (verified: zero hits repo-wide outside unrelated
NPC-conversation state names).

### 2.4 `atlas-channel` — the handler arm

Placed after the `CashSlotTypeSeal` block in
`character_cash_item_use.go` (the arms are ordered loosely by type value).

```go
CashSlotItemTypeExpirationExtender    = CashSlotItemType(61) // GMS < 95, JMS
CashSlotItemTypeExpirationExtenderV95 = CashSlotItemType(62) // GMS >= 95

// expirationExtenderCashSlotItemType returns the version-scoped
// CashSlotItemType for classification 550. Plain 61 also denotes the
// otherCategory==7 megaphone arm on GMS >= 95 (see GetCashSlotItemType's
// ClassificationMegaphones branch), so this check must remain version-scoped.
// IDA-verified: gms_v83 get_cashslot_item_type @0x48645B case 550 -> 61;
// gms_v95 @0x488C70 case 550 -> 62.
func expirationExtenderCashSlotItemType(t tenant.Model) CashSlotItemType {
	if t.Region() == "GMS" && t.MajorVersion() >= 95 {
		return CashSlotItemTypeExpirationExtenderV95
	}
	return CashSlotItemTypeExpirationExtender
}
```

Arm body:

```go
if it == expirationExtenderCashSlotItemType(t) {
	sp := cashsb.NewItemUseItemTag(updateTimeFirst)
	sp.Decode(l, ctx)(r, readerOptions)
	targetSlot := sp.Slot()                       // int16, negative when equipped
	// … gates (§3) … formula (§4) … saga (§5)
}
```

The compartment is `inventory.TypeValueEquip`, unconditionally — the client
hard-codes it and sends no inventory type.

### 2.5 `libs/atlas-saga` + `atlas-saga-orchestrator`

New saga type and action, registered in every table that
`ItemTagUse`/`SealingLockUse`/`IncubatorUse` appear in:

| Location | Addition |
|---|---|
| `libs/atlas-saga/model.go:40` | `ExpirationExtenderUse Type = "expiration_extender_use"` |
| `libs/atlas-saga/model.go:220` | `ExtendAssetExpiration Action = "extend_asset_expiration"` |
| `libs/atlas-saga/payloads.go` | `ExtendAssetExpirationPayload` — same shape as `ApplyAssetLockPayload:1097` (`CharacterId`, `InventoryType`, `Slot int16`, `Expiration time.Time`) |
| `libs/atlas-saga/unmarshal.go:582` | decode case for the new action |
| `.../saga/model.go:42-44` | re-export alias |
| `.../saga/timer.go:174-176, 205, 237` | all three lists |
| `.../saga/compensator.go:267` | the cash-item-use compensation branch |
| `.../saga/handler.go` | step handler → emit `EXTEND_EXPIRATION` |

A saga type absent from `timer.go` gets no timeout and no compensation — a
silently stuck saga. This is the single highest-risk omission in the change and
the plan should make each of these an individually checkable item.

### 2.6 `atlas-inventory` — expiration-only mutation

New command in `kafka/message/compartment/kafka.go` beside
`CommandApplyLock` (`:35`, body at `:186`):

```go
CommandExtendExpiration = "EXTEND_EXPIRATION"

type ExtendExpirationCommandBody struct {
	Slot       int16     `json:"slot"`
	Expiration time.Time `json:"expiration"`
}
```

Consumer arm in `kafka/consumer/compartment/consumer.go`; compartment method
`ExtendAssetExpiration` / `ExtendAssetExpirationAndEmit` shaped exactly like
`ApplyAssetLock` (`compartment/processor.go:1045-1075`) — same lock registry,
same `GetBySlot`, same transaction/emit wrapper.

The asset-level method is **new**, not `asset.ApplyLock`:

```go
// ExtendAssetExpiration sets the expiration on an asset in place without
// touching its flags, emitting the existing UPDATED status event. It is the
// mirror of ApplyLock for genuinely time-limited (unlocked) items:
// ApplyLock unconditionally adds FlagLock and rejects exactly the asset shape
// this feature targets (asset/processor.go:332 — "asset has a non-lock
// expiration").
func (p *ProcessorImpl) ExtendExpiration(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) func(a Model, expiration time.Time) error
```

It calls `updateFlagAndExpiration(db, a.Id(), a.Flag(), expiration)` — passing
the asset's **existing** flag through unchanged — and emits
`UpdatedEventStatusProvider`, the same event the sealing-lock path emits.
Reusing the existing administrator function rather than adding an
expiration-only sibling keeps one write path; the invariant ("flag unchanged")
is then a unit-test assertion rather than a structural guarantee, which is the
deliberate trade: one write path, one test.

Server-side re-validation (PRD FR-4.3) lives here: the command carries an
absolute timestamp computed by the channel, and atlas-inventory clamps it to
`now + maxDays` re-derived from `atlas-data` for the sandglass template id. See
§7 D2 for why the sandglass id must therefore be on the command body.

### 2.7 Client feedback

Nothing new. `atlas-inventory`'s asset `UPDATED` event carries
`AssetData.Expiration` (`kafka/message/asset/kafka.go:32,69-71`); atlas-channel's
`handleAssetUpdatedEvent` (`kafka/consumer/asset/consumer.go:269-296`) rebuilds
the asset and announces an `INVENTORY_OPERATION` add-entry that rewrites the
slot record. The sandglass's removal rides the existing `DestroyAsset` event
path. No `EnableActions` — matching every other inventory-mutating,
non-warping cash arm.

---

## 3. Eligibility gates

Server-side, in this order, each logging at warn with character id, sandglass
item id, target slot, and the reason (PRD FR-5.3). Each maps to a client gate
so a legitimate client can never trip one; they exist because the server does
not trust the client.

| # | Gate | Client counterpart | Rejects |
|---|---|---|---|
| G1 | `GetItemInSlot(EQUIP, targetSlot)` errors | implicit — no item, no drop | empty slot |
| G2 | `target.Expiration().IsZero()` | (b) `CompareFileTime(dateExpire, PERMANENT) >= 0` | permanent equip |
| G3 | `target.CashId() != 0` | (c) `liCashItemSN` | cash equip |
| G4 | `target.Locked()` | (d) `vfunc1()` | sealed / lock-window item |
| G5 | equipment data `notExtend` | (e) `IsNotExtendItem` | WZ-blacklisted equip |
| G6 | `cd.MaxDays == 0` | `maxDays <= 0` short-circuit | mis-authored sandglass |
| G7 | `proposed > cap` | (f) `SP_4564` notice | over-cap extension (see D1) |

`asset.Model.CashId()` is at
`services/atlas-channel/atlas.com/channel/asset/model.go:100`; the
inventory-side model states the same predicate as `IsCashEquipment()`
(`services/atlas-inventory/atlas.com/inventory/asset/model.go:106-108`).

G2 and G4 together are what keeps sandglass semantics and Sealing Lock
semantics from being conflated: a locked item's expiration is a *protect
window*, and the two are already kept apart at `asset/processor.go:329-332`.

**Dropped from the PRD:** FR-3.2(1) — "reject when `inventoryType != Equip`".
There is no inventory type on the wire to check.

---

## 4. Extension formula

```
cap      = now + (maxDays * 24h)
proposed = target.Expiration() + (addTime seconds)
new      = proposed                        // only reachable when proposed <= cap
```

Matching the client bit for bit (§1.3): the cap anchors to **now**, not to the
target's current expiration, and `proposed > cap` is a rejection rather than a
clamp.

Table-driven test cases: under cap; exactly at cap (`proposed == cap` →
accept, the client's `!= 1` comparison admits equality); over cap → reject;
`maxDays == 0` → reject; target already past cap → reject via the same
comparison; 99-day sandglass against a 30-day ceiling → reject on every
possible target.

---

## 5. Saga

Two steps, mirroring `SealingLockUse` (`character_cash_item_use.go:296-330`):

```go
saga.Saga{
	TransactionId: transactionId,
	SagaType:      saga.ExpirationExtenderUse,
	InitiatedBy:   "CASH_ITEM_USE",
	Steps: []saga.Step{
		{StepId: "consume_expiration_extender", Action: saga.DestroyAsset,
		 Payload: saga.DestroyAssetPayload{CharacterId, TemplateId: uint32(itemId), Quantity: 1}},
		{StepId: "extend_asset_expiration", Action: saga.ExtendAssetExpiration,
		 Payload: saga.ExtendAssetExpirationPayload{CharacterId, InventoryType: byte(inventory.TypeValueEquip),
		                                            Slot: targetSlot, Expiration: newExpiration,
		                                            ExtenderTemplateId: uint32(itemId)}},
	},
}
```

Consume first, extend second — the same order as the sealing lock, so a failed
extension compensates the sandglass back via the existing cash-item-use
compensation branch (`compensator.go:267`).

**Idempotency.** The extension is a set-to-absolute-value, never an increment,
so a redelivered `EXTEND_EXPIRATION` command writes the same timestamp twice.
Combined with the saga's step-completion guard this makes double delivery a
no-op rather than a double extension. This is the reason the command body
carries an absolute `expiration` and not a duration — a duration-shaped
command would stack under the at-least-once delivery this cluster has
(see the redelivery-dupes pattern in the known-bugs notes).

---

## 6. Version scope

| Version | Family present | Slot-item-type | Action |
|---|---|---|---|
| gms_v48 | no | — | none; arm unreachable |
| gms_v61 | no | — | none; arm unreachable |
| gms_v72 | yes | 61 | covered by the pre-95 resolver branch |
| gms_v79 | yes | 61 | covered |
| gms_v83 | yes | 61 | covered |
| gms_v84 | yes | 61 (dispatch-arm corroborated 2026-08-13 †, §1.1) | covered by the pre-95 resolver branch |
| gms_v87 | yes | 61 | covered |
| gms_v92 | yes | 61 (dispatch-arm corroborated 2026-08-13 †, §1.1) | covered by the pre-95 resolver branch |
| gms_v95 | yes | 62 | covered by the ≥95 resolver branch |
| jms_v185 | yes | 39 on the wire; Atlas computes 61 | covered — Atlas derives the type from the item id, it does not read it (§1.4) |

† Not a direct classifier decompile like the other rows. §1.1 located a
reachable case-61 arm in `SendConsumeCashItemUseRequest`'s dispatch table,
shape-matched to v83's confirmed arm, but `get_cashslot_item_type` itself was
never located in either IDB — see §1.1 for the full evidentiary-tier
distinction.

No socket-config template edits. `CASH_ITEM_USE` is an existing registered
handler and this task adds no opcode — the arm lives inside the existing
dispatcher. Confirmed 2026-08-13 via
`grep -l "CharacterCashItemUseHandle" services/atlas-configurations/seed-data/templates/*.json`
— all ten in-scope templates (`gms_48_1`, `gms_61_1`, `gms_72_1`, `gms_79_1`,
`gms_83_1`, `gms_84_1`, `gms_87_1`, `gms_92_1`, `gms_95_1`, `jms_185_1`)
already carry the `CharacterCashItemUseHandle` binding — including v48/v61,
which have the shared `CASH_ITEM_USE` opcode registered even though their
client has no case-61 arm for this specific classification. No template
edit is needed and no gap was found.

Per-version `addTime` / `maxDays` values are **data**, read at runtime from
`atlas-data` per tenant. No code branches on them, so no per-version value
verification gates this change. The PRD's §4 table (v83 local extract) is
retained as the reader unit-test fixture.

---

## 7. Decisions needing your sign-off

### D1 — Over-cap use: reject, or clamp-and-consume?

PRD FR-3.4 says clamp and consume. **The client says reject.** When
`proposed > cap`, `ModifyEquipItem` shows *"You cannot extend the effective
date beyond %d days"* and returns without sending anything (§1.3, branch (f)),
on both v83 and JMS v185.

**Recommendation: reject (G7).** Reasons, in order:

1. It matches the client. Clamping is unreachable through a legitimate client,
   so the clamp branch would be dead code that only ever runs for a forged
   packet — and for a forged packet, rejecting is the safer response anyway.
2. It removes the player-visible loss the PRD itself worries about in
   FR-3.2(6). Under clamp-and-consume, a 20-day sandglass on an item with 25
   days left burns the item for 5 days of value with no warning; the client
   would have shown a refusal.
3. It makes 5500005 / 5500006 coherent: they simply never apply, exactly as
   the client behaves.

If you want clamp-and-consume anyway, say so and the design becomes
`new = min(proposed, cap)` with G7 narrowed to "reject only when
`cap <= E`" — everything else is unchanged.

### D2 — Where the server-side cap re-validation gets `maxDays`

FR-4.3 requires atlas-inventory to re-validate rather than trust the channel's
absolute timestamp. To re-derive `cap` it needs `maxDays`, which lives in the
**sandglass's** cash data — but the extension step's payload is about the
**target**. Options:

- **(a) Carry `ExtenderTemplateId` on the payload and command body**
  (shown in §5), and have atlas-inventory `GET /data/cash/{id}` for `maxDays`.
  One extra service call on a cold path. **Recommended** — the re-validation is
  genuinely independent, which is the point of a trust boundary.
- (b) Re-validate only the weaker invariant atlas-inventory can check without
  extra data: `expiration > now` and `expiration > current expiration`. Cheaper,
  but it does not bound the extension, so a forged command could still set an
  expiration years out.
- (c) Skip re-validation. Rejected — FR-4.3 is explicit and the channel is not
  a trust boundary.

### D3 — Naming the reused codec

Reusing `ItemUseItemTag` for the sandglass arm means the type name no longer
describes all of its uses. Options: keep the name and fix the doc comment
(**recommended** — one wire layout, one type, zero churn); or rename it to
something layout-descriptive like `ItemUseTargetSlot` and update both call
sites. The rename is honest but touches a verified codec and its fixtures for
no behavioural gain.

---

## 8. Testing

| Area | Test |
|---|---|
| `atlas-data` cash reader | table-driven over the five v83 ids from PRD §4 (`addTime` 86400 / 604800 / 1728000 / 4320000 / 8553600, `maxDays` 30) plus the absent-field default of 0 |
| `atlas-data` equipment reader | `notExtend` present-true, present-false, absent-defaults-false |
| Extension formula | under cap, exactly at cap, over cap, `maxDays==0`, already past cap, 99-day-vs-30-day |
| Each gate G1–G7 | one case per gate asserting no saga is created |
| `ItemUseItemTag` codec | added case: negative slot round-trips (equipped target) |
| `atlas-inventory` | `ExtendExpiration` leaves `FlagLock` and every other flag bit unchanged; `UPDATED` carries the new expiration |
| `atlas-inventory` | forged over-cap `EXTEND_EXPIRATION` is rejected/clamped server-side (D2) |
| Saga registration | the new type appears in the timer list, the timeout switch, and the compensator branch |
| Idempotency | replaying `EXTEND_EXPIRATION` writes the same absolute value, not a second extension |

Live verification (not inferrable, per the acceptance criteria): drag a
sandglass onto a time-limited equipped item on a v83 tenant and confirm the
tooltip shows the new expiration without a relog.

Build gates per CLAUDE.md: `go test -race ./...` and `go vet ./...` in
atlas-data, atlas-channel, atlas-inventory, atlas-saga-orchestrator,
libs/atlas-constants, libs/atlas-packet, libs/atlas-saga; `docker buildx bake`
for every service whose `go.mod` moves; `tools/lint.sh --check`,
`redis-key-guard.sh`, `goroutine-guard.sh`, `skill-job-id-guard.sh`,
`buff-duration-guard.sh` from the repo root.

---

## 9. Risks

| Risk | Mitigation |
|---|---|
| A bare `61` literal survives somewhere and swallows v95 megaphones | the resolver is the only place the value appears; the arm compares against `expirationExtenderCashSlotItemType(t)` |
| The new saga type is missed in one of `timer.go`'s three lists → silently stuck saga, no timeout, no compensation | each list is its own plan task with its own file:line |
| `asset.ApplyLock` gets reused "because it's close" and stamps `FlagLock` on every extended item | the new method's doc comment names the trap; the flag-preservation unit test fails loudly |
| v84 / v92 turn out not to map 550 → 61 | confirmed by the same jump-table method as v48/v61 before the branch lands; both sit inside a band where four binaries agree |
| `notExtend` is absent from some version's WZ | `GetBool(..., false)` defaults to permissive, matching the client's `!int32 → return 0` |
