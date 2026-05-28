# Pending / Deferred Audit Items — GMS v95

Items deferred from the per-packet audit loop. Each row captures what is unknown,
why it was deferred, and what evidence is needed to resolve it.

---

## OP-FAMILY-messenger-serverbound

| Field | Value |
|---|---|
| Packet | `messenger/serverbound/operation.go` — `Operation` |
| Atlas file | `libs/atlas-packet/messenger/serverbound/operation.go` |
| Reason | Op-byte dispatcher — the single `mode` byte routes to sub-ops 0 (AnswerInvite), 2 (Leave/Destroy), 3 (Invite), 5 (DeclineInvite), 6 (Chat). The full op-family enum (all valid mode values and their semantics) has not been exhaustively verified in IDA. |
| Evidence needed | Confirm that no other mode values exist beyond 0, 2, 3, 5, 6; verify server-side routing in atlas-messengers matches. |
| Verdict | ⚠️ |

---

## OP-FAMILY-messenger-decline

| Field | Value |
|---|---|
| Packet | `messenger/clientbound/invite_declined.go` — `InviteDeclined` |
| Atlas file | `libs/atlas-packet/messenger/clientbound/invite_declined.go` |
| Reason | The `declineMode` byte sub-enum in `OnBlocked` (mode=5) distinguishes between "declined" (0) and "blocked" (non-zero). IDA shows: `if v3` branching to two different StringPool strings (0x31Au vs 0x31Bu). The exact numeric meaning of non-zero values is not confirmed. |
| Evidence needed | Verify declineMode values in server-side atlas-messengers event emissions; confirm whether only 0/1 are used or additional values exist. |
| Verdict | ⚠️ |

---

## AUDIT-TOOL-avatarlook

| Field | Value |
|---|---|
| Affected packets | `messenger/clientbound/add.go` — `Add`; `messenger/clientbound/update.go` — `Update` |
| Reason | The packet audit tool cannot align atlas `WriteByteArray` (AvatarLook encoded as []byte) with IDA `DecodeBuf`. Both use the same AvatarLook encoding; the mismatch is a tool limitation, not a wire bug. Reports show ❌ for Add and Update but the actual encoding is correct. |
| Evidence needed | Tool enhancement to recognize DecodeBuf as opaque byte-blob and compare structurally rather than field-by-field. |
| Verdict | ⚠️ tool limitation — atlas wire is correct |

---

## Still pending — world domain (task-068 Phase 2c, field/clientbound)

### SETFIELD-old-driver-id

| Field | Value |
|---|---|
| Affected packets | `field/clientbound/set_field.go` — `SetField`; `field/clientbound/warp_to_map.go` — `WarpToMap` |
| Atlas files | `libs/atlas-packet/field/clientbound/set_field.go`, `libs/atlas-packet/field/clientbound/warp_to_map.go` |
| IDA | `CStage::OnSetField` @0x71a0a0 (SET_FIELD, GMS v95 opcode 0x8D/141) |
| Reason | v95 reads `m_dwOldDriverID` as a `Decode4` (line 129) immediately after `m_nChannelID` (line 128), unconditionally — before the `bCharacterData` split, so it affects BOTH the SetField (full) and WarpToMap (warp) paths. Atlas writes only the channelId int32, then emits a `WriteByte(0)+WriteInt(0)` pair under the JMS-only guard; it never emits a 4-byte old-driver-id for GMS. A GMS v95 client therefore reads the following envelope bytes shifted by 4. Every other envelope field matches v95 exactly. |
| Why deferred (not fixed) | The version-introduction point of `m_dwOldDriverID` is unknown. Atlas runs production GMS at v83/v87; only the v95 IDB is loaded. Adding a `(GMS && MajorVersion>83)`-gated `WriteInt` could be correct for v95 yet wrong for v87/v92 if the field was introduced later, breaking the very versions atlas serves. A speculative version gate is riskier than the current (also-wrong-for-v95) state. |
| Evidence needed | v83 / v87 / v92 GMS IDA for `CStage::OnSetField` to pin the exact version where `m_dwOldDriverID` (Decode4 after channelId) was added; then add the correctly-gated 4-byte write to both set_field.go and warp_to_map.go. |
| Verdict | ❌ cross-version structural divergence (v95 confirmed; gate unverifiable) |

---

## Still pending — world domain (task-068 Phase 2e, npc/clientbound)

### OP-FAMILY-npc-shop-operation

| Field | Value |
|---|---|
| Affected packets | `npc/clientbound/shop_operation.go` — `ShopOperationSimple`, `ShopOperationLevelRequirement`, `ShopOperationGenericError` (writer `NPCShopOperation`) |
| Atlas files | `libs/atlas-packet/npc/clientbound/shop_operation.go`, `libs/atlas-packet/npc/clientbound/shop_operation_body.go` |
| IDA | `CShopDlg::OnPacket` @0x6eb7d0 (CONFIRM_SHOP_TRANSACTION, GMS v95 opcode 0x130/304), `nType==365` switch on `Decode1(mode)` |
| Reason | The clientbound shop-operation result is a mode-byte family: a single leading byte selects 19+ arms. Verified arms — mode-only (cases 0,1,2,3,5,8,9,0xA,0xD,0x10,0x11,0x12,default → `ShopOperationSimple`), `Decode4`-level (cases 0xE/0xF → `ShopOperationLevelRequirement`), and `Decode1(hasReason)+optional DecodeStr` (case 0x13 → `ShopOperationGenericError`). Each atlas struct's per-mode wire shape was confirmed against the matching IDA case. The exhaustive mode-value → atlas-code mapping (which `operations` template code resolves to each numeric mode, and whether modes 4/6/7/0xB/0xC carry bodies) was not fully enumerated. |
| Evidence needed | Cross-check the `operations` resolver table in the GMS v95 template against every `CShopDlg::OnPacket` case; confirm cases 4 and 8 (early `return`, no Notice) and any unhandled modes have no atlas emitter. |
| Verdict | ⚠️ op-byte family — per-struct wire shapes verified; full mode enum unenumerated |

---

## Still pending — world domain (task-068 Phase 2g, npc/serverbound)

### OP-FAMILY-npc-shop-serverbound

| Field | Value |
|---|---|
| Affected packets | `npc/serverbound/shop.go` — `Shop` (op-byte dispatcher); `npc/serverbound/shop_buy.go` — `ShopBuy`; `npc/serverbound/shop_sell.go` — `ShopSell`; `npc/serverbound/shop_recharge.go` — `ShopRecharge` (handler `NPCShopHandle`) |
| Atlas files | `libs/atlas-packet/npc/serverbound/shop.go`, `shop_buy.go`, `shop_sell.go`, `shop_recharge.go`; dispatcher `services/atlas-channel/.../socket/handler/npc_shop.go` |
| IDA | `CShopDlg::SendBuyRequest` @0x6e9bb0, `CShopDlg::SendSellRequest` @0x6e7260, `CShopDlg::SendRechargeRequest` @0x6e4e90 (NPC_SHOP, GMS v95 opcode 66/0x42); each builds `COutPacket(66) + Encode1(op) + body` |
| Reason | NPC_SHOP serverbound is an op-byte family: a single leading `Encode1(op)` discriminator (BUY=0, SELL=1, RECHARGE=2 in the client) selects the per-op body. Atlas models this as a `Shop` struct that reads only the op byte, then the channel handler (`npc_shop.go`) delegates to `ShopBuy`/`ShopSell`/`ShopRecharge`. Each per-op body wire shape was verified field-for-field against the matching client `Send*` function (✅ in all three reports). The op-byte VALUES atlas matches against (`NPCShopOperationBuy/Sell/Recharge/Leave`) are loaded from the runtime `operations` config map, NOT the static template, so the analyzer cannot confirm atlas's op-byte values equal the client's (0/1/2). |
| Evidence needed | Cross-check the channel `operations` resolver config (BUY/SELL/RECHARGE/LEAVE → numeric op) against the client values: BUY=0, SELL=1, RECHARGE=2 (verified in IDA). LEAVE has no client `Send*` site in this binary's shop dialog (the SetRet/leave path was not located); confirm the LEAVE op value and that no client body trails it. |
| Verdict | ⚠️ op-byte family — per-op body wire shapes verified; op-byte values are runtime config (unverifiable by analyzer) |

### ROUTING-npc-continue-conversation-discriminator

| Field | Value |
|---|---|
| Affected packets | `npc/serverbound/continue_conversation.go` — `ContinueConversation` (handler `NPCContinueConversationHandle`) and its trailing-field structs `ContinueConversationSelection`, `ContinueConversationText` |
| Atlas files | `libs/atlas-packet/npc/serverbound/continue_conversation*.go`; dispatcher `services/atlas-channel/.../socket/handler/npc_continue_conversation.go` |
| IDA | NPC_TALK_MORE reply built inside the `CScriptMan::On*` dialog handlers (opcode 65/0x41): `OnSay` @0x6dc110 (msgType 0), `OnAskYesNo` @0x6dc5a0 (msgType 2/13), `OnAskText` @0x6dc790 (msgType 3), `OnAskMenu` @0x6dce00 (msgType 5), `OnAskAvatar` @0x6dcff0 (msgType 8) |
| Reason | The packet STRUCTS are wire-correct (`ContinueConversation` = msgType byte + action byte; `ContinueConversationText` = string; `ContinueConversationSelection` = int32/byte selection). HOWEVER the channel dispatcher `npc_continue_conversation.go` branches on `lastMessageType == 2` to decide text-vs-selection. Per IDA, msgType **2** is AskYesNo (its reply carries NO trailing field — just msgType+action), while the TEXT reply is msgType **3** (AskText, `EncodeStr`) / **14** (AskBoxText). So the channel handler's discriminator value looks wrong: it would attempt to read a text string after an AskYesNo reply (msgType 2, which has no body) and treat AskText (msgType 3) as a selection. This is a channel HANDLER-LOGIC concern, NOT a wire-shape bug in `libs/atlas-packet` (out of this bucket's struct-audit scope). |
| Evidence needed | Audit `atlas-channel` NPC continue-conversation routing: verify the msgType→body mapping (3/14 = text via `ContinueConversationText`; 5/8/9/etc = selection via `ContinueConversationSelection`; 0/1/2/13 = no trailing body). The current `== 2` branch should likely be `== 3` (and/or cover 14). Track as a channel-service fix, not a packet-lib fix. |
| Verdict | ⚠️ channel handler routing concern — packet structs verified correct |
