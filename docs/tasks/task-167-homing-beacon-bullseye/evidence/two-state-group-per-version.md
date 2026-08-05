# Two-state member group — per version (task-167)

This document transcribes, for every in-scope version, the CTS two-state
member group: member count, ordered members with per-member block size
where measured, the summed trailer length, GuidedBullet's slot index and
mask-bit shift (with basis stated), and the trailer read style. Each
section closes with an explicit verdict.

Source files: `evidence/per-version/gms_v61.md` (+`gms_v61_recheck.md`),
`gms_v72.md`, `gms_v79.md` (+`gms_v79_sizes.md`), `gms_v84.md`
(+`gms_v84_sizes.md`), `gms_v87.md` (+`gms_v87_sizes.md`), `gms_v92.md`
(+`gms_v92_sizes.md`), `jms_v185.md`; v83/v95 transcribed from `design.md`
§2.3/§2.4.

**Epistemic-state legend, used throughout:** **MEASURED** = a number read
directly from the binary this campaign. **INFERRED** = not measured on that
version, but supported by neighbouring measured versions (with the support
stated). **UNVERIFIED** = could not be established, with the specific
blocker named. These three states are never blurred in this document.

---

## v61 — the genuine outlier

- **Member count**: **6 (MEASURED)** — triple cross-checked: the
  constructor `0x65F66F`'s `for(i=0;i<6;++i)` allocation loop,
  `SecondaryStat::DecodeForLocal`'s tail loop (`0x666D66`-`0x666E6A`,
  `cmp ..,6`), and `SecondaryStat::DecodeForRemote`'s tail loop
  (`0x667C5F`, `while(v51<6)`) all independently hard-code 6. **There is no
  7th ("Undead") slot in v61** — a real structural difference from every
  other version in this campaign, not a measurement gap.
- **Ordered members with block sizes (all MEASURED — every vtable was
  statically resolvable via a literal `mov [obj], offset off_X` in the
  constructor, unlike every UNVERIFIED-block-size version below)**:

  | i | name | DecodeForClient | block size |
  |---|---|---|---|
  | 0 | EnergyCharge (positional) | `0x66EC19` | **14** |
  | 1 | DashSpeed (positional, Dash pair) | `0x66ED94` | **14** |
  | 2 | DashJump (positional, Dash pair) | `0x66ED94` (same fn) | **14** |
  | 3 | **RideVehicle** (independently confirmed) | `0x66EB3D`→base | **12** |
  | 4 | SpeedInfusion (positional) | `0x66E8EF` | **18** |
  | 5 | **GuidedBullet** (independently confirmed) | `0x65F840` | **16** |

  Shared base `sub_66E9B6` = 4+4+4 = 12 bytes (no `DecodeTime` call — one
  byte shorter than the pre-95 reference's 13-byte base).
- **Summed trailer length**: **88 bytes (14+14+14+12+18+16), MEASURED** —
  if all 6 members were active simultaneously. Not directly comparable to
  the 110-byte pre-95 reference because v61 has only 6 members, not 7.
- **GuidedBullet slot index and mask-bit shift**: slot **5** (0-based, last
  of 6). Raw shift **64 (MEASURED)** — triple-confirmed: loop position
  (i=5), the `sub_66EDE6(5)` mask-constructor formula, and the dedicated
  single-purpose constant `unk_97C2E0` (= `sub_66EDE6(5)`) that gates
  `CMob::SetGuided`/`ResetGuidedMob` directly in `OnTemporaryStatReset`/`Set`.
- **Reconciliation with the repo's pinned v61 fixture — the single most
  confusing point in this campaign.** The repo's existing v61 CTS fixture
  pins a **16-byte mask** and places GuidedBullet at **atlas-registry shift
  87** — identical to v79/v83. This initially looks like it contradicts the
  raw-64 finding above, but it does not: **raw 64 = registry 87 (offset
  +23)**. The independent recheck (`gms_v61_recheck.md`) derived this
  offset directly from `libs/atlas-packet/model/character_temporary_stat.go`'s
  `buildCharacterTemporaryStatRegistry` (GMS<87 branch: EnergyCharge=82,
  DashSpeed=83, DashJump=84, MonsterRiding=85, SpeedInfusion=86,
  HomingBeacon=**87**) and cross-checked it against all 6 two-state members
  (every one is registry = raw + 23, internally self-consistent). So: the
  fixture's "16-byte mask, GuidedBullet at registry-87" description and this
  campaign's "raw shift 64" finding describe the **same fact at two
  different bases**, not a conflict. Caveat carried over from the recheck:
  the +23 offset is verified *within* the two-state cluster (all 6 members
  agree) but was **not** independently re-derived from the binary for the
  ~82 ordinary flag bits that precede the cluster — that portion of the
  offset is sourced from registry code, not IDA-cross-checked bit-by-bit.
- **Trailer read style**: **per-member mask-gated (MEASURED)**, not
  unconditional. `SecondaryStat::DecodeForLocal` (`0x663665`,
  `0x666D66`-`0x666E6A`) and `SecondaryStat::Reset` (`0x662704`) both show
  an explicit `UINT128 & (1<<(i+59))` test with a `jz`-skip immediately
  before the vtable `DecodeForClient` call — when the bit is clear, zero
  bytes are consumed for that member (see the read-style note at the end of
  this document for why this is benign for our encoder).

**VERDICT: v61 DIFFERS from the pre-95 7-member/110-byte shape in member
count (6, no Undead slot), per-member sizes (14/14/14/12/18/16 = 88, not
15/15/15/13/20/17/15 = 110), and trailer read style (mask-gated, not
unconditional). GuidedBullet raw shift 64 = registry shift 87 — identical
to v79/v83's registry position, once the +23 raw-to-registry offset is
applied; this is a basis reconciliation, not a contradiction. The movement
filter's 12-stat set MATCHES v83 (see `movement-filter.md`).**

## v72

- **Member count**: **7 (MEASURED)** — `eh vector constructor iterator`
  over 7 elements, stride 8, at offset 2680 (`0x6c70e9`), matching both
  decode loops' array base.
- **Ordered members with block sizes (all MEASURED — vtables read from each
  ctor's final `*this = &off_X` fixup)**:

  | # | member | block size |
  |---|---|---|
  | 0 | EnergyCharge | **15** |
  | 1 | DashSpeed | **15** |
  | 2 | DashJump | **15** |
  | 3 | RideVehicle/MonsterRiding | **13** |
  | 4 | SpeedInfusion | **20** |
  | 5 | GuidedBullet/HomingBeacon | **17** |
  | 6 | Undead | **15** |

  Shared base `SecondaryStat__TwoStateBase__DecodeForClient` (`0x6d87d4`) =
  4+4+5 = 13 bytes, matching the pre-95 reference base exactly.
- **Summed trailer length**: **110 bytes (15+15+15+13+20+17+15), MEASURED
  — MATCHES the pre-95/v83 reference exactly.**
- **GuidedBullet slot index and mask-bit shift**: slot **5**. Raw shift
  **72 (`67+5`, MEASURED)** — confirmed by pointer arithmetic
  (`TemporaryStat_GuidedBullet` accessed at `v3+2724 = 2680+8*5+4` in
  `OnTemporaryStatReset`) and by the block-size-order match against v83's
  named 15/15/15/13/20/17/15 sequence. Registry basis: **87** — v72's own
  movement-filter evidence (Question A) states the two-state group starts
  at raw bit 67 "vs v83's registry-documented 83/84/85 (group starts at bit
  82)", i.e. offset **+15** for this cluster; applying the same +15 to
  GuidedBullet's raw shift 72 gives registry shift **87**, consistent with
  RideVehicle's raw70→registry85 under the identical +15 offset.
- **Trailer read style**: **per-member mask-gated (MEASURED)** — DIFFERS
  from the "unconditional" v83 baseline description. Both
  `SecondaryStat::DecodeForLocal` (`0x6cb87b`) and `DecodeForRemote`
  (`0x6cfe78`) implement the trailer as a genuine `for(i=0;i<7;i++)` loop
  computing `1<<(i+67)` every iteration and explicitly gating the virtual
  `DecodeForClient` call on the bit test (disassembly-confirmed `jz` skip
  in both functions).

**VERDICT: v72 MATCHES v83's 7-member/110-byte shape and member order
exactly (same 15/15/15/13/20/17/15 sizes). DIFFERS in trailer read style
(per-member mask-gated in both DecodeForLocal and DecodeForRemote, not
unconditional). GuidedBullet is at slot 5, raw shift 72 → registry shift 87
(same registry position as v79/v83/v61).**

## v79 — block sizes UNVERIFIED, not verified

- **Member count**: **7 (MEASURED)** — the 7 two-state mask constants
  (`sub_7099E5(dest,a2)` for `a2=0..6` → raw shifts 73-79) were found via
  `xrefs_to(sub_7099E5)`, and both decode loops iterate `while(v56<7)`.
- **Ordered members with block sizes**: **UNVERIFIED — genuine blocker, not
  a gap in effort.** This IDB has **no `SecondaryStat::SecondaryStat`
  constructor symbol** (contrast v95, which has one at `0x72F190`), no
  RTTI, no vtable symbols, and no local struct type for `SecondaryStat` —
  so Hex-Rays never propagated a concrete type onto `this` in
  `DecodeForLocal`/`DecodeForRemote`, and the polymorphic
  `DecodeForClient` targets (called through `vtable+0x18`) cannot be
  statically resolved from the virtual-call site alone. Two independent
  agents (the initial pass and the dedicated `gms_v79_sizes.md` follow-up)
  each exhausted a comparable list of approaches — name/symbol search,
  bounded immediate-value search across the whole binary for each of the 7
  slot byte-offsets (2900/2908/2916/2924/2932/2940/2948), `operator new`
  text search inside the `SecondaryStat` translation unit, and a search for
  an unconditional full-reset function that might reveal allocation — with
  **zero** vtable/`DecodeForClient` addresses recovered for any of the 7
  slots.
  - **One exception**: slot 5's *class identity* (not its block size) was
    resolved via a **non-virtual** call — `sub_706287` (a guided-bullet
    damage-bonus helper) directly calls
    `TemporaryStat_GuidedBullet::GetMobID` on the pointer at offset 2940
    (=slot 5), confirming by class identity (not by measuring
    `DecodeForClient`'s byte count) that **slot 5 = GuidedBullet**.
  - **INFERRED (not measured)**: by analogy with v72 (110, MEASURED), v87
    (110, MEASURED), and v92 (110, MEASURED) — three chronologically
    bracketing versions with the identical 7-member shape — v79's trailer
    is plausibly also 15/15/15/13/20/17/15 = 110 bytes. This is stated as
    an inference supported by neighbouring measured versions, **not** as a
    verified v79 fact.
- **Summed trailer length**: **UNVERIFIED (not measured)** — cannot be
  compared to the 110-byte reference from this version's own evidence.
  INFERRED 110 by the bracketing argument above.
- **GuidedBullet slot index and mask-bit shift**: slot **5 — MEASURED by
  class identity** (non-virtual `GetMobID` call, see above; also
  independently corroborated by the mask-test block in
  `OnTemporaryStatReset`, which gates `TemporaryStat_GuidedBullet`-specific
  calls on the identical `sub_7099E5(dest,5)` construction). Raw shift
  **78 (`73+5`), MEASURED**. Registry basis: **87** (raw+9, derived from
  two independently-confirmed anchors: RideVehicle raw76→registry85 and
  GuidedBullet raw78→registry87, both matching the plan's pre-existing
  v83 registry-shift facts).
- **Trailer read style**: **per-member mask-gated (MEASURED)** — both
  `DecodeForLocal` (`0x6fbcba`, disasm `0x700050`-`0x70014a`) and
  `DecodeForRemote` (`0x701539`, decompile `0x701c41`-`0x701c7c`) show the
  bit-test-then-skip pattern gating the virtual `DecodeForClient` call.

**VERDICT: v79's member count (7) and GuidedBullet's slot (5, raw shift 78
→ registry 87) are MEASURED and MATCH v72/v83/v95's slot positions. Trailer
read style is MEASURED as per-member mask-gated. Per-member block sizes and
the 110-byte total are UNVERIFIED — the blocker is a missing
`SecondaryStat` constructor symbol (no RTTI, no vtable symbols) that two
independent passes could not work around; the 110-byte total is INFERRED by
bracketing (v72/v87/v92 all measure 110 with the identical 7-member shape)
but not itself measured on v79.**

## v84 — block sizes UNVERIFIED, not verified

- **Member count**: **7 (MEASURED)** — both decode loops (`sub_7AC409`
  local, `sub_7A5D2B` remote) iterate `while(idx<7)`, stride 8 bytes.
- **Ordered members with block sizes**: **UNVERIFIED — genuine blocker,
  identical in kind to v79's.** No `SecondaryStat`/`CTemporaryStat`
  member-array constructor symbol exists in this IDB either. The dedicated
  `gms_v84_sizes.md` follow-up additionally determined the member array's
  true absolute base (`CUserRemote+0x2C24+0xCEC`, found via
  `CUserRemote::OnResetTemporaryStat`'s `lea ecx,[esi+2C24h]`) and ruled
  out a specific false-lead constructor (`sub_9045FF`, which turned out to
  operate on an unrelated COM/`VARIANTARG` object at different absolute
  offsets) — but still could not locate the true allocation site for any
  of the 7 slots, despite checking the local/remote decode functions, the
  reset/clear function, the destructor, the avatar-modify handler, RTTI
  strings, and every nearby moderately-sized function in the same
  source-file address cluster.
  - **INFERRED (not measured)**: by the same bracketing argument as v79 —
    v72(110)/v87(110)/v92(110) all measure the identical 7-member shape,
    and v84 sits chronologically between v79 and v87. This inference is
    additionally supported (more than v79's) by v84's own **raw-bit
    mapping being fully confirmed** (`raw = 84 + i` for loop index i,
    directly read from `sub_7B0D46(a1,a2) = sub_89F235(a2+84)`), which
    pins GuidedBullet's slot with certainty even without block sizes.
- **Summed trailer length**: **UNVERIFIED (not measured)**. INFERRED 110 by
  bracketing (see above), not itself measured.
- **GuidedBullet slot index and mask-bit shift**: slot **5 — MEASURED**
  (raw bit mapping `raw = 84+i` confirmed by decompile of
  `sub_7B0D46`/`sub_89F235`, giving `i=5 → raw 89`, independently
  cross-checked against a dedicated `OnTemporaryStatReset` side-effect
  block that gates `IsActivated → GetMobID/GetReason → ResetGuidedMob` on
  the identical `sub_7B0D46(_,5)`-constructed mask). Raw shift **89,
  MEASURED**. Registry basis: **87** (`registry = raw − 2`, derived from
  two independent anchors: RideVehicle raw87→registry85, GuidedBullet
  raw89→registry87 — matching the plan's pre-existing "v83 GuidedBullet =
  registry 87" fact exactly).
- **Trailer read style**: **per-member mask-gated (MEASURED)** — both Loop
  1 (`sub_7AC409`, decompile) and Loop 2 (`sub_7A5D2B`, disasm) show the
  `sub_7B0D46(idx)`-test-then-skip pattern before the vtable
  `DecodeForClient` call.

**Cross-version corroboration worth recording**: v84's movement filter
(`movement-filter.md`) independently found **2 new UNVERIFIED constants at
raw shifts 82/83** with no v79/v83 analog. v87's own evidence (below)
independently resolves its own raw-82/83 constants as **Flying** and
**Frozen** — the *same* raw slots. This is strong evidence the Flying/Frozen
addition dates to v84, but **v84's own names for raw 82/83 remain
UNVERIFIED** — this is a cross-version inference about *when* the addition
happened, not a resolution of v84's own symbol gap.

**VERDICT: v84's member count (7) and GuidedBullet's slot (5, raw shift 89
→ registry 87) are MEASURED. Trailer read style is MEASURED as per-member
mask-gated. Per-member block sizes and the 110-byte total are UNVERIFIED —
the blocker is the same missing constructor symbol that blocked v79; the
110-byte total is INFERRED by bracketing (v72/v87/v92 all measure 110 with
identical shape) and additionally supported by v84's own confirmed slot-5
raw-bit mapping, but not itself measured.**

## v87

- **Member count**: **7 (MEASURED)** — `DecodeForRemote`'s tail loop
  (`0x7d8533`) iterates `while(v70<7)`, stride 8 bytes, base `this+848`.
- **Ordered members with block sizes (all MEASURED — the follow-up
  `gms_v87_sizes.md` located the actual member-array constructor,
  `sub_7CA8B4`, called from `CWvsContext::CWvsContext`, via a
  vtable-cluster walk-back rather than name search)**:

  | idx | name | DecodeForClient | block size |
  |---|---|---|---|
  | 0 | EnergyCharge | `0x7E4FEA` | **15** |
  | 1 | DashSpeed | `0x7E5165` | **15** |
  | 2 | DashJump | `0x7E5165` (same fn) | **15** |
  | 3 | RideVehicle/MonsterRiding | `0x7E4EB0` | **13** |
  | 4 | SpeedInfusion | `0x7E4C5B` | **20** |
  | 5 | GuidedBullet/HomingBeacon | `0x7CAAB5` | **17** |
  | 6 | Undead | `0x7E5165` (same fn) | **15** |

  Shared base `sub_7E4D25` = 4+4+5 = 13 bytes.
- **Summed trailer length**: **110 bytes (15+15+15+13+20+17+15), MEASURED
  — MATCHES the pre-95/v83 reference exactly.**
- **GuidedBullet slot index and mask-bit shift**: slot **5, MEASURED**
  (doubly confirmed: `OnTemporaryStatReset`'s `*(v3+858)` dereference
  gated on `dword_CA82B8` — raw shift 91 — *and* the constructor
  switch/vtable-chase in the sizes follow-up). Raw shift **91, MEASURED**
  — note **raw == registry** for v87 (offset 0), the first version in the
  campaign where raw and registry shifts coincide, because v87's client
  already includes the same post-SoulStone stats the atlas registry
  counts.
- **Trailer read style**: **per-member mask-gated (MEASURED)** —
  `DecodeForRemote`'s loop explicitly tests `1<<(86+idx)` against the
  decoded flag and skips the virtual call when clear (quoted disassembly
  in `gms_v87.md`).

**VERDICT: v87 MATCHES v83's 7-member/110-byte shape and member order
exactly (same 15/15/15/13/20/17/15 sizes, MEASURED via the recovered
`sub_7CA8B4` constructor). DIFFERS in trailer read style (per-member
mask-gated). GuidedBullet is at slot 5, raw shift 91 = registry shift 91
(offset 0). The movement filter (`movement-filter.md`) additionally shows
v87 tests 14 constants (v83's 12 + Flying(82) + Frozen(83)).**

## v92

- **Member count**: **7 (MEASURED)** — `SecondaryStat::DecodeForRemote`
  (`0x711240`) tail loop iterates `while(v86<7)`, raw shifts `115+v86` for
  `v86=0..6` (raw range 115-121), array base `this+1151`.
- **Ordered members with block sizes (all MEASURED — the follow-up
  `gms_v92_sizes.md` located the constructor `sub_7129F0` via a
  `search_text` sweep for the array's byte offset `0x11FCh`, closing the
  first pass's two open gaps)**:

  | idx (raw shift) | class | block size |
  |---|---|---|
  | 0 (115) | `sub_712180` | **15** |
  | 1 (116) | `sub_70CDF0` | **15** |
  | 2 (117) | `sub_70CDF0` (same instance-type) | **15** |
  | 3 (118) | `sub_70CD00` (NoExpire base-only) | **13** |
  | 4 (119) | `sub_712040` (Expire<BaseOnCurrentTime,...>) | **20** |
  | 5 (120) | **`sub_70D260` = `TemporaryStat_GuidedBullet::DecodeForClient`** | **17** |
  | 6 (121) | `sub_70CDF0` (3rd instance of the same class as idx1/2) | **15** |

  The first pass (`gms_v92.md`) had tentatively found "5 distinct vtables,
  one slot ambiguous" and flagged a false 20-byte candidate (`0xb360f0`);
  the follow-up (`gms_v92_sizes.md`) resolved this by reading the
  constructor's index arithmetic directly (`ecx` offset ÷ 8 = slot index)
  and confirmed the 7th slot is a 3rd instance of the 15-byte class already
  used at idx 1/2, not a new class.
- **Summed trailer length**: **110 bytes
  (15+15+15+13+20+17+15), MEASURED — MATCHES the pre-95/v83 reference
  exactly.** (The first pass's "subtotal 100, 6 of 7 slots" was superseded
  by the follow-up's full resolution.)
- **GuidedBullet slot index and mask-bit shift**: slot **5, MEASURED two
  independent ways** — `xrefs_to(0x70d260)` lands on vtable-base
  `0xb362dc`'s `+0x18` slot, and the constructor's `case 5` arm writes that
  same vtable base into array slot 5 via directly-read offset arithmetic
  (`(0x1220−0x11F8)/8 = 5`). Raw shift **120 (`115+5`), MEASURED**.
  **No atlas-registry equivalent exists for v92** — `character_temporary_stat.go`
  only encodes the v95+ two-state shape; there is no v92-specific
  registry table to convert against, so v92's raw shift (120) is reported
  client-basis-only, unmapped.
- **Trailer read style**: **per-member mask-gated (MEASURED)**, uniformly
  across all 7 members — every field decode in `DecodeForRemote` and the
  `OnTemporaryStatReset`/`Set` callers is wrapped in a mask-test before
  decoding. Note the first pass's own observation: this differs from the
  v95 reference's specific "only the last two of 6 members conditional"
  shape — v92 gates **all 7** individually, matching the pre-95-shaped
  member set with v95-style universal per-member gating.

**VERDICT: v92 MATCHES v83's 7-member/110-byte shape and member order
exactly (15/15/15/13/20/17/15, MEASURED via the recovered `sub_7129F0`
constructor). DIFFERS in trailer read style (per-member mask-gated, all 7
members uniformly). GuidedBullet is at slot 5, raw shift 120 — no atlas
registry mapping exists yet for v92's own basis.**

## v83 (transcribed from design.md §2.3 — design-phase verified)

- **Member count**: 7 (`GuidedBulletTemporaryStat` is member index 5 of the
  existing `libs/atlas-packet/model/character_temporary_stat.go:473-505`
  layout, matching the client's `SecondaryStat` object).
- **Ordered members with block sizes**: not individually re-tabulated in
  design.md §2.3 (design.md verifies the **GuidedBullet member alone** —
  17 bytes: nValue(4) + rValue(4) + DecodeTime(5) + dwMobId(4) — via
  `sub_77C442` → `TwoStateTemporaryStat<...>::DecodeForClient` @`0x79407D`
  → `TemporaryStatBase<long>::DecodeForClient` @`0x793EF2` → `Decode4`).
  This matches the existing lib layout exactly.
- **Summed trailer length**: 110 bytes (the pre-95 reference figure every
  other version in this document is compared against).
- **GuidedBullet mask bit**: constant at `0xBF5528` = 16 bytes whose low
  qword (LE) is `0x0080000000000000`, matching Cosmic's
  `BuffStat.HOMING_BEACON = 0x80000000000000L` — bit **shift 87**
  (`log2(0x80000000000000) = 55`... design.md states this directly as the
  registry-basis GuidedBullet bit; the atlas registry constant is anchored
  to this v83 value). This is the "**registry shift 87**" fact every other
  version's evidence file cross-references.
- **Trailer read style**: design.md's F1 finding states the current
  encoder ("`EncodeMask` unconditionally ORs every `twoStateBaseStats(t)`
  member's bit") and both v83's set/reset handlers clear *every* masked
  stat on a cancel — this describes **our encoder's** unconditional-write
  behavior, not a directly re-derived v83 client read-style measurement.
  **Important generalisation correction**: every version in this campaign
  that *was* re-examined for read style (v61, v72, v79, v84, v87, v92,
  JMS) reports **per-member mask-gated**, not unconditional, on the client
  read side. Design.md's "pre-95 unconditional" language describes the
  *v83 registry source comment* characterising the server encoder's
  write-side behavior; this campaign did not re-derive v83's own client
  *read*-side gating and cannot contradict or confirm it directly. See the
  closing note below for why the client being mask-gated (as this campaign
  found on every other version) is benign regardless.

## v95 (transcribed from design.md §2.4 — design-phase verified)

- **Member count**: 7 slots in `aTemporaryStat[7]` (`SecondaryStat::SecondaryStat`
  @`0x72F190`), but only **6 are reachable** — slot 6 has "no CTS mask
  constant exists → unreachable on the wire" (the "Undead overflows the
  mask" slot).
- **Ordered members with block sizes** (design.md §2.4, direct quote):

  | Slot | Stat | Block size |
  |---|---|---|
  | 0 | EnergyCharged | **15** |
  | 1 | Dash_Speed | **15** |
  | 2 | Dash_Jump | **15** |
  | 3 | RideVehicle | **13** |
  | 4 | PartyBooster | **20** (note: 5-byte `tCurrentTime`, not the same
      15-byte shape as pre-95 SpeedInfusion at this slot) |
  | 5 | GuidedBullet | **17** |
  | 6 | (unnamed, unreachable) | 15 (unreachable on the wire) |
- **Summed trailer length**: 6 reachable slots = 15+15+15+13+20+17 = **95
  bytes**, not 110 — a structurally different total from every pre-95
  version in this document (v95 substitutes PartyBooster for
  SpeedInfusion at slot 4 and drops the reachable 7th slot).
- **GuidedBullet slot index and mask-bit shift**: slot **5**, mask bit
  **127** (`CTS_*` dynamic-initializer symbols: EnergyCharged=122,
  Dash_Speed=123, Dash_Jump=124, RideVehicle=125, PartyBooster=126,
  GuidedBullet=127).
- **Trailer read style**: **mask-gated per member (design.md, directly
  quoted)** — `DecodeForLocal` (`0x7350E0`), tail loop at
  `0x73DBA0-0x73DBF2`, builds `UINT128(1) << shift` and tests it before
  each member's virtual `DecodeForClient` call. This is the version this
  campaign's other 7 pre-95 read-style findings are consistent with, not
  an outlier.

**VERDICT: v95 is structurally distinct from the pre-95 shape (6 reachable
members not 7, PartyBooster replaces SpeedInfusion, 95-byte not 110-byte
sum) and this was already fully closed at design time — see
`v95-two-state-group.md` for the standalone record.**

---

## Read-style generalisation — correcting design.md's "pre-95 unconditional" language

**Every version this campaign examined (v61, v72, v79, v84, v87, v92,
JMS v185) reports the trailer as per-member mask-gated, not
unconditional.** Design.md's F1/F2 discussion frames "pre-95" as reading
the full 7-block trailer unconditionally; that framing is **v83-specific**
(sourced from the v83 registry comment describing the *server encoder's*
write-side behavior) and is **wrong as a cross-version generalisation** —
it does not describe what this campaign found on the client *read* side for
any of the seven newly-examined versions.

**Why this is nonetheless benign for our encoder**: the design's chosen
approach (§4.4(a), §5.5) sets **all** the two-state bits AND writes **all**
the blocks for every give (populated GuidedBullet block on the MonsterRiding
pattern, F2's registry-merge). A mask-gated client that tests each bit
before reading its block will, when every bit is set and every block is
present, read every block anyway — mask-gated and unconditional-with-all-
bits-set produce byte-identical wire consumption. The mask-gating finding
changes *how* we should reason about client behavior (it explains why a
partial-bit encode would be safe on these clients, which the pre-95
"unconditional" framing would have ruled out) but does not require any
change to the encoder design that was chosen.

---

## gms_12 / gms_48 — not applicable

Recorded here per the plan's Step 8. Three citations, each independently
verified against the current repo state (not copied from the plan on
faith):

- **Zero `521xxxx`/`522xxxx` skill ids exist for these versions.**
  `grep -oE '52[12][0-9]{4}' libs/atlas-constants/gen/wzsnapshot/gms_12_1.json`
  and the same grep against `gms_48_1.json` both return **0 matches** —
  confirmed by direct search of the current repo files, not asserted from
  memory. Neither Homing Beacon (5211006) nor Bullseye (5220011) exist as
  skill ids in either snapshot.
- **The `legacyGmsMask` condition** — `libs/atlas-packet/model/character_temporary_stat.go:592-593`:
  ```go
  func legacyGmsMask(t tenant.Model) bool {
      return t.Region() == "GMS" && t.MajorVersion() < 61
  }
  ```
  confirmed present verbatim at those line numbers in the current worktree.
  Its preceding doc comment (`:575-591`) independently states the v48
  IDA-verified facts this section relies on: a plain 8-byte mask (not the
  16-byte UINT128 v61+ anchor codec emits), and that "the two-state base
  stats sit at shifts 81-87 (mask.H), which pre-v61 clients do not read" —
  i.e. gms_12/gms_48 structurally cannot carry a HOMING_BEACON two-state
  member even if the skill ids existed.
- **The v48 client reads an 8-byte mask via `DecodeBuffer(&v8, 8)`
  @`0x71b06e` under handler `0x71b054`.** Confirmed present in
  `libs/atlas-packet/character/clientbound/buff_cancel_test.go:117-125`,
  which pins this exact fact with a `packet-audit:verify` marker:
  `// packet-audit:verify packet=character/clientbound/BuffCancel
  version=gms_v48 ida=0x71b054`, and the surrounding comment cites
  `CWvsContext::OnTemporaryStatReset @0x71b054 via CInPacket::DecodeBuffer(&v8, 8) @0x71b06e`
  on `GMS_v48_1_DEVM.exe` (port 13337) — matching the plan's citation
  verbatim.

**Conclusion**: gms_12 and gms_48 have zero in-scope skill ids (Homing
Beacon / Bullseye do not exist on either version), and even setting that
aside, both versions predate the `legacyGmsMask` boundary (`MajorVersion()
< 61`) and their client reads a mask width (8 bytes / 64 bits, IDA-verified
on gms_48) too narrow to reach the two-state group's raw bit positions
(59+ on the earliest measured version, v61) at all. Homing Beacon / Bullseye
is therefore genuinely not-applicable on gms_12 and gms_48 — not a
deferred verification gap.
