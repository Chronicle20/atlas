# DropSpawn (← `CDropPool::OnDropEnterField`)

- **IDA:** 0x516670
- **Atlas file:** `libs/atlas-packet/drop/clientbound/spawn.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `nEnterType (v3 — 0=disappear, 1=fresh, 2=existing-on-map, 3/4=animations)` | ✅ |  |
| 1 | int32 | int32 `dwDropID (drop id)` | ✅ |  |
| 2 | byte | byte `nIsMoney (atlas: isMeso bool)` | ✅ |  |
| 3 | int32 | int32 `itemId or meso amount (atlas: meso if isMeso else itemId)` | ✅ |  |
| 4 | byte | int32 `ownerCharId (owner)` | ❌ | width mismatch |
| 5 | int32 | byte `dropType` | ❌ | width mismatch |
| 6 | int32 | int16 `drop x` | ❌ | width mismatch |
| 7 | byte | int16 `drop y` | ❌ | width mismatch |
| 8 | int16 | int32 `sourceObjectId (dropperId — mob or character)` | ❌ | width mismatch |
| 9 | int16 | int16 `sourceX — gated nEnterType != 2` | ✅ |  |
| 10 | int32 | int16 `sourceY — gated nEnterType != 2` | ❌ | width mismatch |
| 11 | int16 | int16 `tDelay — gated nEnterType != 2` | ✅ |  |
| 12 | int16 | bytes `cashItemSN (8 bytes _FILETIME-like; atlas writes WriteInt64(-1)) — gated !isMoney` | ❌ | width mismatch |
| 13 | int16 | byte `questId / pre-pet flag (atlas: !characterDrop bool)` | ❌ | width mismatch |
| 14 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |
| 15 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

