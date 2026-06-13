# MonsterStatSet (← `CMob::OnStatSet`)

- **IDA:** 0x682603
- **Atlas file:** `libs/atlas-packet/monster/clientbound/stat.go`
- **Variant:** GMS/v84
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | bytes `` | ✅ |  |
| 1 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 2 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 3 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 4 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 5 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 6 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 7 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 8 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 9 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 10 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 11 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 12 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 13 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 14 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 15 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 16 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 17 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 18 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 19 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 20 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |

