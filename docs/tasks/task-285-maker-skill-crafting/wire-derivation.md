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
| `MAKER_SKILL` | IDENTICAL | IDENTICAL | IDENTICAL | IDENTICAL † | IDENTICAL | IDENTICAL † | REFERENCE | IDENTICAL |
| `MAKER_RESULT` | IDENTICAL | IDENTICAL | IDENTICAL | IDENTICAL † | IDENTICAL | IDENTICAL † | REFERENCE | IDENTICAL |

† `gms_v84` and `gms_v92` carry no `CUIItemMaker` symbols; both functions were
located structurally rather than by name. See "The two unsymbolized versions"
below for the identification chain — these two verdicts rest on a weaker
evidence chain than the other six.

**C-2 verdict: no version gate is required for either op.** Across all eight
versions the encode order of `CUIItemMaker::RequestItemMake` and the decode
order of `CUserLocal::OnMakerResult` are field-for-field identical in every
arm, including the field widths, the guard predicates and the counted-loop
shapes. The only per-version difference is the opcode passed to the
`COutPacket` constructor, which the registry already carries.

**C-3 verdict: confirmed on all eight versions.** There is exactly **one** mode
encode per arm, emitted as the first `Encode4` *inside* the arm. There is no
pre-switch mode encode and therefore no per-arm echo. The double-encode
rendering in `evidence-maker-skill-v72-v79.md` and in the PRD's §4.3 "Craft
operations" snippet (`prd.md:117-136`, an unlabeled illustrative block — *not*
`FR-4.3` at `prd.md:184`, which is the unrelated `MAKER_RESULT` dispatcher
requirement) is a transcription artefact and is **not** what any client does.

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
| `gms_v84` [^unsym] | `0x8524b7` | one, in-arm (`0x8525da` / `0x8525c0` / `0x85257f`) | same | IDENTICAL |
| `gms_v87` | `0x88afd1` | one, in-arm (`0x88b0f4` / `0x88b0da` / `0x88b099`) | same | IDENTICAL |
| `gms_v92` [^unsym] | `0x7afdc0` | one, in-arm | `switch (v4) { case 1: case 2: …; case 3: …; case 4: …; default: break; }` | IDENTICAL |
| `gms_v95` | `0x7d58d0` | one, in-arm | `switch` | REFERENCE |
| `jms_v185` | `0x8b1040` | one, in-arm (`0x8b1163` / `0x8b1149` / `0x8b1108`) | `if (v6>0) { … }` | IDENTICAL |

[^unsym]: `gms_v84` and `gms_v92` carry **no** `CUIItemMaker` symbols. Both
functions were located structurally, not by name — see "The two unsymbolized
versions" above for the identification chain and its confirmation. Every
`IDENTICAL` verdict for these two versions rests on that weaker chain plus the
decompiled body, never on a symbol.

The three mode-encode addresses per row are, in order, the mode-1/2 arm's
`Encode4(mode)`, the mode-3 arm's `Encode4(3)` and the mode-4 arm's
`Encode4(4)`. Verbatim from the `gms_v84` decompilation (`sub_8524B7`, session
`46c2a2eb`) — the mode-1/2 arm, showing the single in-arm mode encode:

```c
    if ( v4 <= 2 )                                              /*0x85256c*/
    {
      COutPacket::Encode4((COutPacket *)v14, v4);                /*0x8525da*/
      COutPacket::Encode4((COutPacket *)v14, *(_DWORD *)(this + 1508)); /*0x8525e8*/
      COutPacket::Encode1((COutPacket *)v14, *(_BYTE *)(this + 2000));  /*0x8525f9*/
      COutPacket::Encode4((COutPacket *)v14, *(_DWORD *)(this + 1964)); /*0x852607*/
```

and the `gms_v87` mode-3/mode-4 arms (`CUIItemMaker::RequestItemMake`, session
`c0829805`), showing the literal mode value encoded first inside each arm:

```c
    if ( v5 == 3 )                                              /*0x88b08a*/
    {
      COutPacket::Encode4(&a3, 3u);                             /*0x88b0da*/
      v14 = TSecType<long>::GetData(*(this + 1584) + 12);        /*0x88b0ed*/
      goto LABEL_16;
    }
    if ( v5 == 4 )                                              /*0x88b08f*/
    {
      COutPacket::Encode4(&a3, 4u);                             /*0x88b099*/
      v6 = TSecType<long>::GetData(*(this + 1584) + 12);         /*0x88b0a7*/
      COutPacket::Encode4(&a3, v6);                             /*0x88b0b0*/
      COutPacket::Encode4(&a3, *(this + 2028));                 /*0x88b0be*/
      v14 = *(this + 2032);                                     /*0x88b0c3*/
LABEL_16:
      COutPacket::Encode4(&a3, v14);                            /*0x88b0c9*/
    }
```

`jms_v185` is the same shape at `0x8b1163` / `0x8b1149` / `0x8b1108`
(`COutPacket::COutPacket(v16, 0x6C)` at `0x8b10db` fixes the opcode as 108).

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

if nResult <= 1:                    // UNSIGNED compare; compiled as
                                    //   cmp eax,0 / jz  →  take body
                                    //   cmp eax,1 / jnz →  skip body
                                    // i.e. body iff nResult ∈ {0, 1}
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

### Evidence — quoted decompilation

Every claim below was re-derived from the live IDB sessions listed above; each
of the eight functions was decompiled in full and the three load-bearing sites
(`nResult` guard, `bNoItemGain`, `bUsedCatalyst`) were additionally read at the
instruction level with `insn_query`.

#### 1. The `nResult` guard is an unsigned test, and it is **not** a `<=`

Hex-Rays renders `if ( v53 <= 1 )`, but the compiled form on **all eight**
versions is a pair of equality tests, not a magnitude compare. `gms_v95`
(session `ecc757f4`), disassembly at the function's first `Decode4`:

```asm
0x910337  call ?Decode4@CInPacket@@QAEKXZ
0x91033c  mov  [esp+44h+nResult], eax
0x910340  cmp  eax, esi              ; esi == 0
0x910342  jz   short loc_91034D      ; nResult == 0 -> body
0x910344  cmp  eax, 1
0x910347  jnz  loc_910B1E            ; nResult != 1 -> skip body entirely
0x91034d  mov  this, edi
0x91034f  call ?Decode4@CInPacket@@QAEKXZ   ; nMode
```

and `gms_v72` (session `99e435d8`), the same two-equality shape:

```asm
0x86a17c  call ?Decode4@CInPacket@@QAEKXZ
0x86a180  cmp  eax, esi              ; esi == 0
0x86a182  mov  [ebp+var_30], eax
0x86a185  jz   short loc_86A190
0x86a187  cmp  eax, 1
0x86a18a  jnz  loc_86A75E
0x86a190  mov  ecx, ebx
0x86a192  call ?Decode4@CInPacket@@QAEKXZ   ; nMode
```

**A body follows iff `nResult` is exactly 0 or 1.** Every other value —
including any negative value — takes the bodyless path. That is precisely the
semantics of an *unsigned* `<= 1`, which is why the "UNSIGNED" characterization
is correct; it is also why a signed `<= 1` decoder would be wrong (it would
expect a body for `nResult == -1`). Four of the eight IDBs render the local as
`unsigned int` directly (`v72` `unsigned int v53`, `v79` `unsigned int v53`,
`v83` `unsigned int v42`, `jms185` `unsigned int v35`); `v92` renders the cast
explicitly as `if ( (unsigned int)v48 <= 1 )`. `v95`'s local is typed `int` in
that IDB, so Hex-Rays prints a signed compare — the disassembly above shows the
type annotation, not the machine code, is what differs.

Per-version guard instructions (`jz` / `jnz` addresses; the two `cmp`s
immediately precede them, and the `nMode` `Decode4` immediately follows):

| version | session | `nResult` `Decode4` | `cmp 0`/`jz` | `cmp 1`/`jnz` | `nMode` `Decode4` |
|---|---|---|---|---|---|
| `gms_v72` | `99e435d8` | `0x86a17c` | `0x86a185` | `0x86a18a` | `0x86a192` |
| `gms_v79` | `5a1cd4f3` | `0x8b5b1f` | `0x8b5b28` | `0x8b5b2d` | `0x8b5b35` |
| `gms_v83` | `754107bf` | `0x95dafd` | `0x95db06` | `0x95db0b` | `0x95db13` |
| `gms_v84` [^unsym] | `46c2a2eb` | `0x99bde6` | `0x99bdef` | `0x99bdf4` | `0x99bdfc` |
| `gms_v87` | `c0829805` | `0x9e01dc` | `0x9e01e5` | `0x9e01ea` | `0x9e01f2` |
| `gms_v92` [^unsym] | `019cd393` | `0x8f5db7` | `0x8f5dc2` | `0x8f5dc7` | `0x8f5dcf` |
| `gms_v95` | `ecc757f4` | `0x910337` | `0x910342` | `0x910347` | `0x91034f` |
| `jms_v185` | `a977912e` | `0xa29551` | `0xa2955a` | `0xa2955f` | `0xa29567` |

All eight are the identical instruction sequence. **No version gate.**

#### 2. `bNoItemGain` and `bUsedCatalyst` are real wire bytes

Each is a single `CInPacket::Decode1` call whose *return value* is the branch
condition; the branch body then performs *additional* `Decode4` reads. This is
the claim the prior round could not settle from the flattened export call list
(which cannot distinguish a wire byte from a re-counted guard expression), and
it is settled here at the instruction level.

`gms_v95` `bNoItemGain` — one `Decode1`, tested, and the taken branch does two
further `Decode4`s:

```asm
0x910715  mov   this, edi
0x910717  call  ?Decode1@CInPacket@@QAEEXZ
0x91071c  movzx eax, al
0x91071f  test  eax, eax
0x910721  jnz   loc_91080C            ; byte != 0 -> skip the item-gain fields
```
```c
        if ( !CInPacket::Decode1(v2) )                    /*0x910717*/
        {
          nTargetItem = CInPacket::Decode4(v2);           /*0x910732*/
          v22 = CInPacket::Decode4(v2);                   /*0x910746*/
```

`gms_v95` `bUsedCatalyst` — same shape, one `Decode1`, one conditional
`Decode4`:

```asm
0x9109c4  mov   this, edi
0x9109c6  call  ?Decode1@CInPacket@@QAEEXZ
0x9109cb  movzx eax, al
0x9109ce  cmp   eax, esi              ; esi == 0
0x9109d0  jz    loc_910A9B            ; byte == 0 -> no catalyst id follows
0x9109d6  mov   this, edi
0x9109d8  call  ?Decode4@CInPacket@@QAEKXZ   ; nCatalystItemID
```

`gms_v72` is instruction-for-instruction the same pair:

```asm
0x86a46b  mov   ecx, ebx
0x86a46d  call  ?Decode1@CInPacket@@QAEEXZ   ; bNoItemGain
0x86a472  movzx eax, al
0x86a475  cmp   eax, esi
0x86a477  jnz   loc_86A518
...
0x86a66b  mov   ecx, ebx
0x86a66d  call  ?Decode1@CInPacket@@QAEEXZ   ; bUsedCatalyst
0x86a672  movzx eax, al
0x86a675  test  eax, eax
0x86a677  jz    loc_86A704
0x86a67d  mov   ecx, ebx
0x86a67f  call  ?Decode4@CInPacket@@QAEKXZ   ; nCatalystItemID
```

On the remaining six versions the decompilation shows the same single-call
guard; the call addresses are in the per-version table below. Representative
quotes, one per remaining version, all from the mode-1/2 arm:

```c
/* gms_v79  @0x8b5af5 */  if ( !CInPacket::Decode1(v2) )        /*0x8b5e10*/
                          if (  CInPacket::Decode1(v2) )        /*0x8b6010*/
/* gms_v83  @0x95dad3 */  if ( !CInPacket::Decode1(v2) )        /*0x95de3f*/
                          if (  CInPacket::Decode1(v2) )        /*0x95e06d*/
/* gms_v84  @0x99bdbc */  if ( !CInPacket::Decode1(v1) )        /*0x99c128*/
                          if (  CInPacket::Decode1(v1) )        /*0x99c356*/
/* gms_v87  @0x9e01b2 */  if ( !CInPacket::Decode1(v1) )        /*0x9e04d9*/
                          if (  CInPacket::Decode1(v1) )        /*0x9e06c0*/
/* gms_v92  @0x8f5d70 */  if ( !CInPacket::Decode1((int)v2) )   /*0x8f6197*/
                          if (  CInPacket::Decode1((int)v2) )   /*0x8f6446*/
/* jms185   @0xa29527 */  if ( !CInPacket::Decode1(v2) )        /*0xa2984d*/
                          if (  CInPacket::Decode1(v2) )        /*0xa29a37*/
```

#### 3. Per-version read order, by instruction address

Each column is the address of the `CInPacket::Decode*` call that reads that
field. Reading a row left to right gives the wire order; the eight rows are
field-for-field identical, which is the whole of the C-2 verdict.

`mode 1 / mode 2` arm — `Decode1` columns are marked `¹`, all others `Decode4`:

| version | bNoItemGain¹ | nTargetItemID | nItemCount | nNumUsedItem | used[i].id | used[i].count | nNumUsedGem | gem[i].id | bUsedCatalyst¹ | nCatalystItemID | nMesoCost |
|---|---|---|---|---|---|---|---|---|---|---|
| `gms_v72` | `0x86a46d` | `0x86a486` | `0x86a497` | `0x86a51a` | `0x86a538` | `0x86a545` | `0x86a5ce` | `0x86a5df` | `0x86a66d` | `0x86a67f` | `0x86a70b` |
| `gms_v79` | `0x8b5e10` | `0x8b5e29` | `0x8b5e3a` | `0x8b5ebd` | `0x8b5edb` | `0x8b5ee8` | `0x8b5f71` | `0x8b5f82` | `0x8b6010` | `0x8b6022` | `0x8b60ae` |
| `gms_v83` | `0x95de3f` | `0x95de58` | `0x95de69` | `0x95df1a` | `0x95df38` | `0x95df45` | `0x95dfce` | `0x95dfdf` | `0x95e06d` | `0x95e07f` | `0x95e10b` |
| `gms_v84` [^unsym] | `0x99c128` | `0x99c141` | `0x99c152` | `0x99c203` | `0x99c221` | `0x99c22e` | `0x99c2b7` | `0x99c2c8` | `0x99c356` | `0x99c368` | `0x99c3f4` |
| `gms_v87` | `0x9e04d9` | `0x9e04f2` | `0x9e0503` | `0x9e0599` | `0x9e05b7` | `0x9e05c4` | `0x9e063b` | `0x9e0648` | `0x9e06c0` | `0x9e06d0` | `0x9e0749` |
| `gms_v92` [^unsym] | `0x8f6197` | `0x8f61b2` | `0x8f61c6` | `0x8f628e` | `0x8f62aa` | `0x8f62bd` | `0x8f636f` | `0x8f6384` | `0x8f6446` | `0x8f6458` | `0x8f6522` |
| `gms_v95` | `0x910717` | `0x910732` | `0x910746` | `0x91080e` | `0x91082a` | `0x91083d` | `0x9108ef` | `0x910904` | `0x9109c6` | `0x9109d8` | `0x910aa2` |
| `jms_v185` | `0xa2984d` | `0xa29866` | `0xa29878` | `0xa2990c` | `0xa2992a` | `0xa2993c` | `0xa299b0` | `0xa299be` | `0xa29a37` | `0xa29a45` | `0xa29abe` |

`mode 3` and `mode 4` arms (all `Decode4`):

| version | m3 nTargetItemID | m3 nSourceItemID | m4 nDisassembledItemID | m4 nNumRewardItem | m4 reward[i].id | m4 reward[i].count | m4 nMesoCost |
|---|---|---|---|---|---|---|---|
| `gms_v72` | `0x86a1bd` | `0x86a1c0` | `0x86a2f2` | `0x86a35d` | `0x86a376` | `0x86a383` | `0x86a409` |
| `gms_v79` | `0x8b5b60` | `0x8b5b71` | `0x8b5c95` | `0x8b5d00` | `0x8b5d19` | `0x8b5d26` | `0x8b5dac` |
| `gms_v83` | `0x95db3e` | `0x95db4f` | `0x95dca1` | `0x95dd0c` | `0x95dd25` | `0x95dd32` | `0x95dddb` |
| `gms_v84` [^unsym] | `0x99be27` | `0x99be38` | `0x99bf8a` | `0x99bff5` | `0x99c00e` | `0x99c01b` | `0x99c0c4` |
| `gms_v87` | `0x9e021d` | `0x9e022e` | `0x9e0350` | `0x9e03b2` | `0x9e03cb` | `0x9e03d8` | `0x9e0478` |
| `gms_v92` [^unsym] | `0x8f5dfa` | `0x8f5e0d` | `0x8f5f96` | `0x8f600b` | `0x8f6029` | `0x8f603c` | `0x8f610f` |
| `gms_v95` | `0x91037a` | `0x91038d` | `0x910516` | `0x91058b` | `0x9105a9` | `0x9105bc` | `0x91068f` |
| `jms_v185` | `0xa29592` | `0xa295a4` | `0xa296c5` | `0xa29726` | `0xa2973f` | `0xa2974d` | `0xa297ef` |

Note on apparent widths: several IDBs assign a `Decode4` result to a
byte-width local (`char v4; v4 = CInPacket::Decode4(v2);` at `gms_v72`
`0x86a1c0`, and similarly at `v79 0x8b5b71`, `v83 0x95db4f`, `v84 0x99be38`,
`v87 0x9e022e`, `jms185 0xa295a4`). That is a Hex-Rays register-width
inference on an unused high byte, not a wire width — the callee is
`?Decode4@CInPacket@@QAEKXZ` in every case, so **4 bytes are consumed**. Do not
narrow any of these to a byte in the codec.

#### 4. Verdict table

| version | address | guard shape | mode 1/2 | mode 3 | mode 4 | verdict |
|---|---|---|---|---|---|---|
| `gms_v72` | `0x86a152` | `cmp 0/jz`+`cmp 1/jnz` | 11 reads, order per §3 | 2 reads | 4+2n reads | IDENTICAL |
| `gms_v79` | `0x8b5af5` | same | same | same | same | IDENTICAL |
| `gms_v83` | `0x95dad3` | same | same | same | same | IDENTICAL |
| `gms_v84` [^unsym] | `0x99bdbc` | same | same | same | same | IDENTICAL |
| `gms_v87` | `0x9e01b2` | same | same | same | same | IDENTICAL |
| `gms_v92` [^unsym] | `0x8f5d70` | same | same | same | same | IDENTICAL |
| `gms_v95` | `0x9102f0` | same | same | same | same | REFERENCE |
| `jms_v185` | `0xa29527` | same | same | same | same | IDENTICAL |

"same" here is not shorthand for "assumed": each cell is the row of addresses
in §3, which was read off that version's own decompilation. The one rendering
difference across the eight is the arm-dispatch form — `v72`/`v79`/`v83`/`v84`/
`v92`/`v95` compile the mode dispatch to a `switch`, while `v87` and `jms185`
render as an `if ( v2 == 1 || v2 == 2 ) … else if ( v2 == 3 ) … else if
( v2 != 4 ) goto LABEL_50;` chain. Both forms perform no reads for a mode
outside 1..4 and are therefore wire-identical.

**Step 5 (re-confirmed after re-derivation): no divergence, so no
`docs/packets/gates.yaml` entry and no `MajorAtLeast` gate.** Had any of the
eight rows in §3 differed in field order, count or width, a gate would have
been added here — the asymmetry is deliberate (an unnecessary gate costs a
fixture; a missing gate is a silent wire bug with no CI catch). None differed.

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
