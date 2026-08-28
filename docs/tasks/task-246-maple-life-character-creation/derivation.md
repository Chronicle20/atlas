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

`decompile_sha256`: **RESOLVED** (Task 6). `CUICharacterSaleDlg::SendCreateNewCharacter`
was re-harvested and spliced into `docs/packets/ida-exports/gms_v83.json`
this pass (function was already named in the IDB — no rename needed) — value
`40ac1642353a50211515916e1b0ade437e69256cf827317ae304b554eb067983`. See
`docs/packets/evidence/gms_v83/cash.serverbound.CashItemUseMapleLife.yaml`.

Original PENDING note, kept for context:
`CUICharacterSaleDlg::SendCreateNewCharacter` was **absent as a key** in
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

`decompile_sha256`: **RESOLVED** (Task 6) —
`1b18651d77d7af253eb8e72ab2aef44f0c97ff290917eab9dd9a2b930b75a861`, spliced
into `docs/packets/ida-exports/gms_v87.json` (function already named, no
rename needed). See
`docs/packets/evidence/gms_v87/cash.serverbound.CashItemUseMapleLife.yaml`.

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

`decompile_sha256`: **RESOLVED** (Task 6) —
`05ac2d0ed80fc2d326a63351d8119e39a2dc762004cbbc80a87886327a1ec0d2`, spliced
into `docs/packets/ida-exports/gms_v92.json` (function already named, no
rename needed). See
`docs/packets/evidence/gms_v92/cash.serverbound.CashItemUseMapleLife.yaml`.

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

`decompile_sha256`: **RESOLVED** (Task 6) —
`adbf4df033dab6757dfc025fbe2cb3f51a521c47f50c47f5a67dfd320117dc42`, spliced
into `docs/packets/ida-exports/gms_v95.json` (function already named, no
rename needed). See
`docs/packets/evidence/gms_v95/cash.serverbound.CashItemUseMapleLife.yaml`.

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

<!-- Task 2 appends §4 (OnCheckDuplicatedIDResult decode), §5
     (OnCreateNewCharacterResult decode), and §6 (duplicate-probe opcodes /
     C1) below this line. NOTE: Task 1's placeholder comment above numbered
     these §4=OnCreateNewCharacterResult/§5=OnCheckDuplicatedIDResult (reversed)
     and folded the item-id table into §5; Task 2's actual brief
     (.superpowers/sdd/plan/task-2-brief.md Steps 1–3) numbers them the other
     way and has no item-id-table deliverable (that table is §1.4, already
     written). This task follows the brief's numbering, which is authoritative
     for Task 2's own deliverable. Do not edit §1–§3 above without re-deriving. -->

## §4. `CUICharacterSaleDlg::OnCheckDuplicatedIDResult` — the duplicate-name-check result

All five in-scope versions carry this receiver; v84 is VERSION-ABSENT (§2.0,
re-confirmed below). On every present version the function is field-for-field
identical: `DecodeStr sName` (decoded but only used for formatting/UI, not a
routing key), then `Decode1 nResult` as a **SIGNED** byte, three-way branch —
structurally identical to the sibling `CCashShop::OnCheckDuplicatedIDResult`
precedent in `libs/atlas-packet/cash/clientbound/check_name_change.go:22-53`.

`decompile_sha256`: **RESOLVED** (Task 4, this pass) on all four in-scope
versions. The blocking gap was that `CUICharacterSaleDlg::OnCheckDuplicatedIDResult`
was absent as a key in every in-scope `docs/packets/ida-exports/<version>.json`
(confirmed by direct key-scan of the `functions` map on all four present
versions — each export only carried the unrelated `CLogin::OnCheckDuplicatedIDResult`
and `CCashShop::OnCheckDuplicatedIDResult` keys). This was the same blocking
dependency §2.1 escalated for Tasks 4–6; Task 4 closed it for this receiver by:
renaming the unnamed v83/v87 IDB functions (`sub_7D768A`, `sub_82E12C`) to the
mangled symbol `?OnCheckDuplicatedIDResult@CUICharacterSaleDlg@@AAEXAAVCInPacket@@@Z`
(matching the already-named v92/v95 twin, confirmed via `func_query`), then
harvesting each version with `go run ./tools/packet-audit export -splice
"CUICharacterSaleDlg::OnCheckDuplicatedIDResult" -ida-database <session>
-ida-url http://192.168.20.3:8745/mcp` (the working IDA-MCP endpoint — the
CLI's own `-ida-url` default, `http://192.168.20.3:13337/mcp`, is a known-stale
value per `docs/TODO.md`'s "Tooling defects found in `tools/packet-audit`"
entry) and surgically splicing only that one entry into the committed export
(the CLI's own struct round-trip was found to silently drop a legacy
singular-`note` field present on ~200 unrelated entries per file — worked
around by textual insertion of only the new key, verified byte-identical
elsewhere by post-hoc JSON diff). `evidence pin` then computed the real
hashes: v83 `9643707ade90f59a9f5238724e975b0a78c9ca1400d118d1879b51ae53a0d61a`,
v87 `bfbf959f72435f5b48b6ebe51af277d146bed0593862418a0b9ad5070cd89523`,
v92 `0e5899046bd788bfc4fbdd05d06876d04a590cb0d6746fb5efa0405e7e3e8b19`,
v95 `956ddb8de90e489899a016a58572bb41c81b0455f3bff99e31efc33002940574` — see
`docs/packets/evidence/gms_v8{3,7}/maplelife.clientbound.MapleLifeResult.yaml`,
`docs/packets/evidence/gms_v92/maplelife.clientbound.MapleLifeResult.yaml`,
`docs/packets/evidence/gms_v95/maplelife.clientbound.MapleLifeResult.yaml`.

### §4.1 — how the receiver was located (all versions)

`func_query` for `*CUICharacterSaleDlg*` does **not** return an
`OnCheckDuplicatedIDResult`/`OnCreateNewCharacterResult`/`OnPacket` triad by
name on v83/v87 (IDA/PDB only resolved the class's UI-flow methods —
`ShowWindow`, `OnButtonClicked`, `Send*` — for those two binaries); v92/v95
resolve the full set including `OnPacket` directly. Where the names are
unresolved, the receivers were located by scanning for
`CInPacket::DecodeStr`/`Decode1`/`Decode4` call sites inside the address
range spanned by the class's other known methods, then confirmed by finding
the dialog's own `OnPacket` dispatcher via `xrefs_to` on the two candidate
functions (the dispatcher is always the sole caller of both). This dispatcher
is reached from `CField::OnPacket`'s bound-check ladder via a per-field
vtable-slot-`0x3C` indirect call (`mov eax,[ecx]; call [eax+3Ch]`) — on v83
the field the client's own PDB names `CField::OnItemUpgrade`
(`ecx+0x1F4`, covering `CField::OnPacket`'s `0x15D–0x160` (349–352) bound
check) is the one that actually forwards to the Maple-Life dialog's
`OnPacket`, **not** the field/bound-check pair the client names
`CField::OnCharacterSale` (`ecx+0x1F8`, `0x161–0x164`/353–356) — the PDB's
"ItemUpgrade"/"CharacterSale" function names do not line up with which
field routes which opcode range on v83; this is a client-internal naming
quirk, not a registry error (the *registry's* MAPLELIFE opcodes 349/350 are
independently confirmed correct below).

### §4.2 — gms_v83 (`0x7d768a`, size `0x126`, unnamed `sub_7D768A` — PDB does
not resolve `CUICharacterSaleDlg::OnCheckDuplicatedIDResult` on this binary)

Reached via `CField::OnPacket`(`0x531325`)'s `0x15D–0x160` arm →
`CField::OnItemUpgrade`(`0x537f8c`, `ecx+0x1F4`, virtual+`0x3C`) →
`CUICharacterSaleDlg::OnPacket` (`0x7d7586`, unnamed, confirmed by
`xrefs_to` on both `0x7d768a` and `0x7d77b0`):

```
if (a2 == 349) OnCheckDuplicatedIDResult(this, a3);   // 0x7d768a
else if (a2 == 350) OnCreateNewCharacterResult(this, a3); // 0x7d77b0
```

Opcode **349** confirmed for `MAPLELIFE_RESULT` (matches
`docs/packets/registry/gms_v83.yaml:1805-1808` exactly). Decode order:

1. `DecodeStr sName`
2. `Decode1 nResult` (**SIGNED**)

Branch enumeration (identical shape to `check_name_change.go`'s three-way
arm, disassembly-confirmed `jle`/signed compare pattern via decompiler's
`if (v4 <= 0) { if (v4) {…} else {…} } else {…}`):

| arm | UI action |
|---|---|
| `nResult > 0` | `StringPool` SP_5047 "this name is currently being used, please check again" `Notice`; dialog virtual+0x20 call with code `1001` |
| `nResult == 0` | enables the "next" button (`this[61]+4`'s vtable+28 call with `1`); dialog virtual+0x20 call with code `1000` |
| `nResult < 0` | `StringPool` SP_5595 "unknown error (%d)" `Notice`, formatted with `nResult`; dialog virtual+0x20 call with code `1001` |

(`1000`/`1001` are the dialog's own internal UI-transition codes passed to
its `OnButtonClicked`-family virtual, confirmed by name on v95 §4.5 — not
wire values.)

### §4.3 — gms_v84 — VERSION-ABSENT, independently re-confirmed

`func_query *CUICharacterSaleDlg*` and `*CharacterSale*` on v84
(session `46c2a2eb`) return **zero** matches beyond the unrelated
`CField::OnCharacterSale`(`0x5443af`). That function itself carries a
pre-existing task-129 IDA comment (read directly from the decompile, not
authored by this task): *"IDB symbol says CField::OnCharacterSale, but THIS
is the Vicious Hammer forwarder (functional CField::OnItemUpgrade)... The
IDB-named CField::OnItemUpgrade @0x544395 (this[134]) routes 359/360 to the
name/world-transfer dialog, NOT the hammer."* — independent, pre-existing
confirmation that **no** Maple-Life/CharacterSaleDlg code path exists on
v84 under either PDB-named forwarder. `func_query *DuplicateID*` on v84
returns only the unrelated `CLogin::SendCheckDuplicateIDPacket`
(login-socket ordinary duplicate-name check, opcode 8 — §6.2). This
corroborates §2.0's VERSION-ABSENT finding a second, independent way for
this task's own functions.

### §4.4 — gms_v87 (`0x82e12c`, size `0x126`, unnamed `sub_82E12C`)

Reached via `CUICharacterSaleDlg::OnPacket` dispatcher `0x82e028` (unnamed,
confirmed via `xrefs_to` on `0x82e12c`/`0x82e252`):

```
if (a2 == 370) OnCheckDuplicatedIDResult(this, a3);   // 0x82e12c
else if (a2 == 371) OnCreateNewCharacterResult(this, a3); // 0x82e252
```

Opcode **370** confirmed for `MAPLELIFE_RESULT` (matches
`docs/packets/registry/gms_v87.yaml` exactly, brief's expected value).
Decode order and branch shape **byte-for-byte identical** to v83
(`DecodeStr sName; Decode1 nResult SIGNED`, same three-way arm); only the
`StringPool` string IDs differ (5910 unknown-error, 5058 taken — cosmetic
resource-ID renumbering, not a wire change).

### §4.5 — gms_v92 (`0x756370`, **named**
`CUICharacterSaleDlg::OnCheckDuplicatedIDResult`)

`CUICharacterSaleDlg::OnPacket` is directly named on v92 (`0x757180`):

```c
if (a2 == 404) return OnCheckDuplicatedIDResult(this, a3);
result = a2 - 405;
if (a2 == 405) return OnCreateNewCharacterResult(this, a3);
return result;
```

Opcode **404** confirmed (matches brief's expected value and
`docs/packets/registry/gms_v92.yaml`). Decode order/branch identical to
v83/v87 (`DecodeStr sName; Decode1 nResult SIGNED`; string IDs 6348/5125).

### §4.6 — gms_v95 (`0x777e40`, **named**
`CUICharacterSaleDlg::OnCheckDuplicatedIDResult`)

`CUICharacterSaleDlg::OnPacket` (`0x778c50`):

```c
if (nType == 0x19D) OnCheckDuplicatedIDResult(this, iPacket);
else if (nType == 0x19E) OnCreateNewCharacterResult(this, iPacket);
```

0x19D = **413**, matches brief and `docs/packets/registry/gms_v95.yaml`
exactly. Decode order/branch identical to all prior versions (`DecodeStr
sName; Decode1 nResult SIGNED`; string IDs `0x1A86` unknown-error, `0x13C9`
taken). This version's decompile resolves the dialog-state virtual call by
name — `this->OnButtonClicked(this, 1000u)` / `this->OnButtonClicked(this,
1001u)` — confirming the `1000`/`1001` codes noted in §4.2 are
`OnButtonClicked`-family UI-transition codes, not wire fields.

### §4.7 — cross-version summary and version gate

| version | receiver addr | opcode | decode order | branch shape |
|---|---|---|---|---|
| gms_v83 | `0x7d768a` | 349 | DecodeStr sName; Decode1 nResult(SIGNED) | >0 taken / ==0 available+enable-next / <0 unknown-error(fmt) |
| gms_v84 | — | — | **VERSION-ABSENT** | — |
| gms_v87 | `0x82e12c` | 370 | (same) | (same) |
| gms_v92 | `0x756370` | 404 | (same) | (same) |
| gms_v95 | `0x777e40` | 413 | (same) | (same) |

**No field, width, order, or branch-arm divergence across any present
version** — the codec needs no `MajorAtLeast` gate for its body shape. Only
the **opcode** (a per-tenant-template value, resolved via
`WithResolvedCode`/template `operations` table per DOM-25, never a Go
literal) and v84's absence differ.

---

## §5. `CUICharacterSaleDlg::OnCreateNewCharacterResult` — the full error-code enumeration

`decompile_sha256`: **RESOLVED** (Task 5, carried forward here by Task 6 per
the controller's Task-5-review addendum). Task 5 re-harvested the export the
same way Task 4 did for §4 (rename + splice) and pinned the real hashes into
`docs/packets/evidence/gms_v8{3,7}/maplelife.clientbound.MapleLifeError.yaml`,
`docs/packets/evidence/gms_v92/maplelife.clientbound.MapleLifeError.yaml`,
`docs/packets/evidence/gms_v95/maplelife.clientbound.MapleLifeError.yaml` —
values taken verbatim from those four pinned records, not re-derived here:
v83 `1c9c197af60b2741ba35a05de501060e7c9ea151d54a990476d164d3f6ac60c9`,
v87 `281b1504d26f78299a70c4bd4c3466ae7588a177edc1251d1901803289eed3b0`,
v92 `be406086de1e689fc3dbf876b1e156c46bb32faed09a297aa7634e24dabc28bb`,
v95 `9d64bc8fbd5cb82d6df5c8c2a566c19a55fbe71a71c0366a7aae95191443e079`.

### §5.1 — gms_v83 (`0x7d77b0`, size `0x1b0`, unnamed `sub_7D77B0`)

Reached from the same `0x7d7586` dispatcher as §4.2 (opcode 350). Decode
order:

1. `Decode1 nType` (unsigned byte)
2. `Decode4 nParam` (4-byte)

Then: reset a "request sent" flag and an unrelated field-state cleanup, plus
a dialog virtual+0x34 call with `1` (`(*(*this+52))(this,1)`, UI-state, not
wire). **Full branch enumeration** (exact-equality switch on `nType`, not a
range/signed test):

| `nType` | `nParam` | UI outcome |
|---|---|---|
| 52 (`0x34`) | `== 0` | **SUCCESS** — `StringPool` SP_5048 "creation has completed successfully" `Notice`; increments a `CWvsContext`-local character-slot-usage counter (client-side bookkeeping, not wire) |
| 52 (`0x34`) | `!= 0` | `StringPool` SP_5595 "unknown error (%d)" `Notice`, formatted with `nParam` |
| 54 (`0x36`) | *(any)* | `StringPool` SP_5046 "you can not use this name, please check again" `Notice` — duplicate-name-at-submit |
| *(any other `nType`)* | *(any)* | `StringPool` SP_5595 "unknown error (%d)" `Notice`, formatted with `nParam` — generic fallback |

This is the design §5.4-required deliverable: **SUCCESS = `nType==52 &&
nParam==0`**; the "duplicate at submit" arm is `nType==54`; the generic
fallback carries `nParam` as a formatted diagnostic code, not a fixed
enum member.

### §5.2 — gms_v87 (`0x82e252`, size `0x1b0`, unnamed `sub_82E252`)

Opcode 371 (§4.4). Identical decode order (`Decode1 nType; Decode4 nParam`)
and identical branch **shape**, but the `nType` literals shift by **+2**:
success/error-format arm is `nType==54`, duplicate-at-submit arm is
`nType==56`. String IDs: 5059 (success), 6348 (unknown error fmt), 5057
(duplicate-at-submit).

### §5.3 — gms_v92 (`0x7564f0`, **named**
`CUICharacterSaleDlg::OnCreateNewCharacterResult`)

Opcode 405 (§4.5). Same decode order and shape; literals shift by **+1**
from v87: success/error-format arm `nType==55`, duplicate-at-submit arm
`nType==57`. String IDs: 5126 (success), 6348 (unknown error fmt), 5124
(duplicate-at-submit). `nType` is decoded via `Decode1` and compared as an
`unsigned __int8`-cast value here (vs. plain `char`/signed on v83/v87) —
decompiler artifact of the same kind flagged in §2.3 (does not change the
wire width; `Decode1` is always a 1-byte field on every version).

### §5.4 — gms_v95 (`0x777fc0`, **named**
`CUICharacterSaleDlg::OnCreateNewCharacterResult`)

Opcode 414 (§4.6). Same decode order and shape; literals shift by **+2**
from v92 (matching the v83→v87 shift): success/error-format arm
`nType==56`, duplicate-at-submit arm `nType==58`. String IDs: `0x13CA`
(success), `0x1A86` (unknown error fmt), `0x13C8` (duplicate-at-submit).
Also gates on `g_pStage` being a live `CField`-kind stage before clearing a
per-stage flag — in-game-only client bookkeeping, not a wire field.

### §5.5 — cross-version summary, error-code enumeration, and version gate

| version | receiver addr | opcode | `nType` SUCCESS | `nType` duplicate-at-submit | fallback |
|---|---|---|---|---|---|
| gms_v83 | `0x7d77b0` | 350 | 52 (`nParam==0`) | 54 | 52+`nParam!=0`, or any other `nType` → unknown-error(`nParam`) |
| gms_v84 | — | — | **VERSION-ABSENT** | — | — |
| gms_v87 | `0x82e252` | 371 | 54 (`nParam==0`) | 56 | (same shape) |
| gms_v92 | `0x7564f0` | 405 | 55 (`nParam==0`) | 57 | (same shape) |
| gms_v95 | `0x777fc0` | 414 | 56 (`nParam==0`) | 58 | (same shape) |

The **closed enumeration is three semantic arms** on every version —
SUCCESS, DUPLICATE-NAME-AT-SUBMIT, UNKNOWN-ERROR(param) — with the raw
`nType` literal for each arm being a per-version, per-tenant-template
config value (never a Go literal; resolve via the template `operations`
table, DOM-25) and `nParam` carried through as a formatted diagnostic value
on the UNKNOWN-ERROR arm only. No design-anticipated "invalid look" arm was
found on any version — a client-side validity failure never reaches the
wire on the create-character path; the two rejectable-at-server outcomes
the client renders are duplicate-name-at-submit and the generic
unknown-error fallback. Tasks 5/7's `options` keys should be exactly
`{SUCCESS, NAME_TAKEN_AT_SUBMIT, UNKNOWN_ERROR}` (naming is Task 5's call;
the three-arm closure itself is this task's finding). No `MajorAtLeast` gate
is needed for field shape (identical on all four present versions); only the
opcode and the per-version `nType` literal values are tenant-template
config, not a code branch.

---

## §6. Duplicate-probe sender (`SendCheckDuplicateIDPacket`) and C1

### §6.1 — per-version derivation

| version | sender address | opcode emitted | body encode order | collides with `CHECK_CHAR_NAME` (21)? |
|---|---|---|---|---|
| gms_v83 | `0x7d75ab` | **256** (`0x100`) | `EncodeStr sCharName` only | no |
| gms_v84 | — | — (VERSION-ABSENT) | — | n/a |
| gms_v87 | `0x82e04d` | **270** (`0x10E`) | `EncodeStr sCharName` only | no |
| gms_v92 | `0x756250` (unnamed; reached via `xrefs_to` on `SendRequest@CUICharacterSaleDlg`) | **301** (`0x12D`) | `EncodeStr sCharName` only | no |
| gms_v95 | `0x777d20` | **311** | `EncodeStr sCharName` only | no |

All four present versions: client-side validates the name via
`is_valid_character_name(sCharName, !isUnderCover)` first (rejects locally
with a "cannot use this name" `Notice` + dialog code `1001`, **no packet
sent**, if invalid); only a client-side-valid name is sent to the server.
Every version's wire body is a single `EncodeStr` field — no length prefix
beyond the standard string encoding, no other fields. `decompile_sha256`:
**RESOLVED** (Task 6) on all four in-scope versions —
`CUICharacterSaleDlg::SendCheckDuplicateIDPacket` was spliced into each
version's `docs/packets/ida-exports/<version>.json` this pass: v83
`edb0261730d6141c43fc3f1861a7ebf8bc6e0b8c02213d0b60aa6fdd95f3bfd6`, v87
`4421824b03fb872407d05ccf9745fc55bd9d075e31002469128dc39f72fa7321`, v92
`cc9ce2f7d6f7b1b2ab72b5cae80239040969dba3e572fa681fb15f26376d44b4` (v92's
`sub_756250` was renamed to the mangled
`?SendCheckDuplicateIDPacket@CUICharacterSaleDlg@@QAEXABV?$ZXString@D@@@Z`
symbol this pass, matching the already-named v83/v87/v95 twins), v95
`fd119d83db7897762dee55fc8fa94eb8396b8490e19efd7623e9371d8df681aa`. See
`docs/packets/evidence/gms_v8{3,7}/maplelife.serverbound.MaplelifeCheckName.yaml`,
`docs/packets/evidence/gms_v92/maplelife.serverbound.MaplelifeCheckName.yaml`,
`docs/packets/evidence/gms_v95/maplelife.serverbound.MaplelifeCheckName.yaml`.

Original PENDING note, kept for context (v87/v92 the two that had been
absent from the export's `functions` keys before this pass's splice, same
reasoning as §4/§5): v83's
export (`docs/packets/ida-exports/gms_v83.json`) already carries a
**pre-existing note** on the `CLogin::SendCheckDuplicateIDPacket` key stating
verbatim: *"v83 character-name duplicate check. The actual sender is
CUICharacterSaleDlg::SendCheckDuplicateIDPacket@0x7d75ab (v83 routes name
checks through the character-sale dialog): COutPacket(0x100) +
EncodeStr(sCharName)."* — this independently, exactly corroborates this
task's own v83 finding (address, opcode, and body all match). v95's export
carries no such note but its `functions` map also lacks the direct key;
v92 was independently derived above via the unnamed `sub_756250`.

**Registry cross-check:** v87 opcode 270 and v95 opcode 311 exactly match
the `fname: CUICharacterSaleDlg::SendCheckDuplicateIDPacket` rows already
present in `docs/packets/registry/gms_v87.yaml:3651-3655` and
`gms_v95.yaml:4100-4104` — both currently filed under the op name
`JMS_SLASH_COMMAND` (§6.3 explains why that name is wrong). v83's opcode 256
and v92's opcode 301 have **no row at all** in their registries (confirmed
by direct grep/YAML-parse of both files for `opcode: 256`/`opcode: 301` +
`direction: serverbound` — zero matches on either) — these two are
previously-undiscovered opcodes this task surfaces for the first time.
(v92's registry does have an unrelated **clientbound** `opcode: 301`
row — `MOB_ATTACKED_BY_MOB`/`CMob::OnMobAttackedByMob` — clientbound and
serverbound opcode spaces are numbered independently in this protocol, so
this is not a collision.)

### §6.2 — routing consequence: **(A)**

Every in-scope GMS version with the feature present (v83, v87, v92, v95)
has its **own dedicated Maple-Life-specific probe opcode** — none of them
reuse `CHECK_CHAR_NAME` (21, the ordinary login-socket duplicate-name-check
opcode bound to `CashShopCheckNameChangeHandle`/other login flows per
`template_gms_83_1.json:161-169`). v84 has no probe at all (feature absent).
**Task 12 writes a standalone handler** for this op; the pending-record
disambiguation design (routing consequence B) is **not needed** for this
feature. (v83/v84's other `CLogin::SendCheckDuplicateIDPacket` /
`CCashShop::OnCheckDuplicatedIDResult` functions found during this search
are the pre-existing, unrelated ordinary-login and cash-shop-rename probes
respectively — confirmed by their own distinct opcodes, 8 and 328 — and
must not be confused with the Maple-Life probe.)

### §6.3 — jms_v185 opcode 271: **UNRESOLVED — could not positively identify within this pass**

`func_query *CUICharacterSaleDlg*` on jms_v185 (session `a977912e`) returns
**zero** matches — unlike every in-scope GMS version, JMS does not carry a
distinct `CUICharacterSaleDlg` class at all. The wizard-step methods that on
GMS belong to `CUICharacterSaleDlg` (`SetStep1`–`SetStep5`,
`ShiftNewCharEquip`, `GetSelectedAL`, `LoadNewCharInfo`) are, on JMS,
**folded directly into the `CLogin` class** (`CLogin::LoadNewCharInfo`
`0x671842`, `CLogin::ShiftNewCharEquip` `0x6724bd`,
`CLogin::GetSelectedAL` `0x6725a9`/`0x6725fc` — confirmed via `func_query`)
— an architectural difference from every GMS build checked, where this
functionality lives in its own dialog class. `CField::OnCharacterSale`
(`0x57528c`) exists on JMS with the same `mov ecx,[ecx+214h]; call
[eax+3Ch]` forwarder shape seen on GMS, but the object it forwards to was
not identified.

**Searches performed and their results:**
- `func_query` for `*SendCheckDuplicateIDPacket*`/`*DuplicateID*`: only
  `CLogin::SendCheckDuplicateIDPacket` (`0x66e467`) resolves — decompiled
  and confirmed to be the **ordinary login-socket** duplicate-check
  (`COutPacket::COutPacket(v9, 8)` — opcode **8**, not 271; unrelated to
  Maple Life).
- `insn_query` for `push 271` (0x10F immediate) and for any operand `==271`,
  scoped to the `CLogin`-cluster address range `0x670000–0x674000` (where
  the folded-in wizard-step methods live): **zero matches**.
- `insn_query` for `push 271` unscoped (`allow_broad`): the query's
  200,000-instruction scan cap is exhausted before covering the binary's
  full `.text` section (a multi-megabyte region), so this search is
  **inconclusive**, not a negative result.
- `find` (string) for `"CUICharacterSaleDlg"`: zero matches (no RTTI string
  on this build, matching every GMS build checked too — expected, not
  informative either way).
- A `Decode1`/`DecodeStr`-call scan of the `CLogin` cluster
  (`0x670000–0x673000`) surfaced `sub_671717`, which was decompiled and
  ruled out — it is `CLogin`'s GameGuard-update-check flow (sends opcode
  `0x19`=25), unrelated.

**What is known, not invented:** `docs/packets/registry/jms_v185.yaml:3640-
3644` already carries `fname: CUICharacterSaleDlg::SendCheckDuplicateIDPacket`
for opcode 271 (csv-import provenance, i.e. not yet IDA-verified by any
prior pass either). Given (a) this fname exactly matches the confirmed-real
function this task independently verified at three other GMS opcodes
(270/301/311, §6.1), and (b) JMS folds the surrounding character-sale
wizard functionality into `CLogin` rather than a separate class, the
circumstantial hypothesis is that JMS opcode 271 is this same probe
(implemented as an as-yet-unlocated `CLogin` method on this build) rather
than a genuine JMS-only slash command — but this task did **not** obtain a
decompile-confirmed send site for opcode 271, so this is a hypothesis, not
a derived fact, and per the brief's Step 4 instruction this must not be
treated as settled. **Escalating**: neither the "split" nor the "rename"
resolution in the brief's Step 4 can be safely chosen without positive
confirmation of what jms_v185 opcode 271 actually is. Recommend a follow-up
pass scoped narrowly to locating the `COutPacket` construction site for the
literal `271` inside the `CLogin` translation unit on jms_v185 (likely a
wider address sweep of the `CLogin`-cluster than the `0x670000–0x674000`
window checked here, since `CLogin` is a very large class on this build).
Task 3 should **not** split or rename the `JMS_SLASH_COMMAND` registry row
until this is resolved.

### §6.4 — summary

- **C1 answer for GMS (v83/v87/v92/v95): (A)** — each has its own
  Maple-Life-specific probe opcode (256/270/301/311 respectively); no
  `CHECK_CHAR_NAME`(21) collision on any of them; v84 has no probe (feature
  absent).
- **jms_v185 opcode 271: unresolved**, evidence of a genuine (not
  perfunctory) search recorded above; Task 3's registry-row decision for
  `JMS_SLASH_COMMAND` should wait on a follow-up derivation pass rather than
  guess.

### §2.0-CORRECTION — the v84 VERSION-ABSENT finding is retracted

Filed as part of `docs/tasks/task-246-maple-life-character-creation/bug-maple-life-v84-registration.md`.
**§2.0's conclusion above is wrong. `CUICharacterSaleDlg` exists on gms_v84.**
Every downstream task that honoured "VERSION-ABSENT on gms_v84" was acting on
a false premise. This subsection does not renumber or rewrite §2.0 — it
stands as written, retracted here.

§2.0 rested on three findings; two are artifacts of this IDB's symbol
coverage and the third is not load-bearing:

1. **`func_query` → zero matches (§2.0 finding #1) is a symbol-coverage
   artifact, not an absence.** `func_query name_regex=CUI` returns 24
   functions on v84 versus ~280 on v87. The v87 names are full mangled PDB
   symbols; several of v84's 24 are hand-made RE artifacts from earlier
   tasks. This IDB carries almost no class symbols, so "no
   `CUICharacterSaleDlg` symbol" carries no information about the class.
2. **The RTTI string test (§2.0 finding #2) is void.** `find_regex` for
   `CharacterSale` on the v87 session — where the class provably exists,
   with 15 named methods — also returns zero matches. The test cannot
   distinguish presence from absence on any build. §6.3 of this same
   document already said so ("no RTTI string on this build, matching every
   GMS build checked too — expected, not informative either way"),
   contradicting §2.0's use of the same test as evidence of absence.
3. **Finding #3's `CField` symbols were misapplied, not absent.** v84's IDB
   symbol `?OnCharacterSale@CField@@...` at `0x5443af` forwards through
   `this[135]` (`CField`+0x21C), which task-129 already annotated as
   `CUIItemUpgrade` (the Vicious Hammer path) — it is not the Maple Life
   route. v87's real `CField::OnCharacterSale` (`0x55fa2c`) forwards through
   `this[136]` (`CField`+0x220). The v84 Maple Life route is the *other*
   member — the IDB-named `CField::OnItemUpgrade` at `0x544395`
   (`this[134]`), whose task-129 comment records that it "routes 359/360".
   The two `CField` symbols are swapped on v84, which is what sent §2.0's
   finding #3 down the wrong path.

`CUICharacterSaleDlg` was located on v84 by structural fingerprint (not by
symbol), all four functions renamed in session `46c2a2eb`
(`GMS_v84.1_U_DEVM.i64`):

| v84 addr | new IDB name | v87 counterpart | size |
|---|---|---|---|
| `0x7fd86a` | `CUICharacterSaleDlg__SendCheckDuplicateIDPacket_send_0x107` | `0x82e04d` | `0xbf` both |
| `0x7fd845` | `CUICharacterSaleDlg__OnPacket_recv_0x167_0x168` | (virtual, vtable+0x3C) | `0x25` |
| `0x7fd949` | `CUICharacterSaleDlg__OnCheckDuplicatedIDResult_recv` | `0x82e12c` | — |
| `0x7fda6f` | `CUICharacterSaleDlg__OnCreateNewCharacterResult_recv` | `0x82e252` | — |

The v84 sender (`0x7fd86a`) is instruction-for-instruction identical to
v87's `0x82e04d` (size `0xbf` both), sending `COutPacket::COutPacket(v12,
263)` (`0x107`). The clientbound opcodes are read directly off
`CUICharacterSaleDlg__OnPacket` at `0x7fd845`, which decompiles to exactly:

```c
if ( a2 == 359 )      CUICharacterSaleDlg__OnCheckDuplicatedIDResult_recv(this, a3);
else if ( a2 == 360 ) sub_7FDA6F(a3);   // OnCreateNewCharacterResult
```

So on gms_v84: `MAPLELIFE_CHECK_NAME` = 263 (0x107) serverbound,
`MAPLELIFE_RESULT` = 359 (0x167) clientbound, `MAPLELIFE_ERROR` = 360
(0x168) clientbound. Field orders and operations tables match v83/v87's
(`AVAILABLE: 0, TAKEN: 1, UNKNOWN_ERROR: 255` for `MapleLifeResult`;
`SUCCESS: 52, NAME_TAKEN_AT_SUBMIT: 54, UNKNOWN_ERROR: 255` for
`MapleLifeError`). Corroborated on the data side: `Item.wz/Cash/0543.img`
(item `05430000`) is present in the v82-era pack, and v84 is bracketed by
v83 and v87, both of which ship the dialog.

An earlier prediction of opcodes 349/350 (from a registry-gap argument: on
v83 and v87 both, `MAPLELIFE_RESULT` sits at `MTS_OPERATION+1`, and v84 has
a matching two-slot hole at 349/350) **was wrong** — the dispatcher says
359/360. The gap argument is a lead, never a value.

**Status: retracted.** gms_v84 registers the same three ops as its
neighbours, at its own opcodes, per the fix landed alongside this
correction. jms_v185 is unaffected by this correction and remains
unresolved per §6.3/§6.4 above.

