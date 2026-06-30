# DropSpawn (← `CDropPool::OnDropEnterField`)

- **IDA:** 0x4f25ff
- **Atlas file:** `libs/atlas-packet/drop/clientbound/spawn.go`
- **Variant:** GMS/v79
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | int32 | int32 `` | ✅ |  |
| 2 | byte | byte `` | ✅ |  |
| 3 | int32 | int32 `` | ✅ |  |
| 4 | int32 | int32 `` | ✅ |  |
| 5 | byte | byte `` | ✅ |  |
| 6 | int16 | int16 `` | ✅ |  |
| 7 | int16 | int16 `` | ✅ |  |
| 8 | int32 | int32 `` | ✅ |  |
| 9 | int16 | int16 `` | ✅ |  |
| 10 | int16 | int16 `` | ✅ |  |
| 11 | int16 | int16 `` | ✅ |  |
| 12 | int64 | bytes `` | ✅ |  |
| 13 | byte | byte `` | ✅ |  |

