# MakerResultDisassemble (← `CUserLocal::OnMakerResult#Disassemble`)

- **IDA:** 0xa29527
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/maker_result.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `nResult (read unconditionally; > 1 ends the body)` | ✅ |  |
| 1 | int32 | int32 `nMode (arm selector; this arm is nMode == 4)` | ✅ |  |
| 2 | int32 | int32 `nDisassembledItemID @0xa296c5` | ✅ |  |
| 3 | int32 | int32 `nNumRewardItem @0xa29726 (loop length prefix)` | ✅ |  |
| 4 | int32 | int32 `reward[i].nItemID @0xa2973f` | ✅ |  |
| 5 | int32 | int32 `reward[i].nCount @0xa2974d` | ✅ |  |
| 6 | int32 | int32 `nMesoCost @0xa297ef` | ✅ |  |

