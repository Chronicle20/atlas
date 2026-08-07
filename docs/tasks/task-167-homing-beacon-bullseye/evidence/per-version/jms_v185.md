# JMS v185 — CTS evidence (task-167)

Binary: `MapleStory_dump_SCY.exe.i64` · full path `E:\Programs\Nexon\IDBs_v9\JMS\v185\MapleStory_dump_SCY.exe.i64` · session: `b6864e54`

## Binary provenance caveat

`idb_list` was queried and returned all 10 currently-adopted sessions. There is
**exactly one** JMS IDB among them:

- `b6864e54` → `E:\Programs\Nexon\IDBs_v9\JMS\v185\MapleStory_dump_SCY.exe.i64`

No other JMS entry exists in the session list, and none of the other 9
sessions is a JMS build under any name (they are all GMS: v48, v61, v72,
v79, v83, v84, v87, v92, v95 — 6 of which are literally named `*_U_DEVM.exe.i64`
/ `*_U_DEVM.i64`, confirming that naming convention is in active use for
this IDB set, just not for JMS).

**The task plan requires the JMS `*_U_DEVM` build and explicitly excludes
the SMC/retail (`SCY`) dump. That sanctioned `*_U_DEVM` JMS IDB is not
present/discoverable in this environment — only the `SCY` dump is
available.** Per instructions, I proceeded with the analysis on `SCY` (the
only binary available) rather than blocking, but this is flagged for
controller adjudication: **every finding below comes from the SCY dump, not
the sanctioned U_DEVM build, and that substitution has not been verified as
behaviorally equivalent.**

Mitigating fact discovered during analysis: the SCY binary is *not* stripped
— it carries full C++ mangled symbol names (`CWvsContext::OnTemporaryStatReset`,
`SecondaryStat::DecodeForLocal`, `TemporaryStat_GuidedBullet::DecodeForClient`,
template-instantiated `TwoStateTemporaryStat<...>::DecodeForClient`, etc.),
i.e. PDB-quality naming, not a symbol-free retail strip. It also matches
`libs/atlas-packet/model/character_temporary_stat.go`'s pre-existing
IDA-sourced JMS shift comments exactly (see Question B) — that repo comment
predates this session and cites the same 110-base two-state shift, which is
independent corroboration this SCY dump is a legitimate JMS-v185-derived
build, even though it is not provably the specific `*_U_DEVM` artifact the
plan names.

## Question A — movement-affecting filter

**Reset handler: 0xb07628 — resolved to:** `CWvsContext::OnTemporaryStatReset` (confirmed by
the decompiler's own demangled signature: `void __thiscall
CWvsContext::OnTemporaryStatReset(CWvsContext *this, CInPacket *iPacket)`).
This is a plausible, richly-typed reset handler — it decodes a 16-byte mask
(`CInPacket::DecodeBuffer(iPacket, &p, 0x10u)` @0xb0764e), calls
`SecondaryStat::Reset(v3, v19)` @0xb076fd, `CTemporaryStatView::ResetTemporary`
@0xb07779, and `CWvsContext::ValidateStat` @0xb077ac. So the anchor resolves
cleanly on this binary.

The trailing-byte read is:
```c
UINT128::UINT128(&v19, &p, 0x80u);                 /*0xb077be*/
if ( sub_7F76D1(v19) )                              /*0xb077c3*/
{
  ...
  v14 = CInPacket::Decode1(iPacket);                /*0xb0774d region — actually the
                                                        SetSecondaryStatChangedPoint gate;
                                                        the movement-filter-gated byte
                                                        read is the sub_811B78(v19) block
                                                        below*/
}
```
Correction — precise citation of the *movement-filter-gated* read (the one
this task is about) is the `sub_7F76D1` gate:
```c
if ( sub_7F76D1(v19) )                              /*0xb07738*/
{
  v13 = TSingleton<CUserLocal>::ms_pInstance;        /*0xb07747*/
  v14 = CInPacket::Decode1(iPacket);                 /*0xb0774d*/
  CUserLocal::SetSecondaryStatChangedPoint(v13, v14); /*0xb07755*/
}
```
`v19` here is the full 16-byte (128-bit) mask decoded from the packet
(`UINT128::UINT128(&v19, &p, 0x80u)` @0xb07733). `sub_7F76D1` is the mask
helper.

**Filter helper: 0x7f76d1** (`sub_7F76D1`, `BOOL __cdecl sub_7F76D1(UINT128 a1)`).
On first call it lazily builds a combined mask by OR-ing 13 individual
single-bit `UINT128` constants together via `sub_907A5D` (pairwise OR,
verified: `*(v5+v6) = *v5 | *(v5+v4)` — a per-dword OR of the accumulator with
the passed-in constant), then caches the result in `stru_CD9EC8`. It returns
`sub_907BC6(v2) == 0` where `sub_907BC6` is an **is-zero** predicate
(`for(...) if(!*i) ... return 1` when all 4 dwords are zero, else `0`) — so
`sub_7F76D1` returns **true when `a1 & combined_mask` is non-zero**, i.e.
"packet mask intersects the movement-affecting set."

**Constants tested** (quoted from `sub_7F76D1` @0x7f76d1, all fed to
`sub_907A5D`) and their raw bytes (read via `get_bytes`, 16 bytes each,
little-endian):

| addr | raw bytes (nonzero only) | BASIS: raw client bit (byte·8+bitpos) |
|---|---|---|
| `stru_CD9EA8` | byte13=0x01 | 104 |
| `stru_CD9E98` | byte14=0x02 | 113 |
| `stru_CD9E88` | byte15=0x40 | 126 |
| `stru_CD9E78` | byte8=0x01 | 64 |
| `stru_CD9E68` | byte8=0x02 | 65 |
| `stru_CD9E58` | byte10=0x02 | 81 |
| `stru_CD9E48` | byte8=0x08 | 67 |
| `stru_CD9E38` | byte8=0x80 | 71 |
| `stru_CD9E28` | byte2=0x02 | 17 |
| `stru_CD9E18` | byte1=0x80 | 15 |
| `stru_CD9E08` | byte2=0x01 | 16 |
| `stru_CD9DF8` | byte6=0x01 | 48 |
| `stru_CD9DE8` | byte6=0x02 | 49 |

Each constant carries exactly one set bit (verified by reading all 16 bytes
of each — only one byte is nonzero, and that byte is a power of two), so
this is a genuine 13-single-bit union, not a multi-bit mask per entry. Bit
numbering basis = **raw client bit** (memory byte-index×8 + bit-in-byte of
the compile-time `UINT128` constant; x86 little-endian, consistent with
`UINT128::shiftLeft`'s observed semantics in Question B).

**Resolved stat names.** The binary does not name individual stat bits
(no per-bit symbols exist), so names are resolved by cross-referencing the
**atlas registry's JMS shift assignments**
(`libs/atlas-packet/model/character_temporary_stat.go`,
`buildCharacterTemporaryStatRegistry`), whose JMS branch is itself
IDA-sourced (per its own comments, e.g. "JMS adds 28 (two-state at 110)")
and — per Question B below — was independently re-derived from *this same
SCY binary* and matches to the bit exactly for the two-state group (offset
0 between raw-client and atlas-registry bases). Given that corroboration,
the registry's pre-two-state ordering is used here as the ATLAS REGISTRY
BASIS, which for JMS equals the raw client basis 1:1 (both count from
shift/bit 0 at `WeaponAttack`, with JMS's 82→110 post-SoulStone block
counted the same way in both the registry and the binary's two-state
`1<<(i+110)` computation — see Question B):

| bit | name (atlas registry, JMS) | in v83 movement set? |
|---|---|---|
| 15 | INVINCIBLE | no |
| 16 | SOUL_ARROW | no |
| 17 | STUN | **yes** |
| 48 | MESO_UP_BY_ITEM | no |
| 49 | GHOST_MORPH | no (note: v83's "Ghost" ≠ this bit — see below) |
| 64 | WIND_BREAKER_FINAL | no |
| 65 | ELEMENTAL_RESET | no |
| 67 | EVENT_RATE | no |
| 71 | BODY_PRESSURE | no |
| 81 | SOUL_STONE | no |
| 104 | SWALLOW_DEFENSE | no |
| 113 | MONSTER_RIDING (RideVehicle) | **yes** |
| 126 | **not present in atlas's JMS registry** (registry only defines shifts 0–116 for JMS; bit 126 is UNVERIFIED/unmapped) | UNVERIFIED |

**Comparison to v83 list: DIFFERS substantially.** Only 2 of JMS's 13
movement-filter bits resolve to stats that are also in the v83 12-stat
reference list (Speed, Jump, Stun, Weakness, Slow, Morph, Ghost, BasicStatUp,
Attract, RideVehicle, DashSpeed, DashJump): **Stun (bit 17)** and
**MonsterRiding/RideVehicle (bit 113)**. The other 11 v83 stats
(Speed, Jump, Weakness/Weaken, Slow, Morph, Ghost, BasicStatUp, Attract,
DashSpeed, DashJump) are **absent** from JMS's 13-constant filter, and JMS's
other 11 bits resolve to stats with no obvious movement semantics
(WindBreakerFinal, ElementalReset, EventRate, BodyPressure, SoulStone,
SwallowDefense, MesoUpByItem, GhostMorph, Invincible, SoulArrow) plus one
unmapped bit (126). Two of these named constants (`BasicStatUp`, `Attract`)
do not even exist as distinct entries in atlas's current
`character.TemporaryStatType` enum — they are v83-legacy names not carried
into the JMS/GMS-v95-era naming the registry models, consistent with the
task's warning that JMS v185 is a much later branch and should not be
assumed to match v83.

Grounding note: bit→name resolution for bits 0–109 relies on the atlas
registry's pre-existing sequential ordering rather than a fresh
per-bit IDA symbol (the binary has no such symbols); it is corroborated,
not independently re-derived, for that range. The two-state-group range
(110–116, covering bit 113) *is* independently IDA-re-derived in Question B
below and matches the registry exactly.

## Question B — two-state member group

**SecondaryStat constructor: 0x7f571c** (`SecondaryStat::SecondaryStat`).
Constructs an array of 7 elements at `this+4180` (`` `eh vector constructor
iterator'(this+4180, 8u, 7, sub_812474, sub_81245E) ``, stride 8 bytes,
count 7), looping `i=0..6` and allocating/constructing a distinct object
per index. Matched against `SecondaryStat::DecodeForLocal`/`DecodeForRemote`'s
vtable dispatch and the discovered `DecodeForClient` overrides:

| idx | ctor helper | final vtable | `DecodeForClient` (mangled) | resolved name |
|---|---|---|---|---|
| 0 | `sub_7F5A2B` → `sub_812552` override | `off_BEB620` | `sub_812552` (calls `TemporaryStatBase<long>::DecodeForClient` @0x81228d + `Decode2`) | EnergyCharge |
| 1 | `sub_7F5AE6` | `off_BEB6C0` | `TwoStateTemporaryStat<long,not_equal<long,0>,Expire<BaseOnLastUpdatedTime,DynamicTermSet>,...>::DecodeForClient` @0x8126cd | DashSpeed |
| 2 | `sub_7F5AE6` (same) | `off_BEB6C0` | same @0x8126cd | DashJump |
| 3 | `sub_7F59D3` | `off_BEB5E4` | `TwoStateTemporaryStat<long,not_equal<long,0>,NoExpire,...>::DecodeForClient` @0x812418 | MonsterRiding (RideVehicle) |
| 4 | `sub_7F5ABF` | `off_BEB684` | `TwoStateTemporaryStat<long,not_equal<long,0>,Expire<BaseOnCurrentTime,DynamicTermSet>,...>::DecodeForClient` @0x8121c3 | SpeedInfusion |
| 5 | `sub_7F59D3` + vtable override (`*v4=&off_BEB590`) | `off_BEB590` | `TemporaryStat_GuidedBullet::DecodeForClient` @0x7f591d | HomingBeacon (GuidedBullet) |
| 6 | `sub_7F5AE6` (same as 1/2) | `off_BEB6C0` | same @0x8126cd | Undead |

This order — EnergyCharge, DashSpeed, DashJump, MonsterRiding, SpeedInfusion,
HomingBeacon, Undead — matches `twoStateBaseStats()`'s pre-v95 branch in
`libs/atlas-packet/model/character_temporary_stat.go` exactly.

**Block sizes** (bytes consumed from `CInPacket` per `DecodeForClient`,
via decompiled callees):

- `TemporaryStatBase<long>::DecodeForClient` @0x81228d (the shared base every
  member calls first): `DecodeBuffer(...,4)` + `DecodeBuffer(...,4)` +
  `` `anonymous namespace'::DecodeTime `` @0x7f50cc (`Decode1`+`Decode4` = 5
  bytes) = **13 bytes**.
- idx0 EnergyCharge (`sub_812552`): base(13) + `Decode2` (2) = **15 bytes**.
- idx1/2/6 DashSpeed/DashJump/Undead (@0x8126cd): base(13) + `Decode2` (2) = **15 bytes** each.
- idx3 MonsterRiding (@0x812418, `NoExpire`): base only = **13 bytes**.
- idx4 SpeedInfusion (@0x8121c3): base(13) + `DecodeTime` (5) + `Decode2` (2) = **20 bytes**.
- idx5 HomingBeacon/GuidedBullet (@0x7f591d): calls `NoExpire::DecodeForClient`
  (13 bytes, base only, no extra) + `Decode4` (4) = **17 bytes**.

Order + sizes: EnergyCharge 15, DashSpeed 15, DashJump 15, MonsterRiding 13,
SpeedInfusion 20, HomingBeacon 17, Undead 15.

**Summed trailer length: 15+15+15+13+20+17+15 = 110 bytes** (GMS pre-95 ref = 110). **Matches exactly.**

**GuidedBullet mask-bit shift.** `sub_806BE3(a1, a2)` (called both from
`DecodeForLocal`'s loop and `DecodeForRemote`'s loop, with the loop index as
`a2`) computes `1 << (a2 + 110)`
(`UINT128::UINT128(&v5,1u); v3=UINT128::shiftLeft(v2,a2+110); UINT128::UINT128(a1,v3,128u);`).
GuidedBullet/HomingBeacon is index 5 in the 7-member array (0-based) ⇒
**raw client bit shift = 110 + 5 = 115**.

- BASIS: **raw client** — directly computed by the binary (`1<<(index+110)`), IDA-verified in both `DecodeForLocal` (@0x802d8d–0x802e90 loop) and `DecodeForRemote` (@0x805760–0x805797 loop).
- BASIS: **atlas registry** — `character_temporary_stat.go` line ~239 states "JMS=113" for MonsterRiding (index 3 of the group), which is `110+3=113`; HomingBeacon (index 5) is therefore registry-shift **115**, confirmed directly in source at `newAndIncNonDiseased(character.TemporaryStatTypeHomingBeacon)` immediately after the `SpeedInfusion`/`PartyBooster` slot, i.e. shift 115.
- **Offset between bases: 0.** Raw client bit 115 == atlas registry shift 115 for JMS. (For contrast, v83's two-state group base is raw/registry bit 82, also offset 0 on that version — the two clients simply use different absolute bases, 82 vs 110, both internally offset-0 against atlas's per-version registry.)

**Trailer read style: per-member mask-gated**, not unconditional — confirmed
independently in *both* decode paths:

`SecondaryStat::DecodeForLocal` @0x7fcc73, loop @0x802d96–0x802e90 (disassembly,
`SecondaryStat::DecodeForLocal(...)+6123` through `+621D`):
```
loc_802D96:                          ; var_14 = 0..6 loop
  mov ecx, [var_18]                  ; array[i]
  call sub_81247B                    ; -> element ptr
  ...
  call sub_806BE3                    ; 1 << (i+110)   (per Q.B shift derivation)
  ...
  lea ecx, [uFlagTemp]
  call UINT128::operator&            ; decoded_mask & (1<<(i+110))
  mov ecx, eax
  call UINT128::operator_bool
  test al, al
  jz  loc_802E85                     ; SKIP the DecodeForClient call if bit unset
  mov ecx, [arg_4]
  mov eax, [ecx]
  call dword ptr [eax+18h]           ; virtual DecodeForClient(iPacket) — ONLY if bit set
loc_802E85:
  inc [var_14]
  add [var_18], 8
  cmp [var_14], 7
  jl  loc_802D96
```

`SecondaryStat::DecodeForRemote` @0x804dbf, tail loop (fully decompiled,
`v74`/`v75` loop):
```c
v75 = (this + 4184);
do {
  v76 = sub_806BE3(&v80, v74);                 // 1 << (v74+110)
  v77 = UINT128::operator&(&p, &v79, v76);      // decoded_mask & bit
  if ( UINT128::operator bool(v77) )
    (*(**v75 + 24))(*v75, iPacket);             // vtable+0x18 = DecodeForClient, gated
  ++v74;
  v75 += 2;
} while ( v74 < 7 );
```

Both the local (own-character) and remote (foreign-character) decode paths
loop over all 7 two-state members and call each member's virtual
`DecodeForClient` **only if the corresponding mask bit (`1<<(index+110)`) is
set** in the packet's decoded 128-bit mask. This is **per-member
mask-gated**, uniformly across all 7 members — not the "pre-95 clients read
all 7 blocks unconditionally" pattern documented elsewhere in the repo for
v83 (`character_temporary_stat.go` comment near `EncodeMask`). That
unconditional-read comment is sourced from v83 IDA verification specifically;
it does not hold for this JMS v185 (SCY) binary, whose block *sizes and
order* match the v83/pre-95 shape exactly but whose *read mechanism* is
conditional per member, matching the v95-style per-member gating in shape
(though JMS applies it to all 7 members, not just 2 conditional ones as
GMS v95 does). Because atlas's server-side encoder already sets all 7
two-state mask bits unconditionally (per `EncodeMask`'s existing comment),
this does not desync current wire behavior — but it means the JMS client's
own gating logic is architecturally closer to v95's conditional-read model
than to v83's unconditional one, despite matching v83's block layout.

**VERDICT: MATCHES GMS pre-95 (7/110) for order/sizes/shift-base; DIFFERS: read style is per-member mask-gated (not unconditional) for both local and remote decode paths — IDA-verified in both `DecodeForLocal` and `DecodeForRemote` on the SCY binary.**
