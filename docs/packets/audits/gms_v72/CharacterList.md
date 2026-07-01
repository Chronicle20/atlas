# CharacterList (← `sub_5B3646`)

- **IDA:** 0x5b3646
- **Atlas file:** `libs/atlas-packet/character/clientbound/list.go`
- **Variant:** GMS/v72
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `status @0x5b369e` | ✅ |  |
| 1 | byte | byte `count @0x5b3808` | ✅ |  |
| 2 | int32 | int32 `` | ✅ |  |
| 3 | bytes | int32 `` | ✅ |  |
| 4 | byte | bytes `` | ✅ |  |
| 5 | byte | byte `` | ✅ |  |
| 6 | int32 | byte `` | ❌ | width mismatch |
| 7 | int32 | byte `` | ❌ | width mismatch |
| 8 | int64 | byte `` | ❌ | width mismatch |
| 9 | byte | int32 `` | ❌ | width mismatch |
| 10 | int32 | int32 `` | ✅ |  |
| 11 | int32 | int32 `` | ✅ |  |
| 12 | int32 | int32 `` | ✅ |  |
| 13 | int16 | bytes `` | ✅ |  |
| 14 | int16 | byte `` | ❌ | width mismatch |
| 15 | int16 | int16 `` | ✅ |  |
| 16 | int16 | int16 `` | ✅ |  |
| 17 | int16 | int16 `` | ✅ |  |
| 18 | int32 | int16 `` | ❌ | width mismatch |
| 19 | int16 | int16 `` | ✅ |  |
| 20 | int32 | int16 `` | ❌ | width mismatch |
| 21 | int32 | int16 `` | ❌ | width mismatch |
| 22 | byte | int16 `` | ❌ | width mismatch |
| 23 | int32 | int16 `` | ❌ | width mismatch |
| 24 | int16 | int16 `` | ✅ |  |
| 25 | int32 | int16 `` | ❌ | width mismatch |
| 26 | byte | int32 `` | ❌ | width mismatch |
| 27 | int32 | int16 `` | ❌ | width mismatch |
| 28 | byte | int32 `` | ❌ | width mismatch |
| 29 | int32 | int32 `` | ✅ |  |
| 30 | byte | byte `` | ✅ |  |
| 31 | byte | int32 `` | ❌ | width mismatch |
| 32 | int32 | byte `` | ❌ | width mismatch |
| 33 | byte | byte `` | ✅ |  |
| 34 | int32 | int32 `` | ✅ |  |
| 35 | int32 | byte `` | ❌ | width mismatch |
| 36 | int32 | int32 `` | ✅ |  |
| 37 | byte | byte `` | ✅ |  |
| 38 | byte | int32 `` | ❌ | width mismatch |
| 39 | int32 | byte `` | ❌ | width mismatch |
| 40 | int32 | int32 `` | ✅ |  |
| 41 | int32 | int32 `` | ✅ |  |
| 42 | int32 | bytes `` | ✅ |  |
| 43 | int32 | byte `rankEnabled @0x5b384d` | ❌ | width mismatch |
| 44 | byte | bytes `rank16 @0x5b3868` | ❌ | atlas: short — missing trailing field |
| 45 | byte | int32 `slots @0x5b38ba` | ❌ | atlas: short — missing trailing field |

