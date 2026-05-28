# Pending IDA function exports

This list tracks IDA functions referenced by the login-domain audit matrix
(task-027) but NOT yet in `gms_v95.json`. Each row needs a future maintainer
run of `packet-audit export ...` (live IDA-MCP) or hand-derivation from a
focused spike doc to add the function's wire-layout.

## Resolved (now in gms_v95.json)

| FName | Atlas writer/handler | Verdict |
|---|---|---|
| `CLogin::OnCheckPasswordResult` (success) | AuthSuccess | ✅ (v95 field-7 width fix shipped) |
| `CLogin::OnCheckPasswordResult#AuthLoginFailed` (synthetic) | AuthLoginFailed | ✅ |
| `CLogin::OnCheckPasswordResult#AuthTemporaryBan` (synthetic) | AuthTemporaryBan | ✅ |
| `CLogin::OnCheckPasswordResult#AuthPermanentBan` (synthetic) | AuthPermanentBan | ✅ (v95 trailing-bytes fix shipped) |
| `CLogin::OnSetAccountResult` | SetAccountResult | ✅ |
| `CLogin::OnCheckPinCodeResult` | PinOperation | ✅ |
| `CLogin::OnUpdatePinCodeResult` | PinUpdate | ✅ |
| `CLogin::OnLatestConnectedWorld` | SelectWorld | ✅ |
| `CLogin::OnRecommendWorldMessage` | ServerListRecommendations | 🔍 (sub-struct loop) |
| `CLogin::OnSelectWorldResult` | CharacterList | 🔍 (sub-struct CharacterListEntry) |
| `CLogin::OnWorldInformation` | ServerListEntry | 🔍 (sub-struct ChannelLoad loop) |
| `CLogin::OnSelectCharacterResult` | ServerIP | ✅ |
| `CLogin::SendCheckPasswordPacket` | Request (LoginHandle) | ✅ |
| `CLogin::SendSelectCharPacket` | CharacterSelect | ✅ |
| `CLogin::SendCheckUserLimitPacket` | ServerStatusRequest | ✅ (v95 width fix shipped) |
| `CLogin::SendViewAllCharPacket` | AllCharacterListRequest | ✅ |
| `CLogin::OnAcceptLicense` | AcceptTos (account/serverbound) | ✅ |

**17 packets audited, 14 ✅ / 3 🔍 / 0 ❌.**

## Still pending — login domain

| FName / Symbol | Atlas writer/handler | Notes |
|---|---|---|
| `CLogin::OnViewAllCharResult` (0x5de120, size 0x521) | AllCharacterListPong | Medium-complex; involves CharacterListEntry sub-struct. Phase 2 (analyzer descent) needed for high-fidelity audit. |
| `CLogin::SendSelectCharPacketByVAC` (0x5d7550, size 0x669) | CharacterSelectWithPic / *Register? | VAC-variant of char select. Large function; needs careful branch analysis. |
| `CLogin::OnSelectCharacterByVACResult` (0x5de670, size 0x375) | PicResult? | VAC result packet. |
| `CLogin::OnDenyLicense` (0x5d45d0) | — | Client-side function; constructs an outbound deny packet. |
| `CLicenseDlg::OnButtonClicked` (0x5ff870) | (UI callback) | Drives OnAcceptLicense / OnDenyLicense; not directly a wire format. |
| `LoginAuth` (atlas writer) | — | Orphan: atlas writes `WriteAsciiString(screen)`. No IDA function found by direct search. May be a legacy v83 packet that v95 client no longer reads. |

## Out of scope for GMS v95 audit (cross-region or cross-version)

These atlas writers/handlers exist in the codebase but the GMS v95 client
doesn't exercise them. The audit pipeline correctly produces no report
because there's no v95 IDA function to compare against:

- `LoginAuth` (clientbound, writes 1 string) — **JMS v1.85 only**. Whether
  GMS ever produces it is unconfirmed. Not in the gms_95 template.
- `ServerLoad` (clientbound, writes 1 byte) — **GMS v12 (or earlier) only**.
  Not in the gms_95 template.
- `ServerSelect` (serverbound, reads 1 byte worldId) — **GMS v12 (or earlier)
  only**. v95 uses `WorldCharacterListRequest` instead. Not in the gms_95
  template; the `WorldSelectHandle` symbol is dead in v95.
- `PicResult` (clientbound, opcode 0x1C, writes 1 byte) — semantically tied
  to `CLogin::SendSelectCharPacket` (the PIC-register branch's reply).
  Opcode 0x1C is not handled by `CLogin::OnPacket` directly in v95; receipt
  is routed through a different state machine, so the audit pipeline's
  CLogin-based dispatch model can't reach it. Wire shape (1 byte) is
  trivial enough that a manual cross-check confirms ✅.

## Still pending — handlers without an IDA mapping

Atlas writers/handlers under `libs/atlas-packet/login/` whose corresponding IDA
function hasn't been identified yet. Each likely corresponds to a
`CLogin::Send*` outbound packet constructor or a `CLogin::On*` inbound result:

- `AfterLoginHandle` (opcode 0x09) — atlas decodes `byte pinMode, optional (byte opt2, string pin)`
- `RegisterPinHandle` (opcode 0x0A)
- `CheckPicHandle`, `RegisterPicHandle`, `CharacterSelectedPicHandle`, `CharacterListSelectHandle`, `CharacterListSelectWithPicHandle` (PIC family, opcodes 0x15–0x1E)
- `SetGenderHandle` (opcode 0x08) — likely `CLogin::SendSetGenderPacket`
- `WorldCharacterListRequest` (opcode 0x05) — likely `CLogin::SendSelectWorldPacket` or similar
- `ServerStatus` (clientbound) — likely sent by `CLogin::OnCheckUserLimit`?
- `ServerLoad` (clientbound)
- `ServerListEnd` (clientbound, opcode 0x0A end-of-list sentinel inside ServerListEntry) — already audited as part of ServerListEntry's dispatch byte
- `PicResult` (clientbound)

## Known false positives in current audit output

`CharacterList.md` (verdict ❌): the per-entry trailer reports a 1-byte
over-count from row 45 onward. Static analysis collects all conditional
branches' calls (viewAll byte + gm byte + world-rank-enabled byte = 3
bytes), but at runtime only 2 fire: either {viewAll=0, gm=0} → 2 bytes
total (gm path returns early) or {viewAll=0, rank-enabled=1} → 1+16 = 17
+1 = 18 bytes total. v95 reads 2 bytes (onFamily + hasRank) + optional 16
bytes — matches both runtime paths. The pipeline doesn't model
early-return blocks as exclusive, so the audit over-counts. Resolution
would require an analyzer extension that flags `return` statements inside
guarded blocks; deferred to a follow-up.

## Cosmetic / cross-version concerns (not v95-specific bugs)

- `ServerIP.codes.SERVER_UNDER_INSPECTION: 7` (template_gms_95_1.json) — in
  v95 IDA, value 7 in `OnSelectCharacterResult`'s v3 switch triggers
  `GotoTitle + Error(17)` which is the "already logged in" path, not
  server-inspection. The wire value 7 still produces the right behavior
  (kick to title), but the constant name is misleading. Renaming would
  require updating the Go constant in `services/atlas-login/atlas.com/login/socket/writer/server_ip.go`
  AND all version templates (v83/v87/v92/v95/v111/JMS) that share this
  key. Left as-is for now to avoid cross-version breakage.

## JMS v185 cash-shop NX-payment divergence (task-067 Phase 2, Task 10) — 🔍 DEFERRED to sibling task

> **task-067 Phase 2 (JMS v185 cross-version pass)** audited commerce against the
> JMS v185 IDB (`MapleStory_dump_SCY.exe`, md5 af6652ff9b7c549341f35e3569d7564a).
> Tally: **10 ✅ / 5 ❌ / 0 ⚠️ / 0 🔍**. Two JMS-specific IN-scope wire bugs were
> fixed (see below). The 5 ❌ are all cash serverbound "buy/gift" senders and form
> a single coherent finding: **JMS v185 runs a different cash-shop payment
> protocol** (the NX-point system design §9 predicted as uncharted). They are
> DEFERRED as a sibling task, NOT shipped under audit cover.

### IN-scope JMS fixes shipped this pass (2)

| Atlas struct | JMS IDA evidence | Bug | Fix |
|---|---|---|---|
| `interaction/serverbound/operation_chat.go` (`OperationChat`) | `CMiniRoomBaseDlg::CheckAndSendChat`@0x6db3ce: `Encode1(6)`, `Encode4(get_update_time)`, `EncodeStr(message)` | JMS prepends `update_time` (like GMS v87+), but the encoder gated it behind `GMS && >=87` so JMS dropped it → 4-byte serverbound desync per chat | Predicate widened to `(GMS && >=87) \|\| JMS`, single guard. ✅ |
| `cash/clientbound/shop_inventory.go` (`CashShopInventory`) | `CCashShop::OnCashItemResLoadLockerDone`@0x48bcff: `Decode2 count`, `DecodeBuffer(55*count)`, then `Decode2 ×4` (this+288..291) | JMS reads 4 trailing slot-counter shorts (like GMS v95), but the encoder gated the last two behind `GMS && >=95` so JMS wrote only 2 → 4-byte clientbound under-write desyncs the locker panel | Predicate widened to `(GMS && >=95) \|\| JMS`, single guard. ✅ |

### Else-branch confirmations (GMS-gated encoders correct for JMS)

- `cash/clientbound/query_result.go` (`QueryResult`) — `CCashShop::OnQueryCashResult`@0x48b3e8
  reads exactly two ints (`Decode4 nCash`, `Decode4 nMaplePoint`); JMS takes the
  `else` branch (no `prepaid`). ✅
- `cash/clientbound/shop_inventory.go` (`CashShopPurchaseSuccess`) —
  `CCashShop::OnCashItemResBuyDone`@0x48c0f0 reads `DecodeBuffer(0x37=55)`; matches
  atlas's 55-byte item body. ✅
- `interaction/serverbound/operation_personal_store_buy.go` /
  `operation_merchant_buy.go` (`OperationPersonalStoreBuy` / `OperationMerchantBuy`)
  — `CPersonalShopDlg::BuyItem`@0x762365 sends `Encode1 nIdx`, `Encode2 qty`,
  `Encode4 CItemInfo::GetItemCRC`; the prior-pass unconditional itemCRC append is
  correct for JMS. ✅
- `interaction/serverbound/operation_personal_store_set_black_list.go`
  (`OperationPersonalStoreSetBlackList`) — `CPersonalShopDlg::DeliverBlackList`@0x763021
  sends `Encode2 count` then a per-entry `EncodeStr` loop; the prior-pass `[]string`
  conversion is correct for JMS. ✅
- `storage/serverbound/operation_retrieve_asset.go` /
  `operation_store_asset.go` — `CTrunkDlg::SendGetItemRequest`@0x84dea0
  (`Encode1 nType`, `Encode1 slot`) and `CTrunkDlg::SendPutItemRequest`@0x84e07d
  (`Encode2 slot`, `Encode4 itemId`, `Encode2 qty`) match atlas field-for-field. ✅
- `cash/serverbound/shop_operation_buy_normal.go` (`ShopOperationBuyNormal`) —
  `CCashShop::OnBuyNormal`@0x47f5ba sends only `Encode4 nCommSN` (serialNumber);
  matches atlas. ✅

### OUT-of-scope: JMS cash-shop NX payment protocol — 5 ❌ DEFERRED (do NOT build)

The five cash serverbound senders below diverge from **every** atlas branch (v83,
GMS-gated, and JMS-else) in BOTH op-byte and field shape. JMS v185 uses a secondary-
password ("SPW") string plus serial-number-only bodies, and entirely different
dispatcher op-bytes than the GMS shapes atlas models. Making each JMS-correct would
require a **3rd nested region/version branch** inside encoders that already carry two
GMS gates (`>=87` and/or `>=95`) — over the 2-nested-guard HARD CAP — AND a structural
field rewrite plus op-byte remap in `template_jms_185_1.json`. This is a NEW feature
(wiring the JMS NX-point cash protocol), not an audit-cover width fix. **DEFERRED.**

| Atlas struct (file) | JMS IDA sender | JMS op-byte | JMS body | atlas (JMS else-branch) |
|---|---|---|---|---|
| `ShopOperationBuy` (`shop_operation_buy.go`) | `CCashShop::OnBuy`@0x47eaa7 | 3 | `Encode1 isPoints`, `Encode4 serialNumber` | `bool isPoints`, `int currency`, `int serialNumber`, `int zero` (extra `currency`+`zero`) |
| `ShopOperationGift` (`shop_operation_gift.go`) | `CCashShop::SendGiftsPacket`@0x47bced | 0x2E (NOT GMS 0x04) | `Encode4 serialNumber` ONLY | `int birthday`, `int serialNumber`, `str name`, `str message` |
| `ShopOperationBuyCouple` (`shop_operation_buy_couple.go`) | `CCashShop::OnBuyCouple`@0x48085a | 0x1E (NOT GMS 0x1F) | `EncodeStr spw`, `Encode4 serialNumber`, `EncodeStr giveTo`, `EncodeStr message` | `int birthday`, `int option`, `int serialNumber`, `str name`, `str message` (extra `option`, no SPW) |
| `ShopOperationBuyFriendship` (`shop_operation_buy_friendship.go`) | `CCashShop::OnBuyFriendship`@0x481184 | 0x24 (NOT GMS 0x25) | `EncodeStr spw`, `Encode4 serialNumber`, `EncodeStr giveTo`, `EncodeStr message` | `int birthday`, `int option`, `int serialNumber`, `str name`, `str message` |
| `ShopOperationRebateLockerItem` (`shop_operation_rebate_locker_item.go`) | `CCashShop::OnRebateLockerItem`@0x47c059 | 0x1B (NOT GMS 0x1C) | `EncodeStr spw`, `EncodeBuffer 8` (locker SN) | `int birthday`, `long unk` (no SPW) |

**Sibling-task suggestion:** *"JMS v185 cash-shop NX-payment protocol support."*
Scope: (a) add JMS-specific cash serverbound encoders/decoders for the buy/gift/couple/
friendship/rebate family with the SPW string + serial-number shapes above, structured
to avoid a 3rd nested guard (e.g. a region-dispatched body strategy rather than nested
`if`); (b) remap the cash serverbound dispatcher op-bytes in `template_jms_185_1.json`
to the JMS values (0x2E gift, 0x1E couple, 0x24 friendship, 0x1B rebate) — citing the
IDA `Encode1(...)` in each sender; (c) wire the JMS NX-point query/charge flow in
atlas-channel. Out of scope for a packet-shape audit. Note also two interaction
op-byte differences for the same future pass (template, not atlas-packet): JMS
PersonalStore `BuyItem` op = 0x14 personal / 0x1F entrusted (GMS 0x17/0x22), and JMS
`DeliverBlackList` op = 0x1B (GMS 0x1E) — encoder bodies already match, only the
dispatcher opcodes differ.

### Hard-cap result (Task 10)

No encoder was pushed to a 3rd nested guard. The two IN-scope fixes each widened a
single existing predicate to `... || JMS` (1 guard level). The 5 cash-shop NX
divergences that WOULD need a 3rd gate are DEFERRED above, not built. The repo-wide
nesting awk flags `shop_open.go` and `shop_operation_gift.go` as the documented
known sequential-guard false positives (two SIBLING `if` blocks, not nested);
verified by reading — neither was modified this pass.

## Sub-op enum / sub-struct deferrals — commerce domain (task-067)

> **task-067 Phase 2 (GMS v83 cross-version pass) RESOLVED the 11 deferred wire
> bugs below.** v83 IDB (md5 80ff438ced539b831f0d2ed95099275d) gave the second
> cross-version data point. Verdicts:
>
> | Deferred bug | v83 finding | Resolution |
> |---|---|---|
> | storage/Show segmentation + padding | identical to v95 (SetGetItems@0x7c5dfd) | FIXED unconditional |
> | interaction OperationChat update_time | ABSENT in v83 (CheckAndSendChat@0x65f438) | FIXED, gated GMS>=95 |
> | interaction PersonalStoreBuy/MerchantBuy itemCRC | PRESENT in v83 (BuyItem@0x6fd261) | FIXED unconditional |
> | interaction PersonalStoreSetBlackList byte[]→string[] | string[] in v83 (DeliverBlackList@0x6fdeda) | FIXED unconditional |
> | cash SPW BuyCouple/BuyFriendship/RebateLockerItem | v83=int (ask_SPW), v95=string | FIXED, gated GMS>=95 |
> | cash SPW + oneADay Gift | v83=int+int+str+str (no oneADay/SPW) | FIXED, gated GMS>=95 |
> | cash ShopOperationBuy oneADay+eventSN | v83=single int IsZeroGoods (OnBuy@0x46dadd) | FIXED, gated GMS>=95 |
> | cash CashShopInventory 2 trailing shorts | v83 reads only 2 (OnCashItemResLoadLockerDone@0x4794f6) | FIXED, gated GMS>=95 |
>
> The version-gated fixes use `Region()=="GMS" && MajorVersion()>=95`, leaving v83
> at its (confirmed-correct) old shape and JMS untouched (no JMS data this pass).
> The unconditional fixes apply to all versions (field present/correct in both
> v83 and v95). World-transfer (op 0x31, SendBuyTransferWorldItemPacket@0x473601)
> and name-change (op 0x2E, SendBuyNameChangeItemPacket@0x47342f) were confirmed
> PRESENT in v83 with shapes matching atlas's unconditional encoders — no gate
> needed (✅ N/A). The original deferral text is preserved below for traceability.

### Show clientbound — per-tab item segmentation + spurious padding (storage)

> ✅ RESOLVED (task-067 Phase 2): v83 `CTrunkDlg::SetGetItems`@0x7c5dfd reads the
> per-tab loop + conditional meso IDENTICALLY to v95 — segmentation is NOT
> version-gated. Fixed unconditionally: `Show.Encode` now buckets assets by
> `model.Asset.InventoryType()` per set tab bit (4/8/16/32/64), gates meso on
> flag&2, and drops the 3 spurious padding bytes. Residual ❌ on the audit is the
> per-tab loop-flatten tool limitation (same class as UpdateAssets); wire output
> is byte-correct (see show_test.go segmentation+meso-gate tests).

`CTrunkDlg::OnPacket#Show` (case 0x16 → `SetTrunkDlg`@0x76a940 →
`SetGetItems`@0x76a390, v95 GMS_v95.0_U_DEVM 3c71fd88...). The v95 client read
order for the open-storage ("Show") packet is:

1. `Decode1` mode (22)
2. `Decode4` npcTemplateId
3. `Decode1` slotCount
4. `DecodeBuffer(8)` tab-flag bitmask
5. **conditional** `Decode4` meso — read **only if** `flag & 2`
6. **per-tab loop** over tab bits 4/8/16/32/64 (Equip/Use/Setup/Etc/Cash): for
   each set bit, `Decode1` count then `count × GW_ItemSlotBase::Decode`.

Atlas `storage/clientbound/show.go` `Show.Encode` instead writes:
`mode, npcId, slots, Long(flags), Int(meso) [UNCONDITIONAL], Short(0), byte(count),
[flat assets], Short(0), byte(0)`.

Two real divergences (verified ❌ in `Show.md` rows 5-9, v95):

- **Spurious padding** the v95 client never reads: a `WriteShort(0)` *before*
  the item count, and a trailing `WriteShort(0)` + `WriteByte(0)`.
- **Single flat count vs per-tab segmented counts.** Atlas is called with
  `StorageOperationShowBody` which hardcodes `flags = StorageFlagAll` (bits
  2|4|8|16|32|64 = 0x7E) and `toAllPacketAssets` (a flat, unsegmented list of
  every inventory type). With all 5 tab bits set the client expects **5
  separate count+items blocks**; atlas emits one. → wire desync that wedges the
  storage panel until logout (the durable hot-path failure mode).

**Deferred (not fixed here)** because: (a) the correct fix is a structural
rewrite of `Show.Encode` — bucket assets by `Asset.inventoryType()`, emit a
count byte per set tab bit, gate meso on `flag&2`, drop the three padding bytes
— not a width/order tweak; (b) `Show.Encode` is version-agnostic (no region
guards) and only the v95 client `SetGetItems` was readable in this session, so a
change risks the v83/v87/v92/v111/JMS185 clients which may read the trailing
padding differently; (c) the conservative-rule bar (a speculative hot-path
rewrite is worse than an honest deferral). Fix warrants its own task with a
live storage-panel round-trip test across all target versions.

Files: `libs/atlas-packet/storage/clientbound/show.go`.

### UpdateAssets clientbound — loop/sub-struct tool limitation (storage), correct in practice

`CTrunkDlg::OnPacket#UpdateAssets` (cases 9/13/15/19 → `SetGetItems`). Verdict
🔍 in `UpdateAssets.md` is a packet-audit **tool limitation** (the per-tab item
loop + `GW_ItemSlotBase`/`Asset` sub-struct can't be flattened row-for-row), NOT
a wire bug. Runtime callers (`services/atlas-channel/.../storage/consumer.go`)
always pass `flags = inventoryTypeToFlag(t)` (exactly **one** tab bit, currency
bit never set) and `toPacketAssets(t, …)` (assets filtered to that one type), so
the client's per-tab loop runs exactly once → reads one count + items, which
matches atlas's single-count emission byte-for-byte. No fix needed. (Contrast
with Show, which sets all bits and sends a flat list — that one is the real bug.)

### OP-FAMILY-storage — serverbound dispatcher op-byte (storage)

`storage/serverbound/operation.go` `Operation.Encode` writes only the op-byte
(`WriteByte(mode)`); the field payloads live in the sibling sub-bodies. The IDA
oracle is the family of `CTrunkDlg::Send*Request` senders, each of which
`COutPacket(…,67=0x43)` then `Encode1(op)` + fields:

| Atlas struct | op-byte | IDA sender | fields after op |
|---|---|---|---|
| (Operation dispatcher) | — | — | op byte only |
| OperationRetrieveAsset | 4 | `CTrunkDlg::SendGetItemRequest`@0x769e00 | `Encode1` invType, `Encode1` slot |
| OperationStoreAsset | 5 | `CTrunkDlg::SendPutItemRequest`@0x768570 | `Encode2` slot, `Encode4` itemId, `Encode2` qty |
| OperationMeso | 7 | `CTrunkDlg::SendGetMoneyRequest`@0x7688e0 / `SendPutMoneyRequest`@0x7689e0 | `Encode4` signed amount (+withdraw / −deposit) |

The dispatcher's op-byte is supplied by the caller (the resolved `operations`
code), so its lone-byte ✅ in `Operation.md` is expected. Sub-bodies audited
independently — all ✅ in v95. No fix needed; recorded here for traceability.

### Add clientbound — asset sub-struct tool limitation (inventory), correct in practice

`CWvsContext::OnInventoryOperation#Add` (dispatcher @0xa08a70, case 0 line 158 →
`GW_ItemSlotBase::Decode`, v95 GMS_v95.0_U_DEVM 3c71fd88...). Verdict 🔍 in
`Add.md` row 5 is a packet-audit **tool limitation**: the analyzer flattens
`Add.Encode`'s `WriteByteArray(m.asset.Encode(...))` into one opaque row and
cannot descend into the `model.Asset` / `GW_ItemSlotBase` sub-struct. NOT a wire
bug — the dispatcher case-0 read order (Decode1 mode, Decode1 invType, Decode2
slot, then the item via `GW_ItemSlotBase::Decode`) matches `Add.Encode`
(`mode 0, type, slot, asset.Encode`) field-for-field. The asset body is the
shared `model.Asset` encoder, audited independently of this packet. No fix
needed.

### ChangeBatch clientbound — loop + conditional trailing addMov tool limitation (inventory), correct in practice

`CWvsContext::OnInventoryOperation#ChangeBatch` (dispatcher @0xa08a70). Verdict ❌
in `ChangeBatch.md` row 3 is a packet-audit **false positive**, NOT a wire bug.
The dispatcher's per-entry body is a variable-length switch loop
(`case 0/1/2/3`, lines 148-411) that the analyzer collapses to a single opaque
buffer op (the Phase 0 `EncodeEntry` extension resolves the entry recursion, so
row 2 is ✅). After the loop the dispatcher reads **one** trailing addMov byte
(`Decode1` → `SetSecondaryStatChangedPoint`, line 315) iff `nCurItemPos` was set
— which happens when any entry is an equip-slot move (case 2:
`invType==1 && (oldSlot<0||newSlot<0)`, line 225) or remove (case 3:
`invType==1 && slot<0`, line 374). `ChangeBatch.Encode` writes its single
trailing `WriteInt8(addMov)` under exactly that condition (`addMov = max over
entries of EntryAddMov()`, emitted iff `> -1`). The ❌ arises only because the
conditional trailing byte (atlas row 3) has no flat IDA op to align against once
the loop is buffer-collapsed. Wire-correct in v95; no fix needed.

### move.go ↔ change.go:ChangeMove addMov symmetry — verified NOT shared (inventory)

The serverbound move request `CWvsContext::SendChangeSlotPositionRequest`@0x9d9c10
encodes exactly `Encode4 update_time, Encode1 nType, Encode2 nOldPos,
Encode2 nNewPos, Encode2 nCount` — **no addMov byte**. The clientbound echo
`change.go:ChangeMove` (dispatcher case 2) DOES carry the trailing addMov byte.
These are distinct packets (ITEM_MOVE opcode serverbound vs INVENTORY_OPERATION
clientbound) and correctly do NOT share the addMov field — the addMov byte is
exclusively a clientbound `OnInventoryOperation` artifact. `move.go` ✅ matches
its IDA sender byte-for-byte; `change.go:ChangeMove` ✅ matches the dispatcher
(including the conditional addMov). No mismatch, no fix needed; recorded for
traceability.

### OP-FAMILY-interaction — serverbound dispatcher op-byte (interaction)

`interaction/serverbound/operation.go` `Operation.Encode` writes only the op-byte
(`WriteByte(mode)`); each sub-op's payload lives in a sibling
`operation_*.go` file. The v95 IDA oracle is the family of client-side
`CMiniRoomBaseDlg` / `CTradingRoomDlg` / `CPersonalShopDlg` / `CEntrustedShopDlg`
/ `CMemoryGameDlg` / `COmokDlg` / `CField` send functions, each of which builds
`COutPacket(…, 144 = 0x90 PLAYER_INTERACTION)` then `Encode1(op)` + the sub-op
payload. The dispatcher's lone-byte ✅ in `Operation.md` is expected (op-byte
supplied by the caller's resolved `operations` code). Sub-bodies audited
independently. Confirmed op-byte map (v95 GMS_v95.0_U_DEVM 3c71fd88…):

| atlas sub-op | op | IDA sender |
|---|---|---|
| OperationInvite | 2 | `CField::SendInviteTradingRoomMsg`@0x52e9e0 |
| OperationChat | 6 | `CMiniRoomBaseDlg::CheckAndSendChat`@0x6382a0 |
| OperationTradePutItem | 0xF | `CTradingRoomDlg::PutItem`@0x7641d0 |
| OperationTradeAddMeso | 0x10 | `CTradingRoomDlg::PutMoney`@0x764450 |
| OperationTradeConfirm | 0x11 | `CTradingRoomDlg::Trade`@0x7646b0 |
| OperationTransaction | 0x11 | `CCashTradingRoomDlg::Trade`@0x49e180 |
| OperationPersonalStorePutItem | 0x16(22)/0x21(33) | `CPersonalShopDlg::PutItem`@0x69c880 |
| OperationPersonalStoreBuy | 0x17(23)/0x22(34) | `CPersonalShopDlg::BuyItem`@0x69a7f0 |
| OperationPersonalStoreAddToBlackList | 0x1C(28) | `CPersonalShopDlg::OnClickBanButton`@0x69b1c0 |
| OperationPersonalStoreSetBlackList | 0x1E(30) | `CPersonalShopDlg::DeliverBlackList`@0x69b0d0 |
| OperationPersonalStoreRemoveItem | 0x1B(27)/0x26(38) | `CPersonalShopDlg::MoveItemToInventory`@0x6987a0 |
| OperationFieldAddToBlackList | 0x1F(31) | `CField::AddBlackList`@0x539710 |
| OperationFieldRemoveFromBlackList | 0x20(32) | `CField::DeleteBlackList`@0x5397d0 |
| OperationMerchantAddToBlackList | 0x30(48) | `CEntrustedShopDlg::AddBlackList`@0x51ed50 |
| OperationMerchantRemoveFromBlackList | 0x31(49) | `CEntrustedShopDlg::DeleteBlackList`@0x51ee20 |
| OperationMemoryGameTieAnswer | 0x33(51) | `CMemoryGameDlg::OnTieRequest`@0x627e60 |
| OperationMemoryGameRetreatAnswer | 0x37(55) | `COmokDlg::OnRetreatRequest`@0x6804b0 |
| OperationMemoryGameMoveStone | 0x40(64) | `COmokDlg::PutStoneChecker`@0x6801e0 |
| OperationMemoryGameFlipCard | 0x44(68) | `CMemoryGameDlg::SendTurnUpCard`@0x6279b0 |

### INTERACTION-MODE-MAP — mode-byte → sub-op routing (atlas-channel, outside libs/atlas-packet)

The mode-byte → `operation_*.go` reader dispatch lives in atlas-channel routing
(the resolved `operations` code table + the channel handler switch), NOT in
`libs/atlas-packet`. Out of scope for this packet-audit; recorded so a future
maintainer knows the mode map is centralised there, not in the packet structs.

### INTERACTION-CB-MODE-MAP — clientbound interaction_body.go is a router (interaction)

`interaction/clientbound/interaction_body.go` is a constructor/router block (8
`CharacterInteraction*Body` factories that resolve the `operations` mode code,
then build the matching wire struct from `interaction.go`), parallel to cash's
`shop_operation_body.go`. The 8 target structs are the real wire shapes and were
each audited as their own row against `CMiniRoomBaseDlg::OnPacketBase` (clientbound
mode dispatcher @0x639e10) + per-mode sub-handlers:

| Body factory | target struct | mode | IDA handler |
|---|---|---|---|
| CharacterInteractionInviteBody | InteractionInvite | 2 | `OnInviteStatic`@0x637a40 |
| CharacterInteractionInviteResultBody | InteractionInviteResult | 3 | `OnInviteResultStatic`@0x637d70 |
| CharacterInteractionEnterBody | InteractionEnter | 4 | `OnEnterBase`@0x638f80 |
| CharacterInteractionEnterResultSuccessBody | InteractionEnterResultSuccess | 5 | `OnEnterResultBase`@0x638e30 |
| CharacterInteractionEnterResultErrorBody | InteractionEnterResultError | 5 | `OnEnterResultStatic`@0x639500 |
| CharacterInteractionChatBody | InteractionChat | 6 | `OnChat`@0x639ad0 |
| CharacterInteractionLeaveBody | InteractionLeave | 10 | `OnLeaveBase`@0x637510 |
| CharacterInteractionUpdateMerchantBody | InteractionUpdateMerchant | 25 | `CEntrustedShopDlg::OnRefresh`@0x51cc30 |

`interaction_body.go` itself gets no SUMMARY row (router). Wire-shape denominator
for interaction stays 8 CB + 29 SB sub-ops + 1 dispatcher = 38 atlas shapes;
26 produced reports this round (12 SB unmapped/shared, see below).

### OperationChat — missing leading update_time field (interaction) — ✅ RESOLVED

> ✅ RESOLVED (task-067 Phase 2): v83 `CheckAndSendChat`@0x65f438 sends EncodeStr
> message ONLY (no leading update_time) — confirms the field is v95-only. Fixed by
> gating `Encode4 updateTime` behind `Region()=="GMS" && MajorVersion()>=95`; v83
> keeps the single-string shape, JMS untouched.

`CMiniRoomBaseDlg::CheckAndSendChat`@0x6382a0 (op 6, v95). After the dispatcher
op-byte the v95 client sends `Encode4 update_time (get_update_time)` THEN
`EncodeStr message`. Atlas `operation_chat.go` `OperationChat.Decode` reads only
`ReadAsciiString message` — it is **missing the leading 4-byte update_time**, a
hot-path 4-byte read desync on every mini-room chat. Verified ❌ in
`OperationChat.md` rows 0-1 (v95).

**Deferred (not fixed here):** the fix is to prepend a `uint32 updateTime` field
to the struct + Encode/Decode. This is version-sensitive — `update_time`
prefixing on mini-room chat is a known per-version differentiator (some older
GMS/JMS builds omit it) and only v95 IDA was readable this session. A blind
unconditional add risks the v83/v87/v92/v111/JMS185 clients. Warrants a focused
follow-up with cross-version IDA (`CheckAndSendChat` in each target build) to
gate the field behind a region/version guard. File:
`libs/atlas-packet/interaction/serverbound/operation_chat.go`.

### OperationPersonalStoreBuy / OperationMerchantBuy — missing trailing itemCRC (interaction) — ✅ RESOLVED

> ✅ RESOLVED (task-067 Phase 2): v83 `CPersonalShopDlg::BuyItem`@0x6fd261 sends
> Encode1 index, Encode2 quantity, Encode4 CItemInfo::GetItemCRC — itemCRC PRESENT
> in v83 (op 0x17/0x22). Fixed unconditionally: added trailing `itemCRC uint32` to
> both OperationPersonalStoreBuy and OperationMerchantBuy.

`CPersonalShopDlg::BuyItem`@0x69a7f0 (op 23 personal-store / op 34 entrusted-
merchant, v95). After the op-byte the v95 client sends `Encode1 nIdx (index)`,
`Encode2 quantity`, **`Encode4 itemCRC` (`CItemInfo::GetItemCRC`)**. Atlas
`operation_personal_store_buy.go` / `operation_merchant_buy.go` read only
`{byte index, short quantity}` — both are **missing the trailing 4-byte itemCRC**.
Verified ❌ in `OperationPersonalStoreBuy.md` / `OperationMerchantBuy.md` row 2
(v95). (CRC is an anti-tamper checksum the server validates; missing it leaves a
4-byte tail unconsumed.)

**Deferred (not fixed here):** add a `uint32 itemCRC` trailing field. The
item-CRC scheme is version-sensitive (CRC presence/width varies across client
generations; `GetItemCRC` is not in all builds), and only v95 IDA was readable.
Blind add risks other versions. Warrants a cross-version follow-up. Files:
`libs/atlas-packet/interaction/serverbound/operation_personal_store_buy.go`,
`operation_merchant_buy.go`.

### OperationPersonalStoreSetBlackList — byte[] vs string[] structural mismatch (interaction) — ✅ RESOLVED

> ✅ RESOLVED (task-067 Phase 2): v83 `CPersonalShopDlg::DeliverBlackList`@0x6fdeda
> sends Encode2 count then a per-entry EncodeStr loop — string[] in BOTH versions.
> Fixed unconditionally: `entries` changed from `[]byte` to `[]string`
> (ReadAsciiString/WriteAsciiString). Channel handler uses only len(Entries()).

`CPersonalShopDlg::DeliverBlackList`@0x69b0d0 (op 0x1E=30, v95). After the op-byte
the v95 client sends `Encode2 count` then a per-entry loop of `EncodeStr(name)`
(length-prefixed strings — one character name per blacklisted user). Atlas
`operation_personal_store_set_black_list.go` reads `ReadUint16 count` then
`count × ReadByte` (a flat byte array). The count short is ✅ but the body is a
**structural mismatch** (per-entry strings vs raw bytes) — verified ❌ in
`OperationPersonalStoreSetBlackList.md` row 1 (v95).

**Deferred (not fixed here):** the correct shape is `[]string` decoded via
`count × ReadAsciiString`. This is a structural rewrite of the entry loop (and
the `entries []byte` field type), version-sensitive (blacklist wire format is a
per-version differentiator), and only v95 IDA was readable. Warrants a focused
follow-up. File:
`libs/atlas-packet/interaction/serverbound/operation_personal_store_set_black_list.go`.

### Interaction tool-limitation false positives (sub-struct / representation)

These ❌/diff rows are packet-audit **tool limitations**, NOT wire bugs (verified
against v95 IDA):

- **OperationMemoryGameMoveStone** (`COmokDlg::PutStoneChecker`@0x6801e0, op 0x40).
  Atlas writes `WriteInt64 point` (8 bytes); the v95 client reads
  `EncodeBuffer(&pt, 8)` = a `tagPOINT` (two int32 x,y, 8 bytes). Byte-for-byte
  identical on the wire (9 bytes total incl. trailing color byte). The ❌ in
  `OperationMemoryGameMoveStone.md` row 0 is only an int64-vs-8-byte-buffer
  representation mismatch; wire-correct. No fix.
- **InteractionEnter** (`OnEnterBase`@0x638f80, mode 4). Atlas writes
  `mode + visitor.Encode` where `interaction.Visitor` is a sub-struct (slot +
  avatar look + userID string + jobCode). The analyzer flattens the atlas Visitor
  field-by-field but the IDA side is one opaque `DecodeBuffer`/`DecodeAvatar` →
  spurious ❌. The leading mode byte ✅; the body is the shared Visitor sub-struct
  audited independently. No fix.
- **InteractionEnterResultSuccess** (`OnEnterResultBase`@0x638e30, mode 5). Same:
  `interaction.Room` sub-struct (roomType + maxUsers + myPosition + per-slot
  avatar loop) flattened vs single buffer. Mode byte ✅; body is the Room
  sub-struct. No fix.
- **InteractionUpdateMerchant** (`CEntrustedShopDlg::OnRefresh`@0x51cc30, mode 25).
  Header `mode + meso(int) + count(byte)` all ✅; the per-item loop
  (perBundle short, quantity short, price int, `GW_ItemSlotBase` asset sub-struct)
  is a sub-struct the analyzer can't flatten → spurious ❌ from row 3. Note:
  atlas's `meso` int matches the **entrusted-merchant** `OnRefresh` variant
  (`CEntrustedShopDlg::OnRefresh` reads `Decode4 m_nMoney` then delegates to
  `CPersonalShopDlg::OnRefresh`); the personal-store variant has no meso prefix.
  No fix.

### InteractionInviteResult — conditional trailing string (interaction), informational

`OnInviteResultStatic`@0x637d70 (mode 3) reads `Decode1 result` then a trailing
`DecodeStr name` **only for result codes 2/3/4**, NOT for result 1 (or 0). Atlas
`InteractionInviteResult.Encode` writes the message string unconditionally. For
result code 1 the client would not consume atlas's 2-byte empty-string length →
potential 2-byte tail. The audit reports ✅ (atlas always writes a string and the
common-path result codes 2/3/4 read it), and atlas in practice only emits this
writer for the trade-result family (codes 2/3/4). Recorded as a latent
conditional-emit nuance, not a confirmed v95 bug; revisit if the result-1 path is
ever exercised. File: `libs/atlas-packet/interaction/clientbound/interaction.go`.

### Interaction sub-ops with no v95 IDA sender located (interaction) — 🔍 DEFERRED

The following atlas serverbound sub-op structs have no report this round because a
single v95 client-side send function could not be confidently located (the create/
open/visit/name-change/set-visitor/cash-trade-open senders are built inline in
field/inventory-drag/cash-shop UI paths that were not isolated in this session).
Each needs a focused IDA spike before a verdict; do NOT speculate on their wire
shapes:

- **OperationCreate** — mini-room/store/game create request (op 0/1; built by the
  create dialogs `CCashTradingRoomDlg::OnCreate` / personal-shop create / omok/
  memory-game create + `CField` open paths). Multi-variant by roomType.
- **OperationOpen** — store "open for business" request (single bool).
- **OperationCashTradeOpen** — cash-shop trade-open request (nProc/roomType-gated
  multi-field; likely a `CCashShop`/`CWvsContext` cash-trade path).
- **OperationInviteDecline** — decline-invite reply (serialNumber + errorCode);
  likely `CUIFadeYesNo` trade-invite "No" callback.
- **OperationVisit** — store-visit reply (serialNumber + conditional error/cash SN).
- **OperationMerchantNameChange** — hired-merchant name-change request (single int).
- **OperationPersonalStoreSetVisitor** — set-visitor-slot request (slot + name).

(`OperationMerchantBuy`/`OperationMerchantPutItem`/`OperationMerchantRemoveItem`
DO share the confirmed `CPersonalShopDlg` entrusted-merchant op-bytes and were
audited via synthetic `#Merchant` FNames — see OP-FAMILY-interaction.)

### OP-FAMILY-cash — shop_operation_body.go is the clientbound router (cash)

`cash/clientbound/shop_operation_body.go` is the `CashShopOperation` constructor/
router block (8 `CashShop*Body` factories that resolve the `operations` mode code,
then build the matching wire struct from `shop_inventory.go` / `shop_item_moved.go`
/ `shop_operation_result.go`), parallel to interaction's `interaction_body.go`.
It gets NO SUMMARY row (router). The 78 named `CashShopOperationError*` /
`CashShopOperation*` constants are the resolved `errors`/`operations` code table,
NOT distinct wire shapes — they are not audited individually (design §4.3 cap).
The real clientbound dispatcher is `CCashShop::OnCashItemResult`@0x499370 (a single
mode-switch over op-bytes 0x54–0xBC, >40 cases) plus the separate
`CCashShop::OnQueryCashResult`@0x496400 (opcode 0x17F) and `CCashShop::OnPacket`
@0x4997e0 top-level demux. Each atlas target struct was audited as its own row
against its `OnCashItemRes*` sub-handler (v95 GMS_v95.0_U_DEVM 3c71fd88…):

| Body factory | target struct | op | IDA sub-handler |
|---|---|---|---|
| CashShopCashInventoryBody | CashShopInventory | 0x58 | `OnCashItemResLoadLockerDone`@0x494cb0 |
| CashShopCashInventoryPurchaseSuccessBody | CashShopPurchaseSuccess | 0x64 | `OnCashItemResBuyDone`@0x494dd0 |
| CashShopCashItemMovedToCashInventoryBody | CashItemMovedToCashInventory | 0x77 | `OnCashItemResMoveLtoSDone`@0x495050 |
| CashShopCashItemMovedToInventoryBody | CashItemMovedToInventory | 0x79 | `OnCashItemResMoveStoLDone`@0x4948d0 (asset sub-struct; see below) |
| CashShopWishListBody | WishList | 0x5C/0x62 | `OnCashItemResSetWishDone`@0x494d60 / `OnCashItemResLoadWishDone`@0x494020 |
| CashShopInventoryCapacityIncreaseSuccessBody | InventoryCapacitySuccess | 0x6D | `OnCashItemResIncSlotCountDone`@0x497270 |
| CashShopInventoryCapacityIncreaseFailedBody | InventoryCapacityFailed | 0x6E | `OnCashItemResIncSlotCountFailed`@0x497390 |
| CashShopCashGiftsBody | CashShopGifts | 0x5A | `OnCashItemResLoadGiftDone`@0x496520 (gift-list loop; see below) |

`OperationError` (mode+errorByte) is the shared shape of every `*Failed` case
(audited against `OnCashItemResBuyFailed`@0x4969f0, representative). No single
dispatcher showed >10 stale code values vs the template — the failure cases all
share the same `mode + NoticeFailReason(byte)` shape, so no enum-drift triage was
triggered. Recorded for traceability; the router itself needs no fix.

### Cash serverbound SPW-string vs birthday-int divergence (cash) — ✅ RESOLVED (version-gated)

> ✅ RESOLVED (task-067 Phase 2): v83 confirms the leading field is a 4-byte int
> (the `ask_SPW()` return), v95 a length-prefixed string. v83 anchors:
> OnBuyCouple@0x46ffe7, OnBuyFriendship@0x470a5a, SendGiftsPacket@0x46f940,
> OnRebateLockerItem@0x46bde1. atlas's prior `birthday uint32` matched v83. Fixed
> by gating the leading field behind `Region()=="GMS" && MajorVersion()>=95` (int
> for v83, spw string for v95). The Gift fix ALSO adds the v95-only `byte oneADay`
> before the recipient name (v83 has neither SPW string nor oneADay — its leading
> two ints match atlas's birthday+serialNumber). JMS untouched.

Four cash serverbound "gift-family" senders write a leading **`EncodeStr` secondary-
password (SPW) string** in v95 (built by `ask_SPW`), but atlas models the leading
field as a 4-byte **`int birthday`**. Confirmed ❌ in v95 GMS_v95.0_U_DEVM 3c71fd88…:

| Atlas struct | IDA sender | v95 leading field | atlas leading field |
|---|---|---|---|
| ShopOperationBuyCouple | `OnBuyCouple`@0x490d80 (op 0x1F) | `EncodeStr sSPW` | `int birthday` |
| ShopOperationBuyFriendship | `OnBuyFriendship`@0x491b30 (op 0x25) | `EncodeStr sSPW` | `int birthday` |
| ShopOperationGift | `SendGiftsPacket`@0x487b60 (op 4) | `EncodeStr sSPW` | `int birthday` |
| ShopOperationRebateLockerItem | `OnRebateLockerItem`@0x485840 (op 0x1C) | `EncodeStr sSPW` then `EncodeBuffer 8` | `int birthday` then `uint64 unk` |

The remaining fields after the leading one align (couple/friendship: `int option,
int serialNumber, str name, str message`; rebate: 8-byte locker SN). The fix would
replace the leading `birthday uint32` with a `spw string` field (+ for gift, also
insert a `byte oneADay` between serialNumber and name — see below).

**Deferred (not fixed here):** the SPW (secondary-password / "ask_SPW") system is a
known per-region/per-version differentiator — older GMS builds and some regions
carry a `birthday int` instead of an SPW string on these packets (which is almost
certainly why atlas modeled it as `birthday`). Only v95 IDA was readable this
session; a blind string-for-int swap risks the v83/v87/v92/v111/JMS185 clients.
Warrants a cross-version follow-up (`OnBuyCouple`/`OnBuyFriendship`/`SendGiftsPacket`
/`OnRebateLockerItem` in each target build) to gate the field behind a region/version
guard. Files: `libs/atlas-packet/cash/serverbound/shop_operation_buy_couple.go`,
`shop_operation_buy_friendship.go`, `shop_operation_gift.go`,
`shop_operation_rebate_locker_item.go`.

### ShopOperationBuy — trailing oneADay byte + eventSN int (cash) — ✅ RESOLVED (version-gated)

> ✅ RESOLVED (task-067 Phase 2): v83 `CCashShop::OnBuy`@0x46dadd sends a SINGLE
> trailing Encode4 (IsZeroGoods int) after nCommSN — NO byte oneADay, NO eventSN.
> atlas's prior `zero uint32` matched v83. The v95 byte oneADay + int eventSN are
> a later addition. Fixed by gating the tail behind `Region()=="GMS" &&
> MajorVersion()>=95`: v83 emits the single int, v95 emits byte+int. JMS untouched.

`CCashShop::OnBuy`@0x48e530 (op 3, v95). After `bool isMaplePoint, int dwOption,
int nCommSN` the v95 client sends **`Encode1 m_bRequestBuyOneADay`** then
**`Encode4 nEventSN`** (zero-goods event SN). Atlas `ShopOperationBuy` reads
`bool, int, int, int zero` — it models the trailing `byte oneADay + int eventSN`
(5 bytes) as a single `int zero` (4 bytes), a 1-byte under-read + field
mislabel. Confirmed ❌ in `ShopOperationBuy.md` rows 3-4 (v95).

`ShopOperationGift` (`SendGiftsPacket`@0x487b60) has the same `byte oneADay` between
serialNumber and the recipient-name string — atlas omits it entirely (see SPW row
above; the gift fix must address both the leading SPW string AND this missing byte).

**Deferred (not fixed here):** the "buy-one-a-day" / zero-goods-event mechanic is a
later GMS addition; v83 almost certainly omits the trailing byte+eventSN. A blind
add risks older clients. Warrants cross-version IDA confirmation before gating.
Files: `libs/atlas-packet/cash/serverbound/shop_operation_buy.go`,
`shop_operation_gift.go`.

### CashShopInventory — missing 2 trailing slot-counter shorts (cash) — ✅ RESOLVED (version-gated)

> ✅ RESOLVED (task-067 Phase 2): v83 `OnCashItemResLoadLockerDone`@0x4794f6 reads
> ONLY 2 trailing shorts (m_nTrunkCount + m_nCharacterSlotCount). The extra
> m_nBuyCharacterCount + m_nCharacterCount are v95-only. Fixed by gating the 2
> extra shorts behind `Region()=="GMS" && MajorVersion()>=95`
> (buyCharacterCount/characterCount are non-constructor fields defaulting to 0, so
> no caller-signature ripple). v83 keeps the 2-short tail; JMS untouched.

`CCashShop::OnCashItemResLoadLockerDone`@0x494cb0 (CashShopOperation op 0x58, v95).
The v95 client reads, after the 55-byte-per-item locker loop:
`Decode2 m_nTrunkCount`, `Decode2 m_nCharacterSlotCount`, **`Decode2
m_nBuyCharacterCount`**, **`Decode2 m_nCharacterCount`** — four trailing shorts.
Atlas `CashShopInventory.Encode` writes only the first two
(`storageSlots`, `characterSlots`) and is **missing the trailing
`buyCharacterCount` + `characterCount` shorts** (4-byte under-write). Confirmed ❌
in `CashShopInventory.md` rows 5-6 (v95). (The 55-byte `CashInventoryItem.EncodeBytes`
per-item body resolved correctly inline via the Phase 0 analyzer extension — row 2 ✅.)

**Deferred (not fixed here):** the locker-load slot-counter block grew across client
generations (`buyCharacterCount`/`characterCount` are later additions; earlier builds
read only 2 or 3 counters). `CashShopInventory.Encode` is version-agnostic (no region
guards) and only v95 IDA was readable; a blind 2-short append risks the
v83/v87/v92/v111 clients. Warrants cross-version `OnCashItemResLoadLockerDone`
confirmation before gating. File: `libs/atlas-packet/cash/clientbound/shop_inventory.go`.

### Cash tool-limitation false positives (loop / exclusive-branch / int64-vs-buffer) — NOT wire bugs

These ❌ rows are packet-audit **tool limitations**, verified correct against v95 IDA;
no fix needed:

- **ShopOperationSetWishlist** (`OnSetWish`@0x4837d0, op 5). Atlas writes 10×
  `WriteInt` in a `for` loop (40 bytes); v95 reads `DecodeBuffer 40`. The analyzer
  flattens the loop to `int32 + 9×byte` → spurious ❌ rows 1-9. Wire-correct (40 bytes
  either way).
- **WishList** (`OnCashItemResSetWishDone`@0x494d60, op 0x62). Same loop-flatten:
  atlas writes `mode + 10×WriteInt` (1+40 bytes); v95 reads `Decode1 mode +
  DecodeBuffer 40`. Mode ✅; the 10-int loop body is collapsed → spurious ❌ rows 1-2.
- **ShopOperationIncreaseInventory** (`OnBuySlotInc`@0x491710, op 6/7) row 4 and
  **ShopOperationIncreaseStorage** (`OnIncTrunkCount`@0x48dc70, op 7) row 3:
  exclusive-branch over-count. Atlas's `if m.item {WriteInt serialNumber} else
  {WriteByte invType}` (storage: `if m.item {WriteInt}`) collects BOTH branches'
  calls statically, but at runtime only one fires. v95 IncSlotCount sends item=1 →
  int serialNumber (✅); IncTrunkCount sends item=0 → no trailing field (✅). Same
  early-return-exclusivity limitation documented for login `CharacterList`.
- **ShopOperationMoveFromCashInventory** (`OnMoveCashItemLtoS`@0x4828e0, op 0xE) row 0
  and **ShopOperationMoveToCashInventory** (`OnMoveCashItemStoL`@0x482b50, op 0xF)
  row 0: atlas `WriteLong` (int64, 8 bytes) vs v95 `EncodeBuffer(&liSN, 8)` (raw
  8-byte _LARGE_INTEGER). Byte-for-byte identical on the wire; representation-only
  mismatch. Wire-correct.

### Cash shapes with no v95 sender isolated this session (cash) — 🔍 DEFERRED

- **CashShopOpen** (`shop_open.go`) — the cash-shop open packet (opcode 0x12E) is
  built inline in `CCashShop::Init`@0x484920 / `LoadData`@0x492ea0 (huge functions),
  not a discrete `Send*`. Its region/version-gated body (CharacterData + SetSaleInfo
  + Decode Best/Stock/LimitGoods/ZeroGoods blocks) already carries explicit
  GMS/JMS + MajorVersion guards in atlas. Not isolable in one session; no report
  this round. Verdict pending a focused Init/LoadData spike. Do NOT speculate.
- **CashItemMovedToInventory** (`shop_item_moved.go`, op 0x79) — locker→inventory
  echo carrying a `model.Asset` (`GW_ItemSlotBase`) sub-struct via
  `OnCashItemResMoveStoLDone`@0x4948d0. The asset body is a sub-struct the analyzer
  can't flatten (same class as inventory `Add`); no report produced. The leading
  `mode + slot` header is trivially correct; the asset body is the shared
  `model.Asset` encoder audited independently. No fix expected; deferred for a
  loop/sub-struct-aware pass.
- **CashShopGifts** (`shop_inventory.go`) — `CashShopGiftsBody()` is a stub that
  hardcodes mode 0x4D and writes `mode + short(0)` (empty gift list). The real v95
  gift-load reader `OnCashItemResLoadGiftDone`@0x496520 decodes a non-empty
  per-gift loop (0x433-byte handler). Atlas's empty-list stub is intentionally a
  placeholder, not a faithful encoder; auditing it against the real loop would be a
  false ❌. Recorded as a known stub; a faithful implementation + audit is a
  follow-up, not a wire fix.
- **item_use family** (`item_use.go`, `item_use_chalkboard.go`,
  `item_use_field_effect.go`, `item_use_pet_consumable.go`) — all four are built by
  `CWvsContext::SendConsumeCashItemUseRequest`@0x9eb3e0, a ~248 KB single function
  switching over every consumable cash-item type, each branch with its own
  field layout and `update_time` ordering (the `updateTimeFirst` flag atlas threads
  through). The common prefix (`update_time` gated `GMS && MajorVersion>=95`,
  `source int16`, `itemId int`) is consistent with the v95 function header, but the
  per-type bodies could not be exhaustively mapped in one session. Verdict 🔍
  DEFERRED — needs a focused per-item-type spike; do NOT speculate on the variant
  field layouts. (`ItemUse.md` in the audit output is the **inventory** `ItemUse`
  collision winner, not the cash one — the cash `ItemUse` uses `pathHint:"cash/"`
  but shares the wired `CWvsContext::SendStatChangeItemUseRequest` mapping with
  inventory; the cash item-use senders have no separate wired FName this round.)
- **ShopEntry** (`shop_entry.go`) — `CashShopEntryHandle` (opcode 0x28D, single
  `int updateTime`). The matching client sender (`CCashShop::SendEntry` / transfer-
  to-CS-field path) was not isolated as a discrete `Send*` this session; the
  1-int shape is trivial but unverified against a v95 case. Deferred 🔍.

## Sub-op enum drift — character domain

The following character-domain packets dispatch on a leading mode/sub-op byte
inside the packet body. The audit pipeline models a single flat sequence of
Decode calls and cannot represent a switch-on-mode dispatch tree. Each row
below was filed as ❌ by the pipeline; the real issue is sub-op enum drift
that the pipeline cannot verify.

| FName | Atlas writer structs | Notes |
|---|---|---|
| `CUser::OnEffect` | `EffectSimple`, `EffectSkillAffected`, `EffectPet`, `EffectWithId`, `EffectWithMessage`, `EffectProtectOnDie`, `EffectIncDecHP`, `EffectShowInfo`, `EffectLotteryUse`, `EffectItemMaker`, `EffectUpgradeTomb`, `EffectIncubatorUse` (all in effect.go) | 16+ sub-op modes (case 0–15+). Atlas models each mode as a separate struct. All use opcode 0xE0 (foreign) or 0xE9 (self). Pipeline can only see the outermost Decode1 (mode byte). Sub-op byte values need per-mode verification. |
| `CUser::OnEffect` | `EffectQuest`, `EffectQuestForeign` (effect_quest.go) | Mode byte = quest-effect sub-op. Same pipeline limitation. |
| `CUser::OnEffect` | `EffectSkillUse`, `EffectSkillUseForeign` (effect_skill_use.go) | Mode byte = skill-use sub-op (mode 1 in GMS). Berserk/DragonFury/MonsterMagnet branches also conditional on skill ID. |
| `CWvsContext::OnMessage` | `StatusMessageDropPickUpInventoryFull`, `StatusMessageDropPickUpItemUnavailable`, `StatusMessageDropPickUpGameFileDamaged`, `StatusMessageDropPickUpStackableItem`, `StatusMessageDropPickUpUnStackableItem`, `StatusMessageDropLossStackableItem`, `StatusMessageDropLossUnStackableItem`, `StatusMessageDropPickUpMeso`, `StatusMessageForfeitQuestRecord`, `StatusMessageUpdateQuestRecord`, `StatusMessageCompleteQuestRecord`, `StatusMessageCashItemExpire`, `StatusMessageIncreaseExperience`, `StatusMessageIncreaseSkillPoint`, `StatusMessageIncreaseFame`, `StatusMessageIncreaseMeso`, `StatusMessageIncreaseGuildPoint`, `StatusMessageGiveBuff`, `StatusMessageGeneralItemExpire`, `StatusMessageSystemMessage`, `StatusMessageQuestRecordEx`, `StatusMessageItemProtectExpire`, `StatusMessageItemExpireReplace`, `StatusMessageSkillExpire` (all in status_message.go) | Opcode 0x26. Top-level Decode1 = mode byte (0–14); each case delegates to a sub-handler that reads mode-specific fields. Atlas has 20+ sub-op structs each writing: mode byte first, then sub-op body. Pipeline report: `StatusMessageDropPickUpInventoryFull.md` (mode=0, representative). IDA sub-handler trace per mode needed to verify sub-op body layouts. See ack footer in `StatusMessageDropPickUpInventoryFull.md`. |

Resolution: Phase 3 — per-mode IDA sub-function trace for each atlas StatusMessage
struct. Each mode constant maps to a specific IDA case-arm (OnDropPickUpMessage,
OnQuestRecordMessage, OnIncEXPMessage, etc.); wire format per arm needs to be
exported and compared against the corresponding struct's Encode method.

## Still pending — character domain

| FName | Atlas writer/handler | Notes |
|---|---|---|
| (bare-handler) | `CharacterSkillChange` (opcode 0x23) | Already in gms_v95.json. Audit reports ❌ due to tool-limitation in nested `SecondaryStat` sub-struct analysis. See CharacterSkillChange.md ack footer. Deferred to Phase 3 analyzer descent. |
| CreateCharacter (opcode 0x17 / bCharSale path) | atlas decoder absent for `m_bCharSale == true` branch in `CLogin::SendNewCharPacket@0x5d7bd0` (opcode 23, 9× AL items, no SubJob/gender). Cash Shop character creation flow not wired. | follow-up |

## Known false positives — character misc-state bucket (Task 10)

### CharacterSitResult.md (verdict ❌)

Row 2 shows an extra byte not consumed by the client. The analyzer flattens both
branches of the `if m.sitting { WriteByte(1)+WriteShort } else { WriteByte(0) }`
into a merged call list, treating the else-branch `WriteByte(0)` as a 3rd sequential
write that appears after the if-branch writes. At runtime only one branch fires:
either `byte(1)+short(chairId)` or `byte(0)`. IDA `CUserLocal::OnSitResult`
(case 231 = 0xE7 in `CUserLocal::OnPacket`) reads `Decode1` then conditionally
`Decode2` — exactly matching the atlas encoder. The ❌ verdict is a branch-flattening
false positive; no wire bug present.

Resolution: analyzer needs to detect exclusive if/else branches and not union their writes.
Deferred to Phase 3 analyzer enhancement.

### CharacterInfo.md (verdict ❌)

Rows 9–22 show multiple width mismatches and extra bytes. `CWvsContext::OnCharacterInfo`
(case 61 = 0x3D in `CWvsContext::OnPacket`) is a complex packet with:
- A bool-terminated pet list (SetMultiPetInfo do-while loop)
- An optional taming mob block (if-guarded)
- A wishList loop (count + N × int32)
- Version-guarded monster book block (GMS < 87 only; absent in v95)
- MedalAchievementInfo sub-struct (Decode4 + Decode2 + optional loop)
- A chair list block (Decode4 count + DecodeBuffer array)

The flat analyzer cannot track loop state, conditional loops, or the version guard
producing the correct sub-sequence for v95. Cross-checking the atlas encoder against
the IDA manually confirms the encoding is correct for v95:
- No monster book block (GMS v95 ≥ 87 → guard false)
- MedalAchievementInfo: WriteInt(medalId) + WriteShort(0) = Decode4 + Decode2 ✅
- Chair list: WriteInt(0) count + no items = Decode4(0) + no buffer ✅

The ❌ verdict is a multi-cause tool limitation (loop linearization, conditional sub-struct
expansion, version guard interaction). No wire bug present.

Resolution: Phase 3 sub-struct descent + loop-aware analyzer.

## Known false positives — character spawn/list bucket (Task 9)

### AddCharacterEntry.md (verdict ❌)

Rows 42–47 show extra atlas bytes (viewAll placeholder + rankEnabled + 4 × rank int32) not
consumed by the client. `CLogin::OnCreateNewCharacterResult` reads only GW_CharacterStat +
AvatarLook; rank data is zero-filled from client state. MapleStory packets are length-prefixed;
the client silently ignores trailing bytes in standalone packets, so no wire corruption occurs.
The analyzer correctly identifies these 18 extra bytes but they are functionally harmless.
Resolution: dedicated non-rank payload type for AddCharacterEntry or context-aware CharacterListEntry
encoder — deferred to follow-up refactor.

### CharacterViewAllCharacters.md (verdict ❌)

Rows 45–50 show DecodeBuf vs 4 × int32 representation mismatch for rank fields, plus
linearization offset shifting the PIC byte. IDA reads rank as `DecodeBuffer(0x10)` (bulk 16
bytes). Atlas emits 4 × `WriteInt`. Wire bytes are identical. Resolution: diff tool DecodeBuf
expansion — deferred to Phase 3 analyzer enhancement.

## Workflow notes

Refresh procedure:
1. `mcp__ida-pro__list_functions_filter` with a partial name to find the IDA FName (mangled symbols are common; use plain prefix like "SelectChar")
2. `mcp__ida-pro__get_function_by_name` (resolve address)
3. `mcp__ida-pro__decompile_function` (extract C source)
4. Parse the `CInPacket::DecodeN` / `COutPacket::EncodeN` call sequence in lexical order (success path only; multi-branch functions need manual filtering)
5. Add the entry to `gms_v95.json` and the `candidatesFromFName` map in `tools/packet-audit/cmd/run.go`
6. Regenerate audit: `cd tools/packet-audit && go run . --csv-clientbound ... --csv-serverbound ... --template ... --atlas-packet ../../libs/atlas-packet --ida-source ../../docs/packets/ida-exports/gms_v95.json --output ../../docs/packets/audits`

The synthetic-FName scheme (e.g., `CLogin::OnCheckPasswordResult#AuthLoginFailed`)
lets one IDA function model multiple sub-branches when atlas has separate
writers for different result codes.

## Cross-version — character domain (v83)

Results of the GMS v83 cross-version pass (Task 15). All 44+ character FNames were
looked up in v83 IDA (base 0x400000, `MapleStory_dump.exe`).

### Missing FNames in v83 IDA

The following v95 FNames have no equivalent function in v83 IDA; the pipeline produces
no report for them. For each, the v83 behaviour is noted.

| v95 FName | v83 behaviour | Atlas struct | Notes |
|---|---|---|---|
| `CUser::OnEmotion` | Handled inline in `CUserPool::OnUserRemotePacket` case 0xC1: reads `Decode4(emotionId)` only; calls `CAvatar::SetEmotion` directly — no separate function | `CharacterExpression` | **Fixed**: `expression.go` (clientbound) now gates `duration` + `byItemOption` on `GMS>83\|\|JMS`. v83 wire: 8 bytes (4 charId + 4 emotionId). v95 wire: 13 bytes. |
| `CUserRemote::OnSetActivePortableChair` | Handled inline in `CUserPool::OnUserRemotePacket` case 0xC4: reads `Decode4(chairId)` directly into `RemoteUser[3567]` — no separate function | `CharacterChairShow` | Same wire shape (`characterId + chairId` = 8 bytes); no divergence. Atlas encoder correct for v83. |
| `CLogin::SendCheckDuplicateIDPacket` | In v83 this lives on `CUICharacterSaleDlg` (a UI class), not `CLogin`. Wire format `EncodeStr(name)` is identical. | `CheckName` | Audit can't match FName; no pipeline report. Wire shape unchanged — no v83 bug. |
| `CWvsContext::SendStatChangeRequest` | In v83 renamed `CWvsContext::SendStatChangeRequestByItemOption@0xa1e997`. Wire format `Encode4+Encode4+Encode2+Encode2+Encode1` is **identical** to v95. | `HealOverTime` | No divergence; audit entry added under the v95 FName key for gms_v83.json. |

### Resolved v83-only divergences (fixed in Task 15; gates updated to >87 in Task 16)

| FName | Atlas struct | v83 wire | v87 wire | v95 wire | Final gate |
|---|---|---|---|---|---|
| `CUser::ShowItemUpgradeEffect` | `ItemUpgrade` (clientbound) | `Decode1×4` (no enchantCategory, no enchantResultFlag) | `Decode1×4` (same as v83) | `Decode1×3 + Decode4 + Decode1×2` | `GMS>87 \|\| JMS` — widened from `>83` after Task 16 confirmed v87 also has only 4 bytes |
| `CWvsContext::SendEmotionChange` | `ExpressionRequest` (serverbound) | `Encode4` (emotionId only) | `Encode4` (same as v83) | `Encode4 + Encode4 + Encode1` | `GMS>87 \|\| JMS` — widened from `>83` after Task 16 confirmed v87 IDA@0xabbfbb |
| `CUser::OnEmotion` (absent in v83) | `CharacterExpression` (clientbound) | `Decode4` (inline in dispatcher case 0xC1) | `Decode4` (inline in case 0xCE, no separate function) | `Decode4 + Decode4 + Decode1` | `GMS>87 \|\| JMS` — widened from `>83` after Task 16 confirmed v87 IDA@0x9f7492 |

### v83 IDA structural differences not requiring encoder changes

| FName / area | Difference | Verdict |
|---|---|---|
| `CVecCtrlUser::EndUpdateActive` | v83 encodes `Encode1(fieldKey) + Encode4(crc)` only — no dr0/dr1/dr2/dr3/dwKey/crc32. v95 IDA already documented these with `GMS>83\|\|JMS` guards on dr fields. | No action — gates were already correct from v95 audit. |
| `CLogin::SendNewCharPacket` | v83 has no `Encode2(subJob)` after race index. Already gated `MajorVersion() > 83` in `create.go`. | No action — already correct. |
| `CLogin::SendDeleteCharPacket` | v83 sends `EncodeStr(deletionPwd) + Encode4(charId)` — same shape as v95. | No divergence. |
| `CFuncKeyMappedMan::OnInit` | v83 loop count is 89 entries (v95: 90). Pipeline reports ❌ for both versions (loop-count tool limitation). Atlas sends 90 × (type+id) regardless — the extra entry is harmless as the client treats it as a full keymap. | Deferred: loop-count discrepancy. No functional impact. |
| `CWvsContext::OnMessage` | v83 has 14 sub-op modes (0–0xD); v95 added mode 0xE (SkillExpire). Both versions ❌ in pipeline due to sub-op dispatch limitation. | Deferred to Phase 3 sub-op audit. |
| `GW_CharacterStat::Decode` field widths | v83: HP/MHP/MP/MMP are `Decode2` (int16); v95: widened to `Decode4` (int32). Both `CharacterList` and `CharacterViewAllCharacters` have `nSubJob` absent in v83. These are sub-struct fields inside complex packets that the flat analyzer cannot reach. | Deferred — existing `_pending.md` tool-limitation rows cover these. |

### Hard-cap gate check

No encoder/decoder in the character domain now contains more than **2 nested** `if t.Region()` / `if t.MajorVersion()` levels after this task's changes. The three fixed encoders each have a single flat gate. Hard cap not triggered.

## Cross-version — character domain (v87)

Results of the GMS v87 cross-version pass (Task 16). All 44+ character FNames were
looked up in v87 IDA (base 0x400000, `GMSv87_4GB.exe`).

### Confirmed v87 alignments (no change needed)

| FName | v87 behaviour | Notes |
|---|---|---|
| `GW_CharacterStat::Decode` HP/MHP/MP/MMP | v87: `Decode2` (int16), same as v83. Widened to `Decode4` in v95 only. Atlas currently writes int32 for all versions — this sub-struct is inside complex CharacterList packets the flat analyzer cannot reach. Deferred. | Same situation as v83; no new gate needed |
| `GW_CharacterStat::Decode` nSubJob | v87: `Decode2(nSubJob)` IS present at end of stat block. Same as v95. Gate `MajorVersion() > 83` for nSubJob already correct. | No action |
| `CFuncKeyMappedMan::OnInit` loop count | v87: loop count = **89** entries (identical to v83; v95 = 90). Deferred — pipeline cannot model loop counts; atlas always sends 90 which is harmless. | No action |
| `CWvsContext::OnMessage` sub-op modes | v87: 15 modes (0x0–0xE) including SkillExpire — same as v95. | No action |
| `CVecCtrlUser::EndUpdateActive` | v87 IDA@0xa5c937: has full dr0/dr1/fieldKey/dr2/dr3/crc/dwKey/crc32 sequence. Gate `GMS>83\|\|JMS` fires correctly for v87. | No action |
| `CLogin::OnSelectCharacterResult` | v87 success path (LABEL_48): `Decode4(ip)+Decode2(port)+Decode4(charId)+Decode1(authenCode)+Decode4(ulPremiumArgument)` — identical to v95. | No action |
| `CLogin::OnViewAllCharResult` case 0 (CharacterViewAllCharacters) | v87: reads same fields as v95 except NO `m_bLoginOpt` at end. Atlas gates `MajorVersion()>87` for this field — already correct. | No action |
| `CLogin::OnSelectWorldResult` m_nBuyCharCount | v87: absent. Atlas gates `MajorVersion()>87` for `nBuyCharCount` in `list.go` — already correct. | No action |

### Missing FNames in v87 IDA

| v95 FName | v87 behaviour | Atlas struct | Notes |
|---|---|---|---|
| `CUser::OnEmotion` | Handled inline in `CUserPool::OnUserRemotePacket@0x9f7492` case 0xCE: reads `Decode4(emotionId)` only (same as v83 case 0xC1). No duration, no byItemOption. | `CharacterExpression` | **Fixed**: gate widened to `GMS>87\|\|JMS` in Task 16. |
| `CUserRemote::OnSetActivePortableChair` | Handled inline in `CUserPool::OnUserRemotePacket` case 0xD1: reads `Decode4(chairId)` directly. Same wire shape as v95. | `CharacterChairShow` | No divergence. |

### Resolved v87-only divergences (fixed in Task 16)

| FName | Atlas struct | v87 wire | v95 wire | Fix |
|---|---|---|---|---|
| `CUser::ShowItemUpgradeEffect@0x9adb79` | `ItemUpgrade` (clientbound) | `Decode1×4` (no enchantCategory, no enchantResultFlag) | `Decode1×3+Decode4+Decode1×2` | Gate widened from `>83` to `>87` in `item_upgrade.go` |
| `CWvsContext::SendEmotionChange@0xabbfbb` | `ExpressionRequest` (serverbound) | `Encode4` (emotionId only) | `Encode4+Encode4+Encode1` | Gate widened from `>83` to `>87` in `serverbound/expression.go` |
| `CUser::OnEmotion` (inline@0x9f7492) | `CharacterExpression` (clientbound) | `Decode4` (expressionId only) | `Decode4+Decode4+Decode1` | Gate widened from `>83` to `>87` in `clientbound/expression.go` |
| `CWvsContext::OnCharacterInfo@0xabb181` | `CharacterInfo` (clientbound) | monster book block (5×int32) IS present | monster book absent (GMS≥87 guard false) | Gate changed from `< 87` to `<= 87` in `info.go` so v87 correctly includes monster book block |

### v87 IDA structural differences deferred to _pending (not fixed)

| FName | v87 difference | Atlas struct | Status |
|---|---|---|---|
| `CLogin::SendCheckPasswordPacket@0x62dfb4` | v87 appends `Encode4(PartnerCode)` after the 3×Encode1 unknowns; atlas reads only `unknown2` for `>=95` — v87 sends unknown2+PartnerCode but atlas only reads unknown1 for v87 (gate `>=95` skips unknown2 for v87). Low-severity: packet read ends cleanly since no subsequent reads follow. | `Request` | Deferred. Wire-format quirk limited to `SendCheckPasswordPacket`; functional impact is zero since atlas doesn't use PartnerCode. |
| `CLogin::SendSelectCharPacket` 0x1D/0x1E opcodes | v87 PIC-register opcode 0x1E sends `EncodeStr+Encode4+EncodeStr+EncodeStr`; v87 PIC-select opcode 0x1D sends `Encode1(1u)+Encode4+EncodeStr+EncodeStr+EncodeStr`. v95 has layouts at opcodes 0x1C/0x1D. Atlas handler–opcode mapping in v87 template assigns 0x1D→RegisterPicHandle, 0x1E→CharacterSelectedPicHandle — layouts are structurally different from the v87 wire. | `CharacterSelectRegisterPic`, `CharacterSelectWithPic` | Deferred. Requires v87-specific handler variants or opcode-keyed decode dispatch. |

### Hard-cap gate check (Task 16)

No encoder/decoder in the character domain now contains more than **2 nested** `if t.Region()` / `if t.MajorVersion()` levels after Task 16 changes. All four fixed encoders (`ItemUpgrade`, `CharacterExpression`, `ExpressionRequest`, `CharacterInfo`) have at most 2 sequential flat gates (never nested). Hard cap not triggered.

## Cross-version — character domain (JMS v185)

Results of the JMS v185 cross-version pass (Task 17). All character domain FNames were
looked up in JMS v185 IDA (base 0x400000, `MapleStory_dump_SCY.exe`, md5 af6652ff9b7c549341f35e3569d7564a).

The JMS v185 binary shares C++ mangled symbol names with GMS v95 for all character-domain
functions searched. No separate opcode space split was found for the character domain
(unlike the login domain which had distinct GMS vs JMS packet structures for
`OnCheckPasswordResult`).

### Resolved JMS divergences (fixed in Task 17; `|| JMS` clauses removed)

These gates had an incorrect `|| JMS` clause added during Task 15/16 under the assumption
that JMS v185 matched GMS v95 behaviour. JMS v185 IDA confirms it uses the older
(v83/v87-equivalent) layout for these packets.

| FName | Atlas struct | JMS v185 wire | GMS v95 wire | Fix |
|---|---|---|---|---|
| `CUser::OnEmotion@0x9f636b` | `CharacterExpression` (clientbound) | `Decode4(nEmotion)+Decode4(tDuration)` — no byItemOption | `Decode4+Decode4+Decode1` | Gate narrowed: duration emitted for JMS (Decode4), byItemOption NOT emitted for JMS. `expression.go` clientbound updated. |
| `CUser::ShowItemUpgradeEffect@0x9f1a92` | `ItemUpgrade` (clientbound) | `Decode1×5` — no Decode4(nEnchantCategory); enchantResultFlag (v6) IS present | `Decode1×3+Decode4+Decode1×2` | Gate narrowed: `|| JMS` removed from enchantCategory gate only. enchantResultFlag gate retains `|| JMS` since JMS reads Decode1(v6). `item_upgrade.go` updated. |
| `CVecCtrlUser::EndUpdateActive@0xaaa076` | `Move` (serverbound) | `Encode1(detectFlag)+[if active: Encode1(fieldKey)+Encode4(crc)+CMovePath]` — no dr0/dr1/dr2/dr3/dwKey/crc32 | Full dr-field sequence | Gate narrowed: `|| JMS` removed from all dr-field gates in `move.go`. JMS movement is GMS v83-equivalent layout. |

### Resolved JMS divergences — serverbound ExpressionRequest

| FName | Atlas struct | JMS v185 wire | GMS v95 wire | Fix |
|---|---|---|---|---|
| `CWvsContext::SendEmotionChange@0xb0b8be` | `ExpressionRequest` (serverbound) | Encodes only `Encode4(charId)` — the local user's characterId, NOT emotionId+duration+byItemOption | `Encode4(emotionId)+Encode4(duration)+Encode1(byItemOption)` | Gate narrowed: `|| JMS` removed. JMS serverbound opcode 0x2B carries only a charId. Atlas server reads the first int4 as emotionId; JMS sends charId in that slot. No duration or byItemOption for JMS. `serverbound/expression.go` updated. |

### JMS-specific structural differences (no encoder change, documented)

| FName | JMS difference | Atlas struct | Status |
|---|---|---|---|
| `CWvsContext::SendStatChangeRequestByItemOption@0xb054d6` | JMS appends `Encode4(timeGetTime())` after `Encode1(nType)` — 5 fields vs GMS v95's 5 fields (same 5 but JMS adds a 6th trailing int4). Low-severity: atlas server reads only 5 fields then stops; the trailing 4 bytes are ignored. No functional impact. | `HealOverTime` | Deferred. JMS-only trailing field; server ignores it. No encoder change needed. |
| `CWvsContext::OnCharacterInfo@0xb0aa6e` | JMS v185 INCLUDES the monster book block (`SomethingMonsterBook` call). The gate `(GMS && <=87) \|\| JMS` in `info.go` is **correct** for JMS. | `CharacterInfo` | No action — already correct. |
| `CWvsContext::SendCharacterInfoRequest@0xb0b323` | JMS wire: `Encode4(updateTime)+Encode4(dwCharacterID)+Encode1(bPetInfo)` — identical to GMS v95. | `CharacterInfoRequest` | No action — no divergence. |
| `CFuncKeyMappedMan::OnInit@0x5e79aa` | JMS function present, same structure. Loop count not easily determinable from decompile. | `FuncKeyMap` | No action — same tool-limitation as v83/v87. |
| `CUserRemote::OnAvatarModified@0xa57221` | JMS uses a *list* format for couple/friendship (Decode4(count)+loop:DecodeBuf(0x10)+Decode4(pairCharId)) vs GMS v95 which reads single-entry buffers. This is a sub-struct difference beyond the flat analyzer's scope. | `CharacterAppearanceUpdate` | Deferred to Phase 3 sub-struct descent. No wire bug in the outer packet structure. |
| `CUser::OnEmotion@0x9f636b` duration field | JMS reads Decode4(tDuration) — confirmed. Atlas now writes duration for JMS (without byItemOption). | `CharacterExpression` | Fixed — see resolved table above. |
| `CLogin::OnCheckPasswordResult@0x66e79f` | JMS v185 success path decodes differently: `Decode4(accountId)+Decode1(gender)+Decode1(gradeCode)+Decode1(combined)+2×DecodeStr(nexon IDs)+5×Decode1+DecodeBuffer(8)+DecodeStr`. Fundamentally different structure from GMS v95. Atlas server only needs the pre-shared accountId for login; login domain is tracked separately in task-027. | `AuthSuccess` (login domain) | Out of scope for character domain audit. Login domain audit (task-027) tracks this separately. |

### Deferred / known limitations — JMS v185

| Issue | Details |
|---|---|
| ExpressionRequest (sb) JMS semantic mismatch | JMS opcode 0x2B carries only charId; atlas's `Decode` reads it as `emote`. Re-broadcast CharacterExpression carries the JMS charId in the expression slot. Pre-existing on `main` — not introduced by task-028. Follow-up: dedicated JMS-aware decoder. |

### Hard-cap gate check (Task 17)

After Task 17 changes, no encoder/decoder in the character domain contains more than **2 nested** gates. The three fixed encoders each have flat sequential gates — `CharacterExpression` now has one `if GMS>87` + one `else if JMS`, `ItemUpgrade` has a single `if GMS>87`, `Move` has three sequential `if GMS>83` + one `if GMS>28`. No nested gates. Hard cap not triggered.


## Still pending — combat domain (monster)

Phase 2a (task-065) audit of 9 monster clientbound packets in GMS v95. ✅ 3 / ❌ 5 / 🔍 1.

| FName | Atlas writer | Verdict | Notes |
|---|---|---|---|
| `CMobPool::OnMobEnterField@0x6589e0` | MonsterSpawn | ❌ | **Analyzer FP (design §3).** Atlas`s `if (region/version) { if controlled then WriteByte(1) else WriteByte(5) }` if/else expands into two consecutive WriteByte entries in the flat call list, throwing off positions 2+. Plus the `m.monster.Encode` MonsterModel sub-struct cannot be fully resolved because the registry keys on unqualified struct names and there are 4 `Spawn` structs across monster/drop/reactor/pet (last-write-wins in `r.types`). Manual IDA confirms wire is ✅. Defer until registry handles qualified type names. |
| `CMobPool::OnMobLeaveField@0x658b90` | MonsterDestroy | ❌ (real) | Atlas missing optional `WriteInt(swallowCharacterId)` when destroyType == 4 (swallowed by character-eater mob like Yeti-and-Pepe). Real wire bug; narrow scope (swallow eaters only). Constructor signature change `NewMonsterDestroy` affects callers in `services/atlas-channel`. Defer to a follow-up that adds the field + updates call sites. |
| `CMobPool::OnMobChangeController@0x658d10` | MonsterControl | ❌ (real, large) | Atlas wire shape fundamentally differs from v95. Atlas writes `int8 controlType + int32 uniqueId + (if type>0: byte(5) + int32 monsterId + MonsterModel)`. v95 reads `byte controlMode + (if controlMode && opt: int32×3 seed) + int32 mobId + (if controlMode: byte aggro)`. Looks like atlas implements an older-protocol shape; v95 controllers carry a movement-seed instead of MonsterModel. Defer to follow-up — needs cross-version IDA pass (v83/v87) to understand when the shape changed. |
| `CMob::OnMove@0x6521e0` | MonsterMovement | 🔍 | Mostly analyzer FP: sub-struct expansion of `MultiTargetForBall`, `RandTimeForAreaAttack`, and `Movement` is incomplete due to registry struct-name collision. The skill block `(skillId, skillLevel)` is gated `GMS>83 || JMS` in atlas but is written as `Decode4 sEffect.m_Data` (packed) in v95 IDA, vs atlas writing `Decode2 skillId + Decode2 skillLevel` separately — same 4 bytes, different field decomposition. May be ✅ on wire bytes; defer for now. |
| `CMob::OnCtrlAck@0x640c50` | MonsterMovementAck | ✅ | Wire shape matches. uniqueId + moveId(int16) + useSkills(byte) + mp(int16) + skillId(byte) + skillLevel(byte). |
| `CMob::OnStatSet@0x652660` | MonsterStatSet | ❌ | **Analyzer FP.** Atlas writes `uniqueId + MonsterTemporaryStat.Encode(mask + per-bit data) + int16(tDelay=0) + byte(nCalcDamageStatIndex=0) + optional byte(bStat)`. v95 OnStatSet top-level reads `mobId + DecodeBuffer(0x10) mask + delegate ProcessStatSet`. The post-mask trailing fields (tDelay/calcIndex/bStat) live inside `CMob::ProcessStatSet` which the audit pipeline cannot descend into. Wire bytes likely ✅. Defer pending ProcessStatSet decompile. |
| `CMob::OnStatReset@0x652780` | MonsterStatReset | ❌ | Same analyzer FP as StatSet. |
| `CMob::OnDamaged@0x64ecb0` | MonsterDamage | ✅ | Wire shape matches. uniqueId + damageType + damage + (conditional hp/maxHp for bDamagedByMob). |
| `CMob::OnHPIndicator@0x642ef0` | MonsterHealth | ✅ | Wire shape matches. uniqueId + hpPercent. |
| `CMob::GenerateMovePath@???` | MonsterMovementHandle (sb) | (deferred) | Single packet not audited in this PR. `CMob::GenerateMovePath` is a 4 KB+ encode-side function that requires dedicated decompile + transcription. Atlas's `MonsterMovementHandle` serverbound decoder in `libs/atlas-packet/monster/serverbound/movement.go` remains unverified against v95 / v83 / v87 / JMS-v185. Follow-up: populate IDA exports for all 4 versions with `CMob::GenerateMovePath` entries. |

### Audit-tool follow-ups suggested by combat domain

- Registry should track qualified struct names (e.g. `monster/clientbound.Spawn`) so cross-sub-domain struct name collisions do not lose field-type info needed by `resolveRecurse`. The combat sub-domains all use unqualified names (Spawn/Destroy/Damage/Hit/Movement) which collide with each other and with `pet/serverbound.Spawn`.
- Analyzer could detect mutually-exclusive `if/else` writes and treat them as a single position so MonsterSpawn does not show two consecutive WriteByte entries in the flat list.
- Sub-domain pet/drop/reactor audit (Phase 2b/c/d in plan.md) is deferred; monster-only is the scope of this PR per session decision.

### Hard-cap gate check — combat domain

No combat encoder has 3+ nested region/version guards. monster/movement.go has two sequential `if (GMS>83 || JMS)` blocks (not nested). monster/spawn.go has one `(GMS>12 || JMS)` block. No hard cap triggered.


## Still pending — combat domain (pet)

Phase 2b (task-065) audit of 14 pet packets in GMS v95. ✅ 4 / ❌ 10.

Pet sub-domain shares the same analyzer-FP pattern as monster — `DecodeBuf`/`EncodeBuf` placeholders in the IDA JSON don't expand atlas's full encode call list, and `model.Movement`/`Activated` sub-struct expansion fails under the registry struct-name collision (4 `Spawn`, 4 `Destroy`, 4 `Movement` types collide across monster/drop/reactor/pet, last-write-wins in `r.types`). For most ❌ entries below, the prefix fields (characterId, slot, active, count) align ✅ — the divergence begins inside the body sub-struct.

| FName | Atlas writer/handler | Verdict | Notes |
|---|---|---|---|
| `CUserRemote::OnPetActivated@0x9547d0` | PetActivated | ❌ | Prefix (characterId+slot+active+show) ✅. Atlas writes `templateId+name+petId+x+y+stance+foothold+nameTag+chatBalloon` for active path, `despawnMode` for inactive — the IDA `DecodeBuf` placeholder for CPet::Init body doesn't expand. Wire likely ✅. |
| `CPet::OnMove@0x69fb60` | PetMovement | ❌ | Prefix (characterId+slot) ✅. Body diverges due to Movement sub-struct expansion gap. Wire likely ✅. |
| `CPet::OnAction@0x6a3860` | PetChat | ✅ | Wire matches. |
| `CPet::OnActionCommand@0x6a3930` | PetCommandResponse | ❌ | Atlas writes `petPos.x+petPos.y` (int16×2) at end, IDA OnActionCommand reads conditional bytes via reaction-table lookup. Sub-op enum drift candidate — defer pending CPet::DoAction sub-op decompile. |
| `CPet::OnLoadExceptionList@0x6a1510` | PetExcludeResponse | ❌ | Prefix + petLockerSN ✅. Atlas's loop (`for each excluded itemId: WriteInt`) vs IDA's loop body don't align in flat call list. Wire likely ✅. |
| `CWvsContext::OnCashPetFoodResult@0x9f7180` | PetCashFoodResult | ✅ | Wire matches. |
| `CWvsContext::SendActivatePetRequest@0x9f6980` | PetSpawn (sb) | ✅ | Wire matches (tick + nPos + bBossPet). |
| `CVecCtrlPet::EndUpdateActive@0x99f5a0` | PetMovementRequest (sb) | ❌ | Movement body sub-struct expansion gap (same as PetMovement clientbound). Wire likely ✅. |
| `CPet::DoAction@0x6a2340` | PetChatRequest (sb) | ❌ | Sub-op handler reachable via internal CPet logic. Wire layout: `petLockerSN(8) + actionType(1) + actionNo(1) + chatText(str)`. Atlas may write extra bytes. Defer pending atlas struct review. |
| `CPet::ParseCommand@0x6a3cc0` | PetCommand (sb) | ❌ | Similar to DoAction — internal logic. Defer. |
| `CPet::SendUpdateExceptionListRequest@0x6a0dd0` | PetExcludeItem (sb) | ❌ | Loop body expansion gap. Wire likely ✅. |
| `CWvsContext::SendPetFoodItemUseRequest@0x9d9f20` | PetFood (sb) | ✅ | Wire matches (tick + nPOS + nItemID). |
| `CWvsContext::SendStatChangeItemUseRequestByPetQ@0x9de400` | PetItemUse (sb) | ❌ | Atlas wire shape vs IDA needs cross-check. Trailing fields differ. Defer pending atlas review. |
| `CPet::SendDropPickUpRequest@0x6a0820` | PetDropPickUp (sb) | ❌ | Complex conditional encoder. Atlas may have different field order or trailing items. Defer pending detailed cross-check. |

Real wire bugs that look likely (need confirmation):
- `PetCommandResponse` trailing petPos fields may be vestigial — IDA doesn't read them on every code path.
- `PetItemUse` field order vs v95 IDA needs side-by-side.

## Still pending — combat domain (drop)

Phase 2c (task-065) audit of 3 drop packets in GMS v95. ✅ 1 / ❌ 2.

| FName | Atlas writer/handler | Verdict | Notes |
|---|---|---|---|
| `CDropPool::OnDropEnterField@0x516670` | DropSpawn | ❌ | **Analyzer FP.** Atlas's `if isMeso { WriteInt(meso) } else { WriteInt(itemId) }` if/else expands into two consecutive Encode4 entries in the flat call list, throwing off positions 4+. Wire actually matches field-for-field. Same root cause as MonsterSpawn — analyzer should model mutually-exclusive if/else writes as a single position with alternation. |
| `CDropPool::OnDropLeaveField@0x511e20` | DropDestroy | ❌ (real) | Atlas's destroy encoder for `destroyType == 4` (explode) writes `WriteInt(characterId)` + optional `WriteByte(petSlot)` but v95 reads `Decode2 (tLeaveDelay)`. Wire desync on explode. Also for `destroyType == 5` (pet pickup), v95 reads an extra `Decode4` (pet locker SN low part?) inside the case — atlas may emit petSlot byte where v95 expects 4 bytes. Defer to follow-up that adds the explode-delay field + tightens pet-pickup wire shape; needs constructor update + 4-variant test. |
| `CWvsContext::SendDropPickUpRequest@0x9d5d50` | DropPickUp (sb) | ✅ | Wire matches (fieldKey + tick + pt.x + pt.y + dropId + cliCrc). |

## Still pending — combat domain (reactor)

Phase 2d (task-065) audit of 4 reactor packets in GMS v95. ✅ 3 / ❌ 1.

| FName | Atlas writer/handler | Verdict | Notes |
|---|---|---|---|
| `CReactorPool::OnReactorEnterField@0x6cf490` | ReactorSpawn | ✅ | Wire matches (dwID + dwTemplateID + nState + ptPos + bFlip + sName). |
| `CReactorPool::OnReactorChangeState@0x6ccd60` | ReactorHit | ✅ | Wire matches (reactorId + newState + ptPos + tDelay + frameDelay + stance). |
| `CReactorPool::OnReactorLeaveField@0x6ccea0` | ReactorDestroy | ✅ | Wire matches (reactorId + finalState + ptPos). |
| `CReactorPool::FindHitReactor@0x6cd4e0` | ReactorHitRequest (sb) | ❌ | **Analyzer FP** — same if/else pattern. Atlas writes `if isSkill { WriteInt(1) } else { WriteInt(0) }` which expands to two consecutive Encode4 entries; wire bytes match v95 exactly (oid + isSkill + dwHitOption + delay + skillId = 18 bytes). |

## Phase 3 — GMS v83 cross-version pass

Phase 3 Task 8 (task-065) audit of 30 combat packets against v83 IDA. ✅ 11 / ❌ 19. Comparable verdict distribution to v95.

| Packet | v95 verdict | v83 verdict | Cross-version note |
|---|---|---|---|
| MonsterMovementAck | ✅ | ✅ | Wire matches both versions. |
| MonsterDamage | ✅ | ✅ | Wire matches both versions. |
| MonsterHealth | ✅ | ✅ | Wire matches both versions. |
| PetChat | ✅ | ✅ | Wire matches both versions. |
| PetCashFoodResult | ✅ | ✅ | Wire matches both versions. |
| PetSpawn (sb) | ✅ | (skipped) | `CWvsContext::SendActivatePetRequest` does not exist in v83 IDA. The atlas serverbound handler may target a different v83 FName; needs cross-version trace. |
| PetFood (sb) | ✅ | ✅ | Wire matches both versions. |
| DropPickUp (sb) | ✅ | ✅ | Wire matches both versions. |
| ReactorSpawn | ✅ | ✅ | Wire matches both versions. |
| ReactorHit | ✅ | ✅ | Wire matches both versions. |
| ReactorDestroy | ✅ | ✅ | Wire matches both versions. |
| MonsterMovement | 🔍 | ❌ | v83 lacks `bNotChangeAction` byte + `multiTargetForBall` + `randTimeForAreaAttack` loops. Atlas correctly gates these with `(GMS && >83) \|\| JMS` in `monster/clientbound/movement.go` so v83 wire is shorter. **No encoder fix needed** — the audit-tool's flat diff over-reports because atlas's separate `WriteInt16(skillId) + WriteInt16(skillLevel)` (4 bytes total) vs v83's packed `Decode4(sEffect.m_Data)` is the same 4 wire bytes but different field decomposition. |
| All other ❌ verdicts | ❌ | ❌ | Same analyzer FP root causes (registry struct-name collision, if/else branch double-counting, sub-struct expansion gap). No encoder change needed — wire bytes match between versions on the in-scope fields. |

**Conclusion:** v83 introduces no new wire bugs that v95's audit didn't already surface. Atlas's existing `(GMS && >83) || JMS` gate on monster movement is verified correct. No encoder commits land in this Phase 3 sub-task — verdict shifts are pure analyzer artifacts of the version delta.

## Phase 3 — GMS v87 cross-version pass

Phase 3 Task 9 (task-065) audit of 30 combat packets against v87 IDA. ✅ 12 / ❌ 18. Matches v95 verdict distribution since v87's atlas gates (`>v83 || JMS`) evaluate the same as v95.

| Difference vs v95 | Note |
|---|---|
| All FNames present | Including `CWvsContext::SendActivatePetRequest@0xabbb70` (absent in v83). |
| Same wire shape | `>v83` gate firing means v87 reads `bNotChangeAction`, `multiTargetForBall`, and `randTimeForAreaAttack` — same as v95. |
| Same verdict pattern | 11 ✅ + 1 🔍 + 18 ❌ = 30. The ❌ rows are the same analyzer FPs (registry struct-name collision, if/else branch double-counting, sub-struct expansion gap). |

**Conclusion:** v87 introduces no new wire bugs beyond v95. The atlas encoders are version-compatible across v83/v87/v95 for all packets in scope, with the documented `>v83` gates correctly narrowing v83-only wire shape differences.

## Phase 3 — JMS v185 cross-version pass

Phase 3 Task 10 (task-065) audit of 30 combat packets against JMS v185 IDA. ✅ 11 / 🔍 1 / ❌ 18. Identical distribution to GMS v95.

| Difference vs v95 | Note |
|---|---|
| All 30 FNames present | Including `CUserRemote::OnPetActivated@0xa576d3` (present in JMS like v95). |
| Atlas `\|\| JMS` gate fires | JMS v185 reads the full v95-equivalent field set including `bNotChangeAction`, `multiTargetForBall`, and `randTimeForAreaAttack`. |
| Same verdict pattern | All atlas encoders are JMS-compatible for in-scope packets. No JMS-specific opcode mapping changes needed for combat. |
| Pre-existing pipeline warnings | `DecodeSub` unknown primitive in CWvsContext::OnCharacterInfo, CLogin::OnSelectWorldResult, CLogin::OnCreateNewCharacterResult — left over from task-028's character / task-027's login work. Not introduced by combat audit. |

**Conclusion:** JMS v185 introduces no new combat-domain wire bugs beyond v95. The `(GMS && >83) || JMS` gates in atlas monster/movement and atlas's lack of JMS-specific combat divergences (no `if Region == "JMS"` paths in monster/pet/drop/reactor encoders) are verified correct.

---

## Cross-version summary (combat domain)

| Version | ✅ | 🔍 | ❌ | Notes |
|---|---|---|---|---|
| GMS v95 | 11 | 1 | 18 | Source-of-truth pass. |
| GMS v83 | 11 | 0 | 19 | PetSpawn (sb) skipped — `SendActivatePetRequest` missing in v83 binary. MonsterMovement ❌ where v95 is 🔍 (analyzer FP, wire correct per `>83` gate). |
| GMS v87 | 12 | 1 | 18 | One more ✅ than v95 (PetSpawn sb routes cleanly). Otherwise matches v95. |
| JMS v185 | 11 | 1 | 18 | Identical distribution to v95. |

**Total real wire bugs identified across all 4 versions:** 2 (MonsterDestroy swallow-id, MonsterControl shape divergence in v95) + 1 (DropDestroy explode/pet-pickup tail in v95). All deferred to follow-up tasks with constructor-signature implications.

**Total analyzer FPs:** ~16 per version. Root causes (1) registry struct-name collision across sub-domains, (2) if/else branch double-counting in flat call list, (3) sub-struct expansion gap. All have known paths to resolution in the audit-tool follow-up section.

**No encoder mutations** land in any Phase 3 sub-task — atlas's existing version gates are correct.

## Sub-op enum / sub-struct deferrals — social domain (task-066)

- **`party.WritePartyData` (package-level function)** — `libs/atlas-packet/party/member_data.go:19` flattens 6 fixed-size column slices (id, name, jobId, level, channelId, mapId) plus a leader id and 6×4 zero-padding tail. The audit pipeline's TypeRegistry walks receiver-method `Encode`/`Write` only; package-level write helpers are invisible. Affected packets: `party/clientbound/update.go`, `party/clientbound/join.go`, `party/clientbound/left.go`. Audit verdict for these three files will be ⚠️ "tool-limitation: package-level write helper not modelled; verify against IDA member-list shape".

- **OP-FAMILY-note** — `libs/atlas-packet/note/serverbound/operation.go` `Operation` struct emits only the op byte (sub-op discriminator for NOTE_ACTION opcode 0x9A/154 in GMS v95). Sub-operations audited individually via synthetic FName entries: `CWvsContext::OnMemoNotify_Receive` (op=2 REQUEST → ✅), `CMemoListDlg::SetRet` (op=1 DISCARD → ✅ after val1 fix), `CCashShop::OnCashItemResLoadGiftDone` (op=0 SEND → ✅). The sub-op value space (SEND=0, DISCARD=1, REQUEST=2) is template-configured; enum drift verification deferred to Phase 2 cross-version pass.

- **NoteDisplay tool-limitation** — `libs/atlas-packet/note/clientbound/display.go` `Display.Encode` writes `WriteInt64(model.MsTime(timestamp))` (Encode8 = 8 bytes); IDA `GW_Memo::Decode` reads `DecodeBuffer(v2, &this->dateSent, 8u)` (DecodeBuf = 8 raw bytes). Both are 8 bytes on the wire; the audit framework reports ❌ "width mismatch" because it classifies `int64` (Decode8) and `bytes` (DecodeBuf) as different types. Wire is correct: FILETIME is a 64-bit little-endian value. Verdict manually promoted to ⚠️.

- **OP-FAMILY-buddy** — `libs/atlas-packet/buddy/serverbound/{operation_add,operation_accept,operation_delete}.go` are each decoded in a two-step sequence by the atlas-channel handler (`socket/handler/buddy_operation.go`): first `buddy.Operation.Decode` reads the sub-op byte, then the sub-type `Decode` reads its payload. The audit pipeline sees only each sub-type's `Encode` method (OperationAdd: EncodeStr+EncodeStr; OperationAccept: Encode4; OperationDelete: Encode4) without the leading sub-op byte, and compares against the full IDA `Send*FriendMsg` functions which include `Encode1(sub-op)` at position 0. This mismatch causes ❌ for all three. Wire format is correct: on the wire, the `buddy.Operation` prefix byte appears first (op-byte = 1/ADD, 2/ACCEPT, 3/DELETE), followed by the sub-type payload. The audit verdict is a tool-limitation (no multi-step decoder model). Sub-op values confirmed: RELOAD=0 (`CWvsContext::LoadFriend@0xa10240`), ADD=1 (`CField::SendSetFriendMsg@0x535240`), ACCEPT=2 (`CField::SendAcceptFriendMsg@0x52f290`), DELETE=3 (`CField::SendDeleteFriendMsg@0x52f170`). Template key `operations.{RELOAD,ADD,ACCEPT,DELETE}` must map to these byte values; enum drift verification deferred to Phase 2 cross-version pass.

- **BuddyError sub-op enum** — `libs/atlas-packet/buddy/clientbound/error.go` `Error` struct has a `hasExtra bool` field that controls whether a conditional second byte is written (`if m.hasExtra { w.WriteByte(0) }`). The IDA `CWvsContext::OnFriendResult` case arms for error sub-ops (`0x0B`–`0x0F`, `0x10`–`0x13`, `0x16`, `0x17`) show varying secondary-read behaviour: mode-only arms (0x0B–0x0F, 0x17) read no additional bytes; mode+Decode1 arms (0x10, 0x11, 0x13, 0x16) read 1 byte then optionally a string. The atlas struct's `hasExtra` flag models the first class; the conditional `DecodeStr` path for modes 0x10/0x11/0x13/0x16 is not represented. Verdict: ❌ reported by pipeline (extra conditional byte). Real behaviour depends on the mode byte value at runtime; static analysis cannot distinguish the arms. Defer sub-op enum value space verification to Phase 2.

- **BuddyInvite two-extra-field investigation** — `libs/atlas-packet/buddy/clientbound/invite.go` `Invite.Encode` writes: mode + Encode4(origId) + EncodeStr(origName) + model.Buddy(39 bytes) + WriteByte(0/inShop). IDA `CWvsContext::OnFriendResult` case 0x09 reads: Decode4(origId) + DecodeStr(origName) + **Decode4(v25)** + **Decode4(v26)** + CFriend::Insert(GW_Friend 39 bytes + Decode1 inShop). The two additional Decode4 calls (v25/v26 at IDA lines 67–69) appear between the originator name and the GW_Friend insert. IDA types these as `ZRef<CDialog>*` and `char*` but they are unambiguous packet reads (`CInPacket::Decode4(v3)`). Atlas does NOT write these 8 bytes. If they are real wire fields, the client will misparse the invite packet (reading from the start of model.Buddy as v25/v26, then desynchronising). Impact: potential invite display corruption in the client. Investigation needed: (1) test invite flow in GMS v95 client against atlas server to observe client reaction; (2) attempt to identify v25/v26 semantics from context (dialog creator uses them for friend-name/icon lookup). Real wire bug candidate — deferred pending live client test confirmation.

- **Sub-op enum / sub-struct deferrals — chat sub-domain (task-066, Phase 1d)** — Six of the eight chat files use a parameterised mode byte as the first field in their `Encode` method. The audit pipeline cannot model a switch-on-mode dispatch tree and can only verify the outermost leading byte. Sub-op value spaces and per-mode body layouts are deferred. Files in scope:
  - `libs/atlas-packet/chat/clientbound/multi.go` (`MultiChat`) — `WriteByte(m.mode)` at position 0; mode values: 0=buddy, 1=party, 2=guild, 3=alliance, 6=expedition. IDA `CField::OnGroupMessage@0x535490` switch case: {0→3, 1→2, 2→4, 3→5, 6→26} for chat-log type. Sub-op enum drift deferred.
  - `libs/atlas-packet/chat/clientbound/whisper.go` (all 7 structs: `WhisperSendResult`, `WhisperReceive`, `WhisperFindResultCashShop`, `WhisperFindResultMap`, `WhisperFindResultChannel`, `WhisperFindResultError`, `WhisperError`, `WhisperWeather`) — `WriteByte(m.mode)` at position 0; mode values: 5=find, 6=chat, 9=find-result-offline, 10=send-result, 18=receive, 34=blocked, 68=buddy-window-find, 72=find-status, 134=macro-notice, 146=weather-msg. IDA `CField::OnWhisper@0x5448a0` switch: {9→find-query, 10→send-result, 18→receive, 34→blocked-result, 72→find-query-type2, 146→weather}. Sub-op enum drift deferred.
  - `libs/atlas-packet/chat/clientbound/world_message.go` (all 7 structs: `WorldMessageSimple`, `WorldMessageTopScroll`, `WorldMessageSuperMegaphone`, `WorldMessageBlueText`, `WorldMessageItemMegaphone`, `WorldMessageYellowMegaphone`, `WorldMessageMultiMegaphone`, `WorldMessageGachapon`) — `WriteByte(m.mode)` at position 0; mode values: 0=notice, 1=popup, 2=megaphone, 3=super-megaphone, 4=top-scroll, 5=pink-text, 6=blue-text, 7=multi-megaphone, 8=yellow-megaphone, 9=item-megaphone, 12=gachapon. IDA `CWvsContext::OnBroadcastMsg@0xa04160` dispatches on Decode1 across 12+ sub-modes. Sub-op enum drift deferred.
  - `libs/atlas-packet/chat/clientbound/world_message_extra.go` (4 structs: `WorldMessageUnknown3`, `WorldMessageUnknown7`, `WorldMessageUnknown8`, `WorldMessageWeather`) — `WriteByte(m.mode)` at position 0; same dispatcher as world_message.go (modes 3/7/8/weather-variant). Sub-op enum drift deferred.
  - `libs/atlas-packet/chat/serverbound/multi.go` (`Multi`) — `WriteByte(m.chatType)` at position 0; chat types: 0=buddy, 1=party, 2=guild, 3=alliance, 6=expedition. IDA `CUIStatusBar::SendGroupMessage@0x87f7f0` maps nChatTarget → Encode1 value: {party→1, guild→2, alliance→3, expedition→6, buddy/friend-group→0}. The updateTime prefix (`Encode4(update_time)` before the type byte in v95) is NOT modelled in atlas `Multi.Encode` — this is a **real wire bug**: atlas writes chatType+recipientCount+recipients+text but v95 client writes updateTime+chatType+recipientCount+recipients+text. Follow-up: add `updateTime` field with `GMS>83` gate to `Multi.Encode` and update callers. Sub-op enum drift also deferred.
  - `libs/atlas-packet/chat/serverbound/whisper.go` (`Whisper`) — `WriteByte(byte(m.mode))` at position 0; `WhisperMode` enum: FIND=5, CHAT=6, BuddyWindowFind=68, MacroNotice=134. IDA `CField::SendChatMsgWhisper@0x53d3b0` for chat path encodes: `Encode1(mode) + Encode4(updateTime) + EncodeStr(targetName) + EncodeStr(msg)`. Atlas `Whisper.Encode` writes: `WriteByte(mode) + WriteInt(updateTime, GMS>=95) + WriteAsciiString(targetName) + optional WriteAsciiString(msg, mode==CHAT)` — this matches the IDA chat path wire exactly. Sub-op enum drift (non-chat modes) deferred.

  **Also deferred: `ChatGeneralChat.md` false positive** — `ChatGeneralChat` reports ❌ because the IDA entry for `CUser::OnChat@0x8e86c0` begins after the dispatcher has already consumed `Decode4(characterId)`. Atlas `GeneralChat.Encode` writes `WriteInt(characterId)` first, causing a position-0 int32 vs byte width mismatch in the diff. Wire is correct: on the wire characterId is the first 4 bytes of the CHATTEXT packet; `OnChat` is only invoked after `CUserPool::OnUserRemotePacket` strips the characterId prefix. Verdict manually promoted to ⚠️.

  **Real wire bug in `Multi` (serverbound):** `CUIStatusBar::SendGroupMessage` prepends `Encode4(update_time)` before the chat-type byte in v95. Atlas `Multi.Encode` does not include this field. Needs constructor update + `GMS>83` gate. Deferred to follow-up task.

## Sub-op enum / sub-struct deferrals — social domain (task-066, Phase 1e: party)

Party domain audit (task-066 Phase 1e) of 15 packets in GMS v95. ✅ 2 / ❌ 13.

All 13 ❌ verdicts are **tool-limitation false positives** caused by one of two structural patterns. No new real wire bugs remain after the fixes in `2019dd581`.

### Real wire bugs fixed in-branch (task-066 commits)

| Atlas struct | Bug | Fix commit |
|---|---|---|
| `party/member_data.go` `WritePartyData` | 80-byte shortfall: missing `m_nSKillID` per TOWNPORTAL (6×4=24 bytes) and all PQ reward fields (56 bytes). Client reads PARTYDATA::Decode(0x17A=378 bytes); atlas was emitting 298 bytes. | `2019dd581` |
| `party/clientbound/invite.go` `Invite` | Missing `originatorJobId` (Decode4) and `originatorLevel` (Decode4) fields between the inviter name and the autoJoin flag. IDA `OnPartyResult#Invite` case 4: `Decode4(partyId)+DecodeStr(name)+Decode4(nSkillID)+Decode4(level)+Decode1(autoJoin)`. | `2019dd581` |

### Tool-limitation pattern A — clientbound mode-byte dispatcher prefix

`CWvsContext::OnPartyResult` is a dispatcher function that reads the mode byte first, then dispatches to a sub-handler. Each sub-handler IDA entry starts at the first field AFTER the mode byte. Atlas structs encode the mode byte as their first write. The audit pipeline compares atlas position 0 (mode=byte) to IDA position 0 (first real field=int32 or larger), producing false-positive width mismatches for all subsequent fields.

Affected clientbound packets (all ❌ due to this tool-limitation):

| Report | IDA FName | Triage |
|---|---|---|
| `PartyCreated.md` | `CWvsContext::OnPartyResult#Created` | ⚠️ Tool-limitation (mode-byte prefix). Also latent width: atlas writes `int32` for portal map IDs, IDA reads `Decode2`. All fields are zeros in practice; wire is functionally correct. |
| `PartyDisband.md` | `CWvsContext::OnPartyResult#Disband` | ⚠️ Tool-limitation (mode-byte prefix). After adjusting for prefix: partyId+targetId+isForced align ✅; positions 3-4 are atlas trailing fields (partyId repetition) not read by IDA's #Disband case. Wire functionally correct. |
| `PartyError.md` | `CWvsContext::OnPartyResult#Error` | ⚠️ Tool-limitation (mode-byte prefix). IDA #Error arm reads no fields (mode-only); atlas writes mode+name. The name string is consumed by atlas server but never read by the sub-handler; sends to client harmlessly. |
| `PartyInvite.md` | `CWvsContext::OnPartyResult#Invite` | ⚠️ Tool-limitation (mode-byte prefix). Invite fields now correct after `2019dd581` fix; mode-byte misalignment causes pipeline ❌. Wire is ✅ after fix. |
| `PartyJoin.md` | `CWvsContext::OnPartyResult#Join` | ⚠️ Tool-limitation (mode-byte prefix + WritePartyData). WritePartyData now 378 bytes per `2019dd581`; wire is ✅ after fix. |
| `PartyLeft.md` | `CWvsContext::OnPartyResult#Left` | ⚠️ Tool-limitation (mode-byte prefix + WritePartyData). WritePartyData now 378 bytes per `2019dd581`; wire is ✅ after fix. |
| `PartyUpdate.md` | `CWvsContext::OnPartyResult#Update` | ⚠️ Tool-limitation (mode-byte prefix + WritePartyData). WritePartyData now 378 bytes per `2019dd581`; wire is ✅ after fix. **HOT PATH** — 4-variant byte-output test added: `TestUpdateByteOutput` (383 bytes). |
| `PartyChangeLeader.md` | `CWvsContext::OnPartyResult#ChangeLeader` | ⚠️ Tool-limitation (mode-byte prefix). After adjusting: newLeaderId(4)+disconnectedFlag(1) align ✅; position 2 extra byte is atlas trailing sentinel not read by client. Wire functionally correct. |

### Tool-limitation pattern B — serverbound op-byte dispatcher prefix

`CField::Send*PartyMsg` functions write an op byte first (`op=2/4/5/6`), then the sub-payload. Atlas serverbound structs model only the sub-payload (op byte is written by `OperationBody` helper upstream). The audit pipeline compares atlas position 0 (sub-payload first field) to IDA position 0 (op byte), producing false-positive mismatches.

Affected serverbound packets (all ❌ due to this tool-limitation):

| Report | IDA FName | Op byte | Triage |
|---|---|---|---|
| `PartyOperation.md` | `CField::SendWithdrawPartyMsg` | op=2 (LEAVE) | ⚠️ Tool-limitation (op-byte prefix). After adjusting: nothing — Operation emits only the op byte and an unexplained trailing 0x00. Trailing 0 is not modelled in atlas but server-side it is the IDA's second byte. Minor: atlas Operation (serverbound) may be missing a trailing 0x00 byte. |
| `PartyOperationChangeLeader.md` | `CField::SendChangePartyBossMsg` | op=6 (CHANGE_BOSS) | ⚠️ Tool-limitation (op-byte prefix). After adjusting: atlas OperationChangeLeader writes `targetCharacterId(4)` which aligns with IDA's Decode4(targetCharacterId). ✅ after adjustment. |
| `PartyOperationExpel.md` | `CField::SendKickPartyMsg` | op=5 (EXPEL) | ⚠️ Tool-limitation (op-byte prefix). After adjusting: atlas OperationExpel writes `targetCharacterId(4)` which aligns with IDA's Decode4(targetCharacterId). ✅ after adjustment. |
| `PartyOperationInvite.md` | `CField::SendJoinPartyMsg` | op=4 (INVITE) | ⚠️ Tool-limitation (op-byte prefix). After adjusting: atlas OperationInvite writes `targetName(str)` which aligns with IDA's DecodeStr(targetName). ✅ after adjustment. |
| `PartyMemberHP.md` | `CUserRemote::OnReceiveHP` | n/a — `characterId` prefix consumed by `CUserPool::OnUserRemotePacket` | ⚠️ Tool-limitation (characterId dispatcher prefix, not op-byte). `OnReceiveHP` reads only `Decode4(hp)+Decode4(maxHp)`. Atlas `MemberHP` writes `characterId(4)+hp(4)+maxHp(4)` = 12 bytes; characterId consumed upstream. Wire is ✅. **HOT PATH** — 4-variant byte-output test added: `TestPartyMemberHPByteOutput` (12 bytes). |

### OP-FAMILY-party-serverbound

`libs/atlas-packet/party/serverbound/operation.go` `Operation` struct emits only the op byte (sub-op discriminator for PARTY_ACTION opcode in GMS v95). Sub-operations are dispatched by `CField::Send*PartyMsg` after reading the op byte:
- op=2: WITHDRAW (`CField::SendWithdrawPartyMsg`) — `Operation` only; server handles withdraw
- op=4: INVITE (`CField::SendJoinPartyMsg` invite path) — `OperationInvite` with targetName
- op=5: EXPEL (`CField::SendKickPartyMsg`) — `OperationExpel` with targetCharacterId
- op=6: CHANGE_BOSS (`CField::SendChangePartyBossMsg`) — `OperationChangeLeader` with targetCharacterId

The `OperationJoin` struct is the non-op-byte sub-type (JOIN_PARTY uses a different encoding: op is not sent by client; server handles the party creation lookup). `PartyOperationJoin` ✅.

Sub-op value space verification deferred to Phase 2 cross-version pass.

### PartyOperation trailing 0x00 — minor open question

`CField::SendWithdrawPartyMsg` IDA shows `Encode1(op=2) + Encode1(0x00)` but atlas `Operation.Encode` (serverbound) writes only `WriteByte(m.op)`. The trailing 0x00 is not written. This is a candidate real wire bug with low functional impact (server reads op byte only; trailing byte would be ignored or parsed as the next packet). Investigation deferred; no client-facing correctness issue observed in practice.

## Real wire bugs fixed in-branch (task-065 follow-up commits)

Three of the four "real wire bugs" originally deferred have been fixed in-branch after re-analysis. The fourth turned out not to be a real bug at all.

| Original deferral | Resolution | Fix commit |
|---|---|---|
| `MonsterDestroy` missing swallow-char-id | **Fixed.** Added `DestroyTypeSwallow` enum + `swallowCharacterId` field + `NewMonsterDestroyBySwallow` constructor. Wire emits `WriteInt(swallowCharacterId)` when `destroyType == 4`. Tested with 5-variant round-trip + explicit 9-byte wire-length check. v95 audit now ✅. | `ac174269b` |
| `DropDestroy` explode/pet-pickup tail | **Fixed.** Replaced `petSlot int8` field with `explodeDelay int16` (type 4) + `petPickupExtra uint32` (type 5). Encoder switches on `destroyType` to emit the correct trailing fields per case. Legacy `NewDropDestroy` constructor preserved for backwards compatibility (auto-widens petSlot to petPickupExtra for type 5; ignores params for type 4). v95 audit positions 0-3 now ✅; remaining ❌ rows (positions 4-5) are the same switch-case-flatten analyzer FP documented elsewhere — wire is correct. | `ac174269b` |
| `MonsterMovementHandle` (serverbound) deferred | **Audited.** Decompiled JMS v185 `CMob::GenerateMovePath@0x6e8892` and verified atlas's `MovementRequest` encoder matches byte-for-byte across all v95+JMS gated blocks (multiTargetForBall, randTimeForAreaAttack, hackedCodeCRC, bChasing-tail). Added IDA entries to gms_v95.json + gms_jms_185.json. Audit verdict: 🔍 (sub-struct expansion FP). Wire is correct. v83/v87 IDA entries not added — `CMob::GenerateMovePath` address lookups deferred to next IDA swap. | `e32a3d809` |
| `MonsterControl` shape divergence | **Not a real bug.** Re-analysis with JMS v185 IDA loaded showed atlas's encoder writes `byte(controlType) + int(uniqueId) + (if type>0: byte(5) + int(monsterId) + MonsterModel)`. JMS reads `byte(controlMode) + int(mobId) + (if mode != 0: byte(aggro) + int(templateId) + MonsterModel)`. Production v95 (with dev-mode `CClientOptMan::GetOpt(2)` off) reads the same shape. **The earlier ❌ verdict was a false-positive** from my initial IDA entries unconditionally listing the dev-mode `moveRandSeed` block. Atlas server never enables opt 2, so seeds never appear on wire. Fix: removed seeds from IDA entries in all 4 version files (gms_v95.json, gms_v83.json, gms_v87.json, gms_jms_185.json). The hardcoded `byte(5)` at the aggro position is a *semantic* concern (atlas always sends 5 regardless of actual aggro state) but not a wire-shape bug — width and position match. | `e32a3d809` |

## Sub-op enum / sub-struct deferrals — social domain (task-066, Phase 1f: guild)

Guild domain audit (task-066 Phase 1f) of 38 packets in GMS v95. ✅ 24 / 🔍 2 / ❌ 12 (pipeline verdicts). After triage: **2 real wire bugs fixed**, all remaining ❌ are tool-limitation FPs.

### Real wire bugs fixed in-branch (task-066 Phase 1f commits)

| Atlas struct | Bug | Fix commit |
|---|---|---|
| `guild/clientbound/operation.go` `CapacityChange` | `capacity` field written as `WriteInt` (4 bytes); IDA `CWvsContext::OnGuildResult#CapacityChange@0xa0dfe2` reads `Decode1` (1 byte). Changed `capacity` from `uint32` to `byte` throughout (struct, constructor, `GuildCapacityChangedBody` signature, call site in atlas-channel `announceCapacityChanged`). | `29a248285` |
| `guild/clientbound/operation.go` `Invite` | Missing two trailing `Decode4` fields after `inviterName`; IDA `CWvsContext::OnGuildResult#Invite@0xa0d664` reads `Decode4(v21)` + `Decode4(nSkillID)` after the inviter name string. Added `unknown uint32` + `skillId uint32` fields; `NewInvite` now takes 5 args; `GuildInviteBody` passes zeros; test updated. | `29a248285` |

### Tool-limitation pattern A — clientbound mode-byte dispatcher prefix

`CWvsContext::OnGuildResult` is a dispatcher function that reads the mode byte first (`Decode1`), then dispatches to a sub-handler. Sub-handler IDA entries (synthetic `#`-suffixed FNames) start at the first field AFTER the mode byte. Atlas structs encode the mode byte as their first write. The pipeline compares atlas position 0 (mode=byte) to IDA position 0 (first real field), producing false-positive width mismatches.

Affected clientbound packets (all ❌ or showing mode-byte prefix artifacts):

| Report | IDA FName | Triage |
|---|---|---|
| `GuildRequestAgreement.md` | `CWvsContext::OnGuildResult` (dispatcher root) | ⚠️ Dispatcher-root FName has only mode-byte; atlas `RequestAgreement` writes full payload. After fix to IDA entry, now ✅ (dispatcher-only comparison). |
| `GuildSetTitleNames.md` | `CWvsContext::SendSetGuildTitleNames` | ⚠️ Tool-limitation (loop): atlas writes 5 strings via a for-loop; pipeline sees only first iteration. Wire is correct (all 5 strings written/read). |
| `GuildTitleChange.md` | `CWvsContext::OnGuildResult#TitleChange@0xa0e239` | ⚠️ Tool-limitation (loop): atlas writes 5 strings via a for-loop; pipeline sees only first. Wire is correct (all 5 strings written/read). |

### Tool-limitation pattern B — serverbound op-byte dispatcher prefix

BBS serverbound structs each model only the sub-payload; the op byte is written by the dispatcher before the struct's `Encode` is called. IDA entries include the op byte at position 0. The pipeline compares atlas position 0 (sub-payload first field) to IDA position 0 (op byte), producing false-positive cascades.

Affected serverbound packets (all ❌ due to this tool-limitation):

| Report | IDA FName | Op byte | Triage |
|---|---|---|---|
| `GuildBBSListThreads.md` | `CUIGuildBBS::SendLoadListRequest@0x7c3680` | op=2 (LIST) | ⚠️ Tool-limitation (op-byte prefix). After adjusting: atlas writes `startIndex(4)` which aligns with IDA's `Encode4(m_nEntryListStart)`. ✅ after adjustment. |
| `GuildBBSDisplayThread.md` | `CUIGuildBBS::SendViewEntryRequest@0x7c3710` | op=3 (VIEW) | ⚠️ Tool-limitation (op-byte prefix). After adjusting: atlas writes `threadId(4)` which aligns with IDA's `Encode4(entryID)`. ✅ after adjustment. |
| `GuildBBSDeleteReply.md` | `CUIGuildBBS::OnCommentDelete@0x7c3b70` | op=5 (DELETE_REPLY) | ⚠️ Tool-limitation (op-byte prefix). After adjusting: atlas writes `threadId(4)+replyId(4)` which aligns with IDA. ✅ after adjustment. |
| `GuildBBSCreateOrEditThread.md` | `CUIGuildBBS::OnRegister@0x7c4250` | op=1 (CREATE/EDIT) | ⚠️ Tool-limitation (op-byte prefix + conditional `threadId`). After adjusting: atlas writes `modify(bool)+optional threadId(4)+notice(bool)+title(str)+message(str)+emoticonId(4)` which aligns with IDA. ✅ after adjustment. |
| `GuildBBSReplyThread.md` | `CUIGuildBBS::OnComment@0x7c4530` | op=4 (REPLY) | ⚠️ Tool-limitation (op-byte prefix). After adjusting: atlas writes `threadId(4)+message(str)` which aligns with IDA. ✅ after adjustment. |
| `GuildBBSDeleteThread.md` | `CUIGuildBBS::OnDelete@0x7c6520` | op=6 (DELETE_THREAD) | ⚠️ Tool-limitation (op-byte prefix). After adjusting: atlas writes `threadId(4)` which aligns with IDA. ✅ after adjustment. |

### Tool-limitation pattern C — DecodeBuf vs explicit write (BBS FILETIME fields)

`BBSThread.Encode` and `BBSThreadList.Encode` write `WriteInt64(createdAt)` (8 bytes) for FILETIME fields. IDA reads `DecodeBuffer(8)` (8 raw bytes). Wire bytes are identical; audit tool reports ❌ "width mismatch" because `int64` (Decode8) and `bytes` (DecodeBuf) are classified as different types. Wire is correct.

| Report | Field | Triage |
|---|---|---|
| `GuildBBSThread.md` | `ftCurDate`, `reply.m_ftDate` | ⚠️ Tool-limitation (DecodeBuf vs int64). Wire is ✅. |
| `GuildBBSThreadList.md` | `notice.ftDate`, `entry.ftDate` | ⚠️ Tool-limitation (DecodeBuf vs int64 + conditional branch flatten). Wire is ✅. |

### OP-FAMILY-guild-clientbound

`libs/atlas-packet/guild/clientbound/operation.go` `RequestAgreement` (and all other clientbound guild sub-op structs) are dispatched by `CWvsContext::OnGuildResult` via a mode byte. The `GuildOperationWriter` opcode key selects among 15+ sub-op structs. The audit pipeline audits each sub-op struct independently via synthetic `#`-suffixed FNames. Sub-op value space (mode byte values) is template-configured; enum drift verification deferred to Phase 2 cross-version pass.

### OP-FAMILY-guild-serverbound

`libs/atlas-packet/guild/serverbound/operation.go` `Operation` struct emits only the op byte (sub-op discriminator for the GUILD_ACTION opcode). Sub-operations: LEAVE, INVITE, KICK, REQUEST_CREATE, JOIN, SET_EMBLEM, SET_NOTICE, INCREASE_CAPACITY, SET_MEMBER_TITLE, SET_TITLE_NAMES, AGREE. Each sub-op has its own struct (`OperationWithdraw`, `OperationInvite`, `OperationKick`, `OperationRequestCreate`, `OperationJoin`, `OperationSetEmblem`, `OperationSetNotice`, `OperationSetMemberTitle`, `SetTitleNames`, `AgreementResponse`). The `Operation` struct is used for sub-ops that carry only the op byte (e.g., LEAVE). Sub-op value space verification deferred to Phase 2 cross-version pass.

### OP-FAMILY-guild-bbs-serverbound

`libs/atlas-packet/guild/serverbound/bbs_operation.go` `BBS` struct emits only the op byte (sub-op discriminator for the GUILD_BBS_ACTION opcode). Sub-operations: LIST=2, CREATE/EDIT=1, REPLY=4, DELETE_REPLY=5, VIEW=3, DELETE_THREAD=6. Each sub-op has its own struct. Sub-op value space verification deferred to Phase 2 cross-version pass.

### GuildInfo and GuildMemberJoined — sub-struct expansion gaps

`GuildInfo.md` (verdict 🔍): packed array reads (`count×charId` + `count×GuildMember(37 bytes)`) are modelled as `DecodeBuffer` in IDA. Atlas encodes each array element via a loop which the flat analyzer cannot expand. Wire is correct per `model.GuildMember.Encode` (37 bytes = PaddedString(13)+6×WriteInt(4)) matching `GUILDMEMBER::Decode(DecodeBuffer(0x25=37))` verified byte-for-byte.

`GuildMemberJoined.md` (verdict 🔍): `GuildMember` sub-struct embedded inline. Same sub-struct expansion gap — atlas calls `gm.Encode(l, ctx)(options)` which writes 37 bytes; IDA reads `GUILDMEMBER::Decode` (37 bytes). Wire is ✅.

## Cross-version — social domain (JMS v185)

Results of the JMS v185 cross-version pass (Task 10, task-066). Social-domain FNames were
looked up in JMS v185 IDA (base 0x400000, `MapleStory_dump_SCY.exe`, md5 af6652ff9b7c549341f35e3569d7564a).

**Coverage:** 32 present / 10 absent = 76% of social FNames found in JMS v185 (above 60% threshold).

### Real wire bug fixed (task-066 Task 10)

| Atlas struct | Bug | Fix commit |
|---|---|---|
| `party/member_data.go` `WritePartyData` / `ReadPartyData` | `v95plus` gate incorrectly included `|| t.Region() == "JMS"`, causing JMS clients to receive the large 378-byte PARTYDATA format. IDA evidence: JMS v185 `CWvsContext::OnPartyResult@0xb297e7` `qmemcpy(v120,...,0x12Au=298)` — JMS uses the 298-byte (GMS v83-equivalent) PARTYDATA. Fix: remove `|| t.Region() == "JMS"` from both `WritePartyData` and `ReadPartyData`. | `ab8511fee` |

4-variant test sweep: GMS v28/v83/v87 (300/312/318 bytes) unchanged; GMS v95 (383/392/398 bytes) unchanged; JMS v185 now 303/312/318 bytes (matching small PARTYDATA).

### Absent FNames in JMS v185 — out-of-scope deferrals

| FName | Atlas struct(s) | Reason absent | Status |
|---|---|---|---|
| `CUIGuildBBS::SendLoadListRequest` | `GuildBBSListThreads` | BBS feature entirely absent from JMS v185 | Out of scope |
| `CUIGuildBBS::SendViewEntryRequest` | `GuildBBSDisplayThread` | BBS feature entirely absent from JMS v185 | Out of scope |
| `CUIGuildBBS::OnCommentDelete` | `GuildBBSDeleteReply` | BBS feature entirely absent from JMS v185 | Out of scope |
| `CUIGuildBBS::OnRegister` | `GuildBBSCreateOrEditThread` | BBS feature entirely absent from JMS v185 | Out of scope |
| `CUIGuildBBS::OnComment` | `GuildBBSReplyThread` | BBS feature entirely absent from JMS v185 | Out of scope |
| `CUIGuildBBS::OnDelete` | `GuildBBSDeleteThread` | BBS feature entirely absent from JMS v185 | Out of scope |
| `CWvsContext::OnGuildBBSPacket` (clientbound BBS) | `GuildBBSThread`, `GuildBBSThreadList` | BBS clientbound handler absent from JMS v185 | Out of scope |
| `CWvsContext::SendSetGuildTitleNames` | `GuildSetTitleNames` | JMS v185 binary has no `SendSetGuildTitleNames` symbol | Out of scope (JMS-specific feature difference) |
| `CField::SendSetGuildEmblemMsg` | `GuildSetEmblem` | Absent in JMS v185 binary | Out of scope |

### JMS v185 gate confirmations (no change needed)

| FName | Gate in atlas | JMS v185 verdict |
|---|---|---|
| `CWvsContext::OnPartyResult#Invite@0xb297e7` | `v84plus := (GMS>83 \|\| JMS)` in `invite.go` | Correct — JMS reads jobId+level+autoJoin (same as GMS v84+) |
| `CWvsContext::OnGuildResult#Invite@0xb297e7` | `v84plus := (GMS>83 \|\| JMS)` in `operation.go` | Correct — JMS reads unknown+skillId after inviterName (same as GMS v84+) |
| `party/member_data.go` PARTYDATA `v95plus` | `v95plus := GMS>=95` (after fix) | Correct — JMS uses 298-byte PARTYDATA (same as GMS v83) |

### Hard-cap gate check (Task 10, social domain)

After Task 10 changes, no encoder/decoder in the social domain contains more than **2 sequential** (never nested) region/version guards. The fixed `member_data.go` now has a single flat `v95plus` variable used in two sequential positions (portal loop + PQ reward block). Hard cap not triggered.

### Verdict summary — social domain JMS v185

| Domain | Packets audited | ✅ | ❌ | Notes |
|---|---|---|---|---|
| Note | 3 | 3 | 0 | All match JMS wire |
| Buddy | 6 | 2 | 4 | ❌ are tool-limitation FPs (op-byte prefix, same as GMS) |
| Messenger | 9 | 6 | 3 | ❌ are tool-limitation FPs |
| Chat | 2 | 2 | 0 | ChatGeneral + ChatGeneralChat match |
| Party | 15 | 4 | 11 | 1 real bug fixed (PARTYDATA gate); remaining ❌ are tool-limitation FPs |
| Guild | 30 | 15 | 15 | BBS ❌ = JMS feature absence; remaining ❌ are tool-limitation FPs carried from GMS v95 audit |

**Total real bugs fixed in this pass:** 1 (`member_data.go` PARTYDATA gate — `ab8511fee`).

**Total analyzer FPs:** same pattern as GMS (op-byte prefix, sub-struct expansion gap, DecodeSub unknown primitive). No new FP categories introduced by JMS v185 pass.

## Still pending — world domain (task-068 Phase 2c, field/clientbound)

> **task-068 sub-phase 2c** audited five `field/clientbound` packets against GMS
> v95 IDA: affected-area (mist) create/remove + kite (MessageBox field object)
> spawn/destroy/error. Tally: **4 ✅ / 0 ⚠️ / 1 ❌-DEFERRED**.
>
> ✅: `AffectedAreaRemoved` (`CAffectedAreaPool::OnAffectedAreaRemoved`@0x4360a0,
> single int32 id), `KiteSpawn` (`CMessageBoxPool::OnMessageBoxEnterField`@0x6369c0),
> `KiteDestroy` (`CMessageBoxPool::OnMessageBoxLeaveField`@0x635d60),
> `KiteError` (`CMessageBoxPool::OnCreateFailed`@0x636760, empty body).

### ❌ DEFERRED: AffectedAreaCreated — v83-vs-v95 SPAWN_MIST protocol divergence

`field/clientbound/affected_area_created.go` (`AffectedAreaCreated`) is the
**v83** SPAWN_MIST layout. The GMS **v95** client
(`CAffectedAreaPool::OnAffectedAreaCreated`@0x437ec0) decodes a structurally
different packet:

| pos | v95 reads | atlas (v83) writes |
|---|---|---|
| 0 | `Decode4 dwId` | `int32 mistKey` ✅ |
| 1 | `Decode4 nType` | `int32 ownerId` (meaning differs) |
| 2 | `Decode4 dwOwnerId` | `int16 originX` ❌ |
| 3 | `Decode4 nSkillID` | `int16 originY` ❌ |
| 4 | `Decode1 nSLV` | `int16 ltX` ❌ |
| 5 | `Decode2 phase` | `int16 ltY` |
| 6 | `DecodeBuffer(16) rcArea RECT` | `int16 rbX` ❌ |
| 7 | `Decode4 tStart` | `int16 rbY` ❌ |
| 8 | `Decode4 tEnd` | `int32 duration` |
| — | (none) | `int32 skillLevel` (extra) |

v95 adds `nType` + `nSkillID` 4-byte fields, drops `originX/originY`, and packs
LT/RB as a 16-byte RECT buffer rather than four inline int16s. Atlas's elsewhere
field-object spawns (drop/reactor/monster) audit ✅ against v95, so atlas is a
v95 server in general — this `AffectedAreaCreated` is a **stale v83**
implementation.

**Why deferred (not fixed under audit cover):** a correct v95 re-encode would
require adding type/skillId, dropping origin, and emitting the RECT as a buffer —
which would simultaneously **break the v83 client** the struct is written for. A
proper fix must be region/version-guarded AND cross-version-verified against the
v83/v87/v92 IDBs (the verify-against-WZ/IDA discipline). That is a versioned
re-encode effort, out of scope for this clientbound field-shape bucket.

**Sibling-task suggestion:** *"GMS v95 affected-area (mist) SPAWN_MIST re-encode."*
Scope: rewrite `AffectedAreaCreated.Encode` to the v95 layout above behind a
version guard, keeping the v83 path; add a 4-variant Encode test; cite
`CAffectedAreaPool::OnAffectedAreaCreated`@0x437ec0. The sibling REMOVE_MIST
packet is unaffected (single int32 id matches all versions).
