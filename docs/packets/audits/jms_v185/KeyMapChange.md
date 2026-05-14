# KeyMapChange (← `CFuncKeyMappedMan::SaveFuncKeyMap`)

- **IDA:** 0x5e7b48
- **Atlas file:** `libs/atlas-packet/character/serverbound/key_map_change.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `mode (0 = key mapping change)` | ✅ |  |
| 1 | int32 | int32 `count (number of changed key slot indices)` | ✅ |  |
| 2 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

