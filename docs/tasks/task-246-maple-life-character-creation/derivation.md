# Maple Life (`Cash/0543`) — wire derivation

Task 1 of the task-246 plan. Read-only against the GMS v83/v84/v87/v92/v95
IDA sessions; the only repo write for this task is this file. §1–§3 below are
produced by Task 1. §4–§6 (clientbound `OnCreateNewCharacterResult` decode,
the `CUICharacterSaleDlg::OnCheckDuplicatedIDResult` decode, and the
`Cash/0543` item-id → cash-slot-type table) are appended by Task 2 — do not
renumber the sections above when appending.

## IDA sessions used (all confirmed `is_analyzing: false` via `idb_list`)

| version | session_id | binary |
|---|---|---|
| gms_v83 | `754107bf` | `E:\Programs\Nexon\IDBs_v9\GMS\v83_Me\MapleStory_dump.exe.i64` |
| gms_v84 | `46c2a2eb` | `E:\Programs\Nexon\IDBs_v9\GMS\v84_1\GMS_v84.1_U_DEVM.i64` |
| gms_v87 | `c0829805` | `E:\Programs\Nexon\IDBs_v9\GMS\v87\GMSv87_4GB.exe.i64` |
| gms_v92 | `019cd393` | `E:\Programs\Nexon\IDBs_v9\GMS\V92_1\GMS_v92_1_DEVM.exe.i64` |
| gms_v95 | `ecc757f4` | `E:\Programs\Nexon\IDBs_v9\GMS\v95_0\GMS_v95.0_U_DEVM.exe.i64` |

`gms_v83` was discovered but not yet adopted at task start (pid 10044, port
13339, `MapleStory_dump.exe.i64`); `idb_list` reported it already carrying
`session_id: 754107bf` with `adopted: true` and `is_analyzing: false` by the
time this task ran, so no explicit `idb_open` call was needed — it was
already live.

---

## §1. `get_cashslot_item_type` — the 543 classification branch

`get_cashslot_item_type` is the free function `?get_cashslot_item_type@@YAJJ@Z`
that maps a raw `itemId` to the `CashSlotItemType` enum consumed by
`CWvsContext::SendConsumeCashItemUseRequest` (see §2) to select which
sub-body shape to encode for a `USE_CASH_ITEM` request. Case `itemId/10000 ==
543` is the arm that gates character creation.

### §1.1 — v83 (`0x48645b`, size `0x2fc`)

Decompiled switch, `case 507` region omitted (irrelevant), case 543/542/546
region verbatim:

```c
LABEL_87:
  if ( a1 / 1000 == 5420 )      /*0x48671a*/
    return 54;                  /*0x48671c*/
LABEL_89:
  if ( a1 / 1000 < 5431 || a1 / 1000 > 5432 )  /*0x486734*/
    return 57;                  /*0x48673d*/
  else
    return 65;                  /*0x486736*/
case 542:
  goto LABEL_87;
case 543:
  goto LABEL_89;
...
case 546:
  return 57;
```

Disassembly of the comparison at `0x486720`–`0x486746`:

```
0x486723  mov eax, ecx        ; jumptable case 543
0x486725  cdq
0x486726  idiv edi            ; eax = itemId / 1000 (SIGNED divide)
0x486728  cmp eax, 1537h      ; 1537h = 5431
0x48672d  jl  short loc_48673D   ; SIGNED less-than -> 57
0x48672f  cmp eax, 1538h      ; 1538h = 5432
0x486734  jg  short loc_48673D   ; SIGNED greater-than -> 57
0x486736  push 41h            ; 'A' = 65 -> 65
```

Two explicit **signed** bound checks (`jl`/`jg`). Equivalent to: `itemId/1000
in [5431,5432] -> 65`, else `-> 57` (57 is the same numeric value `case 546`
returns unconditionally — the arm is a shared value-space, not a
character-creation-specific number).

### §1.2 — v95 (`0x488c70`, size `0x3c9`)

```c
case 543:
$LN12_9:
  if ( (nItemID / 1000 - 5431) > 1 )   /*0x489009*/
    goto $LN8_8;                        /*0x489009*/
  result = 66;                          /*0x48900b*/
  break;
...
case 546:
$LN8_8:
  result = 58;                          /*0x489012*/
  break;
```

Disassembly at `0x488ff0`–`0x489012`:

```
0x488ff0  mov eax, 10624DD3h    ; magic-number reciprocal for /1000
0x488ff5  imul ecx
0x488ff7  sar edx, 6
0x488ffa  mov eax, edx
0x488ffc  shr eax, 1Fh
0x488fff  add eax, edx          ; eax = itemId / 1000
0x489001  add eax, 0FFFFEAC9h   ; eax += -5431 (0xFFFFEAC9 == -5431 mod 2^32)
0x489006  cmp eax, 1
0x489009  ja  short $LN8_8      ; UNSIGNED above -> 58
0x48900b  mov eax, 42h          ; 'B' = 66 -> 66
```

Single **unsigned** compare (`ja`) against the pre-subtracted value — the
standard `(x - lo) > (hi - lo)` two-sided-bound-in-one-compare compiler
idiom.

### §1.3 — signedness and OQ-2

v83 compiles the bound check as two **signed** comparisons (`jl`/`jg`)
against literal 5431/5432. v95 compiles it as one **unsigned** comparison
(`ja`) against `(itemId/1000) - 5431`. The instruction *form* differs, but
for every real `itemId` (always positive, so `itemId/1000` is always
non-negative and the subtraction never produces a value a signed check would
classify differently from the unsigned-wraparound trick) the two forms are
**functionally identical**: `itemId/1000 in {5431, 5432}` reaches
65 (v83) / 66 (v95); every other `itemId/1000` value in the `543` prefix,
**including 5430xxx**, falls through to 57 (v83) / 58 (v95).

**OQ-2 answer:** the 57/58 branch **is reachable with shipped `Cash/0543`
data**. `Cash/0543` ships exactly `05430000`, `05431000`, `05432000` (§1.4).
`05430000` has `itemId/1000 == 5430`, which is outside `[5431,5432]` on
**both** versions regardless of the signed/unsigned instruction-form
difference — it reaches the 57/58 arm, not 65/66. Only `05431000` and
`05432000` (`itemId/1000` in `{5431,5432}`) reach 65/66. This means
`Cash/0543` is **not uniformly a "character creation" item family at the
`get_cashslot_item_type` classification level**: `05430000` shares the
57/58 numeric slot with the unrelated `ClassificationPetMultiConsumable`
category (see `services/atlas-channel/.../character_cash_item_use.go:1487-1489`),
while only `05431000`/`05432000` drive the 65/66 (character-creation)
sub-body. Atlas's existing Go re-implementation at
`character_cash_item_use.go:1457-1469`
(`if itemId/1000-5431 > 1 { return 57/58 } else { return 65/66 }`) already
matches this exactly, **provided** the Go subtraction is performed in
unsigned 32-bit arithmetic (Go evaluates `itemId/1000-5431` per `itemId`'s
declared type — `item.Id`; if that type is `uint32`, the wraparound for
`5430-5431` reproduces the client's classification correctly for all three
shipped ids; this needs re-confirming against `item.Id`'s underlying type
before Task 3+ treat it as settled, but is out of this task's scope to
re-derive).

**Consequence for Task 6:** whatever plan document assumed all three
`Cash/0543` ids drive `CUICharacterSaleDlg::SendCreateNewCharacter`'s
sub-body must be re-checked — `05430000` classifies as type 57/58, not
65/66, at the `get_cashslot_item_type` level. Confirming whether Atlas's
handler currently *routes* 57/58 to the same dialog-open path as 65/66 (or
to something else entirely) is outside this task's IDA-derivation scope and
should be checked against `character_cash_item_use.go`'s handler dispatch
before Task 3 assumes `05430000` opens `CUICharacterSaleDlg`.

### §1.4 — `Item.wz` `Cash/0543` spec differences (OQ-3)

Read from two independent local WZ corpora (`Cosmic` and `HeavenMS`, both
byte-identical for these three entries; a third corpus, `ms_1172`, carries
two additional out-of-PRD-scope ids `05431001`/`05433000` not shipped per the
PRD's "ships only 5430xxx/5431xxx/5432xxx" framing — the `05433000` entry
there does carry a `jumplevel=50` int, noted only for completeness, not
acted on):

```xml
<imgdir name="05430000">
  <imgdir name="info">
    <canvas name="icon" width="32" height="28"><vector name="origin" x="0" y="28"/></canvas>
    <canvas name="iconRaw" width="32" height="23"><vector name="origin" x="0" y="28"/></canvas>
    <int name="cash" value="1"/>
  </imgdir>
</imgdir>
<imgdir name="05431000">
  <imgdir name="info">
    <canvas name="icon" width="31" height="30"><vector name="origin" x="0" y="31"/></canvas>
    <canvas name="iconRaw" width="31" height="26"><vector name="origin" x="0" y="31"/></canvas>
    <int name="cash" value="1"/>
  </imgdir>
</imgdir>
<imgdir name="05432000">
  <imgdir name="info">
    <canvas name="icon" width="31" height="30"><vector name="origin" x="0" y="31"/></canvas>
    <canvas name="iconRaw" width="31" height="30"><vector name="origin" x="0" y="31"/></canvas>
    <int name="cash" value="1"/>
  </imgdir>
</imgdir>
```

All three carry only `cash=1` under `info` — **no `spec` node at all**: no
`job`, `reqLevel`, `time`, `only`, `tradeBlock`, `slotMax`, or any other
gameplay-relevant tag. The only differences between the three entries are
icon canvas dimensions (cosmetic, not gameplay). This confirms the design
doc's §10 framing: the WZ spec carries no data distinguishing what each id
*does* — that distinction lives entirely in the client's compiled
classification logic (§1.1–§1.3), not in data. This gates nothing per design
§10, recorded per Step 5.

---

## §2. `CUICharacterSaleDlg::SendCreateNewCharacter`

### §2.0 — version coverage and the v84 finding

Searched on all five versions. Found on v83, v87, v92, v95. **Not found on
v84**, and the absence is corroborated three independent ways:

1. `func_query` with `name_regex` for `charactersale`, `newcharacter`,
   `cashslot`, and filter patterns `*SaleDlg*`, `*CharacterSale*`,
   `*CUI.*(Sale|CharCreate|NewChar)*` all return zero matches on v84
   (session `46c2a2eb`).
2. `find_regex` for the literal string `CharacterSaleDlg` (RTTI class-name
   string, present as a plain string wherever the class exists) returns zero
   matches on v84 — the class is not merely unanalyzed, its name string is
   absent from the binary.
3. v84's `CWvsContext::SendConsumeCashItemUseRequest` (`0xa54a2f`, size
   `0x4499` — essentially the same size as v83's byte-identical counterpart
   at `0xa0a63f`, size `0x4495`) contains **zero** occurrences of the
   operand values 543, 5431, or 5432 anywhere in its instruction stream
   (`insn_query` with `func` scoped to the whole function, `op_any` for each
   literal, all three `count: 0`). The `get_cashslot_item_type` classifier
   also has no symbol on v84 (`func_query` for `*get_cashslot_item_type*`
   returns empty, vs. a resolved symbol on v83/v87 — v92/v95 also don't
   resolve this specific mangled name, but v92/v95 clearly ship the feature
   via other evidence in §2.2–§2.4, so the missing symbol alone isn't
   dispositive there the way it is on v84 combined with findings 1–3).

v84's only "character creation" related code is
`CLogin::OnCreateNewCharacterResult` (`0x60f268`, receive-only, reached from
`CLogin::OnPacket`) — the **ordinary login-socket character creation flow**
(`CLogin::SendNewCharPacket`, i.e. `charsb.CreateCharacter`), unrelated to
the cash-shop Maple Life feature. `CField::OnCharacterSale` also exists on
v84 (`0x5443af`) but is a distinct, unrelated field-effect handler (not a
cash-item-use send path).

**Finding: `Cash/0543` in-game character creation via `CUICharacterSaleDlg`
is VERSION-ABSENT on gms_v84.** The client binary simply does not contain
this feature — no class, no classification arm, no send function. This is a
genuine client-side absence, not a symbol-resolution gap, per the
triangulated evidence above. Downstream tasks (3–15) must treat v84 as
having no wire behavior to derive for this feature; the plan's in-scope
version list (v83/v84/v87/v92/v95) should route v84 to whatever the plan's
"n-a" / version-absent handling is for this packet family, or the plan
should be revisited for this cell — that decision is out of this
derivation-only task's scope.

### §2.1 — gms_v83 (`0x7d7960`, size `0x121`)

`decompile_sha256`: **PENDING — not computable this task.**
`CUICharacterSaleDlg::SendCreateNewCharacter` is **absent as a key** in
`docs/packets/ida-exports/gms_v83.json`'s `functions` map (confirmed by
direct grep of the export — zero matches for `SendCreateNewCharacter` or
`CharacterSaleDlg` in any of the five in-scope exports). `tools/packet-audit
evidence pin`'s hash (`internal/evidence/hash.go: FunctionHash`) is a SHA-256
over the canonical JSON of `functions[<fname>]` **from the checked-in
export**, not over ad hoc decompile text — so this task cannot produce the
canonical hash without re-harvesting the export (task-081 playbook), which is
out of this task's file-write scope (derivation.md only). **Escalating this
as a blocking dependency for Tasks 4–6**: before any of those tasks calls
`evidence pin --ida CUICharacterSaleDlg::SendCreateNewCharacter` (or the
`OnCreateNewCharacterResult` / `OnCheckDuplicatedIDResult` counterparts Task
2 derives), the relevant version's `docs/packets/ida-exports/<version>.json`
must be re-harvested to include these functions, or the pin step will fail
exactly as `IMPLEMENTING_A_PACKET.md`'s "Export-resolvability is a
precondition" warns.

Opcode: `COutPacket::COutPacket(&v12, 0x4F)` — **0x4F = 79**. Matches
`docs/packets/registry/gms_v83.yaml` `USE_CASH_ITEM` (serverbound, opcode
79) exactly.

Complete encode order as emitted (guard: `SendCreateNewCharacter` no-ops if
`*(this+720)` truthy — an already-sent flag — or if the name edit box is
empty, in which case it shows `CLoginUtilDlg::Error(10, 0)` and returns
without sending):

1. `Encode2 nPOS` — `*(this+216)`, `int16`
2. `Encode4 nItemID` — `*(this+220)`, `int32`
3. `EncodeStr sName` — from `CCtrlEdit::GetText` on the name edit control
4. `Encode4 al[0]` — `sub_7D5C57(this, 0)` (avatar-look selection 0)
5. `Encode4 al[1]` — `sub_7D5C57(this, 1)`
6. `Encode4 al[2]` — `sub_7D5C57(this, 2)`
7. `Encode4 al[3]` — `sub_7D5C57(this, 3)` (loop `v2 <= 3`, 4 iterations)
8. `Encode4 nGender` — `*(this+504)`
9. `Encode4 nCurrentClass` — `*(this+440)`
10. `Encode4 nSP` — `*(this+448)`
11. `Encode4 update_time` — `get_update_time()` result — **TRAILING, only
    occurrence of update_time in this version**

No leading `update_time` — matches `UpdateTimeFirst` (item_use.go:22-24):
GMS v83 trails.

### §2.2 — gms_v87 (`0x82e402`, size `0x12f`)

`decompile_sha256`: **PENDING**, same reason as §2.1 (function absent from
`gms_v87.json`'s export).

Opcode: `COutPacket::COutPacket(v12, 0x52)` — **0x52 = 82**. Matches
`docs/packets/registry/gms_v87.yaml` `USE_CASH_ITEM` (serverbound, opcode
82) exactly.

Encode order:

1. `Encode4 update_time` — `get_update_time()` — **LEADING**
2. `Encode2 nPOS` — `*(this+232)`
3. `Encode4 nItemID` — `*(this+236)`
4. `EncodeStr sName`
5. `Encode4 al[0..3]` — `sub_82C6F7(v2)`, loop `v2 <= 3`
6. `Encode4 nGender` — `*(this+520)`
7. `Encode4 nCurrentClass` — `*(this+456)`
8. `Encode4 nSP` — `*(this+464)`
9. `Encode4 update_time` — `get_update_time()` — **TRAILING (second
    occurrence)**

`sub_82E10C(v12)` (called instead of a directly-visible `SendRequest` call)
was decompiled separately (`0x82e10c`) and confirmed to be exactly
`CClientSocket::SendPacket(...)` + set-sent-flag — i.e. inlined
`SendRequest`, no additional wire fields.

Leading update_time matches `UpdateTimeFirst`: GMS v87+ leads. **New
finding not covered by `UpdateTimeFirst`'s doc comment**: this packet also
writes update_time a **second time, trailing**, on v87 (and v92/v95, §2.3–
§2.4) — `UpdateTimeFirst` only describes the generic `ItemUse` common-prefix
behavior; `SendCreateNewCharacter`'s own tail independently re-reads and
re-encodes `get_update_time()` after the SP field, on every version except
v83 (which only has the trailing copy, i.e. v83 has exactly one, v87/v92/v95
have exactly two, one leading + one trailing).

### §2.3 — gms_v92 (`0x758770`, size `0x178`)

`decompile_sha256`: **PENDING**, same reason.

Opcode: `COutPacket::COutPacket((COutPacket*)&v13, 0x56u)` — **0x56 = 86**.
Matches `docs/packets/registry/gms_v92.yaml` `USE_CASH_ITEM` (serverbound,
opcode 86) exactly.

Encode order:

1. `Encode4 update_time` — `sub_936E80(v6)` — **LEADING**. `sub_936E80` is
   unnamed (v92 export lacks a symbol here) but decompiles to
   `return *(_DWORD*)(dword_C2F258 + 24);` — a bare global-struct-offset
   read with the identical shape as `get_update_time()` on the other
   versions (a fixed offset off a process-global timer struct); treated as
   the same function by strong structural analogy, not by symbol name.
2. `Encode2 nPOS` — `*((_WORD*)this+116)`
3. `Encode4 nItemID` — `*((_DWORD*)this+59)`
4. `EncodeStr sName`
5. `Encode4 al[0..3]` — `sub_757220(v2)`, loop `v2 <= 3`
6. `Encode4 nGender` — `*((unsigned __int8*)this+520)` — decompiler infers a
   1-byte source read here (vs. a 4-byte `_DWORD` read on v83/v87/v95 for
   the same field); this is a **struct-layout inference artifact**, not a
   wire-width change — `COutPacket::Encode4` always emits 4 bytes regardless
   of the width of the value passed to it, so the wire shape is unaffected.
   Flagged for Task 3+ in case the narrower read is instead evidence of an
   actual value-range change on v92, but no such gate was found in this
   pass.
7. `Encode4 nCurrentClass` — `*((_DWORD*)this+114)`
8. `Encode4 nSP` — `*((_DWORD*)this+116)`
9. `Encode4 update_time` — `sub_936E80(v9)` — **TRAILING (second
    occurrence)**

Same double-update_time shape as v87.

### §2.4 — gms_v95 (`0x77a240`, size `0x178`)

`decompile_sha256`: **PENDING**, same reason.

Opcode: `COutPacket::COutPacket(&oPacket, 85)` — **85 decimal = 0x55**.
Matches `docs/packets/registry/gms_v95.yaml` `USE_CASH_ITEM` (serverbound,
opcode 85) exactly. See §3 for why this is decisive for OQ-1.

Encode order (this version has named fields via RTTI-resolved struct
members, no raw offsets):

1. `Encode4 update_time` — `get_update_time()` — **LEADING**
2. `Encode2 nPOS` — `this->m_nPOS`
3. `Encode4 nItemID` — `this->m_nItemID`
4. `EncodeStr sName` — `CCtrlEdit::GetText(this->m_pEdit.p, ...)`
5. `Encode4 al[0..3]` — `CUICharacterSaleDlg::GetSelectedAL(this, v2)`, loop
   `v2 <= 3`
6. `Encode4 nGender` — `this->m_nGender`
7. `Encode4 nCurrentClass` — `this->m_nCurrentClass`
8. `Encode4 nSP` — `this->m_nSP`
9. `Encode4 update_time` — `get_update_time()` — **TRAILING (second
    occurrence)**

Same double-update_time shape as v87/v92.

### §2.5 — cross-version summary table

| version | address | opcode (dec) | registry op | update_time | field count after nItemID/nPOS |
|---|---|---|---|---|---|
| gms_v83 | `0x7d7960` | 79 | `USE_CASH_ITEM` | trailing only | sName, al×4, nGender, nCurrentClass, nSP, update_time |
| gms_v84 | — | — | — | — | **VERSION-ABSENT (§2.0)** |
| gms_v87 | `0x82e402` | 82 | `USE_CASH_ITEM` | leading + trailing | update_time(lead), sName, al×4, nGender, nCurrentClass, nSP, update_time(trail) |
| gms_v92 | `0x758770` | 86 | `USE_CASH_ITEM` | leading + trailing | (same shape as v87) |
| gms_v95 | `0x77a240` | 85 | `USE_CASH_ITEM` | leading + trailing | (same shape as v87) |

The `MajorAtLeast` boundary for the update_time-leads-and-doubles behavior is
`t.IsRegion("GMS") && t.MajorAtLeast(87)` (v83 is the sole exception below
that line; v84 has no wire to gate at all, §2.0). This matches
`UpdateTimeFirst`'s existing `>= 87` boundary in
`libs/atlas-packet/cash/serverbound/item_use.go:22-24` for the *leading*
copy; the *trailing second copy* on v87+ is a new finding this task
surfaces and Task 3+ must account for (it is not currently modeled by
`ItemUse`, which only carries one `updateTime` field).

### §2.6 — does the layout match `charsb.CreateCharacter`?

**No.** `libs/atlas-packet/character/serverbound/create.go`'s
`CreateCharacter` (`CLogin::SendNewCharPacket`, the ordinary login-socket
character creation packet) carries: `name`, `jobIndex`, `subJobIndex`,
`face`, `hair`, `hairColor`, `skinColor`, `topTemplateId`,
`bottomTemplateId`, `shoesTemplateId`, `weaponTemplateId`, `gender`,
`strength`, `dexterity`, `intelligence`, `luck` — a from-scratch
job/appearance/starting-equipment/stat-roll packet with no `nPOS`/`nItemID`
cash-slot addressing and no `update_time` at all.
`CUICharacterSaleDlg::SendCreateNewCharacter` carries `nPOS`/`nItemID` (the
cash-slot being consumed), `sName`, four `SelectedAL` appearance-selection
values, `nGender`, `nCurrentClass`, `nSP`, and update_time (once or twice
depending on version) — a cash-item-consumption packet whose payload
happens to also create a character, sent under the shared `USE_CASH_ITEM`
opcode family (§3), not the login-socket `CLogin` opcode family at all.
The two packets share no field for field, no opcode, and no encode-order
structure. FR-1.1 is answered: **the layouts are unrelated; nothing about
`CreateCharacter` may be assumed for the Maple Life codec.**

---

## §3. `USE_MAPLELIFE` (registry opcode 303) and OQ-1

`docs/packets/registry/gms_v95.yaml:4038-4042` carries:

```yaml
- op: USE_MAPLELIFE
  direction: serverbound
  opcode: 303
  fname: ""
  provenance: csv-import
```

`fname` is empty and `provenance: csv-import` — this op name was never
resolved against the client; it sits in a run of `UNNAMED_R4xx` placeholders
(opcode 302 = `UNNAMED_R412`, opcode 304 = `UNNAMED_R414`) that a CSV import
speculatively labeled.

**Search performed:** enumerated every `CUICharacterSaleDlg::` method on
gms_v95 (`func_query` filter `*CUICharacterSaleDlg*`, 50 results, full list
captured). Exactly two methods construct a `COutPacket` with a literal
opcode: `SendCreateNewCharacter` (opcode 85, §2.4) and
`SendCheckDuplicateIDPacket` (`0x777d20`, opcode **311** —
`COutPacket::COutPacket(&oPacket, 311)`, for the name-availability probe).
Neither is 303. No other method in the class sends anything.

Separately, and decisively: `CWvsContext::SendConsumeCashItemUseRequest`
(the generic cash-item-use sender, `0x9eb3e0` on v95) constructs exactly one
`COutPacket` in its entire 21,258-byte body (confirmed via `insn_query`
scoped to the function, searching for calls to the `COutPacket(this,int)`
constructor — exactly one match), at `0x9eb4aa`, with the literal operand
`push 55h` immediately before the call (`0x9eb4a4`) — **opcode 85, the exact
same literal `CUICharacterSaleDlg::SendCreateNewCharacter` uses**, and the
exact opcode `docs/packets/registry/gms_v95.yaml` names `USE_CASH_ITEM`.

This same cross-check was repeated for v83/v87/v92 (§2.5 table): in every
version that has `CUICharacterSaleDlg`, the literal opcode
`SendCreateNewCharacter` passes to `COutPacket::COutPacket` is **identical**
to that version's registry `USE_CASH_ITEM` opcode (79/82/86/85 respectively).
This is not a coincidence across four independent binaries — it is the
client reusing the generic cash-item-use opcode for the character-creation
sub-body, keyed by the `543`-classification arm derived in §1.

**OQ-1 answer, stated exactly as required:** ***v95 sends only the
`USE_CASH_ITEM` 543 sub-body*** — `CUICharacterSaleDlg::SendCreateNewCharacter`
is not a standalone opcode-303 packet; it is a `USE_CASH_ITEM` (opcode 85)
request whose leading `update_time`/`nPOS`/`nItemID` triple is exactly the
shape `libs/atlas-packet/cash/serverbound/item_use.go`'s `ItemUse` common
prefix already models, followed by the `05431000`/`05432000`-specific
sub-body (`sName`, 4× `SelectedAL`, `nGender`, `nCurrentClass`, `nSP`,
trailing `update_time`). Evidence: identical opcode literal at the
`CUICharacterSaleDlg::SendCreateNewCharacter` call site and at
`CWvsContext::SendConsumeCashItemUseRequest`'s single `COutPacket`
construction site, both `0x55` (85) on gms_v95; cross-confirmed on
gms_v83/v87/v92 against those versions' own `USE_CASH_ITEM` opcodes.

**Registry finding to flag (not fixed by this task — derivation.md only):**
`docs/packets/registry/gms_v95.yaml`'s `USE_MAPLELIFE` (opcode 303) label is
very likely a **CSV-import mislabel** unconnected to this feature — no
`CUICharacterSaleDlg` method or the generic cash-item sender ever constructs
a packet with that opcode. Task 3+ (or a follow-up registry-correction
commit, own change per `IMPLEMENTING_A_PACKET.md`'s "escalate, don't
auto-fix" guidance) should not treat opcode 303 as related to Maple Life
character creation. Consequently **Task 6 should not write
`libs/atlas-packet/maplelife/serverbound/use.go` as a standalone-opcode
codec** — the serverbound wire for this feature belongs inside the
`cash/serverbound` `USE_CASH_ITEM` family (a new sub-body variant keyed on
the `543`/`5431`-`5432` classification, extending or sitting alongside
`ItemUse`), not as a new top-level `maplelife` serverbound package. This is
a load-bearing recommendation for Task 6's package layout and should be
treated as settled unless a later task finds live-traffic evidence
contradicting it.

---

<!-- Task 2 appends §4 (OnCreateNewCharacterResult decode),
     §5 (OnCheckDuplicatedIDResult decode / item-id table), and
     §6 below this line. Do not edit §1–§3 above without re-deriving. -->
