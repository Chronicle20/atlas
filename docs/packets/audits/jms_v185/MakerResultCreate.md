# MakerResultCreate (← `CUserLocal::OnMakerResult#Create`)

- **IDA:** 0xa29527
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/maker_result.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `nResult (read unconditionally; > 1 ends the body)` | ✅ |  |
| 1 | int32 | int32 `nMode (arm selector; this arm is nMode == 1)` | ✅ |  |
| 2 | byte | byte `bNoItemGain @0xa2984d (truthy SUPPRESSES the pair below)` | ✅ |  |
| 3 | int32 | int32 `nTargetItemID @0xa29866` | ✅ |  |
| 4 | int32 | int32 `nItemCount @0xa29878` | ✅ |  |
| 5 | int32 | int32 `nNumUsedItem @0xa2990c (loop length prefix)` | ✅ |  |
| 6 | int32 | int32 `used[i].nItemID @0xa2992a` | ✅ |  |
| 7 | int32 | int32 `used[i].nCount @0xa2993c` | ✅ |  |
| 8 | int32 | int32 `nNumUsedGem @0xa299b0 (loop length prefix)` | ✅ |  |
| 9 | int32 | int32 `gem[i].nItemID @0xa299be (id only, no count)` | ✅ |  |
| 10 | byte | byte `bUsedCatalyst @0xa29a37` | ✅ |  |
| 11 | int32 | int32 `nCatalystItemID @0xa29a45` | ✅ |  |
| 12 | int32 | int32 `nMesoCost @0xa29abe (rendered as a loss by the client)` | ✅ |  |

