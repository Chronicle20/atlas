# DropSpawn (← `CDropPool::OnDropEnterField`)

- **IDA:** 0x527b4c
- **Atlas file:** `libs/atlas-packet/drop/clientbound/spawn.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `nEnterType` | ✅ |  |
| 1 | int32 | int32 `dwDropID` | ✅ |  |
| 2 | byte | byte `nIsMoney` | ✅ |  |
| 3 | int32 | int32 `itemId or meso` | ✅ |  |
| 4 | byte | int32 `ownerCharId` | ❌ | width mismatch |
| 5 | int32 | byte `dropType` | ❌ | width mismatch |
| 6 | int32 | int16 `drop x` | ❌ | width mismatch |
| 7 | byte | int16 `drop y` | ❌ | width mismatch |
| 8 | int16 | int32 `sourceObjectId` | ❌ | width mismatch |
| 9 | int16 | int16 `sourceX — gated nEnterType != 2` | ✅ |  |
| 10 | int32 | int16 `sourceY — gated nEnterType != 2` | ❌ | width mismatch |
| 11 | int16 | int16 `tDelay — gated nEnterType != 2` | ✅ |  |
| 12 | int16 | bytes `cashItemSN (8 bytes) — gated !isMoney` | ❌ | width mismatch |
| 13 | int16 | byte `questId / pre-pet flag` | ❌ | width mismatch |
| 14 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |
| 15 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

