# DropSpawn (← `CDropPool::OnDropEnterField`)

- **IDA:** 0x505900
- **Atlas file:** `../../libs/atlas-packet/drop/clientbound/spawn.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `nEnterType` | ✅ |  |
| 1 | int32 | int32 `dwDropID` | ✅ |  |
| 2 | byte | byte `nIsMoney` | ✅ |  |
| 3 | int32 | int32 `itemId or meso` | ✅ |  |
| 4 | int32 | int32 `ownerCharId` | ✅ |  |
| 5 | byte | byte `dropType` | ✅ |  |
| 6 | int16 | int16 `drop x` | ✅ |  |
| 7 | int16 | int16 `drop y` | ✅ |  |
| 8 | int32 | int32 `sourceObjectId` | ✅ |  |
| 9 | int16 | int16 `sourceX — gated nEnterType != 2` | ✅ |  |
| 10 | int16 | int16 `sourceY — gated nEnterType != 2` | ✅ |  |
| 11 | int16 | int16 `tDelay — gated nEnterType != 2` | ✅ |  |
| 12 | int64 | bytes `cashItemSN (8 bytes) — gated !isMoney` | ❌ | width mismatch |
| 13 | byte | byte `questId / pre-pet flag` | ✅ |  |

