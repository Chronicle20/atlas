# Changed (← `CWvsContext::OnStatChanged`)

- **IDA:** 0xab6e77
- **Atlas file:** `libs/atlas-packet/stat/clientbound/changed.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bExclRequestSent` | ✅ |  |
| 1 | int32 | int32 `dwStatMask (GW_CharacterStat::DecodeChangeStat @0x502252)` | ✅ |  |
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
| 14 | byte | int16 `nHP (v87 int16; v95 widened to int32)` | ❌ | atlas: short — missing trailing field |
| 15 | byte | int16 `nMHP (v87 int16; v95 widened to int32)` | ❌ | atlas: short — missing trailing field |
| 16 | byte | int16 `nMP (v87 int16; v95 widened to int32)` | ❌ | atlas: short — missing trailing field |
| 17 | byte | int16 `nMMP (v87 int16; v95 widened to int32)` | ❌ | atlas: short — missing trailing field |
| 18 | byte | int16 `nAP` | ❌ | atlas: short — missing trailing field |
| 19 | byte | int16 `nSP (non-extendSP job branch)` | ❌ | atlas: short — missing trailing field |
| 20 | byte | int32 `nEXP` | ❌ | atlas: short — missing trailing field |
| 21 | byte | int16 `nPOP (fame)` | ❌ | atlas: short — missing trailing field |
| 22 | byte | int32 `nMoney (meso)` | ❌ | atlas: short — missing trailing field |
| 23 | byte | int32 `nTempEXP (gachaponExperience)` | ❌ | atlas: short — missing trailing field |
| 24 | byte | byte `bSecondaryStatChangedPoint flag (@0xab6f0b; mask 0x180008; ONE trailing byte only — v87 has no battle-recovery-info second byte)` | ❌ | atlas: short — missing trailing field |


## Manual analysis

**The auto-generated table above is positionally invalid for this packet** — identical situation to v95/v83: mask-driven data-dependent encoder; static diff cannot align the two lists.

**v87 IDA key findings:**

1. **HP/MaxHP/MP/MaxMP width** (`GW_CharacterStat::DecodeChangeStat` @ 0x502252, masks 0x400/0x800/0x1000/0x2000): v87 calls `CInPacket::Decode2` (int16) for all four fields. v95 uses `Decode4` (int32). The atlas gate `v95Plus := GMS && MajorVersion >= 95` correctly writes `WriteInt16` for v87 and `WriteInt` for v95. **Gate CONFIRMED CORRECT ✅**.

2. **Trailing flag bytes** (`CWvsContext::OnStatChanged` @ 0xab6e77, line 97): v87 checks `(v79 & 0x180008) != 0` and reads **ONE** trailing `Decode1` byte (bSecondaryStatChangedPoint). v87 has NO `battle-recovery-info` second trailing byte — identical to v83. The atlas gate that writes the second trailing byte only on `v95Plus` is **CONFIRMED CORRECT ✅**.

**Net result:** No code change needed. Both v95-era gates are correct as-is for v87. v87 mirrors v83 exactly for this packet.

Ack: misc-audit Phase 3 v87 on 2026-06-03
