# Changed (← `CWvsContext::OnStatChanged`)

- **IDA:** 0xb06632
- **Atlas file:** `libs/atlas-packet/stat/clientbound/changed.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bExclRequestSent` | ✅ |  |
| 1 | int32 | int32 `dwStatMask (GW_CharacterStat::DecodeChangeStat @0x50f16a)` | ✅ |  |
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
| 14 | byte | int16 `nHP (JMS v185 int16 — NOT widened to int32 unlike GMS v95)` | ❌ | atlas: short — missing trailing field |
| 15 | byte | int16 `nMHP (JMS v185 int16)` | ❌ | atlas: short — missing trailing field |
| 16 | byte | int16 `nMP (JMS v185 int16)` | ❌ | atlas: short — missing trailing field |
| 17 | byte | int16 `nMMP (JMS v185 int16)` | ❌ | atlas: short — missing trailing field |
| 18 | byte | int16 `nAP` | ❌ | atlas: short — missing trailing field |
| 19 | byte | int16 `nSP (non-extendSP job branch) or sub_50E8B0 (ExtendSP)` | ❌ | atlas: short — missing trailing field |
| 20 | byte | int32 `nEXP` | ❌ | atlas: short — missing trailing field |
| 21 | byte | int16 `nPOP (fame)` | ❌ | atlas: short — missing trailing field |
| 22 | byte | int32 `nMoney (meso)` | ❌ | atlas: short — missing trailing field |
| 23 | byte | int32 `nTempEXP (gachaponExperience)` | ❌ | atlas: short — missing trailing field |
| 24 | byte | byte `bSecondaryStatChangedPoint flag (OnStatChanged@0xb066c9; conditional on mask 0x180008)` | ❌ | atlas: short — missing trailing field |


## Manual analysis

**The auto-generated ❌ verdict is a static-tool artifact** — identical situation to GMS v83/v95: mask-driven data-dependent encoder; static diff cannot align positionally.

**JMS v185 IDA key findings:**

1. **HP/MaxHP/MP/MaxMP width** (`GW_CharacterStat::DecodeChangeStat` @ 0x50f16a, masks 0x400/0x800/0x1000/0x2000): JMS v185 calls `CInPacket::Decode2` (int16) for all four fields — same as GMS v83. The atlas gate `v95Plus := GMS && MajorVersion >= 95` correctly writes `WriteInt16` for JMS (Region != "GMS" → v95Plus = false). **Gate CONFIRMED CORRECT ✅.**

2. **Trailing flag byte(s)**: JMS `OnStatChanged` (@ 0xb06632) reads ONE conditional `Decode1` for `bSecondaryStatChangedPoint` (only when `result & 0x180008 != 0`). No `battle-recovery-info` second byte. Atlas writes 1 unconditional trailing byte (not `v95Plus`). When pet SNs are absent, the trailing byte remains unconsumed — harmless. **Gate CONFIRMED CORRECT ✅.**

3. **Mask layout**: JMS uses the same mask bit assignments as GMS v83/v95 (SKIN=0x1, FACE=0x2, HAIR=0x4, PET_SN1=0x8, LEVEL=0x10, JOB=0x20, STR=0x40, DEX=0x80, INT=0x100, LUK=0x200, HP=0x400, MaxHP=0x800, MP=0x1000, MaxMP=0x2000, AP=0x4000, SP=0x8000, EXP=0x10000, FAME=0x20000, MESO=0x40000, PET_SN2=0x80000, PET_SN3=0x100000, GACHAPON_EXP=0x200000). No mask layout difference.

**Net result:** No code change needed for JMS. Both v95Plus gates remain correctly scoped to `GMS && MajorVersion >= 95`.

**JMS vs GMS: gate confirmed ✅.** JMS uses int16 for HP/MaxHP/MP/MaxMP (same as GMS v83), and 1 trailing byte (same as GMS v83/non-GMS).

Ack: misc-audit Phase 3 JMS185 on 2026-06-03
