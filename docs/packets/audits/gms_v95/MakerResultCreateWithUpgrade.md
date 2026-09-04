# MakerResultCreateWithUpgrade (← `CUserLocal::OnMakerResult#CreateWithUpgrade`)

- **IDA:** 0x9102f0
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/maker_result.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `nResult (read unconditionally; > 1 ends the body)` | ✅ |  |
| 1 | int32 | int32 `nMode (arm selector; this arm is nMode == 2)` | ✅ |  |
| 2 | byte | byte `bNoItemGain @0x910717 (truthy SUPPRESSES the pair below)` | ✅ |  |
| 3 | int32 | int32 `nTargetItemID @0x910732` | ✅ |  |
| 4 | int32 | int32 `nItemCount @0x910746` | ✅ |  |
| 5 | int32 | int32 `nNumUsedItem @0x91080e (loop length prefix)` | ✅ |  |
| 6 | int32 | int32 `used[i].nItemID @0x91082a` | ✅ |  |
| 7 | int32 | int32 `used[i].nCount @0x91083d` | ✅ |  |
| 8 | int32 | int32 `nNumUsedGem @0x9108ef (loop length prefix)` | ✅ |  |
| 9 | int32 | int32 `gem[i].nItemID @0x910904 (id only, no count)` | ✅ |  |
| 10 | byte | byte `bUsedCatalyst @0x9109c6` | ✅ |  |
| 11 | int32 | int32 `nCatalystItemID @0x9109d8` | ✅ |  |
| 12 | int32 | int32 `nMesoCost @0x910aa2 (rendered as a loss by the client)` | ✅ |  |

