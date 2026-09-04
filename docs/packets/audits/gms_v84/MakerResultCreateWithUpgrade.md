# MakerResultCreateWithUpgrade (← `CUserLocal::OnMakerResult#CreateWithUpgrade`)

- **IDA:** 0x99bdbc
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/maker_result.go`
- **Variant:** GMS/v84
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `nResult (read unconditionally; > 1 ends the body)` | ✅ |  |
| 1 | int32 | int32 `nMode (arm selector; this arm is nMode == 2)` | ✅ |  |
| 2 | byte | byte `bNoItemGain @0x99c128 (truthy SUPPRESSES the pair below)` | ✅ |  |
| 3 | int32 | int32 `nTargetItemID @0x99c141` | ✅ |  |
| 4 | int32 | int32 `nItemCount @0x99c152` | ✅ |  |
| 5 | int32 | int32 `nNumUsedItem @0x99c203 (loop length prefix)` | ✅ |  |
| 6 | int32 | int32 `used[i].nItemID @0x99c221` | ✅ |  |
| 7 | int32 | int32 `used[i].nCount @0x99c22e` | ✅ |  |
| 8 | int32 | int32 `nNumUsedGem @0x99c2b7 (loop length prefix)` | ✅ |  |
| 9 | int32 | int32 `gem[i].nItemID @0x99c2c8 (id only, no count)` | ✅ |  |
| 10 | byte | byte `bUsedCatalyst @0x99c356` | ✅ |  |
| 11 | int32 | int32 `nCatalystItemID @0x99c368` | ✅ |  |
| 12 | int32 | int32 `nMesoCost @0x99c3f4 (rendered as a loss by the client)` | ✅ |  |

