# MakerResultCreate (← `CUserLocal::OnMakerResult#Create`)

- **IDA:** 0x95dad3
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/maker_result.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `nResult (read unconditionally; > 1 ends the body)` | ✅ |  |
| 1 | int32 | int32 `nMode (arm selector; this arm is nMode == 1)` | ✅ |  |
| 2 | byte | byte `bNoItemGain @0x95de3f (truthy SUPPRESSES the pair below)` | ✅ |  |
| 3 | int32 | int32 `nTargetItemID @0x95de58` | ✅ |  |
| 4 | int32 | int32 `nItemCount @0x95de69` | ✅ |  |
| 5 | int32 | int32 `nNumUsedItem @0x95df1a (loop length prefix)` | ✅ |  |
| 6 | int32 | int32 `used[i].nItemID @0x95df38` | ✅ |  |
| 7 | int32 | int32 `used[i].nCount @0x95df45` | ✅ |  |
| 8 | int32 | int32 `nNumUsedGem @0x95dfce (loop length prefix)` | ✅ |  |
| 9 | int32 | int32 `gem[i].nItemID @0x95dfdf (id only, no count)` | ✅ |  |
| 10 | byte | byte `bUsedCatalyst @0x95e06d` | ✅ |  |
| 11 | int32 | int32 `nCatalystItemID @0x95e07f` | ✅ |  |
| 12 | int32 | int32 `nMesoCost @0x95e10b (rendered as a loss by the client)` | ✅ |  |

