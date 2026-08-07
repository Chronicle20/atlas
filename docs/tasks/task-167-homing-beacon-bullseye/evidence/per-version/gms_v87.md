# GMS v87 — CTS evidence (task-167)

Binary: `GMSv87_4GB.exe.i64` · session: `d51ecbd3` · identity confirmed: yes — `mcp__ida-pro__idb_list`
reports session `d51ecbd3` → `input_path: E:\Programs\Nexon\IDBs_v9\GMS\v87\GMSv87_4GB.exe.i64`,
`filename: GMSv87_4GB.exe.i64`. Matches the expected binary exactly.

## Question A — movement-affecting filter

Reset handler: `0xab7dc1` (`CWvsContext::OnTemporaryStatReset`) — already verified/marker-pinned per
task instructions; decompiled fresh here only to locate the filter-helper call, not re-derived.

```c
UINT128::UINT128(&v18, &v22, 0x80u); /*0xab7f00*/
v14 = v24; /*0xab7f05*/
sub_8054D0(0, 0, v18.m_data[0], v18.m_data[1], v18.m_data[2], v18.m_data[3]); /*0xab7f12*/
...
CWvsContext::ValidateStat(v14); /*0xab7f45*/
UINT128::UINT128(&v18, &v22, 0x80u); /*0xab7f57*/
if ( sub_7E46F6(v18.m_data[0]) ) /*0xab7f5c*/
```

The trailing-byte gate itself is a separate test earlier in the same function:

```c
UINT128::UINT128(&v18, &v22, 0x80u); /*0xab7ecc*/
if ( sub_7CC3E2(v18.m_data[0]) ) /*0xab7ed1*/
{
  LOBYTE(v13) = CInPacket::Decode1(a2); /*0xab7ee6*/
  sub_9DDE99(v13); /*0xab7eee*/
}
```

Filter helper: **`0x7cc3e2` (unnamed in this IDB — named `sub_7CC3E2` here, not previously named; the
v83 equivalent per task instructions is `sub_77DC78`).** Note: although the parameter is typed `char a1`
by Hex-Rays, the function's body takes `&a1` and ANDs it against a 16-byte (`UINT128`) constant — this is
the decompiler's imprecise rendering of a by-value 128-bit struct argument (MSVC pushes all 4 dwords on
the stack; the call site prints only the low dword `v18.m_data[0]`). The full 128-bit decoded mask is what
is actually tested, consistent with the 12-constant v83 mechanism.

### Mechanism

`sub_7CC3E2` builds a static OR-accumulated 128-bit mask (dynamic-initializer, guarded by `byte_CA2BD0`)
from a chain of `sub_8DCEB7` calls (confirmed by decompile of `0x8dceb7` to compute `*a2 = *this | *a3`,
a 128-bit OR returning the destination), then tests the caller's mask against it via `UINT128::operator&`
+ `sub_8DD020`(a zero-test), matching the "chained OR then AND-test" shape used across all client versions
audited for this task:

```c
BOOL __cdecl sub_7CC3E2(char a1)
{
  ...
  if ( (byte_CA2BD0 & 1) == 0 ) /*0x7cc3f2*/
  {
    byte_CA2BD0 |= 1u; /*0x7cc435*/
    v1 = sub_8DCEB7(byte_CA2BB0, v28, &unk_CA2BA0); /*0x7cc48d*/
    v2 = sub_8DCEB7(v1, v18, &unk_CA2B90); /*0x7cc494*/
    v3 = sub_8DCEB7(v2, v20, &unk_CA2B80); /*0x7cc49b*/
    v4 = sub_8DCEB7(v3, v22, &unk_CA2B70); /*0x7cc4a2*/
    v5 = sub_8DCEB7(v4, v24, &unk_CA2B60); /*0x7cc4a9*/
    v6 = sub_8DCEB7(v5, v26, dword_CA2B50); /*0x7cc4b0*/
    v7 = sub_8DCEB7(v6, v27, &unk_CA2B40); /*0x7cc4b7*/
    v8 = sub_8DCEB7(v7, v19, &unk_CA2B30); /*0x7cc4be*/
    v9 = sub_8DCEB7(v8, v23, &unk_CA2B20); /*0x7cc4c5*/
    v10 = sub_8DCEB7(v9, v17, &unk_CA2B10); /*0x7cc4cc*/
    v11 = sub_8DCEB7(v10, v25, &unk_CA2B00); /*0x7cc4d3*/
    v12 = sub_8DCEB7(v11, v16, &unk_CA2AF0); /*0x7cc4da*/
    v13 = sub_8DCEB7(v12, v21, &unk_CA2AE0); /*0x7cc4e1*/
    UINT128::UINT128(dword_CA2BC0, v13, 0x80u); /*0x7cc4ec*/
    atexit(nullsub_16); /*0x7cc4f6*/
  }
  v14 = UINT128::operator&(&a1, v28, dword_CA2BC0); /*0x7cc508*/
  return sub_8DD020(v14) == 0; /*0x7cc519*/
}
```

This is **14 source constants** (`byte_CA2BB0` base + 13 chained `sub_8DCEB7` OR-folds), not 12.

### Decompiled constants tested (14 total) — BASIS: raw client bit shift

Each source global is itself a dynamic-initializer that computes `(UINT128)1 << N`. Two construction
families were found, both confirmed by decompile of the shared primitives:

- **`sub_8DC983(1)` → `sub_8DCAE0(v, a2)`** (confirmed: `sub_8DCAE0` shifts a 128-bit value of `1` left by
  `a2` bits — direct family, `a2` IS the raw shift).
- **`sub_7DCED7(dest, a2)`** (confirmed: `sub_7DCED7` calls `sub_8DC983(1)` then `sub_8DCAE0(v, a2+86)` —
  i.e. `1 << (a2+86)`, an "offset family" with base 86).

| Global | Ctor | Construction | Raw shift |
|---|---|---|---|
| `byte_CA2BB0` (base) | `sub_7DD05D`@`0x7dd05d` | `sub_8DCAE0(1, 7)` | **7** |
| `unk_CA2BA0` | `sub_7DD08D`@`0x7dd08d` | `sub_8DCAE0(1, 8)` | **8** |
| `unk_CA2B90` | `sub_7DD23D`@`0x7dd23d` | `sub_8DCAE0(1, 17)` | **17** |
| `unk_CA2B80` | `sub_7DD4AD`@`0x7dd4ad` | `sub_8DCAE0(1, 30)` | **30** |
| `unk_CA2B70` | `sub_7DD50D`@`0x7dd50d` | `sub_8DCAE0(1, 32)` | **32** |
| `unk_CA2B60` | `sub_7DD53D`@`0x7dd53d` | `sub_8DCAE0(1, 33)` | **33** |
| `dword_CA2B50` | `sub_7DE36B`@`0x7de36b` | `sub_8DCAE0(1, 49)` | **49** |
| `unk_CA2B40` | `sub_7DD59D`@`0x7dd59d` | `sub_8DCAE0(1, 35)` | **35** |
| `unk_CA2B30` | `sub_7DD65D`@`0x7dd65d` | `sub_8DCAE0(1, 39)` | **39** |
| `unk_CA2B20` | `sub_7E21C7`@`0x7e21c7` | `sub_7DCED7(_,3)` → `86+3` | **89** |
| `unk_CA2B10` | `sub_7E2173`@`0x7e2173` | `sub_7DCED7(_,1)` → `86+1` | **87** |
| `unk_CA2B00` | `sub_7E219D`@`0x7e219d` | `sub_7DCED7(_,2)` → `86+2` | **88** |
| `unk_CA2AF0` | `sub_7DF1FC`@`0x7df1fc` | `sub_8DCAE0(1, 82)` | **82** |
| `unk_CA2AE0` | `sub_7DF22C`@`0x7df22c` | `sub_8DCAE0(1, 83)` | **83** |

Raw shift set tested: **{7, 8, 17, 30, 32, 33, 35, 39, 49, 82, 83, 87, 88, 89}** — 14 constants.

### Resolved stat names

**Raw shift == atlas-registry shift for v87 (offset 0)** — established as follows. `libs/atlas-packet/model/character_temporary_stat.go`
builds its `CharacterTemporaryStatType` registry with a monotonically-incrementing `shift` counter, one
increment per stat, starting at `TemporaryStatTypeWeaponAttack = 0`. Counting that declaration order
(`character_temporary_stat.go:80-166`) places `TemporaryStatTypeSpeed` at index **7** — this is an
independent, code-only cross-check (not IDA), but it matches the client-verified `byte_CA2BB0` = raw shift
7 exactly (Speed is the only 8-bit-width, `ValueAsByteForeignValueWriter`-tagged stat at that position in
both the client's direct-family base entry and the registry). Continuing the same sequential count through
the registry file for each of the other 8 direct-family shifts reproduces the client's raw values exactly:

| Raw shift | Registry stat (sequential count, `character_temporary_stat.go:80-166`) |
|---|---|
| 7 | Speed |
| 8 | Jump |
| 17 | Stun |
| 30 | Weaken (task's "Weakness") |
| 32 | Slow |
| 33 | Morph |
| 35 | MapleWarrior (task's "BasicStatUp") |
| 39 | Seduce (task's "Attract") |
| 49 | GhostMorph (task's "Ghost") |

These 9 are exactly v83's 12-name list minus RideVehicle/DashSpeed/DashJump — **all 9 match by raw shift
with zero offset.**

The 3 offset-family shifts (87, 88, 89) and the 2 new direct-family shifts (82, 83) resolve against the
same file's version-gated block (`character_temporary_stat.go:180-250`), whose own comment states the
verified fact this pass is re-confirming: *"v87 adds 4 (two-state at 86) ... MonsterRiding/RideVehicle
lands at v83=85, v87=89 ... exactly where each client reads it"* (lines 171-172, 238-239) and gates
`Flying`/`Frozen`/`AssistCharge` on `t.MajorAtLeast(87)` (the `post87` block, lines 180-184) immediately
before the two-state group (lines 240-250: `EnergyCharge, DashSpeed, DashJump, MonsterRiding,
SpeedInfusion, HomingBeacon, Undead` in that literal order). Sequential counting from SoulStone (the last
pre-post87 entry) gives: `Flying=82, Frozen=83, AssistCharge=84, MirrorImage=85, EnergyCharge=86,
DashSpeed=87, DashJump=88, MonsterRiding=89, SpeedInfusion=90, HomingBeacon=91, Undead=92` — this matches
raw 82/83/87/88/89 exactly, and **89=MonsterRiding/RideVehicle is independently confirmed from this same
IDB, not just from the registry** — see the corroboration below.

**RideVehicle (raw 89) — independently confirmed inside `OnTemporaryStatReset` itself, not just by
registry cross-reference:**

```c
v4 = UINT128::operator&(&v22, v19, &unk_CA82D8); /*0xab7dfe*/
if ( UINT128::operator bool(v4) ) /*0xab7e05*/
{
  v5 = TSingleton<CUserLocal>::ms_pInstance; /*0xab7e14*/
  v6 = sub_76A987(*(v3 + 854)); /*0xab7e1a*/
  CUser::ShowRideVehicleEffect(v5, *v6); /*0xab7e23*/
}
```

`dword_CA82D8`'s own dynamic initializer (`sub_AB4771`@`0xab4771`): `sub_7DCED7(v2, 3)` → raw shift
`86+3 = 89` — the identical offset-family construction independently verified as movement-filter entry
#`unk_CA2B20`. Two independently-built globals encoding the same shift, one of them directly gated to
`ShowRideVehicleEffect` (the canonical "you are now riding a vehicle" client visual), is conclusive.

**Flying (raw 82) and Frozen (raw 83) — new in v87's filter, absent from v83's 12.** Both are members of
the "direct family" (not the two-state offset family), confirmed by their own dynamic initializers
(`sub_7DF1FC`@`0x7df1fc` → shift 82; `sub_7DF22C`@`0x7df22c` → shift 83) shown above. These raw shifts
correspond to the registry's `Flying`/`Frozen` slots, both gated `t.MajorAtLeast(87)` in the Atlas source
(`character_temporary_stat.go:180-184`) — i.e. new post-SoulStone stats that exist only from v87 onward.
Flying (a flight-movement mode) and Frozen (an immobilizing status) are both plausibly movement-affecting
by name; no v87-client string table exists to confirm the names bit-for-bit beyond the registry
cross-reference, but the count (14 = 12 + 2) and the fact both are the *only* two new direct-family
entries is unambiguous from the IDA disassembly alone, independent of the registry.

**DashSpeed (raw 87) / DashJump (raw 88) — resolved by elimination + registry order, not independently
bit-named.** With RideVehicle confirmed at arg3=89, the remaining two offset-family entries (arg1=87,
arg2=88) are, by registry declaration order (`EnergyCharge, DashSpeed, DashJump, MonsterRiding` —
`character_temporary_stat.go:240-243`), DashSpeed and DashJump respectively. **UNVERIFIED at the
individual-bit level**: no dedicated single-purpose mask (analogous to `dword_CA82D8` for RideVehicle)
was found in this IDB to independently pin which of {87, 88} is DashSpeed vs. DashJump — this mirrors the
same open point in the v79 evidence file for this task.

### Comparison to v83 list

**DIFFERS: v87's filter tests 14 constants, not v83's 12.** All 12 of v83's reference stats (Speed, Jump,
Stun, Weakness/Weaken, Slow, Morph, Ghost/GhostMorph, BasicStatUp/MapleWarrior, Attract/Seduce,
RideVehicle, DashSpeed, DashJump) are present and match by raw shift with **zero offset** to the atlas
registry. v87 additionally tests **Flying (raw 82) and Frozen (raw 83)** — two stats introduced at v87
(`t.MajorAtLeast(87)` gate in the Atlas registry) that did not exist in v83's client at all. This is a
genuine client-behavior difference, not a naming/offset artifact: v87's `sub_7CC3E2` literally has 2 more
`sub_8DCEB7` OR-folds in its accumulator chain than the 12-constant shape described for v83.

## Question B — two-state member group

SecondaryStat constructor: **UNVERIFIED — not found.** `func_query` for `.*SecondaryStat.*` in this IDB
returns only `Reset` (`0x7d089e`), `DecodeForLocal` (`0x7d1ef5`), `DecodeForRemote` (`0x7d8533`),
`CheckByTime` (`0x7d8e94`), `IsRidingTamedMob`, `IsRidingSkillVehicle`, and two unrelated
free-function matches (`CQuestMan::CheckStartDemand`, `get_weapon_mastery`) — no
`SecondaryStat::SecondaryStat` symbol. `list_globals` for `*GuidedBullet*` / `*TemporaryStat*` and
`search_structs` for `SecondaryStat` / `TemporaryStat` all returned empty (no vtable/RTTI/struct symbols
survived in this build). `CWvsContext::CWvsContext` (`0xa96caa`, decompiled in full) was checked as a
candidate host for an inlined member-constructor call, but its explicit field-initialization writes stop
well short of the byte offset where the SecondaryStat's 7-member sub-object array lives (member array base
`this+848` dwords relative to the SecondaryStat object, which itself sits at `CWvsContext_this+2129`
dwords — i.e. far beyond `CWvsContext::CWvsContext`'s own initialization range) — the sub-objects'
vtables are set by code this pass could not locate.

### Two-state group: 7 members, found via the mask-gated decode loop (not a constructor)

`DecodeForRemote` (`0x7d8533`) ends with a genuine loop (not unrolled) over the 7-member array:

```c
v70 = 0; /*0x7d8e3c*/
v71 = this + 848; /*0x7d8e3e*/
do
{
  v72 = sub_7DCED7(v76, v70); /*0x7d8e49*/
  v73 = UINT128::operator&(&v77, v75, v72); /*0x7d8e58*/
  if ( UINT128::operator bool(v73) ) /*0x7d8e5f*/
    (*(**v71 + 24))(*v71, iPacket); /*0x7d8e6d*/
  ++v70; /*0x7d8e70*/
  v71 += 2; /*0x7d8e71*/
} while ( v70 < 7 ); /*0x7d8e77*/
```

`sub_7DCED7(v76, v70)` computes `1 << (v70+86)` (the same offset-family primitive confirmed in Question A),
so member index `v70` (0..6) maps to raw shift `86..92` — **raw == registry shift, offset 0**, same basis
established in Question A. `v71 += 2` (member stride = 2 dwords / 8 bytes) starting at `this+848`.

### Ordered members (name, index → shift, evidence)

| Idx | Raw/registry shift | Name (Atlas registry order, `character_temporary_stat.go:240-250`) | Evidence |
|---|---|---|---|
| 0 | 86 | EnergyCharge | by position only (registry order) |
| 1 | 87 | DashSpeed | by position only (registry order) |
| 2 | 88 | DashJump | by position only (registry order) |
| 3 | **89** | **MonsterRiding / RideVehicle** | **CONFIRMED** — see below |
| 4 | 90 | SpeedInfusion | by position only (registry order) |
| 5 | **91** | **HomingBeacon / GuidedBullet** | **CONFIRMED** — see below |
| 6 | 92 | Undead | by position only (registry order) |

**Idx 3 (raw/registry 89) = RideVehicle — CONFIRMED by address correspondence.** Member-array offset for
index 3 is `848 + 2*3 = 854` dwords. `OnTemporaryStatReset` reads exactly `v3 + 854` (where `v3` is the
`SecondaryStat` base) gated by `dword_CA82D8` (raw shift 89, per Question A) immediately before
`CUser::ShowRideVehicleEffect` — quoted above. Index arithmetic, mask shift, and the client-visible
"riding a vehicle" effect all agree on member 3 = RideVehicle.

**Idx 5 (raw/registry 91) = HomingBeacon / GuidedBullet — CONFIRMED by address correspondence.**
Member-array offset for index 5 is `848 + 2*5 = 858` dwords — exactly the offset read in
`OnTemporaryStatReset`:

```c
v7 = UINT128::operator&(&v22, v19, &unk_CA82B8); /*0xab7e34*/
if ( UINT128::operator bool(v7) ) /*0xab7e3b*/
{
  if ( (*(**(v3 + 858) + 4))(*(v3 + 858)) ) /*0xab7e52*/
  {
    v8 = *(v3 + 858); /*0xab7e59*/
    if ( v8 ) /*0xab7e5d*/
    {
      v18.m_data[3] = sub_69DB9E(v8); /*0xab7e6c*/
      v9 = sub_6AD88E(v8); /*0xab7e6f*/
      sub_6B54C8(*v9, v18.m_data[3]); /*0xab7e78*/
    }
  }
}
```

`dword_CA82B8`'s own dynamic initializer (`sub_AB479B`@`0xab479b`): `sub_7DCED7(v2, 5)` → raw shift
`86+5 = 91`. The 3-call chain (`sub_69DB9E` / `sub_6AD88E` / `sub_6B54C8`, taking the `v3+858` object) is
structurally the same "IsActivated → GetMobID/GetReason → ResetGuidedMob" shape independently documented
for GuidedBullet in the v79 evidence for this task. Index arithmetic (`848+2*5=858`), the mask shift (91),
and the reset-time mob-reset call chain all agree on member 5 = HomingBeacon/GuidedBullet.

**GuidedBullet member's CTS mask-bit shift: raw 91 — BASIS: raw client bit shift** (the literal `N` in
`(UINT128)1 << N`, confirmed twice independently: once via `sub_7DCED7(v76,5)` inside the `DecodeForRemote`
trailer loop, once via `dword_CA82B8`'s own initializer `sub_7DCED7(v2,5)`).

**Idx 0, 1, 2, 4, 6 (EnergyCharge, DashSpeed, DashJump, SpeedInfusion, Undead) — INFERRED by position,
not independently named in this IDB.** No per-member mask constant analogous to `dword_CA82D8` /
`dword_CA82B8` was found for these 5 slots; their names are taken from the Atlas registry's declared
sequential order at the same 7-slot span (`character_temporary_stat.go:240-250`), which the two confirmed
slots (3=RideVehicle, 5=GuidedBullet) match exactly. **UNVERIFIED as direct v87 symbol/string evidence** —
flagged as inference from cross-referenced source, not an independent IDA finding.

### Raw-vs-registry offset

**0 (no offset) for v87.** Established in Question A (Speed raw-7 = registry-7 by sequential count) and
re-confirmed here (RideVehicle raw-89 = registry-89 per the Atlas source's own comment "v87=89"; GuidedBullet
raw-91 = registry-91 by the same sequential count through `character_temporary_stat.go:240-249`). Unlike
the v79 evidence for this task (which needed a `raw+9` conversion because v79 lacks the 4 post-SoulStone
`post87`-gated stats), v87's registry construction already includes the same `post87` stats the client has,
so raw and registry numbering coincide exactly.

### Block sizes — UNVERIFIED

Both `DecodeForLocal` and `DecodeForRemote` dispatch each member's decode through a **virtual call**
(vtable slot `+0x18`, i.e. `DecodeForClient`):

```c
(*(**v71 + 24))(*v71, iPacket); /*0x7d8e6d, DecodeForRemote*/
```

Without a `SecondaryStat` constructor or any vtable/RTTI symbol in this IDB (searched via `func_query`,
`list_globals`, and `search_structs` — all empty; `CWvsContext::CWvsContext` decompiled and found not to
reach the relevant sub-object range), the 7 concrete `DecodeForClient` implementations could not be
statically resolved from the virtual-call site, so **per-member block sizes (expected 15/15/15/13/20/17/15
by analogy with v83/v95) could not be measured in this pass.**
**UNVERIFIED: blocker = no `SecondaryStat` constructor/vtable symbol → cannot resolve the polymorphic
`DecodeForClient` targets from the indirect call alone.**
**Summed trailer length: UNVERIFIED (not measured) — cannot be compared to the 110-byte reference.**

### Trailer read style: **PER-MEMBER MASK-GATED — DIFFERS from "unconditional"**

`DecodeForRemote`'s loop (quoted in full above) explicitly tests `UINT128(1) << (86+idx)` against the
decoded flag and skips the virtual `DecodeForClient` call — and therefore skips that member's packet bytes
entirely — when the bit is not set. This is genuinely mask-gated, the same shape independently found for
v79 and v95 in this task's other evidence files, and is **not** the "always read all 7 blocks
unconditionally" shape the task's leading hypothesis (v83 reference) describes.

**VERDICT: Question A DIFFERS from v83 — 14 constants tested, not 12 (v83's 12 all present at raw shift ==
registry shift, offset 0; v87 additionally tests Flying@82 and Frozen@83, both new v87 stats). RideVehicle
independently confirmed at raw/registry shift 89; DashSpeed/DashJump (87/88) resolved only by
elimination+registry order, individual-bit UNVERIFIED. Question B: member count (7) and read style
(per-member mask-gated, matching v79/v95, not v83's unconditional shape) are directly evidenced; member
order/names match the Atlas registry with RideVehicle (idx 3, shift 89) and GuidedBullet (idx 5, shift 91)
independently confirmed by address correspondence + dedicated mask constants; the other 5 slots are
positional inference from the registry, not independent IDB evidence. Raw-vs-registry offset = 0 for v87
(confirmed via Speed, RideVehicle, and GuidedBullet all landing on their registry-documented shift with no
adjustment). **UNVERIFIED: per-member block sizes and the resulting trailer-length sum** — no
`SecondaryStat` constructor/vtable symbol exists in this IDB to resolve the polymorphic `DecodeForClient`
call targets.**
