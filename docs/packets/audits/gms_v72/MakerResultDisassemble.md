# MakerResultDisassemble (← `CUserLocal::OnMakerResult#Disassemble`)

- **IDA:** 0x86a152
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/maker_result.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `nResult (read unconditionally; > 1 ends the body)` | ✅ |  |
| 1 | int32 | int32 `nMode (arm selector; this arm is nMode == 4)` | ✅ |  |
| 2 | int32 | int32 `nDisassembledItemID @0x86a2f2` | ✅ |  |
| 3 | int32 | int32 `nNumRewardItem @0x86a35d (loop length prefix)` | ✅ |  |
| 4 | int32 | int32 `reward[i].nItemID @0x86a376` | ✅ |  |
| 5 | int32 | int32 `reward[i].nCount @0x86a383` | ✅ |  |
| 6 | int32 | int32 `nMesoCost @0x86a409` | ✅ |  |

