# Changed (← `CWvsContext::OnStatChanged`)

- **IDA:** 0x9fd5d0
- **Atlas file:** `libs/atlas-packet/stat/clientbound/changed.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ❌ (static-diff limitation — see Manual analysis; real verification is the round-trip + wire-width tests)

## Wire-level diff (auto-generated)

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bExclRequestSent` | ✅ |  |
| 1 | int32 | int32 `dwStatMask (GW_CharacterStat::DecodeChangeStat @0x4fa000)` | ✅ |  |
| 2 | byte | byte `nSkin` | ✅ |  |
| 3 | int32 | int32 `nFace` | ✅ |  |
| 4 | int16 | int32 `nHair` | ❌ | width mismatch |
| 5 | int16 | int64 `petLockerSN[0] (DecodeBuffer 8)` | ❌ | width mismatch |
| 6 | int16 | int64 `petLockerSN[1] (DecodeBuffer 8)` | ❌ | width mismatch |
| 7 | int32 | int64 `petLockerSN[2] (DecodeBuffer 8)` | ❌ | width mismatch |
| 8 | int64 | byte `nLevel` | ❌ | width mismatch |
| 9 | byte | int16 `nJob` | ❌ | width mismatch |
| 10 | byte | int16 `nSTR` | ❌ | width mismatch |
| 11 | byte | int16 `nDEX` | ❌ | atlas: short — missing trailing field |
| 12 | byte | int16 `nINT` | ❌ | atlas: short — missing trailing field |
| 13 | byte | int16 `nLUK` | ❌ | atlas: short — missing trailing field |
| 14 | byte | int32 `nHP (v95 widened from int16)` | ❌ | atlas: short — missing trailing field |
| 15 | byte | int32 `nMHP (v95 widened from int16)` | ❌ | atlas: short — missing trailing field |
| 16 | byte | int32 `nMP (v95 widened from int16)` | ❌ | atlas: short — missing trailing field |
| 17 | byte | int32 `nMMP (v95 widened from int16)` | ❌ | atlas: short — missing trailing field |
| 18 | byte | int16 `nAP` | ❌ | atlas: short — missing trailing field |
| 19 | byte | int16 `nSP (non-extendSP job branch)` | ❌ | atlas: short — missing trailing field |
| 20 | byte | int32 `nEXP` | ❌ | atlas: short — missing trailing field |
| 21 | byte | int16 `nPOP (fame)` | ❌ | atlas: short — missing trailing field |
| 22 | byte | int32 `nMoney (meso)` | ❌ | atlas: short — missing trailing field |
| 23 | byte | int32 `nTempEXP (gachaponExperience)` | ❌ | atlas: short — missing trailing field |
| 24 | byte | byte `bSecondaryStatChangedPoint flag (OnStatChanged@0x9fd6c4; conditional Decode1 payload omitted)` | ❌ | atlas: short — missing trailing field |
| 25 | byte | byte `battle-recovery-info flag (OnStatChanged@0x9fd6f0; conditional Decode4+Decode4 payload omitted)` | ❌ | atlas: short — missing trailing field |

## Manual analysis

**The auto-generated table above is positionally invalid for this packet** and the
❌ verdict is a static-analyzer limitation, not a residual bug. Reason: this is a
**mask-driven, data-dependent** encoder. `Changed.Encode` writes only the stats
present in the update set, in the bit order derived from `options["statistics"]`,
whereas the analyzer flattens *every* `switch` case in source order and diffs it
positionally against the IDA bit-ordered field list. The two lists cannot align.
Real verification is by manual IDA comparison (below) plus
`stat/clientbound/changed_test.go` (`TestStatChanged*RoundTrip` for encode/decode
symmetry across all four variants, and `TestStatChangedV95WireWidths` for the
byte-level v95 wire shape).

IDA references: `CWvsContext::OnStatChanged` @0x9fd5d0; mask body
`GW_CharacterStat::DecodeChangeStat` @0x4fa000.

### Real wire bugs found and fixed

1. **HP / MaxHP / MP / MaxMP width (mask bits 0x400 / 0x800 / 0x1000 / 0x2000).**
   v95 `DecodeChangeStat` reads each as `Decode4` (int32); atlas wrote `WriteInt16`
   (2 bytes), desyncing every subsequent field. Fixed: gated to `WriteInt` for
   `GMS && MajorVersion >= 95`, `WriteInt16` otherwise — matching the existing
   precedent in `model/character_statistics.go:112-120`. (`AVAILABLE_AP`, `FAME`,
   `JOB`, `STR/DEX/INT/LUK` remain `Decode2`/int16 — correct.)

2. **Missing trailing flag byte.** `OnStatChanged` reads *two* trailing flags after
   the stat block: `bSecondaryStatChangedPoint` (@0x9fd6c4) then battle-recovery-info
   (@0x9fd6f0). The server leaves both unset (0). atlas wrote only one trailing
   `WriteByte(0)`. Fixed: write/read the second trailing zero byte. Gated to v95+
   (no older-version regression) pending v83/v87/JMS verification in Phase 3.

### Verified-correct under v95

`nSkin`(0x1,byte), `nLevel`(0x10,byte), `nFace`(0x2)/`nHair`(0x4,int32),
`nJob`(0x20)/`nSTR`(0x40)/`nDEX`(0x80)/`nINT`(0x100)/`nLUK`(0x200)/`nAP`(0x4000)/
`nPOP`(0x20000,int16), `nSP`(0x8000,short, non-extendSP job branch),
`nEXP`(0x10000)/`nMoney`(0x40000)/`nTempEXP`(0x200000,int32),
`petLockerSN[0..2]`(0x8/0x80000/0x100000, 8 bytes).

### Open items / follow-ups

- **Mask-bit ORDER is config-driven.** atlas assigns each stat's mask bit from its
  index in `options["statistics"]`. For wire correctness that list MUST match the
  canonical `GW_CharacterStat::DecodeChangeStat` bit layout (SKIN=0x1, FACE=0x2,
  HAIR=0x4, PET_SN_1=0x8, LEVEL=0x10, JOB=0x20, STR=0x40, DEX=0x80, INT=0x100,
  LUK=0x200, HP=0x400, MAX_HP=0x800, MP=0x1000, MAX_MP=0x2000, AVAILABLE_AP=0x4000,
  AVAILABLE_SP=0x8000, EXPERIENCE=0x10000, FAME=0x20000, MESO=0x40000,
  PET_SN_2=0x80000, PET_SN_3=0x100000, GACHAPON_EXPERIENCE=0x200000). The per-tenant
  `statistics` config in `atlas-tenants` should be verified against this layout
  (config-data audit, out of scope for the atlas-packet encoder fix).
- **extendSP branch (0x8000).** When job/1000==3 or job/100==22 or job==2001, the
  v95 client decodes `ExtendSP::Decode` (a variable struct) instead of a plain
  `Decode2`. atlas always writes a short. Not exercised by the current encoder
  (no extend-SP path); recorded as a known gap.
- **Phase 3:** confirm v83/v87/JMS HP width and trailing-flag-byte count; widen or
  narrow the `v95Plus` gates accordingly.

Ack: misc-audit Phase 2b on 2026-06-03
