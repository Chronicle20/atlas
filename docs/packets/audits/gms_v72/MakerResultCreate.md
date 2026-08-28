# MakerResultCreate (← `CUserLocal::OnMakerResult#Create`)

- **IDA:** 0x86a152
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/maker_result.go`
- **Variant:** GMS/v72
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `nResult (read unconditionally; > 1 ends the body)` | ✅ |  |
| 1 | int32 | int32 `nMode (arm selector; this arm is nMode == 1)` | ✅ |  |
| 2 | byte | byte `bNoItemGain @0x86a46d (truthy SUPPRESSES the pair below)` | ✅ |  |
| 3 | int32 | int32 `nTargetItemID @0x86a486` | ✅ |  |
| 4 | int32 | int32 `nItemCount @0x86a497` | ✅ |  |
| 5 | int32 | int32 `nNumUsedItem @0x86a51a (loop length prefix)` | ✅ |  |
| 6 | int32 | int32 `used[i].nItemID @0x86a538` | ✅ |  |
| 7 | int32 | int32 `used[i].nCount @0x86a545` | ✅ |  |
| 8 | int32 | int32 `nNumUsedGem @0x86a5ce (loop length prefix)` | ✅ |  |
| 9 | int32 | int32 `gem[i].nItemID @0x86a5df (id only, no count)` | ✅ |  |
| 10 | byte | byte `bUsedCatalyst @0x86a66d` | ✅ |  |
| 11 | int32 | int32 `nCatalystItemID @0x86a67f` | ✅ |  |
| 12 | int32 | int32 `nMesoCost @0x86a70b (rendered as a loss by the client)` | ✅ |  |

