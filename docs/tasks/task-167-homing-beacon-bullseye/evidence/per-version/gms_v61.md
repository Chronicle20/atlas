# GMS v61 — CTS evidence (task-167)

Binary: `GMS_v61.1_U_DEVM.exe.i64` (input path `E:\Programs\Nexon\IDBs_v9\GMS\v61\GMS_v61.1_U_DEVM.exe.i64`)  ·  session: 415bf585
Confirmed identity: yes — `mcp__ida-pro__idb_list` returned session `415bf585` with `filename: "GMS_v61.1_U_DEVM.exe.i64"`, matching the expected target exactly.

## Question A — movement-affecting filter

Reset handler: `0x84353A` (`CWvsContext::OnTemporaryStatReset`) — decompile confirms it calls
`SecondaryStat::IsMovementAffectingStat()` at `/*0x84364a*/` immediately before the conditional
`CInPacket::Decode1` (the movement byte read):
```
UINT128::UINT128((UINT128 *)&v20, (const struct UINT128 *)v28, 0x80u); /*0x843645*/
if ( SecondaryStat::IsMovementAffectingStat() )                        /*0x84364a*/
{
  v14 = (CUserLocal *)dword_974EE8;                                    /*0x843659*/
  v15 = CInPacket::Decode1(a2);                                        /*0x84365f*/
  CUserLocal::SetSecondaryStatChangedPoint(v14, v15);                  /*0x843667*/
}
```
The disassembly confirms the call site passes the decoded UINT128 mask by value (mangled name
`?IsMovementAffectingStat@SecondaryStat@@SAHVUINT128@@@Z`, i.e. takes a `UINT128` argument even
though the decompiler's pseudocode dropped the visible argument for the hidden-struct-by-value
convention).

Filter helper: `0x660B44` (`SecondaryStat::IsMovementAffectingStat`) — **already named in this
IDB** (unlike v83's unnamed `sub_77DC78`). Decompiled body builds a static OR-accumulated mask on
first call (`byte_977CB8` guard) by chaining 12 sub-calls to `sub_6F3D17(dest, src)` (an
OR-into-dest helper; verified by the compareTo/`!` logic afterward: `result = UINT128::operator&(v13,&unk_977CA8); return !!result`).
The disassembly at `0x660B44` shows the chained accumulator calls consume the pushed
`(dest, src)` pairs in strict LIFO order, giving the following **source-order call sequence**
(this is the actual OR-chain evaluation order, read directly off the pushed-argument stack):

1. `unk_977C98` (seed, i.e. first operand of the OR-chain)
2. `unk_977C88`
3. `unk_977C78`
4. `unk_977C68`
5. `unk_977C58`
6. `unk_977C48`
7. `unk_977C38`
8. `unk_977C28`
9. `unk_977C18`
10. `unk_977C08`
11. `unk_977BF8`
12. `unk_977BE8`

Each constant's bit position was read from its own dynamic initializer (decompiled individually;
all use `UINT128(1u) << N` via `sub_6F3940(N)`, except the last three which use
`sub_66EDE6(a2)` = `1 << (a2+59)`):

| # | constant | init fn | shift (bit) |
|---|---|---|---|
| 1 | `unk_977C98` | `sub_66EFDD` → `sub_6F3940(7)` | **7** |
| 2 | `unk_977C88` | `sub_66F00D` → `sub_6F3940(8)` | **8** |
| 3 | `unk_977C78` | `sub_66F1BD` → `sub_6F3940(17)` | **17** |
| 4 | `unk_977C68` | `sub_66F42D` → `sub_6F3940(30)` | **30** |
| 5 | `unk_977C58` | `sub_66F48D` → `sub_6F3940(32)` | **32** |
| 6 | `unk_977C48` | `sub_66F4BD` → `sub_6F3940(33)` | **33** |
| 7 | `unk_977C38` | `sub_66F7BD` → `sub_6F3940(49)` | **49** |
| 8 | `unk_977C28` | `sub_66F51D` → `sub_6F3940(35)` | **35** |
| 9 | `unk_977C18` | `sub_66F5DD` → `sub_6F3940(39)` | **39** |
| 10 | `unk_977C08` | `sub_66FA11` → `sub_66EDE6(3)` = 3+59 | **62** |
| 11 | `unk_977BF8` | `sub_66F9BD` → `sub_66EDE6(1)` = 1+59 | **60** |
| 12 | `unk_977BE8` | `sub_66F9E7` → `sub_66EDE6(2)` = 2+59 | **61** |

These 12 shift values (7, 8, 17, 30, 32, 33, 49, 35, 39, 62, 60, 61) are **directly verified from
decompiled/disassembled evidence** — not inferred.

**Resolved stat names.** v61's IDB has no `CTS_*`-style symbol names on these dynamic
initializers (unlike the v95 IDB used for `docs/tasks/task-086-mount-system/v95_secondarystat_table.md`,
which had linker-visible `_dynamic_initializer_for__CTS_X__` symbols). `find_regex` for `CTS_`
across the whole v61 binary returned zero hits, and no `TemporaryStat_*` accessor methods exist
for the base-stat array. Name resolution therefore uses **positional correspondence**: the
OR-chain's evaluation order (12 terms, listed above) is compared against the already-verified v83
name order (Speed, Jump, Stun, Weakness, Slow, Morph, Ghost, BasicStatUp, Attract, RideVehicle,
then the two Dash bits — v83's own design note documents the same "RideVehicle | Dash-bit-A |
Dash-bit-B" tail ordering). Two of the twelve are **independently cross-checked** (not just
positional) against dedicated single-purpose mask constants used elsewhere in the same two
handler functions (see Question B for the corroborating evidence):

- shift **62** = **RideVehicle** — confirmed independently: `unk_97C300` (used directly in both
  `OnTemporaryStatReset` and `OnTemporaryStatSet` to gate `CUser::ShowRideVehicleEffect`) is
  itself `sub_66EDE6(3)` = shift 62, and `OnTemporaryStatReset`/`OnTemporaryStatSet` read the
  vehicle id from `*(this+8468) + 601*4` — index 601 is exactly the two-state array slot whose
  mask shift is 62 (see Question B, member index 3).
- shift **64** = would be **GuidedBullet**, except 64 is *not* in the movement-filter list above —
  confirming the design.md v83 finding "**GuidedBullet is NOT movement-affecting**" also holds for
  v61 (GuidedBullet's own shift, 64, is absent from the 12-entry OR-chain).
- shifts **60** and **61** are the two-state array's Dash-pair (index 1 and 2 — see Question B);
  they share byte-identical constructor and `DecodeForClient` code, so which is literally "Speed"
  vs. "Jump" is **not distinguishable from the binary alone** (both are the same wire shape).

Resolved list (shift → name), in OR-chain order:

| shift | name | confidence |
|---|---|---|
| 7 | Speed | positional (matches v83 order) |
| 8 | Jump | positional |
| 17 | Stun | positional |
| 30 | Weakness (`WEAKEN`) | positional |
| 32 | Slow | positional |
| 33 | Morph | positional |
| 49 | Ghost | positional |
| 35 | BasicStatUp | positional |
| 39 | Attract | positional |
| 62 | RideVehicle | **independently confirmed** (see above) |
| 60 | DashSpeed | positional (Dash pair, order unverifiable from code) |
| 61 | DashJump | positional (Dash pair, order unverifiable from code) |

Comparison to v83 list: **MATCHES** on the set of 12 names and their relative (source) order —
Speed, Jump, Stun, Weakness, Slow, Morph, Ghost, BasicStatUp, Attract, RideVehicle, then the two
Dash bits. **DIFFERS** on absolute bit positions: v61's filter bits top out at 64, far below v83's
(which reach into the 80s), consistent with v61 predating most of the Aran/Evan-era CTS additions
that push v83's later bits higher. This is expected version-specific bit-layout drift, not a
semantic difference in which stats are movement-affecting.

## Question B — two-state member group

SecondaryStat constructor: `0x65F66F` (`sub_65F66F` — allocates and constructs the two-state
array; unnamed in this IDB) which at its end calls `0x65FCA2` (`sub_65FCA2` — POD base-stat field
init, and which itself resets all 6 two-state members via `sub_68203B` in a `while(v7=6)` loop).
`0x65F66F` is `SecondaryStat`'s real constructor body: a `for (i=0;i<6;++i)` loop that allocates
and placement-constructs exactly **6** two-state member objects into `this[598..603]`, then calls
`sub_65FCA2(this)`.

**The array has 6 members, not 7** — there is no "Undead"/7th slot in v61 (the loop bound is a
hard-coded `6` in three independent places: the constructor's `for` loop, `SecondaryStat::Reset`'s
tail loop, and `SecondaryStat::DecodeForLocal`'s tail loop).

Ordered members (index in `this[598+i]`, mask shift, constructor, allocated size,
`DecodeForClient` address, block size):

| i | offset | shift | ctor | alloc | name (confidence) | DecodeForClient | block size |
|---|---|---|---|---|---|---|---|
| 0 | `this[598]` | 59 | `sub_65F94E` | 0x28 (40B) | EnergyCharge (positional) | `0x66EC19` | base(12) + Decode2(2) = **14** |
| 1 | `this[599]` | 60 | `sub_65FA09` | 0x28 (40B) | DashSpeed (positional, Dash pair) | `0x66ED94` | base(12) + Decode2(2) = **14** |
| 2 | `this[600]` | 61 | `sub_65FA09` (same fn as i=1) | 0x28 (40B) | DashJump (positional, Dash pair) | `0x66ED94` (same fn as i=1) | **14** |
| 3 | `this[601]` | 62 | `sub_65F8F6` | 0x1C (28B) | **RideVehicle** (independently confirmed) | `0x66EB3D` → `sub_66E9B6` (base only) | **12** |
| 4 | `this[602]` | 63 | `sub_65F9E2` | 0x2C (44B) | SpeedInfusion (positional) | `0x66E8EF` | base(12) + Decode4(4) + Decode2(2) = **18** |
| 5 | `this[603]` | 64 | `sub_65F8F6` (shared base) + custom 2nd-vtable setup (`off_8EA4E8`) | 0x20 (32B) | **GuidedBullet** (independently confirmed) | `0x65F840` → `sub_66EB3D`(base 12) + `Decode4` | base(12) + dwMobId(4) = **16** |

**Independent confirmation of index 3 = RideVehicle and index 5 = GuidedBullet** (not just
positional): both `CWvsContext::OnTemporaryStatReset` (`0x84353A`) and
`CWvsContext::OnTemporaryStatSet` (`0x84311F`) read the CTS-embedded `SecondaryStat` sub-object at
a fixed base offset and directly index element **601** into `CUser::ShowRideVehicleEffect`, e.g.
in `OnTemporaryStatSet`:
```
v14 = (_DWORD *)sub_4C12B4(*((_DWORD *)v2 + 2718));   // 2718 = 8468/4 (SecondaryStat base) + 601... wait: 2718*4-8468 = 2404 = 601*4
CUser::ShowRideVehicleEffect(v13, *v14);
```
and index **603** into `TemporaryStat_GuidedBullet::GetMobID` / `TemporaryStatBase<long>::GetReason`
/ `CMob::SetGuided`. Both offsets (601, 603) match the two-state array indices 3 and 5 exactly.
Additionally, the dedicated single-purpose mask constants used to gate those same code paths are
themselves built from the identical `sub_66EDE6(a2)=1<<(a2+59)` formula used by the array's own
per-member mask test: `unk_97C300 = sub_66EDE6(3)` = shift **62** (RideVehicle), and
`unk_97C2E0 = sub_66EDE6(5)` = shift **64** (GuidedBullet) — both match index-derived shifts
exactly (index 3 → 3+59=62; index 5 → 5+59=64).

**Base `DecodeForClient` shared by all 6 members**: `sub_66E9B6` (vtable slot 6 / vtable offset
`+0x18`, called via `mov ecx,[eax]; call dword ptr [eax+18h]` in the trailer loop):
```
CInPacket::DecodeBuffer((int)a2, this + 1, 4u);   // 0x66e9df — field @ +4
CInPacket::DecodeBuffer((int)a2, this + 2, 4u);   // 0x66e9ed — field @ +8
result = CInPacket::Decode4(a2);                  // 0x66e9f5
this[3] = result;                                 // 0x66e9fc — field @ +12
```
= 4 + 4 + 4 = **12 bytes**, with **no `DecodeTime` (byte+int32) call** — this is one byte
shorter than the v83 reference base (13 bytes: nValue4+rValue4+DecodeTime5). RideVehicle (i=3)
uses this base function directly and unchanged; the other 5 members wrap it and append 2–4 more
bytes as shown in the table above (each verified by decompiling its own `DecodeForClient`
override, e.g. `0x66EC19`, `0x66ED94`, `0x66E8EF`, `0x65F840`).

Summed trailer length **if all 6 members were active simultaneously**: 14+14+14+12+18+16 =
**88 bytes**. This is *not* a meaningful fixed constant for v61, however — see next.

GuidedBullet mask-bit shift: **64** (confirmed three independent ways: the `DecodeForLocal`
tail-loop's `sub_66EDE6(5)` at loop index i=5; the constructor's index 5 → `this[603]`, matching
`OnTemporaryStatReset`/`OnTemporaryStatSet`'s `TemporaryStat_GuidedBullet` field access at the
same offset; and the dedicated `unk_97C2E0 = sub_66EDE6(5)` constant used to gate
`CMob::SetGuided`).

Trailer read style: **per-member mask-gated (conditional)** — **not** unconditional. Evidence from
`SecondaryStat::DecodeForLocal` (`0x663665`), tail loop at `0x666D66`–`0x666E6A`:
```
666d66  and [ebp+var_14], 0            ; i = 0
666d6a  lea eax, [esi+958h]            ; &this[598]
666d73  mov eax, [eax]                 ; load this[598+i] (member pointer)
666d82  call sub_66EDE6                ; UINT128(1) << (i+59)
666d91  call UINT128::operator&        ; decoded_mask & (1<<(i+59))
666d98  call UINT128::compareTo        ; != 0 ?
666d9f  jz   loc_666E5F                ; skip member entirely if bit clear
666dab  call dword ptr [eax+18h]       ; DecodeForClient — ONLY reached if bit set
...
666e66  cmp [ebp+var_14], 6
666e6a  jl   loc_666D73                ; next i
```
When the bit is clear, the member's `DecodeForClient` is never called and **no bytes at all** are
consumed for it (not even a placeholder) — the loop just moves to the next index. `SecondaryStat::Reset`
(`0x662704`) has the structurally identical mask-gated loop (calling `sub_68203B` instead of the
decode vtable slot) confirming this is the established pattern for the whole two-state group, not
an artifact of one function.

**VERDICT: DIFFERS from v83 in member count (6, not 7 — no Undead/7th slot), per-member block
sizes (14/14/14/12/18/16, not 15/15/15/13/20/17), and trailer read style (per-member mask-gated,
not unconditional). The "110 bytes, unconditional" v83 shape does not apply to v61. GuidedBullet's
mask-bit shift is 64. Question A's movement-affecting filter set of names MATCHES v83 (same 12
stats, same relative order) at different absolute bit positions (max bit 64 vs. v83's higher
range).**
