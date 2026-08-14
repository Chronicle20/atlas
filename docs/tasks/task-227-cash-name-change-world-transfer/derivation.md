# task-227 — Derivation

Blocking artifact for plan Task 1. Everything below was read out of the client
IDBs listed in §0 or out of repo source on this branch. Nothing is recalled or
inferred from op names — where a symbol and the code disagree, the code wins and
the symbol was corrected in the IDB.

**Headline:** the coverage matrix's GMS opcode assignment was *right*
(`NAME_TRANSFER` = the 5400000 flow, `WORLD_TRANSFER` = the 5401000 flow); the
**v83 IDB symbols were transposed**, and so was the **jms_v185** one. Three
further defects were found and fixed: the v48/v61/v72/v79 serverbound rows were
missing from the registry entirely (matrix said "op absent"; the client sends
it), v48 uses a different pair of opcodes, and row 409's receiver list carried
three stale `sub_*` names whose registry notes were factually wrong.

---

## 0. IDA sessions used

`select_instance(port)` is dead; every call passed `database=<session>` resolved
from `idb_list` by binary **name**.

| Version | Binary | Session |
|---|---|---|
| gms_v48 | `GMS_v48_1_DEVM.exe.i64` | `93cc947e` |
| gms_v61 | `GMS_v61.1_U_DEVM.exe.i64` | `415bf585` |
| gms_v72 | `GMS_v72.1_U_DEVM.exe.i64` | `c8acae95` |
| gms_v79 | `GMS_v79_1_DEVM.exe.i64` | `1438cecd` |
| gms_v83 | `MapleStory_dump.exe.i64` | `41f13e0d` |
| gms_v84 | `GMS_v84.1_U_DEVM.i64` | `5881cf84` |
| gms_v87 | `GMSv87_4GB.exe.i64` | `d51ecbd3` |
| gms_v92 | `GMS_v92_1_DEVM.exe.i64` | `acdfccff` |
| gms_v95 | `GMS_v95.0_U_DEVM.exe.i64` | `79906a1e` |
| jms_v185 | `MapleStory_dump_SCY.exe.i64` | `b6864e54` |

---

## 1. Authoritative serverbound opcode assignment

### 1.1 The disambiguator

Both sends are byte-identical in shape on GMS (`COutPacket(n)` + two `Encode4`
+ `SendPacket` + set `m_bCashShopRequestSent`), so the shape cannot tell them
apart. Two independent discriminators do:

1. **`CCashShop::ProcessBuy` compares the cash item id against two exact
   constants** and routes to a distinct `OnBuy*` per constant. On v83 the
   comparison renders as an address operand because IDA typed the immediate as
   an offset — the raw disassembly is unambiguous:

   ```
   .text:0047326D   cmp  edi, (offset loc_5265BC+4)   ; 0x5265C0 = 5400000
   .text:00473273   jnz  short loc_47327D
   .text:00473276   call sub_47031E                   ; -> COutPacket(0x10)
   .text:0047327D   cmp  edi, offset off_5269A8       ; 0x5269A8 = 5401000
   .text:00473286   call ?OnBuyNameChange@CCashShop@@QAEXJ@Z  ; -> COutPacket(0x12)
   ```

2. **Only one of the two `OnBuy*` arms calls
   `CCashShop::CheckTransferWorldPossible`** — the function that formats the
   world-transfer refusals. v83 `@0x4734e5`:

   ```c
   ZXString<char>::Format(arg0, aGuildMasterCan);  // "Guild Master can not transfer worlds."
   ZXString<char>::Format(arg0, aGmCanNotTransf);  // "GM can not transfer worlds."
   StringPool::GetString(Instance, &a1, SP_5017_YOU_HAVE_TO_QUIT_FAMILY__R_NTO_MOVE_TO_ANOTHER_WORLD)
   ```

   That gate sits in the **5401000 → `COutPacket(0x12)`** arm. Therefore
   `0x012` is world transfer and `0x010` is name change — which is exactly what
   `docs/packets/audits/STATUS.md` rows 528/532 already said.

### 1.2 Per-version table

| Version | `NAME_TRANSFER` | send address | `COutPacket` line | `WORLD_TRANSFER` | send address | `COutPacket` line |
|---|---|---|---|---|---|---|
| gms_v48 | **0x012** | `0x44f93e` | `push 12h` @`0x44f950` | **0x014** | `0x44fb95` | `push 14h` @`0x44fba7` |
| gms_v61 | 0x010 | `0x45cd72` | `COutPacket::COutPacket((COutPacket *)v5, 16);` | 0x012 | `0x45cefd` | `COutPacket::COutPacket((COutPacket *)v5, 18);` |
| gms_v72 | 0x010 | `0x46bc47` | `COutPacket::COutPacket((COutPacket *)v5, 16);` | 0x012 | `0x46be19` | `COutPacket::COutPacket((COutPacket *)v5, 18);` |
| gms_v79 | 0x010 | `0x46cdbe` | `COutPacket::COutPacket((COutPacket *)v5, 16);` | 0x012 | `0x46cf90` | `COutPacket::COutPacket((COutPacket *)v5, 18);` |
| gms_v83 | 0x010 | `0x4733ca` | `COutPacket::COutPacket(&v4, 16);` | 0x012 | `0x47359c` | `push 12h` @`0x4735ac` |
| gms_v84 | 0x010 | `0x475f14` | `push 10h` @`0x475f24` | 0x012 | `0x4760e6` | `push 12h` @`0x4760f6` |
| gms_v87 | 0x010 | `0x47dd48` | `COutPacket::COutPacket(&a3, 0x10);` | 0x012 | `0x47df1a` | `COutPacket::COutPacket(&a3, 0x12);` |
| gms_v92 | 0x010 | `0x47f830` | `COutPacket::COutPacket((COutPacket *)&v5, 0x10u);` | 0x012 | `0x47f8c0` | `COutPacket::COutPacket((COutPacket *)&v5, 0x12u);` |
| gms_v95 | 0x010 | `0x488190` | `COutPacket::COutPacket(&oPacket, 16);` | 0x012 | `0x4884c0` | `COutPacket::COutPacket(&oPacket, 18);` |
| jms_v185 | **absent** | — | — | **0x009** | `0x484fbf` | `push 9` @`0x484fd3` |

### 1.3 Call chains (per version)

| Version | `ProcessBuy` | 5400000 arm → send | 5401000 arm → send | `CheckTransferWorldPossible` |
|---|---|---|---|---|
| gms_v48 | `0x44f7b8` | `OnBuyNameChange 0x44c5f1` → `0x44f93e` | `OnBuyTransferWorldItem 0x44c707` → `0x44fb95` | `0x44fa5d` |
| gms_v61 | `0x45cb70` | `OnBuyNameChange 0x4595db` → `0x45cd72` | `OnBuyTransferWorldItem 0x459737` → `0x45cefd` | `0x45ce91` |
| gms_v72 | `0x46ba37` | `OnBuyNameChange 0x468a2b` → `0x46bc47` | `OnBuyTransferWorldItem 0x468b93` → `0x46be19` | `0x46bd62` |
| gms_v79 | `0x46cc61` | `OnBuyNameChange 0x469c74` → `0x46cdbe` | `OnBuyTransferWorldItem 0x469dd6` → `0x46cf90` | `0x46ced9` |
| gms_v83 | `0x4731a9` | `0x47031e` → `0x4733ca` | `0x470480` → `0x47359c` | `0x4734e5` |
| gms_v84 | `0x475c9f` | `0x472e14` → `0x475f14` | `0x472f76` → `0x4760e6` | `0x47602f` |
| gms_v87 | `0x47dad3` | `OnBuyNameChange 0x47ab57` → `0x47dd48` | `OnBuyTransferWorldItem 0x47acb9` → `0x47df1a` | (in `0x47acb9`) |
| gms_v92 | `0x48f680` | `0x484280` → `0x47f830` | `0x48d5b0` → `0x47f8c0` | `0x4847e0` |
| gms_v95 | `0x493854` | `OnBuyNameChange 0x491200` → `0x488190` | `OnBuyTransferWorldItem 0x491470` → `0x4884c0` | (in `0x491470`) |
| jms_v185 | `0x484ca8` | **no 5400000 arm** | `0x480bca` → `0x484fbf` | — |

### 1.4 Symbol corrections applied to the IDBs

The read order / call chain is the evidence; these symbols were wrong and were
renamed (`mcp__ida-pro__rename`, `allow_overwrite`) so the next reader is not
misled:

| IDB | Address | Was | Now |
|---|---|---|---|
| gms_v83 | `0x47031e` | `sub_47031E` | `CCashShop::OnBuyNameChange` |
| gms_v83 | `0x4733ca` | `sub_4733CA` | `CCashShop::SendCheckNameChangePossiblePacket` |
| gms_v83 | `0x470480` | `?OnBuyNameChange@CCashShop@@QAEXJ@Z` | `CCashShop::OnBuyTransferWorld` |
| gms_v83 | `0x47359c` | `?SendCheckNameChangePossiblePacket@CCashShop@@QAEXKV?$ZXString@D@@@Z` | `CCashShop::SendCheckTransferWorldPossiblePacket` |
| gms_v84 | `0x475c9f` / `0x472e14` / `0x475f14` / `0x472f76` / `0x4760e6` / `0x47602f` | all `sub_*` | `ProcessBuy` / `OnBuyNameChange` / `SendCheckNameChangePossiblePacket` / `OnBuyTransferWorld` / `SendCheckTransferWorldPossiblePacket` / `CheckTransferWorldPossible` |
| gms_v92 | `0x48f680` / `0x484280` / `0x47f830` / `0x48d5b0` / `0x47f8c0` / `0x4847e0` | all `sub_*` | same six names |
| gms_v48 | `0x44fb95` | `sub_44FB95` | `CCashShop::SendCheckTransferWorldPossiblePacket` |
| gms_v61 | `0x45cefd` | `sub_45CEFD` | `CCashShop::SendCheckTransferWorldPossiblePacket` |
| gms_v72 | `0x46be19` | `sub_46BE19` | `CCashShop::SendCheckTransferWorldPossiblePacket` |
| gms_v79 | `0x46cf90` | `sub_46CF90` | `CCashShop::SendCheckTransferWorldPossiblePacket` |
| gms_v48 | `0x455a7f` | `sub_455A7F` | `CCashShop::OnCheckDuplicatedIDResult` |
| gms_v61 | `0x463900` | `sub_463900` | `CCashShop::OnCheckDuplicatedIDResult` |
| gms_v72 | `0x473519` | `sub_473519` | `CCashShop::OnCheckDuplicatedIDResult` |
| jms_v185 | `0x480bca` | `?OnBuyNameChange@CCashShop@@QAEXJ@Z` | `CCashShop::OnBuyTransferWorld` |
| jms_v185 | `0x484fbf` | `?SendCheckNameChangePossiblePacket@CCashShop@@QAEXKV?$ZXString@D@@@Z` | `CCashShop::SendCheckTransferWorldPossiblePacket` |

### 1.5 jms_v185 is world transfer, not name change

The matrix had `NAME_TRANSFER` × jms_v185 = `0x009` on the strength of the IDB
symbol. Three independent facts say that is wrong:

1. `CCashShop::ProcessBuy @0x484ca8` routes **only** `5401000`
   (`nItemID == (&loc_5269A6 + 2)`) into `0x480bca`. There is no `5400000`
   branch; `find(immediate 5400000)` returns **zero** hits in the whole binary.
   On all nine GMS versions `5401000` is the world-transfer item.
2. jms_v185 contains **no** `CUIChangingCharacterName`, `CUIChangingLicenseNotice`,
   `CCashShop::OnCheckNameChangePossibleResult`, `CCashShop::OnCheckDuplicatedIDResult`,
   or `SendBuyNameChangeItemPacket` (`func_query` over all five patterns → 0 rows).
   The name-change feature does not exist in this client.
3. It *does* contain the entire transfer flow: `CUITransferWorldSelectDlg`,
   `CUITransferWorldLicenseNotice`, `CCashShop::OnCheckTransferWorldPossibleResult`
   (`0x48e7a6`, opcode 0x16C), `SendBuyTransferWorldItemPacket`,
   `OnCashItemResTransferWorldDone/Failed`. The send at `0x484fbf` sets
   `*(this + 7) = 1`; `0x48e7a6` clears `*(this + 7) = 0` — the request/response
   pair.

`docs/packets/registry/jms_v185.yaml` opcode 9 was moved from `NAME_TRANSFER` to
`WORLD_TRANSFER` and the matrix regenerated.

### 1.6 The legacy four were missing, not absent

`NAME_TRANSFER` and `WORLD_TRANSFER` were `⬜` on v48/v61/v72/v79 — the matrix's
"op not in this version" state, which derives purely from registry absence.
Every one of those four clients builds and sends both packets (§1.2/§1.3).
Registry rows were added with `provenance: ida-discovered`; the eight cells now
grade `❌` (unimplemented) instead of `⬜` (absent). **v48 uses 0x012/0x014**, not
0x010/0x012 — a two-slot offset that first-time implementers will otherwise get
wrong from the v61+ pattern.

---

## 2. Field read order, per op, per version

### 2.1 `NAME_TRANSFER` (serverbound)

| Versions | Body |
|---|---|
| gms_v48 … gms_v92 | `Encode4 dwCharacterID` · `Encode4 nBirthDate` |
| gms_v95 | `Encode4 dwCharID` · `EncodeStr sSPW` |
| jms_v185 | n/a (op absent) |

The second field is **not** a pointer despite the `ZXString<char>` in the
pre-v95 mangled prototypes — those prototypes are wrong. v83 `ask_SPW`
(`0x9acf95`) ends `v8 = atoi(*InputStr_Result); … return v8;` and returns `-1`
on cancel, so the value encoded is an **8-digit integer birthday code**. The
prompt is `SP_826_FOR_THE_SAKE_OF_PRIVACY_R_NPLEASE_ENTER_YOUR_8_DIGIT_BIRTHDAY_CODE…`.
v95 is PDB-backed and names the parameter `sSPW`, encoded as a length-prefixed
string.

### 2.2 `WORLD_TRANSFER` (serverbound)

| Versions | Body |
|---|---|
| gms_v48 … gms_v92 | `Encode4 dwCharacterID` · `Encode4 nBirthDate` |
| gms_v95 | `Encode4 dwCharacterID` · `EncodeStr sSPW` |
| jms_v185 | `EncodeStr sSPW` only — **no character id** |

jms_v185's `retn 4` (`0x485035`) proves a single stack argument; the applied
two-argument type is wrong.

### 2.3 `CASHSHOP_CHECK_NAME_CHANGE` (clientbound) — `CCashShop::OnCheckDuplicatedIDResult`

Identical on every version v48…v95:

```
DecodeStr  sName
Decode1    nResult      // > 0 name already in use; == 0 available; < 0 unknown error
```

Receivers: v48 `0x455a7f`, v61 `0x463900`, v72 `0x473519`, v79 `0x4749e5`,
v83 `0x47baea`, v84 `0x47ec88`, v87 `0x4872cb`, v92 `0x493f40`, v95 `0x497fb0`.
On success the client stashes `sName` into the rename dialog
(`CUIChangingCharacterName::SetNameValues` — the PDB-named v95 call at
`0x497fb0`). jms_v185: op absent.

StringPool ids rendered (duplicate / available / unknown): v48 498/499/3222 ·
v61 530/531/3492 · v72 519/520/3546 · v79 519/520/3550 · v83 `SP_517`/`SP_518`/`SP_3570` ·
v84 520/521/3573 · v87 527/528/3579 · v92 534/535/3639 · v95 534/535/3606.

### 2.4 `CASHSHOP_CHECK_NAME_CHANGE_POSSIBLE_RESULT` (clientbound)

Identical on v79…v95:

```
Decode4    dwCharacterID   // read and discarded by the client
Decode1    nResult
Decode4    nBirthDate      // v95 PDB names this local `nBirthDate`
```

`nResult == 0` opens `CUIChangingLicenseNotice` and calls `SetBirthDate(nBirthDate)`.
Receivers: v79 `0x474ab1`, v83 `0x47bbb6`, v84 `0x47ed54`, v87 `0x487397`,
v92 `0x4913a0`, v95 `0x495470`.

### 2.5 `CASHSHOP_CHECK_TRANSFER_WORLD_POSSIBLE_RESULT` (clientbound)

GMS v48…v95:

```
Decode4    dwCharacterID   // read and discarded
Decode1    nResult
Decode4    nBirthDate
Decode1    bHasWorldList
  if bHasWorldList:
    Decode4  nCount
    nCount ×  DecodeStr    // -> CCashShop::m_asWorldName (v95 PDB name)
```

jms_v185 (`0x48e7a6`) **drops `nBirthDate`**:

```
Decode4    dwCharacterID
Decode1    nResult
Decode1    bHasWorldList
  if bHasWorldList: Decode4 nCount; nCount × DecodeStr
```

and its success arm constructs `CUITransferWorldLicenseNotice` with **no**
`SetBirthDate` call. The world-name list is decoded **before** the result switch
on every version, so it is present on failure paths too.

Receivers: v48 `0x455d25`, v61 `0x463ba6`, v72 `0x4737ca`, v79 `0x474c96`,
v83 `0x47bd9b`, v84 `0x47ef39`, v87 `0x48757c`, v92 `0x494040`, v95 `0x4980b0`,
jms `0x48e7a6`.

### 2.6 `CANCEL_NAME_CHANGE_RESULT` (clientbound)

Identical on v61…v95:

```
Decode1    nResult
  nResult == 0x00 -> CUICancelCharacterCouponResults(0), modal, no further read
  nResult == 0xFF -> CUICancelCharacterCouponResults(1), modal, no further read
  otherwise:
    Decode1  bHasMessage
      if bHasMessage: DecodeStr sMessage  -> Notice(sMessage)
      else:                                  Notice(StringPool unknown-error)
```

Receivers: v61 `0x84ace9`, v72 `0x922399`, v79 `0x9744ce`, v83 `0xa2a677`,
v84 `0xa75e3a`, v87 `0xac2313`, v92 `0x9d64a0`, v95 `0xa01b10`.
Unknown-error StringPool id: v61/v72 3492/3546 · v79 3550 · v83 `SP_3570` ·
v84 3573 · v87 3579 · v92 3639 · v95 0xE16 (3606).

### 2.7 `CANCEL_TRANSFER_WORLD_RESULT` (clientbound)

Identical on v61…v95, and identical in shape to §2.6 except the two success
constants:

```
Decode1    nResult
  nResult == 0x00 -> CUICancelCharacterCouponResults(2)
  nResult == 0x01 -> CUICancelCharacterCouponResults(3)     // 0x01, NOT 0xFF
  otherwise: Decode1 bHasMessage [ + DecodeStr sMessage ]
```

Receivers: v61 `0x84ae56`, v72 `0x92254f`, v79 `0x974684`, v83 `0xa2a82d`,
v84 `0xa75ff0`, v87 `0xac24c9`, v92 `0x9d6680`, v95 `0xa01cf0`.

> The `0xFF` / `0x01` asymmetry between the two cancel packets is the single
> easiest way to write two decoders that both pass a fixture and are silently
> wrong. It holds on all eight versions.

### 2.8 `CANCEL_NAME_CHANGE_BY_OTHER` (clientbound)

**Empty body — the client decodes nothing.** It renders one StringPool string
and clears a `CWvsContext` flag plus a timestamp:

```c
// gms_v83 @0xa2a7e6
StringPool::GetString(Instance, &v4, SP_391_THE_CHARACTER_YOU_LOGGED_ON_WITH_R_NHAS_NOT_REQUESTED_A_NAME_CHANGE);
CUtilDlg::Notice(v4, ...);
*&TSingleton<CWvsContext>::ms_pInstance[1].m_Cookie.szCookie[96] = 0;
*&v3[1].m_Cookie.szCookie[100] = get_update_time();
```

Receivers: v72 `0x922508` (SP 393), v79 `0x97463d` (SP 393), v83 `0xa2a7e6`
(`SP_391`), v84 `0xa75fa9` (SP 394), v87 (`0xac2482`, size 0x47 — same shape),
v92 `0x9cc170` (SP 406), v95 `0x9f7620` (SP 0x196 = 406).

---

## 3. The 540 prefix → feature split (OQ-1)

**`5400000` is name change. `5401000` is world transfer.** Route used:
**(b) the client's own classifier + `ProcessBuy` arm.** Route (a) (atlas-data
classification-540 templates) was not needed and was not queried.

Evidence, per version, is the `ProcessBuy` comparison in §1.3 — every GMS
version from v48 to v95 compares against the **exact ids** `5400000` and
`5401000`, not against prefixes, and only the `5401000` arm is gated by
`CheckTransferWorldPossible`.

`get_cashslot_item_type` agrees and recognises exactly two prefixes under
classification 540:

```c
// gms_v95 @0x488c70
case 540:
  if ( nItemID / 1000 == 5400 ) result = 53;
  else { if ( nItemID / 1000 != 5401 ) goto $LN17_9; result = 54; }
```

```
; gms_v83 @0x4866ef  (jumptable case 540)
idiv esi                  ; / 1000
sub  eax, 1518h           ; 5400
jz   short loc_486707     ; -> push 34h  (=52)
dec  eax                  ; 5401
jnz  short loc_48670E     ; -> falls through to case 542
push 35h                  ; (=53)
```

So: v83 `5400 -> 52`, `5401 -> 53`; v95 `5400 -> 53`, `5401 -> 54`. That is
exactly what `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:1117-1131`
already encodes.

**The duplicate at `character_cash_item_use.go:1132-1138` is dead code, not a
placeholder for a third prefix.** No GMS client between v48 and v95 has a third
540 branch, and any `540xxxx` that is neither `5400xxx` nor `5401xxx` falls
through to `0`. Delete the block; do not invent a third id.

**jms_v185 diverges:** it has no `5400000` at all, and maps `5401000` to the
world-transfer flow (§1.5).

---

## 4. Reason-code enumerations (OQ-7)

### 4.1 `CASHSHOP_CHECK_NAME_CHANGE_POSSIBLE_RESULT`

`nResult` is a byte. Four distinct outcomes on every version v79…v95:

| `nResult` | Client behaviour | v79 | v83 | v84 | v87 | v92 | v95 |
|---|---|---|---|---|---|---|---|
| 0 | open `CUIChangingLicenseNotice`, `SetBirthDate(nBirthDate)` | — | — | — | — | — | — |
| 1 | Notice(SP) — "the name change is already submitted due to the item purchase" | 523 | `SP_521` | 524 | 531 | 538 | 538 |
| 2 | Notice(SP) — request-limit, "check if you were recently…" | 524 | `SP_522` | 525 | 532 | 539 | 539 |
| 3 | Notice(SP) — request-limit, "check if you requested…" | 525 | `SP_523` | 526 | 533 | 540 | 540 |
| any other | Notice(SP) — "an unknown error has occurred" | 3550 | `SP_3570` | 3573 | 3579 | 3639 | 3606 |

The v83 enum member names carry the text verbatim:
`SP_521_THE_NAME_CHANGE_IS_ALREADY_SUBMITTED__R_NDUE_TO_THE_ITEM_PURCHASE`,
`SP_522_THIS_APPLIES_TO_THE_LIMITATIONS_ON_THE_REQUEST_R_NPLEASE_CHECK_IF_YOU_WERE_RECEN…`,
`SP_523_THIS_APPLIES_TO_THE_LIMITATIONS_ON_THE_REQUEST_R_NPLEASE_CHECK_IF_YOU_REQUESTED_`,
`SP_3570_AN_UNKNOWN_ERROR_HAS_OCCURED`.

### 4.2 `CASHSHOP_CHECK_TRANSFER_WORLD_POSSIBLE_RESULT`

Nine distinct outcomes from v72 on; **v48 and v61 have no case 8** (their
`switch` ends at 7 and falls to `default`), which matches the family gate
arriving with the family system.

| `nResult` | Client behaviour | v48 | v61 | v72 | v79 | v83 | v84 | v87 | v92 | v95 | jms |
|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | open `CUITransferWorldLicenseNotice` (+`SetBirthDate` except jms) | — | — | — | — | — | — | — | — | — | — |
| 1 | Notice "Cannot find Character Information." (literal, not StringPool, on v61…v95) | * | lit | lit | lit | lit | lit | lit | lit | lit | SP 5459 |
| 2 | Notice(SP) | 3655 | 3927 | 3978 | 3981 | 4002 | 4005 | 4010 | 4076 | 4043 | 5452 |
| 3 | Notice(SP) | 3661 | 3933 | 3985 | 3988 | 4009 | 4012 | 4017 | 4083 | 4050 | 5458 |
| 4 | Notice(SP) | 3656 | 3928 | 3979 | 3982 | 4003 | 4006 | 4011 | 4077 | 4044 | 5453 |
| 5 | Notice(SP) | 3657 | 3929 | 3980 | 3983 | 4004 | 4007 | 4012 | 4078 | 4045 | 5454 |
| 6 | Notice(SP) | 3667 | 3939 | 3991 | 3994 | 4015 | 4018 | 4023 | 4089 | 4056 | 5465 |
| 7 | Notice(SP) | 3662 | 3934 | 3986 | 3989 | 4010 | 4013 | 4018 | 4084 | 4051 | 5460 |
| 8 | Notice(SP) — "you have to quit family to move to another world" | **n/a** | **n/a** | 4996 | 4979 | 5017 | 5022 | 5028 | 5095 | 5035 | 5559 |
| any other | Notice(SP) — unknown error | 3222 | 3492 | 3546 | 3550 | 3570 | 3573 | 3579 | 3639 | 3606 | 5482 |

\* v48's case-1 arm decompiles garbled (`CDialog::DoModal((CDialog *)&v8); CUtilDlg::Notice(...)`)
and the function's string refs contain no `aCannotFindChar`; the rendered text
for v48 case 1 is **unverified**. It does not affect the wire contract.

Case 8's text is confirmed on v83 by the enum name
`SP_5017_YOU_HAVE_TO_QUIT_FAMILY__R_NTO_MOVE_TO_ANOTHER_WORLD`, referenced from
`CCashShop::CheckTransferWorldPossible @0x473519` — the same id the result
handler renders. The other StringPool ids are recorded as **numeric ids only**:
the `StringPoolStrings` enum in these IDBs is not member-enumerable
(`type_inspect` returns `member_count: 0`), and the operand type was applied at
only a handful of v83 sites. The numeric id is what the codec and any future
mapping table need; the rendered text is client-side presentation.

### 4.3 `CANCEL_NAME_CHANGE_RESULT` / `CANCEL_TRANSFER_WORLD_RESULT`

Not a flat enum — a three-way shape switch (§2.6/§2.7):

| Packet | `nResult` | Meaning |
|---|---|---|
| `CANCEL_NAME_CHANGE_RESULT` | `0x00` | cancelled — `CUICancelCharacterCouponResults(0)` |
| | `0xFF` | cancelled, alternate variant — `…(1)` |
| | anything else | failure; body continues `Decode1 bHasMessage` [+ `DecodeStr`] |
| `CANCEL_TRANSFER_WORLD_RESULT` | `0x00` | cancelled — `…(2)` |
| | `0x01` | cancelled, alternate variant — `…(3)` |
| | anything else | failure; body continues `Decode1 bHasMessage` [+ `DecodeStr`] |

### 4.4 `CASHSHOP_CHECK_NAME_CHANGE`

`nResult` is **signed**: `> 0` duplicate, `== 0` available, `< 0` unknown error
(§2.3). Every version tests it with a signed comparison
(`if (v2 > 0) … else if (v2) … else …`).

---

## 5. Is `CASHSHOP_CHECK_NAME_CHANGE` a dispatcher family? — **No**

Per `docs/packets/DISPATCHER_FAMILY.md` a family is one opcode whose **leading
byte switches to N sub-handler arms with distinct bodies**. This op has:

* a single receiver per version (`CCashShop::OnCheckDuplicatedIDResult`),
  reached from one `case` of `CCashShop::OnPacket` — v83 `0x478e2b`:

  ```c
  case 0x148: CCashShop::OnCheckDuplicatedIDResult(this - 2, a3); break;
  case 0x149: CCashShop::OnCheckNameChangePossibleResult(this - 2, a3); break;
  case 0x14B: CCashShop::OnCheckTransferWorldPossibleResult(this - 2, a3); break;
  ```
* a **flat, unconditional** body (`DecodeStr` then `Decode1`) — the result byte
  is read *after* the string and selects only which `Notice` string to render,
  never a different field layout.

The three-extra-receiver appearance in `STATUS.md` row 409 was an artifact: the
v48/v61/v72 registries carried the then-unnamed `sub_455A7F` / `sub_463900` /
`sub_473519` as `fname`, and v48 additionally carried
`CCashShop::OnCheckNameChangePossibleResult` as an `fname_alt`. All three subs
decompile to the same `DecodeStr` + `Decode1` + `Notice` body as v83's
`OnCheckDuplicatedIDResult`; the v48 and v61 registry notes claiming they were
`OnCheckNameChangePossibleResult` were **wrong** (that handler reads
`Decode4`+`Decode1`+`Decode4` and does not exist before v79). The three IDBs
were renamed, the three registry rows corrected, and the matrix regenerated —
row 409 now reads a single receiver.

**No dispatcher-family work is required for this task.**

---

## 6. Matrix / registry amendments made by this pass

`go run ./tools/packet-audit matrix --check` was exit 0 **before** these edits,
and is exit 0 after; `fname-doc --check`, `dispatcher-lint` and
`operations --check` are all exit 0.

| File | Change |
|---|---|
| `docs/packets/registry/gms_v48.yaml` | `CASHSHOP_CHECK_NAME_CHANGE` fname `sub_455A7F` → `CCashShop::OnCheckDuplicatedIDResult`, wrong `fname_alt` and note removed; **added** `NAME_TRANSFER` sb 18 and `WORLD_TRANSFER` sb 20 |
| `docs/packets/registry/gms_v61.yaml` | fname `sub_463900` → `CCashShop::OnCheckDuplicatedIDResult`, note corrected; **added** `NAME_TRANSFER` sb 16 and `WORLD_TRANSFER` sb 18 |
| `docs/packets/registry/gms_v72.yaml` | fname `sub_473519` → `CCashShop::OnCheckDuplicatedIDResult`, note expanded; **added** `NAME_TRANSFER` sb 16 and `WORLD_TRANSFER` sb 18 |
| `docs/packets/registry/gms_v79.yaml` | **added** `NAME_TRANSFER` sb 16 and `WORLD_TRANSFER` sb 18 |
| `docs/packets/registry/jms_v185.yaml` | sb 9 moved `NAME_TRANSFER` → `WORLD_TRANSFER`, fname corrected, full evidence in the note |
| `docs/packets/audits/STATUS.md`, `status.json` | regenerated |

Resulting matrix rows:

```
| NAME_TRANSFER  | … | 0x012 ❌ | 0x010 ❌ | 0x010 ❌ | 0x010 ❌ | 0x010 ❌ | 0x010 ❌ | 0x010 ❌ | 0x010 ❌ | 0x010 ❌ |  ⬜ |
| WORLD_TRANSFER | … | 0x014 ❌ | 0x012 ❌ | 0x012 ❌ | 0x012 ❌ | 0x012 ❌ | 0x012 ❌ | 0x012 ❌ | 0x012 ❌ | 0x012 ❌ | 0x009 ❌ |
```

---

## 7. Hazards for the downstream tasks

1. **The plan's cell count was 59; it is now 67.** The serverbound rows gained
   the four legacy GMS versions and jms moved rows. See `coverage-manifest.yaml`.
2. **v48's serverbound opcodes are 0x012 / 0x014**, not 0x010 / 0x012. Anything
   that copies the v61+ constants into v48 will bind the wrong opcode.
3. **v95 changed the second field from `Encode4` to `EncodeStr`.** A single
   codec with a `MajorAtLeast(95)` gate is required; the field is not merely
   widened, it changes wire type.
4. **jms_v185's `WORLD_TRANSFER` body has no character id** and its
   `CASHSHOP_CHECK_TRANSFER_WORLD_POSSIBLE_RESULT` has no `nBirthDate`.
5. **The second field is the account second password / birthday code.** v83–v92
   send it as an integer, v95+/jms as a string. `design.md §1.7` and
   `context.md §4` are right that it must be redacted inside the struct's
   `String()`, not at the call site.
6. **`CANCEL_TRANSFER_WORLD_RESULT` uses `0x01` where `CANCEL_NAME_CHANGE_RESULT`
   uses `0xFF`.** Both are "cancelled" on the wire.
7. **The gms_v79 IDB's `CCashShop::OnBuy*` symbols are shifted by one arm.**
   `ProcessBuy @0x46cc61` routes `a2/100 == 11120` to `OnBuyCharacter`,
   `a2/10000 == 910` to `OnBuyCouple`, the 8e7..9e7 commodity-SN range to
   `OnBuyPackage`, `11128` to `OnBuyNormal`, `5430` to `OnBuyFriendship`,
   `555` to `OnIncCharacterSlotCount` and `911` to `OnEnableEquipSlotExt` — each
   one position off from the v83/v87/v95 semantics. The two arms this task needs
   (`OnBuyNameChange`/`OnBuyTransferWorldItem`) are **correct**, so nothing was
   renamed, but any future `CASHSHOP_OPERATION` / cash-purchase work on v79 must
   re-derive from the item-id comparisons rather than the symbol names.
8. **`character_cash_item_use.go:1132-1138` must be deleted, not filled in**
   (§3).
