# task-285 — maker packet wire derivation (Steps 2–5)

Per-version IDB derivation of the two maker ops. Every field, width and address
below is read off a Hex-Rays decompilation of the named function in the named
IDB session. Nothing here is transcribed from MapleStory knowledge, from the
checked-in `docs/packets/ida-exports/*.json` flattened call lists, or from the
design phase.

- `MAKER_SKILL` — serverbound, `CUIItemMaker::RequestItemMake`
- `MAKER_RESULT` — clientbound, `CUserLocal::OnMakerResult`

## Summary

| op | v72 | v79 | v83 | v84 | v87 | v92 | v95 | jms185 |
|---|---|---|---|---|---|---|---|---|
| `MAKER_SKILL` | IDENTICAL | IDENTICAL | IDENTICAL | IDENTICAL | IDENTICAL | IDENTICAL | REFERENCE | IDENTICAL |
| `MAKER_RESULT` | IDENTICAL | IDENTICAL | IDENTICAL | IDENTICAL | IDENTICAL | IDENTICAL | REFERENCE | IDENTICAL |

**C-2 verdict: no version gate is required for either op.** Across all eight
versions the encode order of `CUIItemMaker::RequestItemMake` and the decode
order of `CUserLocal::OnMakerResult` are field-for-field identical in every
arm, including the field widths, the guard predicates and the counted-loop
shapes. The only per-version difference is the opcode passed to the
`COutPacket` constructor, which the registry already carries.

**C-3 verdict: confirmed on all eight versions.** There is exactly **one** mode
encode per arm, emitted as the first `Encode4` *inside* the arm. There is no
pre-switch mode encode and therefore no per-arm echo. The double-encode
rendering in `evidence-maker-skill-v72-v79.md` and in PRD FR-4.3 is a
transcription artefact and is **not** what any client does.

**Step 5: no divergence was found, so no `docs/packets/gates.yaml` entry is
added and no `MajorAtLeast` gate is introduced.** `packet-audit gate-check` is
untouched by this task.

## Sessions and addresses

Sessions resolved by loaded-IDB filename via `idb_list`.

| version | session | binary | `RequestItemMake` | size | `OnMakerResult` | size |
|---|---|---|---|---|---|---|
| `gms_v72` | `99e435d8` | `GMS_v72.1_U_DEVM.exe.i64` | `0x760cc3` | `0x1b3` | `0x86a152` | `0x660` |
| `gms_v79` | `5a1cd4f3` | `GMS_v79_1_DEVM.exe.i64` | `0x795dc3` | `0x1ce` | `0x8b5af5` | `0x660` |
| `gms_v83` | `754107bf` | `MapleStory_dump.exe.i64` (v83_Me) | `0x827096` | `0x1ce` | `0x95dad3` | `0x6df` |
| `gms_v84` | `46c2a2eb` | `GMS_v84.1_U_DEVM.i64` | `0x8524b7` | `0x1ce` | `0x99bdbc` | `0x6df` |
| `gms_v87` | `c0829805` | `GMSv87_4GB.exe.i64` | `0x88afd1` | `0x1ce` | `0x9e01b2` | `0x634` |
| `gms_v92` | `019cd393` | `GMS_v92_1_DEVM.exe.i64` | `0x7afdc0` | `0x20a` | `0x8f5d70` | `0x8a0` |
| `gms_v95` | `ecc757f4` | `GMS_v95.0_U_DEVM.exe.i64` | `0x7d58d0` | `0x20a` | `0x9102f0` | `0x8a0` |
| `jms_v185` | `a977912e` | `MapleStory_dump_SCY.exe.i64` | `0x8b1040` | `0x1ce` | `0xa29527` | `0x633` |

Both functions carry MSVC-mangled names in every IDB
(`?RequestItemMake@CUIItemMaker@@IAEHXZ`,
`?OnMakerResult@CUserLocal@@QAEXAAVCInPacket@@@Z`), which is why the prior
agent's plain-name lookups against the checked-in exports missed them.

### The two unsymbolized versions

`gms_v84` and `gms_v92` have **no** `CUIItemMaker` symbols at all (a
`*ItemMaker*` name filter returns zero functions on both), so
`RequestItemMake` was located structurally rather than by name and then
confirmed by decompilation:

- **`gms_v84` → `sub_8524B7`.** `OnMakerResult` tail-calls
  `CUIItemMaker::OnItemMakeResult` as `sub_852685`. On `gms_v83`, where both
  are symbolized, `RequestItemMake` (`0x827096`, size `0x1ce`) ends exactly at
  `OnItemMakeResult` (`0x827264`). Applying the same adjacency,
  `0x852685 - 0x1ce = 0x8524b7`, which `lookup_funcs` reports as a function of
  size `0x1ce`. Its decompilation is the `RequestItemMake` body verbatim
  (same four guards, same `COutPacket` + switch, same three arms), with
  `COutPacket::COutPacket(&v14, 113)` — matching the `gms_v84` registry
  opcode 113.
- **`gms_v92` → `sub_7AFDC0`.** `OnMakerResult` calls `OnItemMakeResult` as
  `sub_7AF6E0` (size `0x2f2`, matching `gms_v95`'s `OnItemMakeResult` size
  `0x2f2`). On `gms_v95` the delta `OnItemMakeResult` → `RequestItemMake` is
  `0x7d58d0 - 0x7d51f0 = 0x6e0`; `0x7af6e0 + 0x6e0 = 0x7afdc0`, a function of
  size `0x20a` — the same size as `gms_v95`'s `RequestItemMake`. Its
  decompilation is the `RequestItemMake` body verbatim, with
  `COutPacket::COutPacket(&v17, 0x7Cu)` — matching the `gms_v92` registry
  opcode 124.

## Risk R-1 — DISCHARGED, both addresses CONFIRMED

Task 1 committed `MAKER_SKILL` `ida.address` from a design-phase note without
IDB confirmation. Both values are correct; **no registry correction is
required.**

| version | committed `ida.address` | hex | IDB `?RequestItemMake@CUIItemMaker@@IAEHXZ` | verdict |
|---|---|---|---|---|
| `gms_v72` | `7736515` | `0x760cc3` | `0x760cc3` | CONFIRMED |
| `gms_v79` | `7953859` | `0x795dc3` | `0x795dc3` | CONFIRMED |

The `MAKER_RESULT` addresses corrected in `2f2361700` are likewise confirmed
against the live IDBs: `gms_v72` `8823122` = `0x86a152`, `gms_v79` `9132789` =
`0x8b5af5`. `gms_v84`'s pre-existing `10075580` = `0x99bdbc` is also correct.

The remaining five `MAKER_RESULT` entries (`gms_v83`, `gms_v87`, `gms_v92`,
`gms_v95`, `jms_v185`) carry `provenance: csv-import` with no `ida.address`.
Nothing in the IDBs contradicts them, so per the task brief they are left
untouched; their confirmed addresses are recorded in the session table above
for whoever adds them.

## Opcode cross-check

Every registry `MAKER_SKILL` opcode equals the literal passed to the
`COutPacket` constructor in that version's `RequestItemMake`. No registry
opcode is wrong.

| version | `COutPacket::COutPacket(..., N)` | decimal | registry `opcode` |
|---|---|---|---|
| `gms_v72` | `0x70` | 112 | 112 |
| `gms_v79` | `111` | 111 | 111 |
| `gms_v83` | `0x71` | 113 | 113 |
| `gms_v84` | `113` | 113 | 113 |
| `gms_v87` | `0x74` | 116 | 116 |
| `gms_v92` | `0x7C` | 124 | 124 |
| `gms_v95` | `125` | 125 | 125 |
| `jms_v185` | `0x6C` | 108 | 108 |

`MAKER_RESULT` opcodes are not derivable from `OnMakerResult` itself (it is a
dispatch target, not a constructor); they are left as registered.

## `MAKER_SKILL` — serverbound encode order

Reference: `gms_v95` `CUIItemMaker::RequestItemMake` @ `0x7d58d0`, whose IDB
carries full struct typing, so it names the fields the other seven versions
address only by offset.

```c
COutPacket::COutPacket(&oPacket, 125);
m_nRecipeClass = this->m_nRecipeClass;
switch ( m_nRecipeClass )
{
  case 1u:
  case 2u:
    COutPacket::Encode4(&oPacket, m_nRecipeClass);                 // the ONLY mode encode
    COutPacket::Encode4(&oPacket, this->m_nTargetItem);
    COutPacket::Encode1(&oPacket, this->m_CatalystSlot.bMounted);
    COutPacket::Encode4(&oPacket, this->m_nNumGem_Mounted);
    if ( this->m_nNumGemSlot > 0 ) {
      p_p = &this->m_aGemSlot[0].pItem.p;
      do {
        if ( *p_p ) {
          Data = TSecType<long>::GetData(&(*p_p)->nItemID);
          COutPacket::Encode4(&oPacket, Data);
        }
        ++v7; p_p += 8;
      } while ( v7 < this->m_nNumGemSlot );
    }
    break;
  case 3u:
    COutPacket::Encode4(&oPacket, m_nRecipeClass);
    m_nSlotPosition_DisassbleItem = TSecType<long>::GetData(&this->m_aRecipeSlot[0].pItem.p->nItemID);
    goto LABEL_20;
  case 4u:
    COutPacket::Encode4(&oPacket, m_nRecipeClass);
    v10 = TSecType<long>::GetData(&this->m_aRecipeSlot[0].pItem.p->nItemID);
    COutPacket::Encode4(&oPacket, v10);
    COutPacket::Encode4(&oPacket, this->m_nTI_DisassbleItem);
    m_nSlotPosition_DisassbleItem = this->m_nSlotPosition_DisassbleItem;
LABEL_20:
    COutPacket::Encode4(&oPacket, m_nSlotPosition_DisassbleItem);
    break;
  default:
    break;
}
```

Derived wire layout — **the same on all eight versions**:

```
i32  nRecipeClass                   // mode; the only mode encode, inside the arm

mode 1, mode 2 (make item / make item with gems):
  i32  nTargetItemID                // m_nTargetItem
  u8   bCatalystMounted             // m_CatalystSlot.bMounted
  i32  nNumGemMounted               // m_nNumGem_Mounted — length prefix for the list below
  [nNumGemMounted x] i32 nGemItemID // one per NON-EMPTY gem slot

mode 3 (make monster crystal):
  i32  nRecipeItemID                // m_aRecipeSlot[0] item id

mode 4 (disassemble):
  i32  nRecipeItemID                // m_aRecipeSlot[0] item id
  i32  nTI_DisassembleItem          // inventory type of the item being disassembled
  i32  nSlotPosition_DisassembleItem

mode <= 0, mode >= 5:
  (no body — opcode only)
```

The gem loop iterates `m_nNumGemSlot` slots and emits an `Encode4` only for
non-null slots, so the number of emitted gem ids equals the count of mounted
gems — which is exactly the `m_nNumGem_Mounted` value encoded immediately
before. **A decoder may treat `nNumGemMounted` as the list length.**

Per-version confirmation, all against the pattern above:

| version | address | mode encode | arm shape | verdict |
|---|---|---|---|---|
| `gms_v72` | `0x760cc3` | one, in-arm (`0x760de7` / `0x760dcd` / `0x760d8c`) | `if (v4>0) { if (v4<=2) …; if (v4==3) …; if (v4==4) … }` | IDENTICAL |
| `gms_v79` | `0x795dc3` | one, in-arm (`0x795ee6` / `0x795ecc` / `0x795e8b`) | same | IDENTICAL |
| `gms_v83` | `0x827096` | one, in-arm (`0x8271b9` / `0x82719f` / `0x82715e`) | same | IDENTICAL |
| `gms_v84` | `0x8524b7` | one, in-arm | same | IDENTICAL |
| `gms_v87` | `0x88afd1` | one, in-arm | same | IDENTICAL |
| `gms_v92` | `0x7afdc0` | one, in-arm | `switch (v4) { case 1: case 2: …; case 3: …; case 4: …; default: break; }` | IDENTICAL |
| `gms_v95` | `0x7d58d0` | one, in-arm | `switch` | REFERENCE |
| `jms_v185` | `0x8b1040` | one, in-arm | `if (v6>0) { … }` | IDENTICAL |

The `if`-chain form (v72–v87, jms185) and the `switch` form (v92, v95) are
semantically the same shape: the `if (v > 0)` outer guard and the `switch`
`default: break;` both emit an empty body for any mode outside 1..4. This is a
Hex-Rays rendering difference, not a wire divergence.

Struct offsets differ between versions (the `CUIItemMaker` layout grows), but
every version reads the same four members in the same order. For the record,
the mode-1/2 arm's four offsets per version:

| version | `m_nRecipeClass` | `m_nTargetItem` | `bMounted` | `m_nNumGem_Mounted` | gem-slot base | `m_nNumGemSlot` |
|---|---|---|---|---|---|---|
| `gms_v72` | +1784 | +1472 | +1964 | +1928 | +1800 | +1924 |
| `gms_v79` | +1788 | +1476 | +1968 | +1932 | +1804 | +1928 |
| `gms_v83` | +1796 | +1484 | +1976 | +1940 | +1812 | +1936 |
| `gms_v84` | +1820 | +1508 | +2000 | +1964 | +1836 | +1960 |
| `gms_v87` | +1844 | +1532 | +2024 | +1988 | +1860 | +1984 |
| `gms_v92` | +2884 | +2572 | +3064 | +3028 | +2900 | +3024 |
| `gms_v95` | (typed) | (typed) | (typed) | (typed) | (typed) | (typed) |
| `jms_v185` | +2808 | +2496 | +2988 | +2952 | +2824 | +2948 |

## `MAKER_RESULT` — clientbound decode order

Reference: `gms_v95` `CUserLocal::OnMakerResult` @ `0x9102f0`, again the
best-typed IDB (its locals are named `nResult`, `nTargetItem`, `nItemNum`,
`nDisassembedItemID`, `bSlotInit`).

Derived wire layout — **the same on all eight versions**:

```
i32  nResult                        // 0 or 1 = a body follows; > 1 = bodyless

if nResult > 1:
  (no further reads — the client falls straight through to
   CUIItemMaker::OnItemMakeResult(nResult, 0, 0, 0))

if nResult <= 1:                    // UNSIGNED compare
  i32  nMode

  mode 1, mode 2:
    u8   bNoItemGain                // guard byte; read UNCONDITIONALLY
      if bNoItemGain == 0:
        i32  nTargetItemID
        i32  nItemCount
    i32  nNumUsedItem               // count for the loop below
    [nNumUsedItem x] { i32 nItemID; i32 nCount }
    i32  nNumUsedGem                // count for the loop below
    [nNumUsedGem x]  { i32 nItemID }
    u8   bUsedCatalyst              // guard byte; read UNCONDITIONALLY
      if bUsedCatalyst != 0:
        i32  nCatalystItemID
    i32  nMesoCost

  mode 3:
    i32  nTargetItemID              // the crystal produced
    i32  nSourceItemID              // the item consumed

  mode 4:
    i32  nDisassembledItemID
    i32  nNumRewardItem             // count for the loop below
    [nNumRewardItem x] { i32 nItemID; i32 nCount }
    i32  nMesoCost

  mode outside 1..4:
    (no further reads — the switch has no matching arm)
```

`bNoItemGain` and `bUsedCatalyst` are **real wire bytes**, not decompiler
artefacts of a guard expression: each appears as a single
`CInPacket::Decode1(v2)` call evaluated once as the `if` condition, and the
`Decode*` calls inside the branch are additional reads. This settles the
caveat the prior agent flagged against the flattened export call list, which
could not distinguish the two.

Per-version confirmation:

| version | address | `nResult <= 1` guard | mode 1/2 | mode 3 | mode 4 | verdict |
|---|---|---|---|---|---|---|
| `gms_v72` | `0x86a152` | yes | as above | as above | as above | IDENTICAL |
| `gms_v79` | `0x8b5af5` | yes | as above | as above | as above | IDENTICAL |
| `gms_v83` | `0x95dad3` | yes | as above | as above | as above | IDENTICAL |
| `gms_v84` | `0x99bdbc` | yes | as above | as above | as above | IDENTICAL |
| `gms_v87` | `0x9e01b2` | yes | as above | as above | as above | IDENTICAL |
| `gms_v92` | `0x8f5d70` | yes | as above | as above | as above | IDENTICAL |
| `gms_v95` | `0x9102f0` | yes | as above | as above | as above | REFERENCE |
| `jms_v185` | `0xa29527` | yes | as above | as above | as above | IDENTICAL |

### Why the function sizes differ (and why that is not a wire divergence)

`OnMakerResult` ranges from `0x633` (jms185) to `0x8a0` (v92/v95). The brief
predicted this comes from chat-log handling, and the decompilations confirm it
exactly. Every size difference is accounted for by one of:

- **`CItemInfo::GetItemTypeName`.** v83, v84, v87, v92, v95 and jms185 fetch an
  extra *type name* string alongside the item name in the mode-1/2 gain block,
  the mode-3 block and the mode-4 reward loop, and format it with the
  four-argument `SP_5437_YOU_HAVE_GAINED_ITEMS_IN_THE_S_TAB_S_D`. v72 and v79
  have no type-name lookup and use the three-argument `SP_292`. **No packet
  field is involved** — the extra string comes from local WZ item data keyed by
  the already-decoded item id.
- **`CUIStatusBar::ChatLogAdd` inlining.** v92 and v95 inline the status-bar
  chat-log append (`if (ms_pInstance) CUIStatusBar::ChatLogAdd(...)`) at three
  call sites instead of calling the `CHATLOG_ADD` helper, which alone accounts
  for the jump from `0x634` (v87) to `0x8a0`.
- **String-pool ids.** The `StringPool::GetString` ids shift per version
  (292/294/295/291 on v72–v79; 5437/293/294/292 on v83; 5429/303/304/302 on
  v87; 5497/306/307/305 on v92; 0x1542/0x132/0x133/0x131 on v95;
  0x122/0x124/0x125/0x121 on jms185) and jms185 logs to chat channel `6`
  instead of `7`. All are display-side.

None of these touch a `CInPacket::Decode*` call. The read sequence is
byte-identical across all eight.

## Mode byte values — for Task 8 and Task 9

Both ops use the **same** four mode values, and on both directions the mode is
a **4-byte little-endian integer**, not a byte, despite the name "mode byte" in
the plan. Task 9's `operations` table must resolve these to `i32` values.

| mode | `MAKER_SKILL` (send) | `MAKER_RESULT` (receive) |
|---|---|---|
| `1` | make item, no gems path (`m_nRecipeClass == 1`) | item-make result |
| `2` | make item, gem path (`m_nRecipeClass == 2`) | item-make result (shares the arm with mode 1) |
| `3` | make monster crystal | crystal result |
| `4` | disassemble item | disassemble result |

Modes 1 and 2 share a single arm in **both** directions — the client encodes
whichever of 1/2 is in `m_nRecipeClass` and decodes them into the same body.
A dispatcher family must therefore register 1 and 2 as two arms with identical
bodies, not collapse them: the value is echoed on the wire and the server has
to preserve which one it received.

`MAKER_RESULT` additionally has a **bodyless** form for `nResult > 1`: the
opcode and the `i32 nResult` alone, no mode field. `nResult` is compared
**unsigned**, so 0 and 1 are the only values carrying a body.

## Verification

```
$ go run ./tools/packet-audit matrix --check
$ go run ./tools/packet-audit fname-doc --check
```

Exit codes recorded in `.superpowers/sdd/plan/task-6-report.md`. No
`gates.yaml` change was made, so `gate-check` is unaffected by this task.
