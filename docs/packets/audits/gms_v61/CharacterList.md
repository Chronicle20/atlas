# CharacterList (← `sub_56688D`)

- **IDA:** 0x56688d
- **Atlas file:** `libs/atlas-packet/character/clientbound/list.go`
- **Variant:** GMS/v61
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `status @0x5668e5` | ✅ |  |
| 1 | byte | byte `count @0x566a4c` | ✅ |  |
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
| 21 | int16 | int16 `` | ✅ |  |
| 22 | byte | int16 `` | ❌ | width mismatch |
| 23 | int32 | int16 `` | ❌ | width mismatch |
| 24 | byte | int16 `` | ❌ | width mismatch |
| 25 | int32 | int16 `` | ❌ | width mismatch |
| 26 | byte | int32 `` | ❌ | width mismatch |
| 27 | int32 | int16 `` | ❌ | width mismatch |
| 28 | byte | int32 `` | ❌ | width mismatch |
| 29 | byte | byte `` | ✅ |  |
| 30 | int32 | byte `` | ❌ | width mismatch |
| 31 | byte | byte `` | ✅ |  |
| 32 | int32 | int32 `` | ✅ |  |
| 33 | int32 | byte `` | ❌ | width mismatch |
| 34 | int32 | int32 `` | ✅ |  |
| 35 | byte | byte `` | ✅ |  |
| 36 | byte | int32 `` | ❌ | width mismatch |
| 37 | int32 | byte `` | ❌ | width mismatch |
| 38 | int32 | int32 `` | ✅ |  |
| 39 | int32 | int32 `` | ✅ |  |
| 40 | int32 | bytes `` | ✅ |  |
| 41 | int32 | byte `rankEnabled @0x566a88` | ❌ | width mismatch |
| 42 | byte | bytes `rank16 @0x566aa3` | ❌ | atlas: short — missing trailing field |
| 43 | byte | int32 `slots @0x566b02` | ❌ | atlas: short — missing trailing field |

