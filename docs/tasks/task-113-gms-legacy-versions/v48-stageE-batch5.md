# v48 Stage E BATCH 5 — interaction (CMiniRoomBaseDlg / miniroom / trade / shop) family

Version `gms_v48`, IDB port 13337 (`GMS_v48_1_DEVM.exe`). Anchor = `gms_v61` (208).
Serverbound opcode = 93 (0x5D). All send-sites body-verified from their
`COutPacket(93)` site. v48 sits below every legacy gate (`chatHasUpdateTime`
false, `tradeCrcPresent` false) so every body encode order equals the verified
v61 fixture; the leading mode byte is dispatcher-framed and never enters the
sub-struct body. Result: **20 serverbound + 1 clientbound promoted =v61, 2
n-a, 0 blocked.**

## Per-cell outcome (serverbound)

| # | Struct | v48 sub (addr) | mode | body | fixture | outcome |
|---|--------|----------------|------|------|---------|---------|
| 1 | OperationChat | sub_546A05 @0x546a05 (send 0x546a30) | 6 | EncodeStr(msg) | 02006869 | ✅ =v61 |
| 2 | OperationFieldAddToBlackList | sub_4CBD0E @0x4cbd0e | 0x1C | EncodeStr(name) | 02006869 | ✅ =v61 |
| 3 | OperationFieldRemoveFromBlackList | sub_4CBD88 @0x4cbd88 | 0x1D | EncodeStr(name) | 02006869 | ✅ =v61 |
| 4 | OperationInvite | sub_4C5100 @0x4c5100 (send 0x4c528d) | 2 | Encode4(id) | 78563412 | ✅ =v61 |
| 5 | OperationMemoryGameFlipCard | sub_53875D @0x53875d | 0x3D | Encode1(first),Encode1(index) | 0102 | ✅ =v61 |
| 6 | OperationMemoryGameMoveStone | sub_578388 @0x578388 | 0x39 | EncodeBuffer(8),Encode1(color) | 010000000000000002 | ✅ =v61 |
| 7 | OperationMemoryGameRetreatAnswer | sub_573A54 @0x573a54 | 0x2C | Encode1(bool) | 01 | ✅ =v61 |
| 8 | OperationMemoryGameTieAnswer | sub_573B11 @0x573b11 | 0x30 | Encode1(bool) | 01 | ✅ =v61 |
| 9 | OperationMerchantBuy | sub_58847F @0x58847f | 0x1F/0x14 | Encode1(idx),Encode2(qty) | 031900 | ✅ =v61 |
| 10 | OperationMerchantPutItem | sub_58883F @0x58883f | 0x1E/0x13 | E1,E2,E2,E2,E4 | 0205006400070040420f00 | ✅ =v61 |
| 11 | OperationMerchantRemoveItem | sub_588B4B @0x588b4b | 0x23/0x18 | Encode2(index) | 0500 | ✅ =v61 |
| 12 | OperationPersonalStoreAddToBlackList | sub_588DFC @0x588dfc | 0x19 | Encode1(slot),EncodeStr(name) | 0202006869 | ✅ =v61 |
| 13 | OperationPersonalStoreBuy | sub_58847F @0x58847f (shared) | 0x14 | Encode1,Encode2 | 031900 | ✅ =v61 |
| 14 | OperationPersonalStorePutItem | sub_58883F @0x58883f (shared) | 0x13 | E1,E2,E2,E2,E4 | 0205006400070040420f00 | ✅ =v61 |
| 15 | OperationPersonalStoreRemoveItem | sub_588B4B @0x588b4b (shared) | 0x18 | Encode2(index) | 0500 | ✅ =v61 |
| 16 | OperationPersonalStoreSetBlackList | sub_588D46 @0x588d46 | 0x1B | Encode2(count),loop EncodeStr | 010002006162 | ✅ =v61 |
| 17 | OperationTradeAddMeso | sub_5E819A @0x5e819a | 0xE | Encode4(amount) | 40420f00 | ✅ =v61 |
| 18 | OperationTradeConfirm | sub_5E836C @0x5e836c | 0xF | bodyless | (empty) | ✅ =v61 |
| 19 | OperationTradePutItem | sub_5E7F74 @0x5e7f74 | 0xD | E1,E2,E2,E1 | 020500640003 | ✅ =v61 |
| 20 | OperationTransaction | sub_5E836C @0x5e836c (shared) | 0xF | bodyless | (empty) | ✅ =v61 |

Clientbound:

| Struct | v48 dispatcher (addr) | outcome |
|--------|-----------------------|---------|
| InteractionInteractionUpdateMerchant | CMiniRoomBaseDlg::OnPacketBase = sub_5459C4 @0x5459c4 | ✅ =v61 (cross-version equality; no gate) |

n-a:

| Struct | fname | disposition |
|--------|-------|-------------|
| OperationMerchantAddToBlackList | CEntrustedShopDlg::AddBlackList | n-a — no v48 send-site; entrusted (hired-merchant) blacklist absent, matches v61 |
| OperationMerchantRemoveFromBlackList | CEntrustedShopDlg::DeleteBlackList | n-a — same; entrusted shop first appears v83+ |

## Shared codecs

- BuyItem (sub_58847F): MerchantBuy + PersonalStoreBuy (isMerchant flag → mode 0x1F/0x14).
- PutItem (sub_58883F): MerchantPutItem + PersonalStorePutItem (0x1E/0x13).
- MoveItemToInventory (sub_588B4B): MerchantRemoveItem + PersonalStoreRemoveItem (0x23/0x18).
- Trade (sub_5E836C, bodyless): TradeConfirm + Transaction (both alias the base CTradingRoomDlg::Trade send; no separate cash-entry-list send exists in v48, same as v61).

## Re-derived v48 mode table

Serverbound (opcode 93, `Encode1(mode)` after `COutPacket(93)`):

| mode | operation | send-site |
|------|-----------|-----------|
| 2 | INVITE | sub_4C5100 @0x4c528d |
| 6 | CHAT | sub_546A05 @0x546a30 |
| 0xA | leave/exit (bodyless) | sub_586E18 / sub_5E69EA … |
| 0xD | TRADE PUT ITEM | sub_5E7F74 @0x5e8109 |
| 0xE | TRADE ADD MESO (PutMoney) | sub_5E819A @0x5e830d |
| 0xF | TRADE CONFIRM (Trade, bodyless) | sub_5E836C @0x5e83ec |
| 0x13 | PERSONAL STORE PUT ITEM | sub_58883F |
| 0x14 | PERSONAL STORE BUY | sub_58847F |
| 0x18 | PERSONAL STORE REMOVE ITEM | sub_588B4B |
| 0x19 | PERSONAL STORE ADD BLACKLIST | sub_588DFC @0x588ea4 |
| 0x1A | (personal store auto-reban) | sub_588F0F @0x588f56 |
| 0x1B | PERSONAL STORE SET BLACKLIST | sub_588D46 @0x588d77 |
| 0x1C | FIELD ADD BLACKLIST | sub_4CBD0E @0x4cbd34 |
| 0x1D | FIELD REMOVE BLACKLIST | sub_4CBD88 @0x4cbdae |
| 0x1E | MERCHANT PUT ITEM | sub_58883F @0x588a51 |
| 0x1F | MERCHANT BUY | sub_58847F @0x5887d1 |
| 0x23 | MERCHANT REMOVE ITEM | sub_588B4B @0x588c18 |
| 0x2A | buy-from-player-shop (Encode4) | sub_69FE41 @0x69ff6a |
| 0x2C | OMOK RETREAT ANSWER (body bool) | sub_573A54 @0x573a7a |
| 0x30 | MEMORY GAME TIE ANSWER (body bool) | sub_573B11 @0x573b37 |
| 0x39 | MEMORY GAME MOVE STONE | sub_578388 @0x5783ad |
| 0x3D | MEMORY GAME FLIP CARD | sub_53875D @0x53877f |

Game-control toggles 0x2B/0x2D/0x2F/0x31/0x32/0x33/0x34/0x35/0x36/0x38 are
bodyless mode-only sends (give-up / leave / ready / skip / timeout), present in
both COmok (0x578xxx) and CMemoryGame (0x53dxxx) button clusters via the shared
CMiniRoomBaseDlg base; not modeled Atlas arms.

Clientbound dispatcher CMiniRoomBaseDlg::OnPacketBase (sub_5459C4 @0x5459c4),
`Decode1(mode)` then switch — byte-identical to v61 (sub_5BEC69):

- no active room: 2 → OnInvite (sub_545EA6), 3 → sub_54607D, 5 → sub_545A60
- active room: 3 → Leave (sub_54607D), 4 → OnAvatar (sub_5462E3), 6 → OnChat
  (vtable+76), 9 → Enter (sub_546433), 10 → InviteResult (sub_54637C),
  default → vtable+60 sub-dispatch (CPersonalShopDlg::OnPacket; mode 0x18 =
  hired-merchant refresh → OnRefresh → UpdateMerchant).

## Divergence resolution — retreat / tie answer

The brief flagged that v48 omok/memory tie/retreat *toggle* functions
(sub_578797 / sub_53D26E @…, modes 0x31 accept / 0x32 decline / 0xA) send the
answer as the MODE with NO body bool. Those are **not** the retreat/tie answer —
they are a different game-control toggle (server request 0x30-family "ready/skip"
handled by a per-game duplicate cluster). The **real** retreat/tie answers carry
the body bool: request-dispatcher sub_5731A9 routes server request 0x2B →
sub_573A54 (answer 0x2C, `Encode1(YesNo==6)`, dialog 442) and request 0x2F →
sub_573B11 (answer 0x30, `Encode1(YesNo==6)`, dialog 446). This is exactly v61's
request/answer pairing shifted −1 (v61 retreat req 0x2C/ans 0x2D, tie req
0x30/ans 0x31). Bodies identical → **outcome (a): fixture == v61 "01"** for both.
Retreat = lower mode (omok), tie = higher mode, matching v61 ordering.

## FlipCard argument order

v48 sub_53875D encodes `Encode1(a2),Encode1(a1)` after the mode; caller
sub_538613 invokes `sub_53875D(index, firstFlag)`, so wire = [first][index],
matching v61 `Encode1(first),Encode1(index)`. Fixture "0102".

## Gates

- `go test ./libs/atlas-packet/interaction/...` — PASS (also `-race`, `go vet` clean).
- `go run ./tools/packet-audit matrix --check` — **exit 0**.
- `grep -ciE 'orphan|dangling|stale|drift|unresolv|malformed' STATUS.md` — **0**.
- gms_v48 conflicts — **0** (unchanged).
- Verified counts: **gms_v48 48 → 69 (+21)**; v61 208 / v72 216 / v79 228 /
  v83 367 / v84 345 / v87 379 / v95 399 / jms 362 — **all unchanged** (no regression).

## Mechanic

Followed the v61-interaction pattern: added 17 `sub_XXXX` entries (16 unique
serverbound send-fns + clientbound sub_5459C4) to `gms_v48.json` (surgical text
splice, +282 lines, no existing entries touched), pinned evidence with those
`sub_XXXX` names, added `// packet-audit:verify` markers in new
`serverbound/v48_test.go` + `clientbound/v48_test.go` with the v48 sub addresses.
The pre-existing per-struct audit-report stubs (unresolved IDAName) are retained —
v61 promotes identically with stub reports + resolved evidence + marker.

## Commit

- `HEAD` — task-113(v48): stage E — verify interaction (miniroom/trade/shop) family (tier-1).
  27 files, +866/−66; touches only interaction tests, gms_v48 evidence/export/
  unimplemented, STATUS.md, status.json.
- Branch verified: `task-113-gms-legacy-versions`.
