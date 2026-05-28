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

## Sub-op enum / sub-struct deferrals — commerce domain (task-067)

### Show clientbound — per-tab item segmentation + spurious padding (storage)

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

### OperationChat — missing leading update_time field (interaction) — DEFERRED

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

### OperationPersonalStoreBuy / OperationMerchantBuy — missing trailing itemCRC (interaction) — DEFERRED

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

### OperationPersonalStoreSetBlackList — byte[] vs string[] structural mismatch (interaction) — DEFERRED

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

### Cash serverbound SPW-string vs birthday-int divergence (cash) — DEFERRED (version-gated)

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

### ShopOperationBuy — trailing oneADay byte + eventSN int (cash) — DEFERRED (version-gated)

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

### CashShopInventory — missing 2 trailing slot-counter shorts (cash) — DEFERRED (version-gated)

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
