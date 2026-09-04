# MakerResultMonsterCrystal (← `CUserLocal::OnMakerResult#MonsterCrystal`)

- **IDA:** 0x8f5d70
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/maker_result.go`
- **Variant:** GMS/v92
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `nResult (read unconditionally; > 1 ends the body)` | ✅ |  |
| 1 | int32 | int32 `nMode (arm selector; this arm is nMode == 3)` | ✅ |  |
| 2 | int32 | int32 `nTargetItemID @0x8f5dfa (crystal produced)` | ✅ |  |
| 3 | int32 | int32 `nSourceItemID @0x8f5e0d (item consumed; Decode4 despite a byte-width local)` | ✅ |  |

