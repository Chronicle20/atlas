# MakerResultCreate (← `CUserLocal::OnMakerResult#Create`)

- **IDA:** 0x8b5af5
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/maker_result.go`
- **Variant:** GMS/v79
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `nResult (read unconditionally; > 1 ends the body)` | ✅ |  |
| 1 | int32 | int32 `nMode (arm selector; this arm is nMode == 1)` | ✅ |  |
| 2 | byte | byte `bNoItemGain @0x8b5e10 (truthy SUPPRESSES the pair below)` | ✅ |  |
| 3 | int32 | int32 `nTargetItemID @0x8b5e29` | ✅ |  |
| 4 | int32 | int32 `nItemCount @0x8b5e3a` | ✅ |  |
| 5 | int32 | int32 `nNumUsedItem @0x8b5ebd (loop length prefix)` | ✅ |  |
| 6 | int32 | int32 `used[i].nItemID @0x8b5edb` | ✅ |  |
| 7 | int32 | int32 `used[i].nCount @0x8b5ee8` | ✅ |  |
| 8 | int32 | int32 `nNumUsedGem @0x8b5f71 (loop length prefix)` | ✅ |  |
| 9 | int32 | int32 `gem[i].nItemID @0x8b5f82 (id only, no count)` | ✅ |  |
| 10 | byte | byte `bUsedCatalyst @0x8b6010` | ✅ |  |
| 11 | int32 | int32 `nCatalystItemID @0x8b6022` | ✅ |  |
| 12 | int32 | int32 `nMesoCost @0x8b60ae (rendered as a loss by the client)` | ✅ |  |

