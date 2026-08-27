# task-240 — Equip-slot extension derivation

Blocking artifact for plan Task 21. Same format as
[`derivation.md`](derivation.md): every fact below was read out of a client IDB
identified in §0 or out of repo source on this branch. Nothing here is recalled
from general MapleStory knowledge, and no value is inferred from a symbol name
alone — for each answer the decompilation or the disassembly is quoted.

Consumer: Task 23 (the ENABLE_EQUIP_SLOT effect half; FR-SLOT-1/2/3).

---

## 0. IDB identity

The brief calls the primary database "the GMS v95.1 IDB". As recorded in
[`derivation.md` §0](derivation.md), **no v95.1 database exists in this
ida-pro-mcp instance**; the v95 answers below come from the v95.0 binary and are
applied to the `gms_95_1` tenant template, the same substitution every other v95
row in the repo makes.

Because E1 turned out to be **version-dependent** (§1), seven databases were
consulted, not one. All values as reported by `idb_list`:

| Tenant column | `filename` | `session_id` |
|---|---|---|
| gms_v72 | `GMS_v72.1_U_DEVM.exe.i64` | `f2a2e7c1` |
| gms_v79 | `GMS_v79_1_DEVM.exe.i64` | `f36df4cd` |
| gms_v83 | `MapleStory_dump.exe.i64` | `41f09cce` |
| gms_v84 | `GMS_v84.1_U_DEVM.i64` | `306dc69a` |
| gms_v87 | `GMSv87_4GB.exe.i64` | `0d9cf8b6` |
| gms_v92 | `GMS_v92_1_DEVM.exe.i64` | `e3328b84` |
| gms_v95 (→ `gms_95_1`) | `GMS_v95.0_U_DEVM.exe.i64` | `32c8836f` |
| jms_v185 | `MapleStory_dump_SCY.exe.i64` | `05eb9c27` |

`idb_list` reports no hash for any session; binary identity is pinned by
filename + session only. The v48 and v61 instances are listed by `idb_list` but
are **not adopted** (empty `session_id`), so they were not queried.

Mode numbers are taken from the tenant template, not from memory.
`services/atlas-configurations/seed-data/templates/template_gms_95_1.json`:

```
2311:            "ENABLE_EQUIP_SLOT": 10,
4839:            "ENABLE_EQUIP_SLOT_EXT_SUCCESS": 117,
4840:            "ENABLE_EQUIP_SLOT_EXT_FAILED": 118,
```

Serverbound mode 10 = `0xA` is confirmed against the client sender in §1.3.

---

## 1. E1 — the slot index — **RESOLVED, with a version-dependence the brief did not anticipate**

**Question (brief Step 1).** From `CCashShop::OnEnableEquipSlotExt`'s result
handler in the v95 IDB: what does the client do with the `slotIndex uint16` it
reads, and which equipped-inventory slot position does the extended pendant
occupy?

**Answer, in two parts:**

- **The wire value `slotIndex` is always `0`.** It is not a body-part number. It
  is an index into a one-element per-character array of expiry `FILETIME`s, and
  every client checked bounds it to exactly one legal value.
- **The equipped-inventory body part is `slotIndex + K`, where `K` is
  version-dependent**: `K = 51` on GMS v79/v83/v84, `K = 59` on GMS v87/v92/v95,
  `K = 36` on JMS v185. For v95 (this task's primary target) the extended
  pendant occupies **body part 59**, i.e. Atlas slot position **−59**.

### 1.1 v95 — the result handler

`CCashShop::OnCashItemResEnableEquipSlotExtDone` @ `0x497490`
(`?OnCashItemResEnableEquipSlotExtDone@CCashShop@@IAEXAAVCInPacket@@@Z`, size
`0x260`), reached from the ENABLE_EQUIP_SLOT_EXT_SUCCESS arm. `decompile(0x497490)`,
abridged to the load-bearing lines with IDA's address markers intact:

```c
  v3 = CInPacket::Decode2(iPacket);              /*0x4974c4*/   // slotIndex
  v4 = CInPacket::Decode2(v2);                   /*0x4974d2*/   // days
  p  = CWvsContext::GetCharacterData(...)->p;    /*0x4974e3*/
  ftNow = Util::FTGetNow();                      /*0x497521*/
  v7 = v3 + 59;                                  /*0x49752f*/   // <-- body part
  if ( v3 || CompareFileTime(&p->aEquipped2[v7 + 13], &ftNow) >= 0 )
  {                                              /*0x497556*/
    v13 = &p->aEquipExtExpire[v3];               /*0x4975d6*/
    v14 = Util::FTAddDay(v13, v4);               /*0x4975df*/   // extend from current expiry
    v13->dwLowDateTime  = v14.dwLowDateTime;     /*0x4975eb*/
    v13->dwHighDateTime = v14.dwHighDateTime;    /*0x4975fb*/
    ZAPI.FileTimeToSystemTime(v22, p_st);        /*0x4975fe*/
    bodyaprt_name = get_bodyaprt_name(&v24, v7); /*0x497612*/   // names body part v3+59
    m_pStr = StringPool::GetString(Instance, &v25, 0x1473u)->_m_pStr;  /*0x49763a*/
    ZXString<char>::Format(&iPacket, m_pStr, v18, st, v30, HIWORD(v31), v32, v33); /*0x49765e*/
  }
  else
  {
    v8 = Util::FTAddDay(&ftNow, v4);             /*0x49755e*/   // fresh, from now
    p->aEquipExtExpire[0].dwHighDateTime = v8.dwHighDateTime;   /*0x497563*/
    v21 = 59;                                    /*0x49756e*/
    p->aEquipExtExpire[0].dwLowDateTime  = v8.dwLowDateTime;    /*0x497570*/
    v9 = get_bodyaprt_name(v20, v21);            /*0x49757f*/   // names body part 59
    v11 = StringPool::GetString(v10, &v24, 0x1474u)->_m_pStr;   /*0x49759e*/
  }
  ...
  CUtilDlg::Notice(v19, v20, v21, v22, p_st);    /*0x4976b6*/
```

So the handler's entire observable effect is: **write one `FILETIME` into
`CharacterData::aEquipExtExpire[slotIndex]`, then pop a `CUtilDlg::Notice`.** It
inserts no item, moves no inventory slot, and sends nothing back.

`days` is a plain day count fed to `Util::FTAddDay` (`0x75ffe0`). The two arms
are *renew* (`0x4975df`, base = the existing expiry) and *first purchase*
(`0x49755e`, base = now).

`p->aEquipped2[v7 + 13]` and `&p->aEquipExtExpire[v3]` are the **same address**;
IDA rendered the first through the neighbouring array. From `type_inspect
CharacterData` (session `32c8836f`):

```
aEquipped2       offset 0x2d9  size 480  ZRef<GW_ItemSlotBase>[60]   (8 bytes/entry)
aEquipExtExpire  offset 0x519  size 8    _FILETIME[1]
m_mEquippedSetItem offset 0x521 size 24  ZMap<long,EQUIPPED_SETITEM,long>
```

`0x2d9 + 8 × (v3 + 72) = 0x519 + 8 × v3` for every `v3`. Both expressions denote
`aEquipExtExpire[v3]`.

### 1.2 Why `slotIndex` is bounded to 0 — three independent checks

1. **The array holds one element.** `aEquipExtExpire` is `_FILETIME[1]` at
   `0x519`; `0x519 + 8 = 0x521` is `m_mEquippedSetItem`. Any `slotIndex > 0`
   would write over the equipped-set-item map.
2. **`CharacterData::Decode` restores exactly one `FILETIME`** (§2), so no
   second entry could survive a relog even if one were written.
3. **The other clients bound it explicitly in the instruction stream.** v84
   (`306dc69a`) and v87 (`0d9cf8b6`) and JMS (`05eb9c27`) all compile the check
   as an equality test on `slotIndex + K`:

   ```
   gms_v84  0x47ded7  lea ebx, [esi+33h]           ; ebx = slotIndex + 51
            0x47deda  cmp ebx, 33h ; '3'
            0x47dedd  jl  loc_47DF6E
            0x47dee3  jg  loc_47DF6E               ; anything but 0 skips the whole effect

   gms_v87  0x48650b  lea ebx, [esi+3Bh]           ; ebx = slotIndex + 59
            0x48650e  cmp ebx, 3Bh ; ';'
            0x486511  jl  loc_4865A2
            0x486517  jg  loc_4865A2

   jms_v185 0x48d90f  lea ebx, [esi+24h]           ; ebx = slotIndex + 36
            0x48d912  cmp ebx, 24h ; '$'
            0x48d915  jl  loc_48D9A6
            0x48d91b  jg  loc_48D9A6
   ```

4. **On v95 the bound is a named predicate.** `CharacterData::IsEquipSlotExpired`
   @ `0x47ce20`, decompiled in full:

   ```c
   BOOL __thiscall CharacterData::IsEquipSlotExpired(CharacterData *this, int nPos, const _FILETIME *ftNow)
   {
     return nPos == 59 && CompareFileTime(this->aEquipExtExpire, ftNow) < 0; /*0x47ce47*/
   }
   ```

   `nPos == 59` is hard-coded. `xrefs_to(0x47ce20)` returns exactly three
   callers — see §1.4.

**Conclusion for E1 (v95):** the server must send `slotIndex = 0`. Any other
value is silently discarded by the client. The equipped-inventory body part the
extension unlocks is **59**.

### 1.3 The purchase send, for completeness

`CCashShop::OnEnableEquipSlotExt` @ `0x48e130`
(`?OnEnableEquipSlotExt@CCashShop@@QAEXJ@Z`, size `0x3f9`):

```c
  if ( CharacterData::IsEquipSlotExpired(p, 59, &ftNow) )   /*0x48e225*/
    ... StringPool 0x1472  ("buy" wording)
  else
    ftExpired = Util::FTAddDay(p->aEquipExtExpire, v6);     /*0x48e34b*/
    bodyaprt_name = get_bodyaprt_name(&v38, 59);            /*0x48e363*/
    ... StringPool 0x16C7 ("extend to <date>" wording)
  if ( CConfirmPurchaseDlg::Confirm(v31, v32, p_dwOption) == 1 )  /*0x48e459*/
  {
    COutPacket::COutPacket(&oPacket, 275);                  /*0x48e464*/
    COutPacket::Encode1(&oPacket, 0xAu);                    /*0x48e477*/  // mode 10
    COutPacket::Encode1(&oPacket, dwOption == 2);           /*0x48e48c*/  // pointType
    COutPacket::Encode4(&oPacket, nCommSN);                 /*0x48e49d*/  // serialNumber
    CClientSocket::SendPacket(...);                         /*0x48e4ad*/
    v39->m_bCashShopRequestSent = 1;                        /*0x48e4b6*/
  }
```

`Encode1(0xAu)` matches `template_gms_95_1.json:2311` (`ENABLE_EQUIP_SLOT: 10`),
and the body `pointType bool` + `serialNumber uint32` matches
`libs/atlas-packet/cash/serverbound/shop_operation_enable_equip_slot.go:52-53`
(encode) / `:68-69` (decode). **The brief's cited `:58-72` is off by a few lines
in the current tree; the non-legacy arm is at `:52-53` and `:68-69`.**

Note the `m_bCashShopRequestSent` latch at `0x48e4b6`: as established in
[`derivation.md` §3](derivation.md), the cash shop wedges until an arm clears
it, so Task 23 must always answer mode 10 with either the success or the failed
arm.

### 1.4 What the body part is actually used for

`xrefs_to(0x47ce20)` → three callers, all on v95:

| Caller | Address | Effect when expired |
|---|---|---|
| `CCashShop::OnEnableEquipSlotExt` | `0x48e225` | picks the "buy" vs "extend" wording |
| `CDraggableItem::WearEquipItem` | `0x504496` | `test eax,eax / jnz loc_50450B` @ `0x50449b`–`0x50449d` — skips the equip entirely, so nothing can be worn in slot 59 |
| `get_real_equip` | `0x73e68b` | `test eax,eax / jnz loc_73E8D9` @ `0x73e690`–`0x73e692` — the item in slot 59 contributes no stats and no avatar layer |

So the "extended equip slot" is not an inventory-capacity change; it is a
**time-gated 60th equipped body part** that the client refuses to use while
`aEquipExtExpire` is in the past.

### 1.5 Naming evidence — the slot is a *second pendant*

`get_bodyaprt_name` maps a body part to a display string. On v95 (`0x59ee20`)
the extended part shares its string with body part 17:

```c
    case 17:
    case 59:
      v4 = 646;
      goto LABEL_3;
```

The v83 IDB has the same function with **named** string constants, which
identifies string 646/629 unambiguously (`0x5ca0b0`):

```c
      if ( a2 != 51 ) { ... }
LABEL_22:
      v4 = SP_629_PENDANT;
   ...
        case 17:
          goto LABEL_22;
```

and JMS v185 (`sub_631F2C` @ `0x631F2C`) does the same with 36:

```c
    case 17:
    case 36:
      v4 = 664;
```

In every client the extended part is an alias of the **pendant** slot. Atlas
already models body part 17 as `{Type: "pendant", Position: -17}`
(`libs/atlas-constants/inventory/slot/constants.go:28`).

### 1.6 The version table

| Tenant column | Result handler | `K` (body part) | Evidence | `CharacterData` expiry offset |
|---|---|---|---|---|
| gms_v72 | **absent** | n/a | see §1.7 | n/a |
| gms_v79 | `0x473c2c` | **51** | `0x473c8a lea ebx,[esi+33h]`; `0x473c8d cmp ebx,33h` | `0x42B` (`0x473cbe`) |
| gms_v83 | `0x47acdb` | **51** | `v3 = v1 + 51;` /*`0x47ad39`*/; `get_bodyaprt_name(v30, v3)` /*`0x47ae07`*/ | `0x45F` (`*(v2 + 1119)`, `0x47ad6d`) |
| gms_v84 | `0x47de79` | **51** | `0x47ded7 lea ebx,[esi+33h]`; `0x47deda cmp ebx,33h` | `0x45F` (`0x47df0b`) |
| gms_v87 | `0x4864ad` | **59** | `0x48650b lea ebx,[esi+3Bh]`; `0x48650e cmp ebx,3Bh` | `0x4E1` (`0x48653f`) |
| gms_v92 | `0x493420` | **59** | `0x4934bf lea ebp,[esi+3Bh]` | not captured |
| gms_v95 | `0x497490` | **59** | `v7 = v3 + 59;` /*`0x49752f`*/; `IsEquipSlotExpired` hard-codes 59 (`0x47ce47`) | `0x519` (`type_inspect`) |
| jms_v185 | `0x48d8b1` | **36** | `0x48d90f lea ebx,[esi+24h]`; `0x48d912 cmp ebx,24h` | `0x385` (`0x48d943`) |

Internal consistency check — every row's `CompareFileTime` base address
reconstructs the same offset from a different literal, which is why the offsets
are quoted:
`v79: 0x293 + 8×51 = 0x42B`; `v83: 711 + 8×51 = 1119 = 0x45F`;
`v84: 0x2C7 + 8×51 = 0x45F`; `v87: 0x309 + 8×59 = 0x4E1`;
`v95: 0x2D9 + 8×72 = 0x519`; `jms: 0x265 + 8×36 = 0x385`.

**The transition is between v84 and v87.** The GMS renumbering coincides with the
v87-era body-part additions (`get_bodyaprt_name` on v95 gains cases 49/50/51 =
medal/belt/shoulder-ish that v83 does not have for 51 — on v83, 51 *is* the
extended pendant).

### 1.7 gms_v72 — RECONCILED 2026-08-19 (task-240 task 24): the send DOES exist; "no effect" is unsettled, not confirmed

**This section originally concluded "no equip-slot-extension effect in the
client" from a name-based search plus a decompiler pretty-name that turned out
to be stale. That conclusion is withdrawn.** See
`review-task-21.md` (non-blocking finding 2) and the reconciliation task that
re-decompiled `0x468e43` directly against the v72 IDB (session `f2a2e7c1`) and
cross-checked v79 (session `f36df4cd`).

Original evidence, still accurate as far as it goes: `func_query(name_regex="OnCashItemRes",
session=f2a2e7c1)` returns 46 named `OnCashItemRes*` handlers;
`OnCashItemResEnableEquipSlotExtDone` is not among them, and
`func_query(name_regex="EquipSlot|EquipExt")` returns nothing.
`search_text("CompareFileTime", 0x473500–0x4739d1)` over the unnamed gap in the
handler block returns 0 hits.

**What was wrong:** this section treated the decompiler's pretty-printed
function header (`CCashShop::OnIncCharacterSlotCount`) as the function's
identity. `lookup_funcs`/`analyze_function` against the v72 IDB's actual name
table (not the decompiler's stale header) return the real linked symbol:

```
?OnBuySlotInc@CCashShop@@QAEXJ@Z   (CCashShop::OnBuySlotInc)   size 0x407
```

`OnBuySlotInc` in v72 is a single pre-split handler that services every
cash-shop slot-purchase tab (equip/use/setup/etc/character) through one body,
keyed on `Data / 1000 % 10` (the tab id). By v79 this one function has been
split into three:

| v79 function | addr | size | shape |
|---|---|---|---|
| `OnBuySlotInc` | `0x466b13` | `0x359` | itemType-gated (1–4), mode=6 constant, `Encode1(0)` + `Encode1(a2)`, no currency field |
| `OnIncCharacterSlotCount` | `0x4673be` | `0x21d` | `CItemInfo::GetEquipExtItem` + `CConfirmPurchaseDlg::Confirm`, mode=9 constant, `Encode1(flag)` + `Encode4(a2)`, no currency field |
| `OnEnableEquipSlotExt` | `0x469fa9` | `0x407` | CS_COMMODITY_EX + `GetSlotIncDelta`, mode 6\|7, pointType+currency+flag+serialNumber |

v72's `0x468e43` matches `OnEnableEquipSlotExt` **exactly** in size (0x407 vs
0x407) and in body: identical StringPool ids (558, 537, 493, 494, 508),
identical `GetSlotIncDelta`/48-96 slot-cap logic, identical `CS_COMMODITY_EX`
construction, identical field order. It does **not** match v79's
`OnIncCharacterSlotCount` (0x21d, structurally divergent — different item-info
call, different dialog, different packet field shape). The send is:

```c
      COutPacket::COutPacket((COutPacket *)v31, 219);
      v19 = TSecType<long>::GetData(v34 + 16);
      COutPacket::Encode1((COutPacket *)v31, (v19 / 1000 == 9110) + 6);   // mode 6 or 7
      COutPacket::Encode1((COutPacket *)v31, v45 == 2);   // pointType
      COutPacket::Encode4((COutPacket *)v31, v45);        // currency
      COutPacket::Encode1((COutPacket *)v31, 1u);         // constant flag
      COutPacket::Encode4((COutPacket *)v31, (unsigned int)a2);  // serialNumber
```

This is the equip-slot-extension purchase-send lineage, not a slot-count
purchase that happens to look similar.

**On "no client-side effect":** the absence of a distinctly-named
`OnCashItemResEnableEquipSlotExtDone` in v72 is now better explained by the
same pre-split structure as the send side: v72 has a generic
`OnCashItemResIncSlotCountDone`/`OnCashItemResIncSlotCountFailed` pair
(`0x472686`/`0x47277a`) that v79 likely narrows into the per-op `...Done`
handlers, the same way `OnBuySlotInc` narrows into three sends. Whether that
generic handler actually applies the equip-slot-extension FILETIME (§1.1–1.6)
on v72 has **not** been decompiled and is not settled by this reconciliation —
it would need `OnCashItemResIncSlotCountDone` decompiled and traced against the
`CharacterData` FILETIME field. Do **not** treat gms_v72 as confirmed "no
effect"; treat it as **send confirmed present, client-side application
unverified** (open question, not "no effect").

---

## 2. E2 — how the extension survives a relog — **RESOLVED: a `FILETIME` in the `CharacterData` blob, gated on `dbcharFlag & 0x100000`. Atlas already encodes this field, mis-named `InventoryData.Timestamp`.**

**Question (brief Step 2).** How does the client learn the extension is active
after a channel change or relog? A field on `GW_CharacterStat`? A re-sent
`CashShopEnableEquipSlotExtSuccess`? An avatar-look consequence?

**Answer: none of those three.** The expiry `FILETIME` is a field of
`CharacterData` (not `GW_CharacterStat`), and it is restored from the
character-data blob that `CStage::OnSetField` carries. The avatar-look
consequence is *derived client-side* from that `FILETIME` (§1.4, `get_real_equip`);
it is not transmitted.

### 2.1 The read site (v95)

`CharacterData::Decode` @ `0x4fcce0`
(`?Decode@CharacterData@@QAE_KAAVCInPacket@@H@Z`, size `0x19a6`).
`search_text("+519h", 0x4fcce0–0x4fe690)` returns exactly one hit, at
`0x4fd177`. `insn_query` over `0x4fd154–0x4fd188`:

```
0x4fd154  mov eax, [esp+84h+dbcharFlag]
0x4fd158  and eax, 100000h
0x4fd15d  xor this, this
0x4fd15f  or  eax, this
0x4fd161  jz  short loc_4FD188
0x4fd163  mov edi, [esp+84h+nCount]
0x4fd16c  call ?Decode4@CInPacket@@QAEKXZ
0x4fd177  mov [esi+519h], eax          ; aEquipExtExpire[0].dwLowDateTime
0x4fd17d  call ?Decode4@CInPacket@@QAEKXZ
0x4fd182  mov [esi+51Dh], eax          ; aEquipExtExpire[0].dwHighDateTime
0x4fd188  mov eax, [esp+84h+dbcharFlag]
0x4fd18c  and eax, 4
...
0x4fd199  mov esi, [esp+84h+var_70]
0x4fd19d  add esi, 0FDh                ; &CharacterData::aEquipped[0].p  (0xF9 + 4)
```

`0x519` is `aEquipExtExpire` (§1.1 layout). So: **flag bit `0x100000`, two
`Decode4`s (low then high half of one `FILETIME`), read immediately before the
`0x4` equipment block.** Exactly one `FILETIME` — corroborating §1.2's bound of
`slotIndex = 0`.

### 2.2 Cross-version confirmation (v79)

`search_text("+42Bh", 0x400000–0x600000, session=f36df4cd)` returns 4 hits, two
in the result handler and one in `CharacterData::Decode` @ `0x4d9f46`.
`insn_query` over `0x4d9f20–0x4d9f60`:

```
0x4d9f2d  and eax, 100000h
0x4d9f36  jz  short loc_4D9F57
0x4d9f3b  call ?Decode4@CInPacket@@QAEKXZ
0x4d9f46  mov [esi+42Bh], eax
0x4d9f4c  call ?Decode4@CInPacket@@QAEKXZ
0x4d9f51  mov [esi+42Fh], eax
0x4d9f5a  and eax, 4
```

Same bit, same shape, same position relative to the `0x4` equipment block, at
v79's expiry offset `0x42B` (§1.6). The mechanism is stable across the GMS
range even though the body part `K` is not.

### 2.3 Atlas already puts this field on the wire — under the wrong name

`libs/atlas-packet/character/data.go`:

```go
 50:	Timestamp     int64
...
446:	// Inventory-update FILETIME: added in the v79 protocol revision (flag 0x100000,
447:	// read before the equip section). Absent v48/v61/v72 — v72 has no 0x100000 block
448:	// before equipment (its only 0x100000 use is the trailing wishlist map). IDA-verified.
449:	if (t.IsRegion("GMS") && t.MajorAtLeast(79)) || t.Region() == "JMS" {
450:		w.WriteInt64(m.Inventory.Timestamp)
451:	}
```

and the mirror decode at `:541`. The gate (`GMS ≥ 79 || JMS`), the flag bit, and
the position (immediately before the equip section) all match §2.1/§2.2
exactly. **`InventoryData.Timestamp` is `CharacterData::aEquipExtExpire[0]`.**
It is not an "inventory-update" timestamp; the existing comment is a
misidentification, and the field is currently always zero (i.e. Atlas already
tells every client "your extended slot expired at FILETIME 0").

**Consequence for FR-SLOT-3.** No new wire field, no new opcode, and no re-sent
`CashShopEnableEquipSlotExtSuccess` is required. Task 23 needs to:

1. persist one expiry timestamp per character (Windows `FILETIME`, 100-ns ticks
   since 1601-01-01 — the same encoding the existing tests use, e.g.
   `Timestamp: 94354848000000000` in `data_test.go:69`);
2. populate `InventoryData.Timestamp` from it when building the character-data
   blob; and
3. on a successful purchase, extend that stored value by `days` — from the
   current stored expiry if it is still in the future, otherwise from now,
   mirroring the client's own two arms at `0x4975df` / `0x49755e` so client and
   server agree on the resulting date shown in the `CUtilDlg::Notice`.

Renaming `InventoryData.Timestamp` to something accurate (e.g.
`EquipSlotExtExpire`) is a correctness improvement Task 23 may make, but it
touches `data_test.go`, `data_evan_test.go` and every builder that sets it —
flagged here rather than decided here.

---

## 3. The `libs/atlas-constants` consequence — **RECOMMEND: add exactly one entry, `{Type: "pendant2", Position: -59}`. Do NOT add −51 or −36.**

E1 resolved, so per the brief this section makes a recommendation. Task 23 makes
the edit; this task edited nothing.

### 3.1 Recommendation

Add one line to `libs/atlas-constants/inventory/slot/constants.go`, after
`pet3ItemIgnore`:

```go
	{Type: "pendant2", Position: -59},
```

Rationale against the file's existing conventions:

- **Sign.** Every entry in `Slots` is the negation of the client body part
  (`pendant` = −17 for body part 17, `medal` = −49, `belt` = −50,
  `shoulder` = −51). Body part 59 → `-59`.
- **Name.** The file numbers repeated instances of a slot with a bare suffix —
  `ring1`/`ring2`/`ring3`/`ring4`, `pet2Equip`, `pet3Ring2`. §1.5 shows the
  extended part is a second pendant (same `StringPool` id as body part 17 in all
  three clients), so `pendant2` matches both the naming convention and the
  client's own semantics. `pendant2` is not currently taken.
- **No collision.** `positionToSlot` is a plain map keyed by `Position`
  (`constants.go:83,98`), so a duplicate position would silently overwrite an
  existing slot. −59 is unused; the table's most negative entry today is −51
  (`shoulder`) among the non-pet slots and −48 (`pet3ItemIgnore`) among the pet
  slots.

### 3.2 The caveat Task 23 must not lose

`Slots` is a **version-agnostic** table, but §1.6 shows the extended pendant's
body part is **not** version-agnostic. Adding the v87+ GMS value is safe;
adding the others is not:

| Would-be entry | Blocked because |
|---|---|
| `-51` (GMS v79/v83/v84) | already `{Type: "shoulder", Position: -51}` (`constants.go:25`) — the *same number* means "extra pendant" on v83 and "shoulder" on v95 |
| `-36` (JMS v185) | already `{Type: "pet2MagicScales", Position: -36}` (`constants.go:56`) — and JMS's body-part numbering diverges wholesale (its `get_bodyaprt_name` has cases 21/22/28/29 and no 49/50/51) |

So: **one entry, `-59`, and the version-dependent mapping must live in the
version-gated packet/tenant layer, never in `slot.Slots`.** If Task 23 needs the
body part for a pre-v87 or JMS tenant, it must resolve it from the tenant
version, not from this table.

---

## 4. Summary

| ID | Question | Status | Answer |
|---|---|---|---|
| E1 | The `slotIndex uint16` value, and the equipped slot it unlocks | **RESOLVED** | Wire value is always **0** (bounded by an equality test in every client, and by `aEquipExtExpire` being a 1-element array at `CharacterData+0x519`). Body part = `slotIndex + K`; **K = 59 on v95** (`0x49752f`, and `IsEquipSlotExpired` hard-codes 59 @ `0x47ce47`) → Atlas position **−59**. The handler's only effect is writing one `FILETIME` + a `CUtilDlg::Notice`; it inserts no item. |
| E1-b | Is K the same on every supported version? | **RESOLVED — no** | 51 on gms_v79/v83/v84, 59 on gms_v87/v92/v95, 36 on jms_v185; gms_v72 has no handler at all (§1.6, §1.7) |
| E2 | How the extension survives a relog | **RESOLVED** | One `FILETIME` in the `CharacterData` blob, `dbcharFlag & 0x100000`, two `Decode4`s immediately before the `0x4` equip block (`0x4fd154`–`0x4fd182` on v95; `0x4d9f2d`–`0x4d9f51` on v79). **Not** `GW_CharacterStat`, **not** a re-sent success arm, **not** an avatar-look field. Atlas already encodes it as `InventoryData.Timestamp` (`data.go:50,450,541`) — same gate, same position — under a wrong name and always zero. |
| E3 | `libs/atlas-constants` consequence | **RESOLVED (recommendation)** | Add `{Type: "pendant2", Position: -59}` and nothing else; −51 and −36 collide with `shoulder` and `pet2MagicScales`, and the version-dependent mapping belongs in the version-gated layer (§3) |

Nothing in E1, E2 or E3 was left UNRESOLVED for the v95 target. Two honest gaps
are flagged in place rather than guessed:

- **gms_v72 / gms_v61 / gms_v48.** v72 has no `OnCashItemResEnableEquipSlotExtDone`
  and no equivalent unnamed handler that touches `CompareFileTime` in the
  handler block (§1.7); v61 and v48 were not queried because their `idb_list`
  entries are not adopted (empty `session_id`). Task 23 should treat these
  columns as *no effect* — the purchase arm may still be answered, but there is
  no client state to persist.
- **gms_v92 expiry offset.** The `lea ebp, [esi+3Bh]` at `0x4934bf` fixes
  `K = 59`, but the window read did not reach the store instruction, so the
  `CharacterData` offset is not pinned for v92. This does not affect any
  server-side decision (§2 shows the wire shape is a flag-gated `FILETIME`
  regardless of the struct offset).
