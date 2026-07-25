# NpcSpawn (← `CNpcPool::OnNpcEnterField`)

- **IDA:** 0x5efdb9
- **Atlas file:** `libs/atlas-packet/npc/clientbound/spawn.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | int32 | int32 `` | ✅ |  |
| 2 | int16 | int16 `` | ✅ |  |
| 3 | int16 | int16 `` | ✅ |  |
| 4 | byte | byte `` | ✅ |  |
| 5 | int16 | int16 `` | ✅ |  |
| 6 | int16 | int16 `` | ✅ |  |
| 7 | int16 | int16 `` | ✅ |  |
| 8 | byte | byte `` | ✅ |  |

