# Movement-affecting filter — per version (task-167)

This document transcribes, for every in-scope version, the client's
`IsMovementAffectingStat`-equivalent function: its address, the constants it
tests (with the shift **basis** stated — raw client bit vs. atlas-registry
shift), the resolved stat names where resolvable, the mapping onto atlas's
`TemporaryStatType`, and an explicit verdict on whether that version's set
matches v83's 12-stat reference list.

Source files: `evidence/per-version/gms_v61.md` (+`gms_v61_recheck.md`),
`gms_v72.md`, `gms_v79.md`, `gms_v84.md`, `gms_v87.md`, `gms_v92.md`,
`jms_v185.md`; v83/v95 transcribed from `design.md` §2.3/§2.4 (design-phase
verified, not re-derived by this campaign).

Where a per-bit stat NAME could not be resolved from symbols in a given IDB,
this is stated explicitly per version — an inferred/positional name is never
presented as resolved.

---

## v61

- **Filter function**: `0x660B44` (`SecondaryStat::IsMovementAffectingStat`,
  already named in this IDB). Called from
  `CWvsContext::OnTemporaryStatReset` @`0x84353A`.
- **Constants tested**: 12, chained via `sub_6F3D17`-style OR-accumulator,
  each `UINT128(1) << N` via `sub_6F3940(N)` or `sub_66EDE6(a2) = 1<<(a2+59)`.
  BASIS: **raw client bit** — shifts {7, 8, 17, 30, 32, 33, 35, 39, 49, 60,
  61, 62}.
- **Resolved names**: Speed(7), Jump(8), Stun(17), Weakness(30), Slow(32),
  Morph(33), Ghost(49), BasicStatUp(35), Attract(39) — resolved
  **positionally** (matched against v83's already-verified name order, not
  independently symbol-named in this IDB). RideVehicle(62) is
  **independently confirmed** (shared single-purpose mask constant
  `unk_97C300` also gates `CUser::ShowRideVehicleEffect`, and the same
  constant indexes two-state array slot 3). DashSpeed(60)/DashJump(61) are
  positional only — the pair shares byte-identical code, so which raw shift
  is Speed vs. Jump is not distinguishable from the binary. GuidedBullet
  itself (shift 64) is confirmed **absent** from this 12-entry list.
- **Atlas `TemporaryStatType` mapping**: Speed, Jump, Stun, Weaken, Slow,
  Morph, GhostMorph, MapleWarrior, Seduce, MonsterRiding, DashSpeed/DashJump —
  the same 12 types v83 maps, at raw shifts 23 lower than the atlas-registry
  basis (registry = raw + 23 for this cluster; see
  `two-state-group-per-version.md` §v61 for the full offset derivation).
- **Matches v83's 12-stat list?** **YES** — same 12 names, same relative
  order; only the absolute raw bit positions differ (max bit 64 vs. v83's
  higher range), which is expected version-specific bit-layout drift, not a
  semantic divergence.

## v72

- **Filter function**: `0x6c87b6` (`SecondaryStat::IsMovementAffectingStat`,
  already named). Called from `CWvsContext::OnTemporaryStatReset` @`0x918f3c`.
- **Constants tested**: 12, OR-chained via `SecondaryStat__TwoStateBitMask`
  (two-state family, `1<<(idx+67)`) and `sub_7998DE` (direct family,
  `1<<shift`). BASIS: **raw client bit** — shifts {7, 8, 17, 30, 32, 33, 35,
  39, 49, 68, 69, 70}.
- **Resolved names**: Speed(7), Jump(8), Stun(17) — high confidence
  (foundational CTS slots present in every GMS era, matching v72's own
  extraction). DashSpeed(68)/DashJump(69)/RideVehicle(70) — confirmed by two
  independent methods: (1) `TemporaryStat_GuidedBullet`'s pointer offset in
  `OnTemporaryStatReset` lands on two-state array slot 5, and (2) the
  per-member block-size order (Question B) exactly matches v83's named
  order, placing RideVehicle at the third two-state-family slot (70).
  **Shifts 30, 32, 33, 35, 39, 49 — stat NAME UNVERIFIED.** No
  `CTS_*`/`dynamic_initializer_for_*` symbols exist in this IDB. A naive
  same-index lookup against the v83-derived atlas registry was
  spot-checked and **falsified**: registry index 35 = `MapleWarrior`
  (expects 0 extra decode bytes), but v72's bit-35 site decodes
  `Decode2()+Decode4()` (6 bytes) — not the NoOp shape. Index-for-index name
  lookup against the v83-tuned registry is therefore unsafe for v72.
- **Atlas `TemporaryStatType` mapping**: 9 main-sequence + DashSpeed +
  DashJump + MonsterRiding (RideVehicle) confirmed by count/family/pointer
  evidence; the 6 middle main-sequence names are UNVERIFIED (see above), so
  they cannot be positively bound to specific `TemporaryStatType` constants
  from this IDB alone.
- **Matches v83's 12-stat list?** **Structurally YES (count/split
  9+3 matches exactly), but 6 of the 9 main-sequence NAMES are UNVERIFIED** —
  cannot confirm or deny they are precisely Weakness/Slow/Morph/Ghost/
  BasicStatUp/Attract at the individual-bit level.

## v79

- **Filter function**: `0x6f852f` (`SecondaryStat::IsMovementAffectingStat`,
  already named). Called from `CWvsContext::OnTemporaryStatReset` @`0x96ab32`.
- **Constants tested**: 12, OR-chained via `sub_7DE63D` (accumulator) with
  members from a "direct family" (`sub_7DE266(1,shift)`) and a
  `sub_7099E5(dest,a2) = 1<<(a2+73)` family. BASIS: **raw client bit** —
  shifts {7, 8, 17, 30, 32, 33, 35, 39, 49, 74, 75, 76}.
- **Resolved names**: RideVehicle(76) **CONFIRMED** — the identical
  `sub_7099E5(dest,3)` construction independently builds the mask gating
  `CUser::ShowRideVehicleEffect` in `OnTemporaryStatReset`. Speed(7)
  **supported by cross-reference** to the plan's already-verified v83/v95
  "Speed is registry shift 7" fact. **Shifts 8, 17, 30, 32, 33, 35, 39, 49 —
  stat NAME UNVERIFIED** (no `TEMPORARY_STAT` enum/string table in this IDB;
  count and family match v83's remaining 8 names but individual bit↔name
  assignment could not be resolved). **Shifts 74, 75 — UNVERIFIED which is
  DashSpeed vs. DashJump** (adjacent triple with RideVehicle(76), matched by
  count/family only).
- **Atlas `TemporaryStatType` mapping**: RideVehicle(MonsterRiding) and
  Speed positively bound; the remaining 10 constants are bound only by
  count/structure to the v83 12-name set, not by individual-bit symbol
  evidence.
- **Matches v83's 12-stat list?** **Matches by count and structure (12
  constants, same 9+3 split); RideVehicle and Speed positions independently
  confirmed; remaining 10 individual name↔shift assignments UNVERIFIED** — no
  behavioral DIFFERS found, but full name-for-name verification is not
  possible from this IDB's symbols.

## v83 (transcribed from design.md §2.3 — design-phase verified)

- **Filter function**: unnamed `sub_77DC78` (`MapleStory_dump.exe`, v83_Me
  IDB). Called from both `CWvsContext::OnTemporaryStatSet`/`OnTemporaryStatReset`
  trailing-byte gates (design.md §2.3).
- **Constants tested**: 12, "chained comparisons" shape. BASIS not stated as
  raw-vs-registry explicitly in design.md, but design.md's own registry
  (`libs/atlas-packet/model/character_temporary_stat.go`) is anchored to
  v83's client shifts directly — i.e. for v83, raw client basis and
  atlas-registry basis coincide by construction.
- **Resolved names** (design.md §2.3, direct quote): Speed, Jump, Stun,
  Weakness, Slow, Morph, Ghost, BasicStatUp, Attract, RideVehicle
  (`0x0020…`), and the two Dash bits (`0x0010…`, `0x0008…`) — 12 names, all
  resolved. **GuidedBullet is explicitly confirmed NOT movement-affecting.**
- **Atlas `TemporaryStatType` mapping**: this is the reference list every
  other version in this document is compared against — Speed, Jump, Stun,
  Weaken, Slow, Morph, GhostMorph, MapleWarrior, Seduce, MonsterRiding,
  DashSpeed, DashJump.
- **Matches v83's 12-stat list?** Reference version — trivially yes.

## v84

- **Filter function**: unnamed `sub_7a07e7` (v84 equivalent of v83's
  `sub_77DC78`, not renamed in this IDB). Called from
  `CWvsContext::OnTemporaryStatReset` @`0xa6bb24`.
- **Constants tested**: **14**, not 12 — confirmed by careful disassembly
  (the decompiler's printed call order omits an implicit-`ecx` first
  operand, `mov ecx, offset unk_C49638` @`0x7a088d`, which undercounts to 13
  if only the pseudocode is trusted). BASIS: **raw client bit** — shifts
  {7, 8, 17, 30, 32, 33, 35, 39, 49, 82, 83, 85, 86, 87}.
- **Resolved names**: the 9 low-range shifts {7,8,17,30,32,33,35,39,49}
  exactly match v79's raw shift set (registry offset 0 in that range) —
  Speed + 8 more names by count/positional correspondence, **individual
  bit↔name UNVERIFIED** (same blocker as v79: no `TEMPORARY_STAT` symbol
  table). RideVehicle(87) **CONFIRMED** — a dedicated single-purpose mask
  (`dword_C4EC20`, same `sub_7B0D46(_,3)` construction) independently gates
  a `ShowRideVehicleEffect`-pattern call in `OnTemporaryStatReset`.
  DashSpeed/DashJump(85/86) — UNVERIFIED which is which, matched only by
  family/adjacency to the confirmed RideVehicle slot. **Raw shifts 82, 83 —
  NEW relative to v79/v83, semantic name UNVERIFIED.** No v79/v83 analog
  exists for these two bits, and no dedicated `OnTemporaryStatReset`
  side-effect block references either individually. Flagged as a genuine
  new-stat addition, not a renumbering artifact.
- **Raw-vs-registry basis**: derived as `registry = raw − 2` for the
  shifted/family range (from two independent anchors: RideVehicle raw87→
  registry85, GuidedBullet raw89→registry87 in Question B) and offset 0 for
  the low/direct range (Speed raw7 = registry7). The offset is
  **non-uniform across the bit range** — consistent with 2 new stat types
  having been inserted into the enum between v83 and v84, immediately below
  the two-state-family block.
- **Atlas `TemporaryStatType` mapping**: the original 12 v83 types are
  present and accounted for (RideVehicle confirmed; the 9 direct-family
  names by count/position). The 2 new raw-82/83 constants have **no
  confirmed mapping** onto any existing `TemporaryStatType` — cross-version
  corroboration (see `two-state-group-per-version.md`) suggests they land
  at the same raw slots v87 independently resolves as Flying/Frozen, but
  v84's own evidence does not itself name them.
- **Matches v83's 12-stat list?** **NO — DIFFERS.** v84 tests 14 constants:
  the original 12 (with 9 names UNVERIFIED individually) plus 2 new
  constants (raw 82, 83) absent from v83 entirely. This is recorded as a
  genuine behavioral difference, not an artifact.

## v87

- **Filter function**: unnamed `sub_7cc3e2` (not previously named). Called
  from `CWvsContext::OnTemporaryStatReset` @`0xab7dc1`.
- **Constants tested**: **14** — a direct family (`sub_8DCAE0(1,a2)`,
  `a2` = raw shift) and an offset family (`sub_7DCED7(dest,a2) =
  1<<(a2+86)`). BASIS: **raw client bit** — shifts {7, 8, 17, 30, 32, 33,
  35, 39, 49, 82, 83, 87, 88, 89}.
- **Resolved names**: **raw shift == atlas-registry shift for v87 (offset
  0)**, cross-checked by sequential-counting the atlas registry's
  declaration order (`character_temporary_stat.go:80-166`) and matching it
  bit-for-bit against the client's own values. Speed(7), Jump(8), Stun(17),
  Weaken(30), Slow(32), Morph(33), MapleWarrior(35), Seduce(39),
  GhostMorph(49) — **all 9 resolved by registry cross-reference matching the
  raw client bits exactly** (this is the first version in the campaign
  where all 9 direct-family names are pinned, via the registry's own
  sequential-count method rather than pure position-matching against v83).
  RideVehicle(89) **independently CONFIRMED inside the IDB itself** — a
  dedicated mask (`dword_CA82D8`, same `sub_7DCED7(_,3)` construction)
  gates `CUser::ShowRideVehicleEffect` directly in `OnTemporaryStatReset`.
  Flying(82) and Frozen(83) — new in v87, resolved via the atlas registry's
  `MajorAtLeast(87)`-gated block, but **not independently bit-named inside
  the IDB** (no v87 string table). DashSpeed(87)/DashJump(88) — resolved
  only by elimination + registry declaration order once RideVehicle(89) is
  pinned; **UNVERIFIED at the individual-bit level** inside this IDB.
- **Atlas `TemporaryStatType` mapping**: Speed, Jump, Stun, Weaken, Slow,
  Morph, MapleWarrior, Seduce, GhostMorph, DashSpeed, DashJump,
  MonsterRiding, Flying, Frozen.
- **Matches v83's 12-stat list?** **NO — DIFFERS.** All 12 of v83's stats
  are present (at raw shift == registry shift, offset 0), but v87
  additionally tests **Flying(82)** and **Frozen(83)** — two stats that did
  not exist in v83's client at all (`MajorAtLeast(87)`-gated in the atlas
  registry). Genuine client-behavior difference: `sub_7cc3e2` literally has
  2 more OR-folds than the 12-constant v83 shape.

## v92

- **Filter function**: `sub_705080` @`0x705080`. Called from
  `CWvsContext::OnTemporaryStatReset` @`0x9c7800`.
- **Constants tested**: **13** OR-folded via `sub_7F5010`. BASIS: **raw
  client bit** — shifts {8, 17, 30, 32, 33, 35, 39, 49, 82, 83, 116, 117,
  118}.
- **Resolved names**: **UNVERIFIED for essentially all 13 bits.** This
  binary carries no per-bit textual/RTTI label for the plain scalar `long`
  stats. Notably, 3 of the 13 bits (116, 117, 118) fall inside the
  two-state member's confirmed raw range (115–121, see Question B / the
  companion doc) — meaning at least 3 of these "movement-affecting" bits
  are actually two-state/pointer-dispatched members, not plain scalars —
  but this evidence file could not conclusively bind any of {116,117,118}
  to a specific named member (e.g. is 118 = raw-shift 120's GuidedBullet
  slot as later resolved by `gms_v92_sizes.md`? Not cross-checked in the
  filter pass itself — GuidedBullet's confirmed raw shift is **120**, which
  is *not* in this 13-bit filter list at all, meaning GuidedBullet is
  correctly excluded from the movement filter on v92, consistent with every
  other version). The remaining 10 bits (8,17,30,32,33,35,39,49,82,83) have
  no name resolution in this pass.
- **Atlas `TemporaryStatType` mapping**: cannot be established from this
  evidence file — no stat names were resolved.
- **Matches v83's 12-stat list?** **NO — DIFFERS by count** (13 vs. v83's
  12), and by raw bit values (8,17,30,32,33,35,39,49,82,83,116,117,118 does
  not look like a simple "v83 ∪ 1 stat" pattern). Without name resolution,
  it cannot be stated which named stats were added or dropped relative to
  v83 — flagged explicitly as UNVERIFIED exact stat-name membership, not
  inferred.

## JMS v185

- **Filter function**: `sub_7f76d1` @`0x7f76d1`. Called from
  `CWvsContext::OnTemporaryStatReset` @`0xb07628`. **Provenance caveat: see
  the dedicated subsection below.**
- **Constants tested**: **13**, each a genuine single-bit `UINT128`
  constant (verified — every constant's 16 raw bytes have exactly one
  nonzero byte, itself a power of two). BASIS: **raw client bit**
  (byte-index×8 + bit-in-byte, little-endian) — shifts {15, 16, 17, 48, 49,
  64, 65, 67, 71, 81, 104, 113, 126}.
- **Resolved names** (via the atlas registry's JMS branch, itself
  independently re-derived and matched exactly for the two-state range in
  Question B — see companion doc): 15=Invincible, 16=SoulArrow,
  **17=Stun**, 48=MesoUpByItem, 49=GhostMorph, 64=WindBreakerFinal,
  65=ElementalReset, 67=EventRate, 71=BodyPressure, 81=SoulStone,
  104=SwallowDefense, **113=MonsterRiding (RideVehicle)**. Bit 126 —
  **UNVERIFIED/unmapped** (the atlas registry's JMS branch only defines
  shifts 0–116; 126 has no corresponding entry).
- **Atlas `TemporaryStatType` mapping**: TemporaryStatTypeInvincible,
  TemporaryStatTypeSoulArrow, TemporaryStatTypeStun,
  TemporaryStatTypeMesoUpByItem, TemporaryStatTypeGhostMorph,
  TemporaryStatTypeWindBreakerFinal, TemporaryStatTypeElementalReset,
  TemporaryStatTypeEventRate, TemporaryStatTypeBodyPressure,
  TemporaryStatTypeSoulStone, TemporaryStatTypeSwallowDefense,
  TemporaryStatTypeMonsterRiding; bit 126 has no mapping.
- **Matches v83's 12-stat list?** **NO — DIFFERS substantially.** Only 2 of
  JMS's 13 bits (Stun, MonsterRiding) are also in v83's 12-stat reference
  list. The other 11 v83 stats (Speed, Jump, Weakness, Slow, Morph, Ghost,
  BasicStatUp, Attract, DashSpeed, DashJump — note Ghost/GhostMorph *is* a
  JMS bit but at a different semantic slot than v83's usage) are absent
  from JMS's filter, and JMS's other 11 bits are stats with no obvious
  movement semantics. This is expected: JMS v185 is a much later branch and
  its movement-affecting set has diverged substantially from v83's.

### JMS provenance caveat

- **What was read**: `MapleStory_dump_SCY.exe.i64` (session `b6864e54`),
  the only JMS IDB present in this environment (`idb_list` returned exactly
  one JMS entry among 10 adopted sessions; the other 9 are all GMS builds).
- **What the plan asked for**: a JMS `*_U_DEVM` build (matching the naming
  convention used for 6 of the 9 GMS IDBs in the same set), explicitly
  excluding the SMC/retail (`SCY`) dump.
- **Disposition**: no JMS `*_U_DEVM` IDB is discoverable in this
  environment. Per instructions, analysis proceeded on the `SCY` dump
  (the only binary available) rather than blocking; the owner has
  **accepted** this substitution with the caveat recorded here — it has not
  been independently proven behaviorally equivalent to a `*_U_DEVM` build.
- **Mitigating evidence** (from `jms_v185.md`):
  - The SCY binary is **not stripped** — it carries full mangled C++
    symbol names (`CWvsContext::OnTemporaryStatReset`,
    `SecondaryStat::DecodeForLocal`, `TemporaryStat_GuidedBullet::DecodeForClient`,
    template-instantiated `TwoStateTemporaryStat<...>::DecodeForClient`), i.e.
    PDB-quality naming, not a symbol-free retail strip.
  - The pinned `0xb07628` anchor resolved cleanly to
    `CWvsContext::OnTemporaryStatReset` by the decompiler's own demangled
    signature.
  - Its shift math (`1 << (index+110)` for the two-state group) matches a
    pre-existing IDA-sourced repo comment in
    `libs/atlas-packet/model/character_temporary_stat.go` ("JMS adds 28
    (two-state at 110)") that predates this session.
  - Its two-state trailer result (110 bytes, order EnergyCharge/DashSpeed/
    DashJump/MonsterRiding/SpeedInfusion/HomingBeacon/Undead, sizes
    15/15/15/13/20/17/15) matches v72/v83/v87/v92 exactly.

## v95 (transcribed from design.md §2.4 — design-phase verified)

- **Filter function**: `SecondaryStat::IsMovementAffectingStat` @`0x7208C0`
  (`GMS_v95.0_U_DEVM.exe`).
- **Constants tested**: 15, all resolved by name (this is the one IDB in
  the whole campaign with full `CTS_*` dynamic-initializer symbol names).
  BASIS: names are given directly by symbol, not shift-derived in
  design.md.
- **Resolved names** (design.md §2.4, direct quote): Speed, Jump, Stun,
  Weakness, Slow, Morph, Ghost, BasicStatUp, Attract, RideVehicle,
  Dash_Speed, Dash_Jump, Flying, Frozen, YellowAura.
- **Atlas `TemporaryStatType` mapping**: the same 12 v83 types plus Flying,
  Frozen, YellowAura.
- **Matches v83's 12-stat list?** **NO — DIFFERS.** v83's 12 are a strict
  subset; v95 adds 3 more (Flying, Frozen, YellowAura). Note the partial
  cross-version corroboration: v87 independently resolves Flying/Frozen at
  its own raw shifts 82/83, and v84's two UNVERIFIED raw-82/83 constants sit
  at the identical raw slots — see `two-state-group-per-version.md` for the
  full discussion of that corroboration (it concerns the movement filter's
  new members dating to v84, even though v84's own names remain
  unresolved).
