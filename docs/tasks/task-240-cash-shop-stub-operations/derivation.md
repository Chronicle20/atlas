# task-240 — Derivation

Blocking artifact for plan Task 1. Every wire-level fact below was read out of the
client IDB identified in §0 or out of repo source on this branch. Nothing here is
recalled from general MapleStory knowledge, and no value is inferred from a symbol
name alone — for each answer the dispatch site or the encode sequence is quoted.

Consumers: Task 2 (D1), Task 8 (D2a/D2b), Task 14 (D3a/D3b), Task 17 (D4a),
Task 20 (D4b).

---

## 0. IDB identity

The plan brief calls this "the GMS v95.1 IDB". **No v95.1 database exists in this
ida-pro-mcp instance.** The database actually used is a **v95.0** binary:

| Field | Value (as reported by `idb_list`) |
|---|---|
| `filename` | `GMS_v95.0_U_DEVM.exe.i64` |
| `input_path` | `E:\Programs\Nexon\IDBs_v9\GMS\v95_0\GMS_v95.0_U_DEVM.exe.i64` |
| `session_id` | `32c8836f` |
| `backend` | `gui` (already adopted; `idb_open` not called) |
| `created_at` | `2026-08-15T17:47:05.503782` |

`idb_list` reports no hash for any session, so binary identity is pinned by
path + filename + session only. Note the path differs from the one quoted in the
task brief (`...\IDBs_v9\GMS95_0\...`); the value in this table is the one the
tool returned.

All answers below are therefore **v95.0** answers applied to the `gms_95_1`
tenant template. That is the same substitution every other v95 row in the repo
already makes (see `docs/tasks/task-227-cash-name-change-world-transfer/derivation.md` §0,
which pins gms_v95 to a `GMS_v95.0_U_DEVM.exe.i64` IDB as well).

Serverbound mode bytes are taken from the tenant table, not from memory:
`services/atlas-configurations/seed-data/templates/template_gms_95_1.json`
carries `SET_WISHLIST 5`, `BUY_PACKAGE 32`, `APPLY_WISHLIST 35` (lines 2307–2319)
and clientbound `LOAD_WISHLIST 92`, `UPDATE_WISHLIST 98`,
`BUY_PACKAGE_SUCCESS 154`, `BUY_PACKAGE_FAILED 155`, `GIFT_PACKAGE_SUCCESS 156`,
`GIFT_PACKAGE_FAILED 157` (lines 4812–4847). The v83 mode numbers that appear in
existing doc comments are **not** the v95 numbers and were not used.

---

## 1. D1 — the v95 opcode for `CStage::OnSetCashShop` — **RESOLVED: `0x8F` (143)**

**Question.** Does `CStage::OnSetCashShop` exist on v95, and which clientbound
opcode dispatches to it?

**Answer.** It exists at `0x71adf0`
(`?OnSetCashShop@CStage@@IAEXAAVCInPacket@@@Z`) and is reachable from exactly one
site: the `CStage::OnPacket` dispatch switch at `0x71b0b0`. The case value is
**143 = `0x8F`**.

`decompile(0x71b0b0)`:

```c
void __thiscall CStage::OnPacket(CStage *this, int nType, CInPacket *iPacket)
{
  switch ( nType ) /*0x71b0b9*/
  {
    case 141: CStage::OnSetField((this - 8), iPacket);    break; /*0x71b0ed*/
    case 142: CStage::OnSetITC((this - 8), iPacket);      break; /*0x71b0dd*/
    case 143: CStage::OnSetCashShop((this - 8), iPacket); break; /*0x71b0cd*/
  }
}
```

`xrefs_to(0x71adf0)` returns a single code xref, `0x71b0cd`, inside that switch —
so 143 is the only route to the handler. `CStage::OnPacket` is itself reached
from `CField::OnPacket` (`0x547166`) and `CLogin::OnPacket` (`0x5dfab3`) with the
raw opcode forwarded as `nType`.

**Independent cross-check that `nType` is the raw opcode.** The two neighbouring
cases are already verified registrations in the v95 template:

| Case in `CStage::OnPacket` | Template writer (`template_gms_95_1.json`) | opCode |
|---|---|---|
| 141 → `CStage::OnSetField` | `SetField`, `fname: CStage::OnSetField` (line 3272) | `0x8D` |
| 142 → `CStage::OnSetITC` | `SetItc`, `fname: CStage::OnSetITC` (line 3285) | `0x8E` |
| 143 → `CStage::OnSetCashShop` | *(absent — this is what Task 2 adds)* | **`0x8F`** |

141 = 0x8D and 142 = 0x8E match the checked-in registrations exactly, so case 143
is opcode 0x8F.

**Slot availability.** The `writers` array in `template_gms_95_1.json` runs
`0x8C` (`CharacterSkillMacro`) → `0x8D` (`SetField`) → `0x8E` (`SetItc`) → `0x93`
(`BlockedMap`); `0x8F` is unclaimed in that array. The `0x8F` that appears at line
987 of the same file is in the **`handlers`** (serverbound) array
(`MessengerOperationHandle` / `CFadeWnd::SendCloseMessage`) — a different
namespace, no conflict.

**Do not copy the v92 value forward.** `template_gms_92_1.json` line 2415 registers
`CashShopOpen` at `0x8E` with `fname: CStage::OnSetCashShop`. On v95 `0x8E` is
`CStage::OnSetITC`. Registering the v92 value on v95 would collide with `SetItc`.

**Consumer:** Task 2 — register `{"opCode":"0x8F","writer":"CashShopOpen",
"fname":"CStage::OnSetCashShop","services":["channel"]}` in
`template_gms_95_1.json`. This is *not* the `n-a` (FR-V95-3) case.

---

## 2. D2a — APPLY_WISHLIST (v95 mode 35 = `0x23`) body — **RESOLVED: empty**

**Question.** Does the client write any bytes after the mode byte?

**Answer.** No. The body is empty; mode 35 is a bare request.

Sender: `CCashShop::ApplyWishListEvent` @ `0x482ea0`. Full decompilation:

```c
void __thiscall CCashShop::ApplyWishListEvent(CCashShop *this)
{
  if ( !this->m_bCashShopRequestSent )                                  /*0x482ec6*/
  {
    COutPacket::COutPacket(&oPacket, 275);                              /*0x482ed5*/
    COutPacket::Encode1(&oPacket, 0x23u);                               /*0x482ee8*/
    CClientSocket::SendPacket(ms_pInstance, v2, &oPacket);              /*0x482ef8*/
    this->m_bCashShopRequestSent = 1;                                   /*0x482f01*/
    ZArray<unsigned char>::RemoveAll(&v4);                              /*0x482f10*/
  }
}
```

`0x23` = 35 = the template's `APPLY_WISHLIST`. Between `Encode1(0x23)` and
`SendPacket` there is no other `Encode*` call and no inlined `ZArray` append —
compare `CCashShop::OnBuyCouple` (§4) where the inlined appends are plainly
visible. Opcode 275 = `0x113` is the `CASHSHOP_OPERATION` serverbound op.

Trigger, for context: the sole caller is `CCSWnd_Best::OnMouseButton` @
`0x4cabf7` (function `0x4caa00`) — a button in the cash shop's "Best" window.

**Consumer:** Task 8 — the serverbound APPLY_WISHLIST body decodes to a
zero-field struct.

---

## 3. D2b — the reply arm APPLY_WISHLIST expects — **RESOLVED: `UPDATE_WISHLIST` (mode 98 = `0x62`)**

**Question.** `LOAD_WISHLIST` (92, `cashcb.CashShopWishListLoadBody`) or
`UPDATE_WISHLIST` (98, `cashcb.CashShopWishListUpdateBody`)?

**Answer.** `UPDATE_WISHLIST` — mode **98** (`0x62`,
`CCashShop::OnCashItemResSetWishDone`).

**Evidence 1 — the request-in-flight latch.** `ApplyWishListEvent` both *guards*
on `!m_bCashShopRequestSent` and *sets* it to 1 (`0x482f01`, field offset
`+0x1C`). Every other cash-shop send is gated on the same field
(`TrySendQueryCashRequest` `0x481be6`, `OnMoveCashItemLtoS` `0x482925`,
`OnMoveCashItemStoL` `0x482b9c`, `OnRemoveWish` `0x483b1c`, `OnStatusCharge`
`0x483c81`, `OnExItemSlot` `0x48d8cb`, `OnIncCharacterSlotCount` `0x48dee8`), so
whichever arm the server sends back must clear it or the cash shop wedges.

A full sweep of `mov dword ptr [reg+1Ch], 0` over the whole `CCashShop` code
range (`search_text` `0x47d000`–`0x493f00`: 0 writes, only `cmp` gates;
`0x493f00`–`0x49a000`: 28 writes) shows exactly **two** wishlist arms clear it:

```
0x494d70  CCashShop::OnCashItemResSetWishDone(CInPacket &)+10   mov dword ptr [esi+1Ch], 0
0x4969c7  CCashShop::OnCashItemResSetWishFailed(CInPacket &)+7  mov dword ptr [esi+1Ch], 0
```

`OnCashItemResLoadWishDone` (`0x494020`) and `OnCashItemResLoadWishFailed`
(`0x496990`) appear in neither range — the LOAD_WISHLIST pair never clears the
latch, which is consistent with it being the unsolicited shop-entry push rather
than a response.

**Evidence 2 — the arm's own body matches an "apply/update" response.**
`decompile(0x494d60)`:

```c
void __thiscall CCashShop::OnCashItemResSetWishDone(CCashShop *this, CInPacket *iPacket)
{
  v9 = 40;
  m_nWishList = this->m_nWishList;
  this->m_bCashShopRequestSent = 0;              /*0x494d70*/
  CInPacket::DecodeBuffer(iPacket, m_nWishList, v9);   // 10 x int32 SNs
  if ( this->m_nCurCategory == 9 )
    CCSWnd_List::ChangePage(&this->m_wndList);
  if ( !this->m_bRemoveWish )
    StringPool::GetString(Instance, &v5, 0x24Eu);      // user-facing confirmation
    CUtilDlg::Notice(...);
}
```

40 bytes = 10 × uint32 — the same payload shape as
`cashcb.CashShopWishListUpdateBody(sns []uint32)`.

**Mode numbers.** From the `CCashShop::OnCashItemResult` dispatch switch at
`0x499370`: `case 0x5Cu → OnCashItemResLoadWishDone` (92 = `LOAD_WISHLIST`),
`case 0x5Du → OnCashItemResLoadWishFailed` (93), `case 0x62u →
OnCashItemResSetWishDone` (98 = `UPDATE_WISHLIST`), `case 0x63u →
OnCashItemResSetWishFailed` (99). These agree with the template's
`LOAD_WISHLIST: 92` / `UPDATE_WISHLIST: 98`.

**Residual caveat (honest).** The client contains no explicit request→response
correlation table; the binding above is inferred from (a) the latch that only the
SetWish pair releases and (b) the payload shape. If a live capture ever shows the
server answering mode 35 with 92, the client would hang on the next cash-shop
action rather than mis-render — that is the observable symptom to look for.

**Consumer:** Task 8 — reply to APPLY_WISHLIST with
`cashcb.CashShopWishListUpdateBody` (mode 98), not `...WishListLoadBody`.

---

## 4. D3a — BUY_OTHER_PACKAGE (v95 mode 33 = `0x21`) body — **RESOLVED: not byte-identical to `ShopOperationBuyPackage`**

**Question.** Is the body identical to `ShopOperationBuyPackage`
(`pointType bool`, `option uint32`, `serialNumber uint32`), or does it carry
extra fields?

**Answer.** Different. Mode 33 is `CCashShop::OnGiftPackage` @ `0x4907b0`, whose
body after the mode byte is:

| # | Field | Encoder | Source variable |
|---|---|---|---|
| 1 | secondary password | `COutPacket::EncodeStr` @ `0x490b93` | `sSPW` (from `ask_SPW`) |
| 2 | commodity serial number (uint32) | inlined `Encode4` @ `0x490be2` | `nCommSN` |
| 3 | recipient character name | `COutPacket::EncodeStr` @ `0x490c01` | `sGiveTo` (`CUISendGift::GetResult`) |
| 4 | gift message | `COutPacket::EncodeStr` @ `0x490c1d` | `sText` (`CUISendGift::GetResult`) |

The mode byte itself is the inlined `Encode1` at `0x490b74`: `a[v46++] = 33;`.
There is **no `pointType` byte and no `option` int** on this send. Excerpt:

```c
      COutPacket::COutPacket(&oPacket, 275);          /*0x490ae1*/
      a[v46++] = 33;                                  /*0x490b74*/  // Encode1(mode)
      ZXString<char>::operator=(v31, &sSPW);
      COutPacket::EncodeStr(&oPacket, v31[0]);        /*0x490b93*/  // SPW
      *&v21[v46] = nCommSN; v46 += 4;                 /*0x490be2*/  // Encode4(SN)
      ZXString<char>::operator=(v31, &sGiveTo);
      COutPacket::EncodeStr(&oPacket, v31[0]);        /*0x490c01*/  // recipient
      ZXString<char>::operator=(v31, &sText);
      COutPacket::EncodeStr(&oPacket, v31[0]);        /*0x490c1d*/  // message
      CClientSocket::SendPacket(ms_pInstance, v25, &oPacket);
      this->m_bCashShopRequestSent = 1;               /*0x490c39*/
```

For contrast, mode 32 (`BUY_PACKAGE`) is `CCashShop::OnBuyPackage` @ `0x48ed40`
and **does** match the existing struct exactly:

```c
      COutPacket::COutPacket(&oPacket, 275);
      COutPacket::Encode1(&oPacket, 0x20u);           // mode 32
      v40 = dwOption;
      COutPacket::Encode1(&oPacket, dwOption == 2);   // pointType
      COutPacket::Encode4(&oPacket, v40);             // option
      COutPacket::Encode4(&oPacket, nCommSN);         // serialNumber
```

So `libs/atlas-packet/cash/serverbound/shop_operation_buy_package.go` is correct
for mode 32 on v95 and must **not** be reused verbatim for mode 33. The nearest
existing shape is `ShopOperationBuyCouple`'s v95 GMS arm (SPW string, option,
SN, name, message) minus the `option` int — see §5.

The `CCashShop::OnGiftPackage` guard also uses NX Prepaid only (`dwOption = 4;`
is never encoded), and it rejects any commodity whose `nItemId / 10000 != 910`
(i.e. it is package-only).

**Consumer:** Task 14 — model BUY_OTHER_PACKAGE as
`spw string, serialNumber uint32, name string, message string` for GMS v95.

---

## 5. D3b — the reply arm BUY_OTHER_PACKAGE expects — **RESOLVED: `GIFT_PACKAGE_SUCCESS` (156) / `GIFT_PACKAGE_FAILED` (157)**

**Question.** Confirm or refute the design §0 hypothesis that mode 33 is answered
by 156/157 rather than 154/155.

**Answer.** Confirmed. `CCashShop::OnCashItemResGiftPackageDone` (dispatch
`case 0x9Cu` = 156 @ `0x499448`) decodes exactly the fields the gift flow
produces, including the recipient name that only mode 33 sends:

```c
void __thiscall CCashShop::OnCashItemResGiftPackageDone(CCashShop *this, CInPacket *iPacket)
{
  CInPacket::DecodeStr(iPacket, &sRcvCharacterName);   // recipient name
  v3 = CInPacket::Decode4(v2);                         // item / package id
  CInPacket::Decode2(v2);                              // unused1
  CInPacket::Decode2(v2);                              // unused2
  v4 = CInPacket::Decode4(v2);                         // NX Prepaid spent
  CItemInfo::GetSpecialName(..., &sItemName, v3);
  ZXString<char>::Format(&iPacket,
    "[ %s ] \r\nwas sent to %s. \r\n%d NX Prepaid \r\nwere spent in the process.",
    sItemName._m_pStr, sRcvCharacterName._m_pStr, v4);
```

That read order — `str, int32, int16, int16, int32` — is field-for-field the
existing `CashShopGiftPackageDoneBody(recipientName string, packageId int32,
unused1, unused2 uint16, nxCashSpent int32)`.

The 154 arm is a different shape entirely and has no recipient name —
`CCashShop::OnCashItemResBuyPackageDone` @ `0x496b60` (dispatch `case 0x9Au` @
`0x49942e`) reads `Decode1` = count, then `DecodeBuffer(55 * count)` of
`GW_CashItemInfo` locker rows, then a `Decode2`. A client that received 154 in
answer to a gift would insert the gifted items into the *sender's own locker*.

Dispatch arms, from `CCashShop::OnCashItemResult` @ `0x499370`:
`0x9A → OnCashItemResBuyPackageDone` (154), `0x9B → ...BuyPackageFailed` (155),
`0x9C → OnCashItemResGiftPackageDone` (156), `0x9D → ...GiftPackageFailed` (157)
— matching the template's `BUY_PACKAGE_SUCCESS/FAILED` and
`GIFT_PACKAGE_SUCCESS/FAILED` values.

**Consumer:** Task 14 — answer mode 33 with 156/157.

---

## 6. D4a — what `option uint32` carries — **RESOLVED: the user's chosen payment method (1 / 2 / 4), not a spare field**

**Question.** What does the client put in `option` on `ShopOperationBuyPackage`,
`ShopOperationBuyCouple`, `ShopOperationBuyFriendship`, and does the server
consume it? Design §13 OQ-O1 requires it be *proven* ignorable.

**Answer.** It is **not** an ignorable field. `option` is the currency the user
selected in the purchase-confirmation dialog, and the client constrains it to
exactly one of three values.

**Step 1 — the client seeds `dwOption` with an affordability bitmask.**
`CCashShop::OnBuyPackage` @ `0x48ed40`:

```c
    nPrice = v58->nPrice; v10 = 0; dwOption = 0;
    if ( _ZtlSecureFuse<long>(m_nNexonCash, m_nNexonCash_CS) >= nPrice )
      { v10 = 1;  dwOption = 1; }                       // bit0 = NX Credit
    if ( _ZtlSecureFuse<long>(m_nMaplePoint, m_nMaplePoint_CS) >= nPrice
         && v58->nLimit != 2 )
      { v10 |= 2u; dwOption = v10; }                    // bit1 = Maple Point
    if ( _ZtlSecureFuse<long>(m_nPrepaidNXCash, ...) >= nPrice )
      { v10 |= 4u; ... }                                // bit2 = NX Prepaid
```

`CCashShop::OnBuyCouple` @ `0x490d80` builds the same mask (bits 1 and 4 only).

**Step 2 — the dialog narrows it to exactly one bit and writes it back.**
`dwOption` is passed **by pointer** into `CConfirmPurchaseDlg::Confirm`
(`0x48c2c0` → `0x48c100`, signature `..., unsigned int *dwOption, ...`), and the
inner overload writes the user's radio selection back through it:

```c
  v11 = CDialog::DoModal(v10);
  if ( v11 == 1 )
  {
    v12 = 0;
    if ( *(... + 72) ) v12 = 2;        // Maple Point checkbox
    if ( *(... + 72) ) v12 |= 1u;      // NX Credit checkbox
    if ( v13 )        v12 |= 4u;       // NX Prepaid checkbox
    if ( !v6 && v12 != 2 && v12 != 1 && v12 != 4 )
    {
      ZXString<char>::ZXString<char>(&v15,
        "You should choose only one\r\nof the three options.", -1);
      CUtilDlg::Notice(...); return 2;
    }
    *dwOption = v12;                   /* 0x48c100 */
  }
```

So on the wire `option ∈ {1, 2, 4}` = {NX Credit, Maple Point, NX Prepaid}, never
a combination and never 0 on a successful confirm.

**Step 3 — `pointType` is derived from it, and is lossy.**
`COutPacket::Encode1(&oPacket, dwOption == 2)` — the bool is true iff the payment
is Maple Point. A server that reads only `pointType` cannot distinguish NX Credit
(1) from NX Prepaid (4); only `option` carries that.

**Step 4 — what the server does today.** `option` is decoded and logged, and
nothing else, at
`services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation.go:167`
(BUY_COUPLE), `:174` (BUY_PACKAGE) and `:184` (BUY_FRIENDSHIP). There is no
consumer that branches on it anywhere in `services/` or `libs/`.

**Verdict for OQ-O1.** `option` is *currently* unconsumed by the server, so a stub
may ignore it without changing observed behaviour — but it is **not semantically
spare**, and it must not be documented as unused/reserved. Any future
implementation that debits a real balance must read `option` (1 = NX Credit,
2 = Maple Point, 4 = NX Prepaid), because `pointType` alone collapses 1 and 4.

**Consumer:** Task 17.

---

## 7. D4b — `oneADay byte` (`m_bRequestBuyOneADay`) — **RESOLVED: a client-set "this is today's One-a-Day item" request marker; the daily state itself is server-owned. Server-side enforcement is NOT determinable from the client.**

**Question.** Server-enforced per-day gift limit, or client-side UI flag?

**Answer.** Neither label is quite right, and the honest split matters for the
stub:

**(a) The flag is a request marker set only by the One-a-Day window.**
Field offset `+0x3B4C` on `CCashShop` (established from
`CCashShop::InitOneADay` @ `0x47d430`, whose decompilation is
`m_bRequestBuyOneADay = 0; m_bIsOneADay = 0;` and whose disassembly is
`mov [this+3B4Ch], eax` / `mov [this+18h], eax`).

A full `search_text` sweep for `3B4Ch` over `0x400000`–`0x700000` returns 11
sites — every write is either a reset (`InitOneADay` `0x47d432`, `CCashShop::Init`
`0x485404`) or inside `CCSWnd_OneADay::OnButtonClicked` (`0x4d14a9`, `0x4d150e`,
`0x4d1553`, `0x4d1577`, `0x4d15af`, `0x4d15dd`), and every read is a send site
(`SendBuyAvatarPacket` `0x4857f2`, `SendGiftsPacket` `0x487e29`, `OnBuy`
`0x48ec8d`). No other UI touches it. `decompile(0x4d1400)`:

```c
  if ( !nId ) {                                  // "Buy today's item"
    ...
    this->m_pCashShop->m_bRequestBuyOneADay = 1;
    CCashShop::ProcessBuy(this->m_pCashShop, Data, *(v10 + 12));
    this->m_pCashShop->m_bRequestBuyOneADay = 0;
  }
  if ( nId == 1 ) {                              // "Gift today's item"
    ...
    this->m_pCashShop->m_bRequestBuyOneADay = 1;
    if ( TSecType<long>::GetData((v13 + 16)) / 10000 == 910 )
      CCashShop::OnGiftPackage(this->m_pCashShop, *(v13 + 12));
    else
      CCashShop::OnGift(this->m_pCashShop, *(v13 + 12));
    this->m_pCashShop->m_bRequestBuyOneADay = 0;
  }
  if ( nId - 2 <= 9 && this->m_nState == 1 ) {    // buying a PREVIOUS day's item
    ...
    this->m_pCashShop->m_bRequestBuyOneADay = 0;  // explicitly cleared
    CCashShop::ProcessBuy(this->m_pCashShop, v6, p->nSN);
  }
```

It is set for exactly the two "today" buttons and explicitly cleared for the
"previous days" buttons — i.e. it tells the server *which product slot* the
purchase is against, discounted-daily vs ordinary.

**(b) The one-a-day state is server-pushed.** `CCashShop::OnPacket` @ `0x4997e0`
routes clientbound `case 0x18B → CCashShop::OnOneADay` @ `0x495950`:

```c
  this->m_nOneADayItemDate = CInPacket::Decode4(iPacket);
  this->m_nOneADayItemSN   = CInPacket::Decode4(v2);
  v4 = CInPacket::Decode4(v2);                     // count
  if ( v4 ) { ZArray<OneADayInfo>::_Realloc(...); CInPacket::DecodeBuffer(v2, ..., 12 * v4); }
  else      { ZArray<OneADayInfo>::RemoveAll(&this->m_aPrevOneADayInfo); }
```

The server chooses the day's item, its date, and the prior-days list; the client
only renders them (`CCSWnd_OneADay::DisableTodayButtons` @ `0x4d0cf0`,
`::ChangeState` @ `0x4d2b90`).

**(c) What is NOT establishable from the client.** Whether the *server* rejects a
second one-a-day purchase in the same day cannot be read out of the binary — the
client contains no such refusal path keyed on the flag, and the refusal would be
a server-side rule. Do **not** document it as "server-enforced per-day limit";
document it as the request marker in (a).

**On the wire (v95 GIFT, mode 4).** `CCashShop::SendGiftsPacket` @ `0x487b60`
confirms the field's position, which matches
`libs/atlas-packet/cash/serverbound/shop_operation_gift.go` for GMS ≥ 95:

```c
  COutPacket::COutPacket(&oPacket, 275);
  COutPacket::Encode1(&oPacket, 4u);                              // mode 4 = GIFT
  ZXString<char>::operator=(v35, &this->m_sg.sSPW);
  COutPacket::EncodeStr(&oPacket, v35[0]);                        // spw
  COutPacket::Encode4(&oPacket, this->m_sg.nCommSN);              // serialNumber
  COutPacket::Encode1(&oPacket, this->m_bRequestBuyOneADay);      // oneADay
  ZXString<char>::operator=(v35, &sDone);
  COutPacket::EncodeStr(&oPacket, v35[0]);                        // recipient name
  ZXString<char>::operator=(v35, &this->m_sg.sText);
  COutPacket::EncodeStr(&oPacket, v35[0]);                        // message
```

**Consumer:** Task 20 — treat `oneADay` as a meaningful request discriminator to
be carried and logged (today's discounted slot vs an ordinary gift), not as a
cosmetic UI byte, and do not assert a server-side daily limit.

---

## 8. Summary

| ID | Question | Status | Answer |
|---|---|---|---|
| D1 | v95 opcode for `CStage::OnSetCashShop` | RESOLVED | `0x8F` (143); handler `0x71adf0`, dispatch `0x71b0cd` in `CStage::OnPacket` `0x71b0b0` |
| D2a | APPLY_WISHLIST (35) body | RESOLVED | Empty — `Encode1(0x23)` then `SendPacket`, `CCashShop::ApplyWishListEvent` `0x482ea0` |
| D2b | APPLY_WISHLIST reply arm | RESOLVED | `UPDATE_WISHLIST` 98 (`OnCashItemResSetWishDone` `0x494d60`) — the only wishlist arm clearing `m_bCashShopRequestSent` |
| D3a | BUY_OTHER_PACKAGE (33) body | RESOLVED | `spw str, serialNumber u32, name str, message str` — `CCashShop::OnGiftPackage` `0x4907b0`; NOT the BuyPackage shape |
| D3b | BUY_OTHER_PACKAGE reply arm | RESOLVED | `GIFT_PACKAGE_SUCCESS` 156 / `GIFT_PACKAGE_FAILED` 157 (`0x496dc0`, `0x496f20`) |
| D4a | `option uint32` semantics | RESOLVED | Selected payment method, exactly one of 1 (NX Credit) / 2 (Maple Point) / 4 (NX Prepaid); `CConfirmPurchaseDlg::Confirm` `0x48c100` writes it back. Server currently only logs it |
| D4b | `oneADay byte` semantics | RESOLVED | Client-set marker for "today's One-a-Day slot" (`CCSWnd_OneADay::OnButtonClicked` `0x4d1400`); daily state pushed by server via `CCashShop::OnOneADay` `0x495950`. Server-side limit enforcement unknowable from the client |

No question was left UNRESOLVED. The two soft spots are flagged in place: the
request→response binding in §3 is inferential (no correlation table exists in the
client), and the server-side one-a-day rule in §7(c) is outside the binary.
