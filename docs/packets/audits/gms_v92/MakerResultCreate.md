# MakerResultCreate (← `CUserLocal::OnMakerResult#Create`)

- **IDA:** 0x8f5d70
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/maker_result.go`
- **Variant:** GMS/v92
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `nResult (read unconditionally; > 1 ends the body)` | ✅ |  |
| 1 | int32 | int32 `nMode (arm selector; this arm is nMode == 1)` | ✅ |  |
| 2 | byte | byte `bNoItemGain @0x8f6197 (truthy SUPPRESSES the pair below)` | ✅ |  |
| 3 | int32 | int32 `nTargetItemID @0x8f61b2` | ✅ |  |
| 4 | int32 | int32 `nItemCount @0x8f61c6` | ✅ |  |
| 5 | int32 | int32 `nNumUsedItem @0x8f628e (loop length prefix)` | ✅ |  |
| 6 | int32 | int32 `used[i].nItemID @0x8f62aa` | ✅ |  |
| 7 | int32 | int32 `used[i].nCount @0x8f62bd` | ✅ |  |
| 8 | int32 | int32 `nNumUsedGem @0x8f636f (loop length prefix)` | ✅ |  |
| 9 | int32 | int32 `gem[i].nItemID @0x8f6384 (id only, no count)` | ✅ |  |
| 10 | byte | byte `bUsedCatalyst @0x8f6446` | ✅ |  |
| 11 | int32 | int32 `nCatalystItemID @0x8f6458` | ✅ |  |
| 12 | int32 | int32 `nMesoCost @0x8f6522 (rendered as a loss by the client)` | ✅ |  |

