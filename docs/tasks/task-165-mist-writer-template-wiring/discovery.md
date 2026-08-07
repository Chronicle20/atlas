# Task 165 — Tier B mist opcode discovery (gms_48, gms_92, gms_12)

Scope: reverse-engineer `AFFECTED_AREA_CREATED` (SPAWN_MIST) and
`AFFECTED_AREA_REMOVED` (REMOVE_MIST) for the three client versions that Tasks
1–7 did not cover, and wire only what could be read out of each version's own
IDB.

Method constraint honoured throughout: **case constants were read from
disassembly, never computed from a neighbouring version's opcode.** For the SEH
handler bodies the disassembly (not the Hex-Rays pseudocode) is the citation.

Anchor for correlation: gms_v61 `CAffectedAreaPool::OnPacket` @0x423eb7
(`a2 == 210` → Created, `a2 == 211` → Removed) and its handler
`CAffectedAreaPool::OnAffectedAreaCreated` @0x423edc, whose read order is the
one the eight already-verified versions encode.

---

## gms_48 — Outcome **A**

**Walked.** The demangled lookup that previously failed (and produced the
`unresolved: true` rows in `docs/packets/ida-exports/gms_v48.json`) fails only
because the v48 IDB carries the *mangled* symbol. A `name_regex` query for
`Pool.*OnPacket` resolved:

- Dispatcher: `?OnPacket@CAffectedAreaPool@@QAEXJAAVCInPacket@@@Z` @**0x42182f**
  — a two-arm dispatcher, sibling to the already-named `CTownPortalPool::OnPacket`
  @0x5e318d in the same IDB.
- Arms enumerated from disassembly at 0x42182f:
  - `sub eax, 0CAh` @0x421833 / `jz` @0x421838 → `?OnAffectedAreaCreated@…` @0x421854
  - `dec eax` @0x42183a / `jnz` (fallthrough) → `?OnAffectedAreaRemoved@…` @0x421e7f

**Opcodes.** `SPAWN_MIST = 202 (0xCA)`, `REMOVE_MIST = 203 (0xCB)`.
(Note the shift vs v61's 210/211 is −8, i.e. *not* the −20 CField-base delta —
which is exactly why the constants had to be read rather than derived.)

**Read order — Created** (`0x421854`, disassembly, EH-prolog function):

| # | addr | primitive | field |
|---|---|---|---|
| 1 | 0x421877 | Decode4 | dwId |
| 2 | 0x421881 | **Decode1** | nType |
| 3 | 0x42188b | Decode4 | dwOwnerId |
| 4 | 0x421895 | Decode4 | nSkillID |
| 5 | 0x42189f | Decode1 | nSLV |
| 6 | 0x4218aa | Decode2 | phase |
| 7 | 0x4218bd | DecodeBuffer(16) | rcArea (4×int32 absolute LTRB) |
| 8 | 0x4218c4 | **Decode1** | trailing `+0x30` slot |

Field-slot correlation with v61: the object stores at 0x4218f5–0x42191b land in
the same slots as v61's (`[0] dwId, [1] nType, [2] dwOwnerId, [3] nSkillID,
[4] nSLV, [8..11] rcArea, [12] trailing`). Reads #4 is confirmed to be
`nSkillID` by the branch compares against the mist skill ids 130, 131, 2111003
and 4221006 (Smokescreen) — read at 0x421954 (`mov eax, 82h` / `sub ecx, eax`),
0x421961 (`dec ecx`), 0x421968 (`sub ecx, 203598h`) and 0x421970
(`sub ecx, 203233h`; the cumulative subtraction resolves to 4221006). v61
@0x423edc branches on the same four.

**Divergence vs the current codec:** two fields narrow to one byte — `nType`
(Decode4 → Decode1) and the trailing `+0x30` slot (Decode4 → Decode1). Body is
33 bytes, not 39.

**Read order — Removed** (`0x421e7f`): a single `CInPacket::Decode4` @0x421e9a.
**Matches** the current codec exactly (4-byte body).

---

## gms_92 — Outcome **A**

**Walked.** `?OnAffectedAreaCreated@CAffectedAreaPool@@` @0x4392a0 and
`?OnAffectedAreaRemoved@…` @0x4371f0 were already named in the v92 IDB; the
dispatcher was located by `xrefs_to` on both, which resolved to a tail chunk of
`?OnPacket@CField@@UAEXJAAVCInPacket@@@Z` @**0x5406b0** (the documented v92
stage dispatcher). The arm block is at 0x439710–0x439737.

**Arms enumerated from disassembly** (Hex-Rays agreed, but disassembly is the
citation):

- `sub eax, 140h` @0x439714 / `jz loc_43972D` @0x439719 → Created @0x4392a0
- `sub eax, 1` @0x43971b / `jnz locret` @0x43971e → Removed @0x4371f0

Correlation basis: the surrounding already-named arms in the same range
(`CMessageBoxPool::*` 317–319, `CTownPortalPool::*` 322–323,
`COpenGatePool::*` 324–325, `CReactorPool::*` 326–329) bracket the pair
positionally exactly as they do in v95.

**Opcodes.** `SPAWN_MIST = 320 (0x140)`, `REMOVE_MIST = 321 (0x141)`.

**Read order — Created** (`0x4392a0`, disassembly): Decode4 @0x4392dd,
Decode4 @0x4392eb, Decode4 @0x4392f4, Decode4 @0x4392ff, Decode1 @0x439308,
Decode2 @0x439316, DecodeBuffer(16) @0x439327, **Decode4 @0x43932e**,
**Decode4 @0x439339**. Object stores at 0x439387–0x4393be put the 8th read at
`+0x30` (the slot v61 fills with its single trailing Decode4) and the 9th at
`+0x48` (the slot that does not exist before v92).

**Divergence vs the current codec:** none in field order, but the gate was
wrong. The extra leading time word ("tStart" in the Atlas model) was gated on
`MajorVersion() >= 95`; v92 carries it too. Gate lowered to
`IsRegion("GMS") && MajorAtLeast(92)` — additive, affects only majors in
[92, 95), i.e. gms_92 alone. Body is 43 bytes, same as v95.

**Read order — Removed** (`0x4371f0`): a single `CInPacket::Decode4` @0x43722a.
**Matches** the current codec exactly.

---

## gms_12 — Outcome **A**

**Walked.** No v12 session was open; the IDB was opened by binary name
(`GMS_v12_1_DEVM.exe.i64`, per the project's v12 packet-naming record) and
resolved:

- Dispatcher: `?OnPacket@CAffectedAreaPool@@QAEXJAAVCInPacket@@@Z` @**0x4166d0**
  — v12 routes the pool through its own two-arm dispatcher, not inline in
  `CField::OnPacket` @0x47502d.
- Arms enumerated from disassembly at 0x4166d0:
  - `sub eax, 8Fh` @0x4166d4 / `jz` @0x4166d9 → `?OnAffectedAreaCreated@…` @0x4166f5
  - `dec eax` @0x4166db / `jnz` (fallthrough) → `?OnAffectedAreaRemoved@…` @0x416cc4

**Opcodes.** `SPAWN_MIST = 143 (0x8F)`, `REMOVE_MIST = 144 (0x90)`.

**Read order — Created** (`0x4166f5`, disassembly; decode primitives per the
v12 naming record: `Decode4 = sub_416B99`, `Decode2 = sub_416B60`,
`DecodeBuffer = sub_416BD1`):

| # | addr | primitive | field |
|---|---|---|---|
| 1 | 0x416718 | Decode4 | dwId |
| 2 | 0x416722 | **Decode1** | nType |
| 3 | 0x41672c | Decode4 | **nSkillID** |
| 4 | 0x416736 | Decode1 | nSLV |
| 5 | 0x416741 | Decode2 | phase |
| 6 | 0x416754 | DecodeBuffer(16) | rcArea |

Nothing further is decoded (verified by continuing the disassembly to 0x41681e).

Class + payload fingerprint (never opcode number) confirms read #3 is
`nSkillID`, not `dwOwnerId`: at 0x4167c9–0x4167e6 that value is compared against
`0x82` (130), `0x83` (131) and `130 + 1 + 0x203598` (2 111 003) — the first
three of the mist skill ids the v48 @0x421854 and v61 @0x423edc handlers branch
on. v12's chain genuinely stops at three: it has no fourth compare, whereas v48
adds 4 221 006 (Smokescreen) at 0x421970. The
object stores confirm the slot map: `[0x00] dwId, [0x04] nType, [0x08] nSkillID,
[0x0C] nSLV, [0x10] phase-derived, [0x14] rcArea` — there is no owner slot and
no `+0x30`-equivalent trailing slot.

**Divergence vs the current codec:** `dwOwnerId` absent; trailing time word
absent; `nType` one byte. Body is 28 bytes.

**Read order — Removed** (`0x416cc4`): a single Decode4 (`sub_416B99`) @0x416cdf.
**Matches** the current codec exactly.

---

## IDB naming

All six target functions were **already named** with the v95 mangled-symbol
convention in their respective IDBs
(`?OnAffectedAreaCreated@CAffectedAreaPool@@IAEXAAVCInPacket@@@Z` /
`?OnAffectedAreaRemoved@…`, plus `?OnPacket@CAffectedAreaPool@@…` on v48 and
v12). No rename was needed; the earlier "not found" result was a *demangled*
name lookup against a *mangled* symbol table, which is why it is not evidence of
absence.

---

## Conditional work performed

| Version | Outcome | Template wired | Registry | Evidence | Codec gate |
|---|---|---|---|---|---|
| gms_48 | A | `0xCA` / `0xCB` | `docs/packets/registry/gms_v48.yaml` (`provenance: ida-discovered`) + export rows de-`unresolved`ed | `docs/packets/evidence/gms_v48/field.clientbound.FieldAffectedArea{Created,Removed}.yaml` (TIER1-FIXTURE) | `nType` byte, trailing byte |
| gms_92 | A | `0x140` / `0x141` | none exists (provenance = this document) | n/a (no matrix column) | leading time word gate lowered to `MajorAtLeast(92)` |
| gms_12 | A | `0x8F` / `0x90` | none exists (provenance = this document) | n/a (no matrix column) | `nType` byte, no `dwOwnerId`, no trailing word |

All gates are additive and expressed with the `MajorAtLeast` idiom. The Task 1
and Task 2 byte-fixture tests were re-run: every already-verified version
(v61/v72/v79/v83/v84/v87/v95/jms185) produces byte-identical output.

## Semantics of the trailing `+0x30` slot — SETTLED (corrected 2026-08-07)

> **Correction.** The original version of this section, written from the
> gms_48/gms_61/gms_92 handlers alone, called `record[+0x14]` an "absolute
> expiry tick" and left `+0x30` open. Both readings were wrong, and acting on
> the first one shipped a regression (mists rendered for a fraction of a second
> and vanished). The record layout below is read from the **gms_v95 PDB-backed
> IDB**, where `AFFECTEDAREA` is a named 76-byte struct, and re-confirmed
> against the gms_v83 handler.

```
AFFECTEDAREA (76 bytes)
  0x00 dwID        0x04 nType     0x08 dwOwnerID   0x0C nSkillID
  0x10 nSLV        0x14 tStart    0x18 tEnd        0x1C bEnd
  0x20 rcArea(16)  0x30 nElemAttr 0x34 nDamage     0x38 bLeftToDraw
  0x3C nFogState   0x40 apLayer   0x44 posList     0x48 nPhase
```

**`+0x14` is `tStart`, not an expiry.** The arithmetic in the table below is
real — `record[+0x14] = Decode2 * 100 + get_update_time()` — but the field it
writes is the tick at which the mist is first *drawn*, not the tick at which it
dies. `CAffectedAreaPool::Update` gates the draw on it and then clears it:

| version | instructions |
|---|---|
| gms_83 | `0x431214 mov ecx, [eax+14h]` · `0x431220 test/js` (skip while `tCur - tStart < 0`) · `0x431238 call FindAndDraw` · `0x431244 mov [edx+14h], 0` |
| gms_95 | `0x4369b8 mov eax, [esi+14h]` · `0x4369c7` same gate · `0x4369e3 call FindAndDraw` · `0x4369e8 mov [esi+14h], 0` |

So the Decode2 field is a **delay before drawing**, in units of 100 ms. Sending
a mist's duration there hides the mist for its entire lifetime. Send 0.

The store into `+0x14` (unchanged from the original reading, retained as
evidence):

| version | instructions | source operand |
|---|---|---|
| gms_48 | `0x421933 mov eax, [ebp+var_38]` · `0x421939 imul eax, 64h` · `0x42193c add eax, [ebp+var_20]` · `0x42193f mov [ecx+14h], eax` | `var_38` = Decode2 @0x4218aa; `var_20` = the call return stored @0x421874 |
| gms_61 | `0x423fbb mov eax, [ebp+var_3C]` · `0x423fc1 imul eax, 64h` · `0x423fc4 add eax, [ebp+var_20]` · `0x423fc7 mov [ecx+14h], eax` | `var_3C` = Decode2; `var_20` = same call return |
| gms_83 | store @0x431b50 (`v101[5] = a2 + 100*v88`), clamp @0x431b56 | `v88` = Decode2 @0x431ac7; `a2` = `get_update_time()` @0x431a7e |
| gms_92 | `0x43936d imul edi, 64h` · `0x439383 add edi, [esp+84h+var_58]` · `0x4393c4 mov [esi+14h], edi` | `edi` = Decode2 @0x439316; `var_58` = `sub_936E80` return @0x4392cb |

Note the v61 IDB carries an inline correction on that call site: the symbol
rendered as `CWvsContext::SetExclRequestSent` there is mislabeled and is really
`get_update_time()`. The clamp (`mov dword ptr [eax+14h], 1` at v48
0x421945–0x42194a, v61 0x423fcd–0x423fd2, v92 0x4393c7–0x4393cb) only guards
`tStart == 0`, which is the "already drawn" sentinel — it is not a lifetime floor.

**`+0x30` is `nElemAttr`.** It is stored raw, with no arithmetic, on every
version: v48 `0x421928 mov [eax+30h], ecx`, v61 `0x423fb0 mov [eax+30h], ecx`,
v83 `0x431b3b`, v92 `0x4393b9 mov [esi+30h], ecx`, v95 `0x437fd9`. The original
section was right that it is not a time value; the v95 struct names it. Atlas
does not model a mist elemental attribute, so the writer emits 0 (and its low
byte on v48, whose slot is one byte wide).

**Where the mist's real lifetime comes from.** Not this packet. The client
computes `tEnd` (+0x18) from its own WZ skill data —
`tEnd = tStart + 1000 * SKILLENTRY::GetLevelData(...)` (v83 @0x43200f, v95
@0x43822a for the area-buff-item arm) — and on the mob-skill arms (skill 130
@0x4321cb, skill 131 @0x43206d, v83) it never sets `tEnd` at all. For mob mists
the client holds the area until the server sends `AffectedAreaRemoved`, which is
exactly what atlas-maps does when the mist expires. There is therefore no
duration to carry on the wire, and no version of this packet has a field for one.

**The second trailing Decode4 (gms_v92+) is `nPhase`** (+0x48, v95 @0x437fde) —
a separate field from the Decode2 draw delay, despite the similar name. Atlas
does not model it; the writer emits 0.
