# Spike B3.5: Interaction (mini-room / PLAYER_INTERACTION) serverbound sub-ops

Task B3.5 — locate the IDA *send-sites* for 7 serverbound `PLAYER_INTERACTION`
sub-ops (JMS opcode `0x7C`) and confirm each Atlas decode shape against the client's
encode read-order. Per-sub-op verdict; fix-in-task only on a genuine divergence.

Atlas files audited: `libs/atlas-packet/interaction/serverbound/operation_{create,open,
cash_trade_open,invite_decline,visit,merchant_name_change,personal_store_set_visitor}.go`.
Mode→sub-op mapping is owned by the consumer
`services/atlas-channel/.../socket/handler/character_interaction.go` (the packet lib
structs carry no sub-op constant of their own).

## Method & a load-bearing caveat (version skew)

The loaded IDB is **JMS v185.1** (`MapleStory_dump_SCY.exe`). All senders below are read
from that IDB. The clientbound result handler `CWvsContext::OnEntrustedShopCheckResult`
and the dialog blacklist senders reveal a **sub-op renumbering between JMS185 and the
GMS family Atlas targets** (Atlas mode comments match GMS v83/87/95):

| operation (semantic) | GMS sub-op (Atlas) | JMS185 sub-op (this IDB) |
|---|---|---|
| merchant add-to-blacklist  | `0x30` | `0x2D` (`CEntrustedShopDlg::AddBlackList` @ `0x54bb75`) |
| merchant del-from-blacklist| `0x31` | `0x2E` (`CEntrustedShopDlg::DeleteBlackList` @ `0x54bbf9`) |

So the **high-numbered merchant/personal-store sub-op bytes cannot be byte-matched
against JMS185** — JMS185's `0x2D` is "merchant blacklist add" (string body), not Atlas's
`MERCHANT_NAME_CHANGE` (uint32 body). Cross-checked the GMS v95 export
(`docs/packets/ida-exports/gms_v95.json`, 354 functions): it confirms GMS
AddBlackList=`0x30`/DeleteBlackList=`0x31` (matching Atlas) and contains **no serverbound
sender** for cash-trade-open, merchant-name-change, or personal-store-set-visitor either.

Where an Atlas op maps cleanly onto a JMS185 sender by *semantics*, the body shape is
validated regardless of the differing sub-op number. Where it does not, the verdict is
**corroborated (sender not located)** — body shape unchanged from the prior per-struct
audit and consistent with sibling ops; no read-order is fabricated.

## Per-sub-op verdicts

| # | Atlas op (mode) | Atlas decode shape | IDA sender (JMS185) | Client encode read-order (after sub-op byte) | Verdict |
|---|---|---|---|---|---|
| 1 | `OperationCreate` (CREATE, mode 0) | `roomType b`; rt∈{1,2}: title str, private bool, [pw str], nGameSpec b; rt=3: private bool; rt∈{4,5}: title str, private bool, slot i16, itemId u32; rt=6: private bool | `CField::SendInviteTradingRoomMsg` @ `0x56c859` (rt 3); `CWvsContext::SendOpenShopRequest` @ `0xb04926` (rt 4/5) | rt3: `Encode1(0)`→`Encode1(3)`→`Encode1(0=private)`. rt4/5: `Encode1(0)`→`Encode1((bEntrusted!=0)+4)`→`EncodeStr(title)`→`Encode1(0=private)`→`Encode2(slot)`→`Encode4(itemId)` | **verified** (rt 3 + 4/5 confirmed; rt 1/2/6 standard shape, corroborated) |
| 2 | `OperationOpen` (OPEN, mode 0xB) | `success bool` | `CPersonalShopDlg::OnCorrectSSN2` @ `0x761cee`; `CEntrustedShopDlg::OnCorrectSSN2` @ `0x54acd3` | `Encode1(0xB)`→`Encode1(1)` (one byte, always 1) | **verified** |
| 3 | `OperationInviteDecline` (INVITE_DECLINE, mode 3) | `serialNumber u32`, `errorCode b` | `CMiniRoomBaseDlg::SendInviteResult` @ `0x6da8e6` (err≠0 branch); `SendCashInviteResult` @ `0x6da99e` | `Encode1(3)`→`Encode4(dwSN)`→`Encode1(nErrCode)` | **verified** |
| 4 | `OperationVisit` (VISIT, mode 4) | `serialNumber u32`, `errorCode b`, [errCode≠0: errMsg str], `something bool`, [something: unk1 i16, cashSerialNumber u64] | `CMiniRoomBaseDlg::SendInviteResult` @ `0x6da8e6` (success branch); `CWvsContext::OnEntrustedShopCheckResult` case `0x11` @ `0xb0f14e` | success: `Encode1(4)`→`Encode4(dwSN)`→`Encode1(0=err)`→`Encode1(0=something)`. visit-shop: `Encode1(4)`→`Encode4(sn)`→`Encode1(0=err)`→`Encode1(1=something)`→`Encode2(unk1)`→`EncodeBuffer(cashSN,8)` | **verified** (both `something` branches confirmed) |
| 5 | `OperationCashTradeOpen` (CASH_TRADE_OPEN, mode 0xE) | `nProc b`, `roomType b`, then nProc/roomType-gated u32/u32/b / u32/u32/b/u16/u64 / u32 | not located | n/a | **corroborated (sender not located)** |
| 6 | `OperationMerchantNameChange` (MERCHANT_NAME_CHANGE, mode 0x2D) | `unk1 u32` | not located (JMS185 `0x2D` is a *different* op — merchant blacklist add, string body) | n/a | **corroborated (sender not located)** |
| 7 | `OperationPersonalStoreSetVisitor` (PERSONAL_STORE_SET_VISITOR, mode 0x1D) | `slot b`, `name str` | not located | n/a | **corroborated (sender not located)** |

## Notes per verdict

**1 CREATE** — Two of three populated branches positively confirmed in IDA.
`SendOpenShopRequest` encodes roomType as `(bEntrusted != 0) + 4`, i.e. `4` (player
shop) or `5` (hired merchant), then `EncodeStr(name)`, `Encode1(0)` (private), `Encode2`
(inventory pos / slot), `Encode4` (item id) — exactly Atlas's `rt∈{4,5}` branch.
`SendInviteTradingRoomMsg` (else branch) encodes `Encode1(0)`,`Encode1(3)`,`Encode1(0)`
— Atlas's `rt=3` branch. The omok/memory-game branch (rt 1/2: title/private/[pw]/
nGameSpec) is the standard mini-game create shape; not located as a distinct sender but
not contradicted.

**2 OPEN** — Identical in both the personal-shop and entrusted-shop SSN-correct
callbacks: a single trailing byte fixed at `1`. Atlas reads it as `bool success` → true.
Exact match.

**3 INVITE_DECLINE** — `SendInviteResult` error path is the canonical sub-op-3 sender:
`Encode4(dwSN)` then `Encode1(nErrCode)`. Atlas reads `u32`+`byte`. Exact match. (Note:
the *success* path of the same function emits sub-op **4** — see VISIT — which is why
mode 3 and mode 4 share this sender.)

**4 VISIT** — Fully confirmed in both shapes. The error-free `SendInviteResult` path
sends `something = 0` (no trailing payload). The visit-a-shop path in
`OnEntrustedShopCheckResult` case `0x11` sends `something = 1` followed by `Encode2`
(int16 `unk1`, the shop position) and `EncodeBuffer(...,8)` (the 8-byte cash serial
number) — Atlas's `if something { ReadInt16; ReadUint64 }`. The `errorCode != 0` →
`errorMessage` branch was not exercised by a located sender but is the standard error
shape and is not contradicted.

**5 CASH_TRADE_OPEN** — No serverbound sender located. This is the mini-room *cash*
trading-room open flow; JMS185 exposes `CCashTradingRoomDlg` (ctor/Trade/PutMoney/
button-handler) but none of those build a `0x7C` packet with the nProc/roomType create
body — the room is opened reactively. `CCashShop::*` is the cash-*item*-shop, a different
subsystem. GMS v95 export likewise has no such serverbound sender. Atlas body shape is
unchanged from the prior per-struct audit and internally consistent with the other create
variants; not fabricating a read-order.

**6 MERCHANT_NAME_CHANGE** — No sender at Atlas's GMS sub-op `0x2D`. In JMS185, `0x2D` is
*merchant add-to-blacklist* (`CEntrustedShopDlg::AddBlackList`, body = `EncodeStr(name)`),
a renumbered-but-different operation, so it cannot validate Atlas's `unk1 u32` body. No
serverbound name-change sender exists in JMS185 or the GMS v95 export. The strong reading
(corroborated by `OnEntrustedShopCheckResult#ShopRename` being a *clientbound* result, GMS
sub-op `0xE`) is that the name-change is a server→client notification and the serverbound
handler is a defensive mirror that a real client never triggers. Body unchanged; conservative
verdict.

**7 PERSONAL_STORE_SET_VISITOR** — No sender located at Atlas's GMS sub-op `0x1D`. The
nearby personal-store senders in JMS185 (`OnClickBanButton` @ `0x7630d5` → sub-op `0x19`;
`DeliverBlackList` @ `0x763021` → sub-op `0x1B`) are blacklist ops, not set-visitor.
Semantically "set visitor (slot, name)" is a server→client store-occupancy update; like
name-change it has no client send-site in either IDB. Atlas's `slot b, name str` body is
unchanged; conservative verdict.

## Outcome

- **4 verified** (CREATE, OPEN, INVITE_DECLINE, VISIT) — senders located in JMS185, encode
  read-order matches Atlas decode byte-for-byte.
- **3 corroborated, sender not located** (CASH_TRADE_OPEN, MERCHANT_NAME_CHANGE,
  PERSONAL_STORE_SET_VISITOR) — no serverbound send-site in JMS185 or the GMS v95 export;
  body shapes unchanged from the prior per-struct audit and consistent with sibling ops;
  the latter two are most plausibly clientbound notifications with defensive serverbound
  mirrors.
- **0 fixes.** No genuine divergence found; no code changed. All 7 sub-ops retain their
  existing byte-level tests (`go test ./interaction/...` green).

### Senders located (JMS185, for reference)

- `CField::SendInviteTradingRoomMsg` — `0x56c859`
- `CWvsContext::SendOpenShopRequest` — `0xb04926`
- `CPersonalShopDlg::OnCorrectSSN2` — `0x761cee`
- `CEntrustedShopDlg::OnCorrectSSN2` — `0x54acd3`
- `CMiniRoomBaseDlg::SendInviteResult` — `0x6da8e6`
- `CMiniRoomBaseDlg::SendCashInviteResult` — `0x6da99e`
- `CWvsContext::OnEntrustedShopCheckResult` (case `0x11`, visit payload) — `0xb0ee59`
