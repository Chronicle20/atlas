# Changed (← `CWvsContext::OnStatChanged`)

- **IDA:** 0x9fd5d0
- **Atlas file:** `../../libs/atlas-packet/stat/clientbound/changed.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bExclRequestSent` | ✅ |  |
| 1 | int32 | int32 `dwStatMask (GW_CharacterStat::DecodeChangeStat @0x4fa000)` | ✅ |  |
| 2 | byte | byte `nSkin` | ✅ |  |
| 3 | int32 | int32 `nFace` | ✅ |  |
| 4 | int32 | int32 `nHair` | ✅ |  |
| 5 | int16 | int64 `petLockerSN[0] (DecodeBuffer 8)` | ❌ | width mismatch |
| 6 | int32 | int64 `petLockerSN[1] (DecodeBuffer 8)` | ❌ | width mismatch |
| 7 | int64 | int64 `petLockerSN[2] (DecodeBuffer 8)` | ✅ |  |
| 8 | byte | byte `nLevel` | ✅ |  |
| 9 | byte | int16 `nJob` | ❌ | width mismatch |
| 10 | byte | int16 `nSTR` | ❌ | atlas: short — missing trailing field |
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

