# task-191 — Movement types, header delta, and move opcodes derived from the GMS clients

Evidence document for Task 1. **Documentation only** — no code, no template edits. Tasks 2, 5, 6 and
8 consume the tables below.

Derived 2026-08-04 by decompiling the GMS v83 / v87 / v92 / v95 and JMS v185 clients over the
IDA-MCP session server. Every value below cites a client function and an instruction address. No
value was carried across from a neighbouring template, from the plan's Appendix A, or from general
MapleStory knowledge.

---

## 1. IDB session map

Sessions were adopted from the running GUI instances with `idb_open {mode: "prefer_gui"}`.
**Session ids rotate** — always re-run `idb_list` and match on `filename` rather than reusing these.

| Version | `filename` | IDB session id (2026-08-04) |
|---|---|---|
| GMS v83 (method control) | `MapleStory_dump.exe.i64` | `41f13e0d` |
| GMS v87 (header control) | `GMSv87_4GB.exe.i64` | `d51ecbd3` |
| GMS v92 | `GMS_v92_1_DEVM.exe.i64` | `acdfccff` |
| GMS v95 | `GMS_v95.0_U_DEVM.exe.i64` | `79906a1e` |
| JMS v185 (header control) | `MapleStory_dump_SCY.exe.i64` | `b6864e54` |

`CMovePath` codec addresses, as returned by
`func_query {queries:[{name_regex:"Encode@CMovePath|Decode@CMovePath|Flush@CMovePath"}]}` — every
address the design cited was confirmed by lookup before any decompile was trusted:

| Version | `CMovePath::Encode` | size | `CMovePath::Decode` | size | `CMovePath::Flush` |
|---|---|---|---|---|---|
| GMS v83 | `0x68a563` | `0x269` | `0x68a33c` | `0x1cb` | `0x68a88d` |
| GMS v87 | `0x6c70fe` | `0x2da` | `0x6c6e86` | `0x214` | `0x6c74a1` |
| GMS v92 | `0x65a260` | `0x552` | `0x65ad60` | `0x31e` | `0x65b5a0` |
| GMS v95 | `0x666e20` | `0x552` | `0x667920` | `0x31e` | `0x668160` |
| JMS v185 | `0x70b6c4` | `0x2b4` | `0x70b3ce` | `0x2b9` | `0x70ba2c` |

v92 and v95 have byte-identical `Encode` (`0x552`) and `Decode` (`0x31e`) body sizes. That is a
hint, not the evidence — the case-for-case comparison in §4 is the evidence.

---

## 2. Method, and the v83 control

**Method.** Decompile `CMovePath::Encode` (the *serverbound* authority — what the client writes and
Atlas must decode) and `CMovePath::Decode` (the *clientbound* authority — what the client reads and
Atlas must encode). Read the `switch (nAttr)` in each and group every `case` by the exact
`Encode2`/`Encode1` (resp. `Decode2`/`Decode1`) sequence it emits. Map each group to a `Type` by the
PRD FR-1.2 field-set table. Requiring the two halves to agree case-for-case is a free consistency
check and is reported for every version below.

**Control.** v83 `CMovePath::Encode` `0x68a563`. Switch dispatched at `0x68a60c`; the element type
byte is written by `COutPacket::Encode1(a2, *v8)` at `0x68a5fb`.

| Group | Arm body (quoted from the decompile) | Cases |
|---|---|---|
| NORMAL | `Encode2(*(v8+1))`@`0x68a61c`, `+2`@`0x68a62a`, `+3`@`0x68a638`, `+4`@`0x68a646`, `+6`@`0x68a654`; `if (*v8 != 15) goto LABEL_10`@`0x68a65c` else `Encode2(*(v8+7))`@`0x68a6a1` | 0, 5, 0xF, 0x11 |
| JUMP | `Encode2(*(v8+3))`@`0x68a66f`, `Encode2(*(v8+4))`@`0x68a729` | 1, 2, 6, 0xC, 0xD, 0x10, 0x12, 0x13, 0x14, 0x16 |
| TELEPORT | `Encode2(*(v8+1))`@`0x68a688`, `+2`@`0x68a696`, `+6`@`0x68a6a1` | 3, 4, 7, 8, 9, 0xB |
| STAT_CHANGE | `Encode1(*(v8+18))`@`0x68a73b`, then `goto LABEL_19` — **skips** the trailing `Encode1(bMoveAction)`@`0x68a6e1` and `Encode2(tElapse)`@`0x68a6ef` | 0xA |
| START_FALL_DOWN | `Encode2(*(v8+3))`@`0x68a710`, `+4`@`0x68a71e`, `+7`@`0x68a729` | 0xE |
| DEFAULT | no arm — falls to `default: break` | 0x15, and anything unlisted |

Decimal: NORMAL {0,5,15,17}; JUMP {1,2,6,12,13,16,18,19,20,22}; TELEPORT {3,4,7,8,9,11};
STAT_CHANGE {10}; START_FALL_DOWN {14}; DEFAULT {21}. Max case `0x16` → 23 entries.

**Compared index-for-index against the committed `template_gms_83_1.json`
`CharacterMoveHandle.options.types`** (23 entries, read programmatically with `python3 -c`, not
eyeballed):

| idx | template `Name` | template `Type` | derived group | ✓ |
|---|---|---|---|---|
| 0 | NORMAL | NORMAL | NORMAL | ✓ |
| 1 | JUMP | JUMP | JUMP | ✓ |
| 2 | IMPACT | JUMP | JUMP | ✓ |
| 3 | IMMEDIATE | TELEPORT | TELEPORT | ✓ |
| 4 | TELEPORT | TELEPORT | TELEPORT | ✓ |
| 5 | HANG_ON_BACK | NORMAL | NORMAL | ✓ |
| 6 | UNKNOWN | JUMP | JUMP | ✓ |
| 7 | ASSAULTER | TELEPORT | TELEPORT | ✓ |
| 8 | ASSASSINATION | TELEPORT | TELEPORT | ✓ |
| 9 | RUSH | TELEPORT | TELEPORT | ✓ |
| 10 | STAT_CHANGE | STAT_CHANGE | STAT_CHANGE | ✓ |
| 11 | SIT_DOWN | TELEPORT | TELEPORT | ✓ |
| 12 | UNKNOWN | JUMP | JUMP | ✓ |
| 13 | UNKNOWN | JUMP | JUMP | ✓ |
| 14 | START_FALL_DOWN | START_FALL_DOWN | START_FALL_DOWN | ✓ |
| 15 | FALL_DOWN | NORMAL | NORMAL (the `nAttr==15` inner branch) | ✓ |
| 16 | START_WINGS | JUMP | JUMP | ✓ |
| 17 | WINGS | NORMAL | NORMAL | ✓ |
| 18 | ARAN_ADJUST | JUMP | JUMP | ✓ |
| 19 | MOB_TOSS | JUMP | JUMP | ✓ |
| 20 | DASH_SLIDE | JUMP | JUMP | ✓ |
| 21 | UNKNOWN | DEFAULT | DEFAULT | ✓ |
| 22 | UNKNOWN | JUMP | JUMP | ✓ |

**Gate: PASS — 23/23 indices reproduce.** The method is validated; the v92/v95 derivations below are
licensed.

---

## 3. v95 element table

Sources: `CMovePath::Encode` `0x666e20` (switch @`0x666f45`, type byte written at `0x666f30`) and
`CMovePath::Decode` `0x667920` (switch @`0x6679ce`, type byte read at `0x6679b8`). The v95 IDB is
PDB-backed, so the arms name their struct fields directly.

**Both halves agree case-for-case.** Arms, quoted:

| Group | `Encode` arm | `Decode` arm | Cases |
|---|---|---|---|
| NORMAL | `Encode2(x)`@`0x666f53`, `y`@`0x666f5f`, `vx`@`0x666f6b`, `vy`@`0x666f77`, `fh`@`0x666f83`; `if (nAttr == 12)`@`0x666f8b` → `Encode2(fhFallStart)`@`0x666f94`; `Encode2(xOffset)`@`0x666fa0`, `Encode2(yOffset)`@`0x666fa9` | `x`@`0x6679de`, `y`@`0x6679e9`, `vx`@`0x6679f4`, `vy`@`0x6679ff`, `fh`@`0x667a03`; `if (nAttr == 12)`@`0x667a17` → `fhFallStart`@`0x667a20`; `xOffset`@`0x667a2d`, `yOffset`@`0x667a36` | 0, 5, 0xC, 0xE, 0x23, 0x24 |
| JUMP | `Encode2(vx)`@`0x666fff`, `Encode2(vy)`@`0x667009` | `vx`@`0x667ae8`, `vy`@`0x667afa` (via `LABEL_16`) | 1, 2, 0xD, 0x10, 0x12, 0x1F, 0x20, 0x21, 0x22 |
| TELEPORT | `x`@`0x667012`, `y`@`0x66701e`, `fh`@`0x667028` | `x`@`0x667b2b`, `y`@`0x667b36`, `fh`@`0x667b3a` | 3, 4, 6, 7, 8, 0xA |
| STAT_CHANGE | `Encode1(bStat)`@`0x667080`, then `goto LABEL_59`@`0x667085` — **skips** `bMoveAction`@`0x6670bf` and `tElapse`@`0x667110` | `Decode1(bStat)`@`0x667bb4`, then `goto LABEL_10`@`0x667bde` — skips `bMoveAction`@`0x667a3a` / `tElapse`@`0x667a4b` | 9 |
| START_FALL_DOWN | `vx`@`0x667031`, `vy`@`0x66703d`, `fhFallStart`@`0x667047` | `vx`@`0x667b73`, `vy`@`0x667b7e`, `fhFallStart`@`0x667b87` | 0xB |
| FLYING_BLOCK | `x`@`0x667053`, `y`@`0x66705f`, `vx`@`0x66706b`, `vy`@`0x666fa9` (via `LABEL_14`) | `x`@`0x667b99`, `y`@`0x667ba2`, then `LABEL_16` `vx`/`vy` | 0x11 |
| DEFAULT | no arm (`default: break`) | explicit no-read arm @`0x667b0d` for `0x14`–`0x1E` (copies the previous `x`/`y`/`vx`/`vy`); `default: break` for `0xF`/`0x13` | 0xF, 0x13, 0x14–0x1E |

Highest case in both halves is `0x24` → **length 37**.

Naming policy applied (design §2.4 — the table was renumbered relative to v83 and the client exposes
no name strings, so only structurally-justified names are assigned; everything else is `UNKNOWN`
rather than invented):

| index | client evidence | `Name` | `Type` |
|---|---|---|---|
| 0 | first NORMAL-group member | `NORMAL` | `NORMAL` |
| 1 | JUMP arm | `UNKNOWN` | `JUMP` |
| 2 | JUMP arm | `UNKNOWN` | `JUMP` |
| 3 | TELEPORT arm | `UNKNOWN` | `TELEPORT` |
| 4 | TELEPORT arm | `UNKNOWN` | `TELEPORT` |
| 5 | NORMAL arm | `UNKNOWN` | `NORMAL` |
| 6 | TELEPORT arm | `UNKNOWN` | `TELEPORT` |
| 7 | TELEPORT arm | `UNKNOWN` | `TELEPORT` |
| 8 | TELEPORT arm | `UNKNOWN` | `TELEPORT` |
| 9 | sole `bStat`-only arm | `STAT_CHANGE` | `STAT_CHANGE` |
| 10 | TELEPORT arm | `UNKNOWN` | `TELEPORT` |
| 11 | sole `vx,vy,fhFallStart` arm | `START_FALL_DOWN` | `START_FALL_DOWN` |
| 12 | NORMAL arm carrying the inner `nAttr == 12` → `fhFallStart` branch (**load-bearing**) | `FALL_DOWN` | `NORMAL` |
| 13 | JUMP arm | `UNKNOWN` | `JUMP` |
| 14 | NORMAL arm | `UNKNOWN` | `NORMAL` |
| 15 | `default` | `UNKNOWN` | `DEFAULT` |
| 16 | JUMP arm | `UNKNOWN` | `JUMP` |
| 17 | sole `x,y,vx,vy` arm | `FLYING_BLOCK` | `FLYING_BLOCK` |
| 18 | JUMP arm | `UNKNOWN` | `JUMP` |
| 19 | `default` | `UNKNOWN` | `DEFAULT` |
| 20–30 | explicit no-read arm | `UNKNOWN` | `DEFAULT` |
| 31 | JUMP arm | `UNKNOWN` | `JUMP` |
| 32 | JUMP arm | `UNKNOWN` | `JUMP` |
| 33 | JUMP arm | `UNKNOWN` | `JUMP` |
| 34 | JUMP arm | `UNKNOWN` | `JUMP` |
| 35 | NORMAL arm | `UNKNOWN` | `NORMAL` |
| 36 | NORMAL arm | `UNKNOWN` | `NORMAL` |

Group totals: NORMAL 6 · JUMP 9 · TELEPORT 6 · STAT_CHANGE 1 · START_FALL_DOWN 1 · FLYING_BLOCK 1 ·
DEFAULT 13 = **37**.

`FALL_DOWN` is the only `Name` that changes wire behaviour (`NormalElement.Decode` reads
`FhFallStart` only when the looked-up `Name` is `FALL_DOWN`), and it lands on index 12, matching the
client's literal `nAttr == 12` test at `0x666f8b`/`0x667a17`.

---

## 4. v92 element table — derived independently

Sources: `CMovePath::Encode` `0x65a260` (switch @`0x65a385`) and `CMovePath::Decode` `0x65ad60`
(switch @`0x65ae0e`). The v92 IDB carries no PDB types for `CMovePath::ELEM`, so the arms were read
by struct offset. Offsets resolve as `+2` x, `+4` y, `+6` vx, `+8` vy, `+10` bMoveAction,
`+12` fh, `+14` fhFallStart, `+16` tElapse, `+18` bStat, `+20` xOffset, `+22` yOffset,
`+24` usRandCnt, `+26` usActualRandCnt — the same field order the v95 PDB names.

| Group | `Encode` arm | `Decode` arm | Cases |
|---|---|---|---|
| NORMAL | `+2`@`0x65a393`, `+4`@`0x65a39f`, `+6`@`0x65a3ab`, `+8`@`0x65a3b7`, `+12`@`0x65a3c3`; `if (*(_BYTE *)v12 == 12)`@`0x65a3cb` → `+14`@`0x65a3d4`; `+20`@`0x65a3e0`, `+22`@`0x65a3e9` | `+2`@`0x65ae1e` … `if (*(_BYTE *)v9 == 12)`@`0x65ae57` → `+14`@`0x65ae60`; `+20`@`0x65ae6d`, `+22`@`0x65ae76` | 0, 5, 0xC, 0xE, 0x23, 0x24 |
| JUMP | `+6`@`0x65a43f`, `+8`@`0x65a449` | `LABEL_16` `+6`@`0x65af28`, `+8`@`0x65af3a` | 1, 2, 0xD, 0x10, 0x12, 0x1F, 0x20, 0x21, 0x22 |
| TELEPORT | `+2`@`0x65a452`, `+4`@`0x65a45e`, `+12`@`0x65a468` | `+2`@`0x65af6b`, `+4`@`0x65af76`, `+12`@`0x65af7a` | 3, 4, 6, 7, 8, 0xA |
| STAT_CHANGE | `Encode1(+18)`@`0x65a4c0`, then `goto LABEL_59`@`0x65a4c5` — skips `bMoveAction` and `tElapse`@`0x65a550` | `Decode1(+18)`@`0x65aff4`, then `goto LABEL_10`@`0x65b01e` — skips `bMoveAction`@`0x65ae7a` / `tElapse`@`0x65ae8b` | 9 |
| START_FALL_DOWN | `+6`@`0x65a471`, `+8`@`0x65a47d`, `+14`@`0x65a487` | `+6`@`0x65afb3`, `+8`@`0x65afbe`, `+14`@`0x65afc7` | 0xB |
| FLYING_BLOCK | `+2`@`0x65a493`, `+4`@`0x65a49f`, `+6`@`0x65a4ab`, `+8`@`0x65a3e9` (`LABEL_14`) | `+2`@`0x65afd9`, `+4`@`0x65afe2`, then `LABEL_16` | 0x11 |
| DEFAULT | no arm | explicit no-read arm for cases 20–30; `default` for 15 and 19 | 0xF, 0x13, 0x14–0x1E |

Highest case `0x24` → **length 37**. Both halves agree case-for-case.

**v92 == v95, index-for-index.** Every group's case list, every field order, the `nAttr == 12`
inner-branch index, and the `bStat`-only arm's skip of the trailing `bMoveAction`/`tElapse` are
identical. The §3 table applies unchanged to v92.

---

## 5. Header delta (v88+ four-field header)

Sequence written at the top of `CMovePath::Encode` and read at the top of `CMovePath::Decode`,
before the element loop.

| Version | `Encode` header | `Decode` header | Fields |
|---|---|---|---|
| GMS v83 | `Encode2(x)`@`0x68a57c`, `Encode2(y)`@`0x68a592`, `Encode1(count)`@`0x68a5c3` | `Decode2`@`0x68a352`, `Decode2`@`0x68a36b`, `Decode1`@`0x68a381` | x, y, count |
| GMS v87 | `Encode2`@`0x6c7118`, `Encode2`@`0x6c712e`, `Encode1`@`0x6c715f` | `Decode2`@`0x6c6e95`, `Decode2`@`0x6c6ead`, `Decode1`@`0x6c6ec6` | x, y, count |
| JMS v185 | `Encode2`@`0x70b6de`, `Encode2`@`0x70b6f4`, `Encode1`@`0x70b725` | `Decode2`@`0x70b3e4`, `Decode2`@`0x70b3fd`, `Decode1`@`0x70b40e` | x, y, count |
| GMS v92 | `Encode2(m_x)`@`0x65a284`, `Encode2(m_y)`@`0x65a29f`, **`Encode2(m_vx)`@`0x65a2ba`**, **`Encode2(m_vy)`@`0x65a2d5`**, `Encode1(count)`@`0x65a306` | `Decode2`@`0x65ad78`, `Decode2`@`0x65ad91`, **`Decode2`@`0x65adb0`**, **`Decode2`@`0x65adbc`**, `Decode1`@`0x65adc7` | x, y, **vx**, **vy**, count |
| GMS v95 | `Encode2(_ZtlSecureTear_m_x)`@`0x666e44`, `Encode2(m_y)`@`0x666e5f`, **`Encode2(m_vx)`@`0x666e7a`**, **`Encode2(m_vy)`@`0x666e95`**, `Encode1(m_lElem._m_uCount)`@`0x666ec6` | `Decode2`@`0x667938`, `Decode2`@`0x667951`, **`Decode2`@`0x667970`**, **`Decode2`@`0x66797c`**, `Decode1`@`0x667987` | x, y, **vx**, **vy**, count |

In v95 the four values come from `_ZtlSecureFuse<short>` over `_ZtlSecureTear_m_x` /
`_m_y` / `_m_vx` / `_m_vy` respectively — the third and fourth fields are demonstrably the velocity
pair, not a repetition of x/y. v92 does the same through the unnamed fuse helper `sub_479AF0`.

**The JMS read is not optional and it came back negative: JMS v185 writes the three-field header,
not the four-field one.** Task 2's gate therefore keeps its region clause —
`IsRegion("GMS") && MajorAtLeast(88)` — and must **not** be relaxed to a bare `MajorAtLeast(88)`.

In the v95 `Decode`, the four header values are also the seed for the DEFAULT/JUMP arms: cases
`0x14`–`0x1E` copy `x`, `y`, `b` (=vx) and `ovy` (=vy) from the header at `0x667b0d`–`0x667b19`, which
is why v88+ needs the velocity pair in the header at all.

---

## 6. Opcodes

### 6.1 Method

`xrefs_to CMovePath::Flush` enumerates every move sender. v95 `Flush` `0x668160` → 6 xrefs
(`more: false`, `xref_count: 6`); v92 `Flush` `0x65b5a0` → 6 xrefs (`more: false`). Each sender's
opcode is the immediate passed to the `COutPacket::COutPacket(long)` constructor.

For the v92 summon block (§6.4) a complete sweep was needed, so the entire
`COutPacket::COutPacket(long)` call-site set was enumerated per IDB (v92 ctor `0x67eb20`, 571 call
sites; v95 ctor `0x68d090`, 597 call sites) and the pushed immediate recovered by reading the 20
bytes preceding each call and decoding the last `push imm8`/`push imm32`. **The recovered map was
validated against 14 independently-decompiled sites (6 v92 + 8 v95) — 14/14 exact.** Ten sites
(5 per IDB) whose immediate the byte scan could not recover were decompiled individually; all ten
resolve to `0x0F` or `0x8D` and none is in the range of interest.

### 6.2 v95 senders (named symbols)

| Role | v95 function | `COutPacket` site | Opcode | vs `template_gms_95_1.json` |
|---|---|---|---|---|
| Mob | `CMob::GenerateMovePath` `0x651100` | `0x651909` | 227 = `0xE3` | matches |
| Npc | `CNpc::GenerateMovePath` `0x671590` | `0x67165b` | 241 = `0xF1` | not routed by Atlas |
| Dragon | `CVecCtrlDragon::EndUpdateActive` `0x996570` | `0x9965b2` | 214 = `0xD6` | not routed by Atlas |
| Pet | `CVecCtrlPet::EndUpdateActive` `0x99f5a0` | `0x99f5e5` | 199 = `0xC7` | matches |
| Summoned | `CVecCtrlSummoned::EndUpdateActive` `0x9a0700` | `0x9a075d` | 207 = `0xCF` | matches |
| **User** | `CVecCtrlUser::EndUpdateActive` `0x9a0d20` | **`0x9a0ee3`** | **44 = `0x2C`** | **the FR-3.1 answer** |

Three of the four Atlas-routed values already match the committed v95 template, which cross-validates
the walk before it is used for the new one. `CVecCtrlUser::EndUpdateActive` writes the eight-field
anti-cheat header before `Flush`: `Encode4(~drInfo[0])`@`0x9a0ef8`, `Encode4(~drInfo[1])`@`0x9a0f06`,
`Encode1(FieldKey)`@`0x9a0f1e`, `Encode4(~drInfo[2])`@`0x9a0f2c`, `Encode4(~drInfo[3])`@`0x9a0f3a`,
`Encode4(Crc)`@`0x9a0f4f`, `Encode4(dwKey)`@`0x9a0f6c`, `Encode4(Crc32)`@`0x9a0f8d`, then
`CMovePath::Flush`@`0x9a0fa3`.

`docs/packets/registry/gms_v95.yaml` already records `MOVE_PLAYER` serverbound opcode 44 — confirmed
correct. Its `fname` is `CUserLocal::OnKey` with `fname_alts: [CVecCtrlUser::EndUpdateActive]`; the
function that actually constructs the packet is `CVecCtrlUser::EndUpdateActive`.

### 6.3 v92 senders

The six v92 `Flush` callers are byte-for-byte size-identical to their v95 counterparts and appear in
the same address order, so the correspondence is not merely positional:

| Role | v92 function | size | v95 counterpart size | `COutPacket` site | Opcode | Structural confirmation |
|---|---|---|---|---|---|---|
| Mob | `sub_6447A0` | `0x10a1` | `0x10a1` | `0x644fa9` | `0xDC` | matches template |
| Npc | `sub_664DC0` | `0x217` | `0x217` | `0x664e8b` | `0xEA` | not routed by Atlas |
| Dragon | `sub_96F190` | `0x95` | `0x95` | `0x96f1d2` | `0xD3` | neither version encodes a prefix |
| **Pet** | `sub_9781A0` | `0xb8` | `0xb8` | `0x9781e5` | **`0xC4`** | `EncodeBuffer(petLockerSN, 8)`@`0x97820c`, mirroring v95 @`0x99f60c` |
| **Summoned** | `sub_9792D0` | `0xc1` | `0xc1` | `0x97932d` | **`0xCC`** | `Encode4(dwSummonedID)`@`0x979345`, mirroring v95 @`0x9a0775` |
| User | `sub_9798F0` | `0x2c8` | `0x2c8` | `0x979ab3` | `0x2E` | same 8-field anti-cheat header, `0x979ac8`–`0x979b5d`, then `Flush`@`0x979b73` |

`template_gms_92_1.json` currently registers `PetMovementHandle` at… nothing — the v92 template has
no pet move handler at all. The value `0xC4` above is the one Task 6 must add.

### 6.4 The rest of the v92 summon block, and who owns `0xC8`

Both clients' summon senders, listed by enclosing-function address, from the complete ctor-immediate
sweep:

| # | v95 function | opcode | v92 function | opcode |
|---|---|---|---|---|
| 1 | `CSummoned::TryDoingHeal` `0x74ad90` | `0xD2` | `sub_72D430` | `0xCF` |
| 2 | `CSummoned::TryDoingGiveBuff` `0x74af50` | `0xD2` | `sub_72D670` | `0xCF` |
| 3 | `CSummoned::TryDoingHealingRobot` `0x74b3e0` | `0xD2` | `sub_72DAB0` | `0xCF` |
| 4 | `CSummoned::TryDoingSummon` `0x74b640` | `0xD2` | `sub_72DD10` | `0xCF` |
| 5 | **`CSummoned::SetDamaged` `0x74b730`** | **`0xD1`** | **`0x72DE00`** (renamed) | **`0xCE`** |
| 6 | `CSummoned::SendRemove` `0x74c170` (size `0x7e`) | `0xD3` | `0x72E690` (size `0x7e`, renamed) | `0xD0` |
| 7 | **`CSummoned::AttackToTargetMob` `0x7501d0`** | **`0xD0`** | **`0x732640`** (renamed) | **`0xCD`** |
| 8 | `CSummoned::TryDoingAttackManual` `0x751240` | `0xD0` | `sub_7332B0` | `0xCD` |
| 9 | `CSummoned::TryDoingTaslaCoilAttack` `0x752780` | `0xD0` | `sub_736DB0` | `0xCD` |
| 10 | — | — | `sub_738490` | `0xCD` |

The v92 block is isomorphic to v95's under a uniform **−3** shift. That shift is independently
anchored by four pairs where the symbol or the size is identical on both sides:

| v95 | opcode | v92 | opcode |
|---|---|---|---|
| `CPet::SendDropPickUpRequest` (size `0x216`) | `0xCA` | `CPet::SendDropPickUpRequest` (size `0x216`, already named in the v92 IDB) | `0xC7` |
| `CWvsContext::SendStatChangeItemUseRequestByPetQ` (`0x17c`) | `0xCB` | `CWvsContext::SendStatChangeItemUseRequestByPetQ` (`0x17d`, already named) | `0xC8` |
| `CPet::SendUpdateExceptionListRequest` (`0x109`) | `0xCC` | `CPet::SendUpdateExceptionListRequest` (`0x109`, already named) | `0xC9` |
| `CPet::ParseCommand` (`0x353`) | `0xC9` | `sub_699230` (`0x353`) | `0xC6` |
| `CSummoned::SendRemove` (`0x7e`) | `0xD3` | `sub_72E690` (`0x7e`) | `0xD0` |

Positional order alone is not evidence, so the two values Task 6 consumes were additionally confirmed
by body comparison:

- **Summon damage — v92 `0xCE`.** v92 `0x72DE00` @`0x72e0ce` writes
  `Encode4(hi(dwSummonedID))`@`0x72e0e6`; then either `Encode1(0xFE)`@`0x72e0f8` + `Encode4`@`0x72e102`,
  or `Encode1(nAttackIdx)`@`0x72e10e` + `Encode4`@`0x72e118` + `Encode4`@`0x72e138` +
  `Encode1(nDir < 0)`@`0x72e14d`; then `CClientSocket::SendPacket`@`0x72e15d`. v95
  `CSummoned::SetDamaged` @`0x74bb6a` emits the identical branch structure at
  `0x74bb82`/`0x74bb94`/`0x74bb9e`/`0x74bbae`/`0x74bbb8`/`0x74bbd8`/`0x74bbed`/`0x74bbfd`.
- **Summon attack — v92 `0xCD`.** v92 `0x732640` @`0x732e15` writes `Encode4(dwSummonedID)`@`0x732e31`,
  `Encode4(~drInfo[0..1])`@`0x732e45`/`0x732e59`, `Encode4(update_time)`@`0x732e6a`,
  `Encode4(~drInfo[2..3])`@`0x732e7e`/`0x732e92`,
  `Encode1(nAction & 0x7F | bLeft << 7)`@`0x732eac`, `Encode4(dwKey)`@`0x732ecf`,
  `Encode4(Crc32)`@`0x732ef6`, `Encode1(1)`@`0x732f03`, four `Encode2`@`0x732f4c`–`0x732fdf`,
  `Encode4`@`0x733004`, then the per-mob loop. v95 `CSummoned::AttackToTargetMob` @`0x750db2` emits
  the same sequence at `0x750dce`–`0x751077`.

`sub_7332B0`, `sub_736DB0` and `sub_738490` are three further confirmed `0xCD` senders in the same
class. They are **not** renamed: v95 has three 2-argument attack senders while v92 has one 2-argument
and two 3-argument ones, so no 1:1 v95 identity is established for them. Their opcode (`0xCD`) is
established by the byte-level sweep and is unaffected.

**Who owns `0xC8` / `0xC9` / `0xCA` in the v92 client (the Step-7 question):**

| Opcode | v92 sender | v95 equivalent |
|---|---|---|
| `0xC8` | `CWvsContext::SendStatChangeItemUseRequestByPetQ` `0x9b3a00`, ctor @`0x9b3abe` — **already named in the v92 IDB** | `0xCB` `PetItemUseHandle` |
| `0xC9` | `CPet::SendUpdateExceptionListRequest` `0x6963a0`, ctor @`0x6963d1` — **already named in the v92 IDB** | `0xCC` `PetItemExcludeHandle` |
| `0xCA` | **nothing.** Zero of the 571 `COutPacket::COutPacket(long)` call sites in the v92 `.text` passes the immediate `0xCA`. `0xCB` is likewise unused. | `0xCD` — also unused in v95 (the same two-slot gap before the summon block) |

So the design's "v95−3 correspondence" hypothesis is **confirmed against the client**, and by named
symbols on both sides rather than by position: `template_gms_92_1.json`'s
`SummonMoveHandle 0xC8` / `SummonAttackHandle 0xC9` / `SummonDamageHandle 0xCA` are misassigned. The
derived values are:

| Handler | current template | **derived** |
|---|---|---|
| `PetMovementHandle` | *absent* | **`0xC4`** |
| `SummonMoveHandle` | `0xC8` | **`0xCC`** |
| `SummonAttackHandle` | `0xC9` | **`0xCD`** |
| `SummonDamageHandle` | `0xCA` | **`0xCE`** |

**No FR-1.6 / design §5.3 escalation.** The escalation condition is "a real, Atlas-routed v92 handler
legitimately colliding at `0xC8`/`0xC9`/`0xCA`". `template_gms_92_1.json` has 46 handlers and
registers **no** `Pet*` handler at any opcode, so nothing Atlas routes contends for those three
slots — they are simply wrong and free to vacate. The destination slots `0xCC`, `0xCD`, `0xCE` are
likewise unoccupied in that template. Task 6 can relocate cleanly.

---

## 7. IDB renames applied

Applied to the v92 IDB (`GMS_v92_1_DEVM.exe.i64`) via the `rename` tool; all 11 returned
`dir: "vibe"` (`summary: {total: 11, ok: 11, failed: 0}`).

| v92 address | old | new (v95 mangled symbol) | basis |
|---|---|---|---|
| `0x6447A0` | `sub_6447A0` | `?GenerateMovePath@CMob@@IAEXJHUTARGETINFO@1@HJJJHH@Z` | `Flush` caller, size-exact `0x10a1` |
| `0x664DC0` | `sub_664DC0` | `?GenerateMovePath@CNpc@@IAEXJJ@Z` | `Flush` caller, size-exact `0x217` |
| `0x96F190` | `sub_96F190` | `?EndUpdateActive@CVecCtrlDragon@@MAEXXZ` | `Flush` caller, size-exact `0x95` |
| `0x9781A0` | `sub_9781A0` | `?EndUpdateActive@CVecCtrlPet@@MAEXXZ` | `Flush` caller, size-exact `0xb8`, `EncodeBuffer(…, 8)` |
| `0x9792D0` | `sub_9792D0` | `?EndUpdateActive@CVecCtrlSummoned@@MAEXXZ` | `Flush` caller, size-exact `0xc1`, `Encode4(dwSummonedID)` |
| `0x9798F0` | `sub_9798F0` | `?EndUpdateActive@CVecCtrlUser@@MAEXXZ` | `Flush` caller, size-exact `0x2c8`, 8-field anti-cheat header |
| `0x697910` | `sub_697910` | `?DoAction@CPet@@AAEXJJVZtl_bstr_t@@HHH@Z` | body match: `EncodeBuffer(petLockerSN,8)`, `Encode1(nType)`, `Encode1(v8 < 9 ? 0 : v8)`, `EncodeStr` |
| `0x699230` | `sub_699230` | `?ParseCommand@CPet@@QAEHV?$ZXString@D@@@Z` | size-exact `0x353` in the pet block |
| `0x72DE00` | `sub_72DE00` | `?SetDamaged@CSummoned@@QAEXJJJPAVCMob@@JJ@Z` | body match (§6.4) |
| `0x72E690` | `sub_72E690` | `?SendRemove@CSummoned@@QAEXXZ` | size-exact `0x7e` |
| `0x732640` | `sub_732640` | `?AttackToTargetMob@CSummoned@@QAEHPAVCMob@@J@Z` | body match (§6.4) |

Not renamed, deliberately: `sub_7332B0`, `sub_736DB0`, `sub_738490` (v92 `0xCD` senders with no
established 1:1 v95 identity — see §6.4). Naming them after a v95 function would be invention.

---

## 8. Unresolved indices

**None.** All 37 indices (0–36) for both v92 and v95 resolve from both codec halves, and the two
halves agree. Both `Encode` switches dispatch through `default` for the unlisted cases, which is a
positive reading (no extra bytes) rather than an absence of evidence — corroborated by v95/v92
`Decode` carrying an *explicit* no-read arm for `0x14`–`0x1E`.

---

## 9. Divergences from the plan's Appendix A

**None.** Appendix A was not consulted until after the derivation above was complete. Compared
group-for-group and name-for-name:

| Group | Appendix A | derived | |
|---|---|---|---|
| `NORMAL` | 0, 5, 12, 14, 35, 36 | 0, 5, 12, 14, 35, 36 | ✓ |
| `JUMP` | 1, 2, 13, 16, 18, 31, 32, 33, 34 | same | ✓ |
| `TELEPORT` | 3, 4, 6, 7, 8, 10 | same | ✓ |
| `STAT_CHANGE` | 9 | 9 | ✓ |
| `START_FALL_DOWN` | 11 | 11 | ✓ |
| `FLYING_BLOCK` | 17 | 17 | ✓ |
| `DEFAULT` | 15, 19, 20–30 | same | ✓ |
| length | 37 | 37 | ✓ |
| named indices | 0 `NORMAL`, 9 `STAT_CHANGE`, 11 `START_FALL_DOWN`, 12 `FALL_DOWN`, 17 `FLYING_BLOCK` | same | ✓ |

Appendix A's literal JSON block may be used verbatim by Tasks 5 and 6 without edit. The opcode
values in the plan's Task 1 tables (v92 pet `0xC4`, summon move `0xCC`, v95 user `0x2C`) also
reproduce; the plan did not pre-state v92 summon attack/damage, which this task derives as `0xCD`
and `0xCE`.

---

## 10. Matrix note (not acted on by this task)

Five movement fixtures currently carry `packet-audit:verify … version=gms_v95` markers (these are
what promote the corresponding `docs/packets/audits/STATUS.md` cells):

- `libs/atlas-packet/character/clientbound/movement_test.go:46` (`ida=0x948a80`)
- `libs/atlas-packet/monster/clientbound/movement_test.go:13` (`ida=0x6521e0`)
- `libs/atlas-packet/monster/serverbound/movement_test.go:19` (`ida=0x651100`)
- `libs/atlas-packet/pet/clientbound/movement_test.go:13` (`ida=0x69fb60`)
- `libs/atlas-packet/pet/serverbound/movement_test.go:12` (`ida=0x99f5a0`)

Those markers pin a header layout the v95 client does not read. `model.Movement.Decode`
(`libs/atlas-packet/model/movement.go:33-40`) reads `StartX`, `StartY`, count — three fields. §5
shows the v95 client writes and reads **five** (x, y, vx, vy, count). The comment block above the
`character/clientbound` marker states outright that the blob is *"byte-identical structure to
v83/v87/v95"*; §4/§5 show it is not — v95 differs both in the header and in the element table
(37 entries vs 23). Those fixtures assert that a byte string equals `model.Movement.Encode`'s own
output rather than pinning client-derived bytes, so they pass regardless — the known
"round-trip fixture ≠ client-validated" failure mode.

Opening a fixture campaign is a PRD §2 non-goal for task-191. The correction is recorded here so a
later matrix pass has the evidence and can demote/re-verify those five cells.

---

## 11. Client-option coupling (`usRandCnt` / `usActualRandCnt`)

Both v92 and v95 read **two extra `int16` per element**, immediately after `tElapse`, when a client
option lookup returns non-zero:

| Version | gate | fields |
|---|---|---|
| v95 `Encode` | `if (CClientOptMan::GetOpt(ms_pInstance, 2u))` @`0x667120` | `usRandCnt`@`0x667173`, `usActualRandCnt`@`0x6671bb` |
| v95 `Decode` | same call @`0x667a57` | `usRandCnt`@`0x667a69`, `usActualRandCnt`@`0x667a72` |
| v92 `Encode` | `if (sub_4A8E80(2))` @`0x65a560` | `+24`@`0x65a5b3`, `+26`@`0x65a5fb` |
| v92 `Decode` | `if (sub_4A8E80(2))` @`0x65ae97` | `+24`@`0x65aea9`, `+26`@`0x65aeb2` |

`CClientOptMan::GetOpt` (v95 `0x4ac700`) is three lines and returns 0 for any absent key:

```
nOpt = 0;
v2 = ZMap<unsigned long,long,unsigned long>::GetAt(&this->m_mOpt, &dwType, &nOpt);
return v2 != 0 ? nOpt : 0;
```

v92's `sub_4A8E80` is structurally the same function (`v3 = 0; v1 = sub_4A8E30(&a1, &v3); return v1 != 0 ? v3 : 0;`).

`m_mOpt` is populated **only** by `CClientOptMan::DecodeOpt` (v95 `0x4acb30`), which clears the map
and then reads a server-sent list:

```
ZMap<...>::RemoveAll(&this->m_mOpt);          /*0x4acb38*/
v4 = CInPacket::Decode2(iPacket);             /*0x4acb48*/   // count
if (v4) do { dwType = Decode4(); val = Decode4(); Insert(...); } while (--v5);
```

Atlas writes a **zero** option count — `libs/atlas-packet/field/clientbound/set_field.go:50`
(`w.WriteShort(0) // decode opt`) and `libs/atlas-packet/field/clientbound/warp_to_map.go:95`. The map
therefore stays empty, `GetOpt(2)` returns 0, and **the two fields are never on the wire.**
`movement.go` correctly omits them.

**Coupling to remember:** if Atlas ever sends a non-empty client-option list that includes type `2`,
`model.Movement`'s element codecs must gain a matching pair of `usRandCnt`/`usActualRandCnt` reads
and writes (both halves — one-sided is a corruption bug), gated the same way as the header.
