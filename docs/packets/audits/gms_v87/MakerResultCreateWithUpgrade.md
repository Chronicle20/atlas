# MakerResultCreateWithUpgrade (← `CUserLocal::OnMakerResult#CreateWithUpgrade`)

- **IDA:** 0x9e01b2
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/maker_result.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `nResult (read unconditionally; > 1 ends the body)` | ✅ |  |
| 1 | int32 | int32 `nMode (arm selector; this arm is nMode == 2)` | ✅ |  |
| 2 | byte | byte `bNoItemGain @0x9e04d9 (truthy SUPPRESSES the pair below)` | ✅ |  |
| 3 | int32 | int32 `nTargetItemID @0x9e04f2` | ✅ |  |
| 4 | int32 | int32 `nItemCount @0x9e0503` | ✅ |  |
| 5 | int32 | int32 `nNumUsedItem @0x9e0599 (loop length prefix)` | ✅ |  |
| 6 | int32 | int32 `used[i].nItemID @0x9e05b7` | ✅ |  |
| 7 | int32 | int32 `used[i].nCount @0x9e05c4` | ✅ |  |
| 8 | int32 | int32 `nNumUsedGem @0x9e063b (loop length prefix)` | ✅ |  |
| 9 | int32 | int32 `gem[i].nItemID @0x9e0648 (id only, no count)` | ✅ |  |
| 10 | byte | byte `bUsedCatalyst @0x9e06c0` | ✅ |  |
| 11 | int32 | int32 `nCatalystItemID @0x9e06d0` | ✅ |  |
| 12 | int32 | int32 `nMesoCost @0x9e0749 (rendered as a loss by the client)` | ✅ |  |

