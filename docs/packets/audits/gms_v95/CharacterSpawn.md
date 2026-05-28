# CharacterSpawn (← `CUserPool::OnUserEnterField`)

- **IDA:** 0x94db40
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/spawn.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId (dwCharacterID)` | ✅ |  |
| 1 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 7 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 8 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 9 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 10 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 11 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 12 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 13 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 14 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 15 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 16 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 17 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 18 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 19 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 20 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 21 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 22 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 23 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 24 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 25 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 26 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 27 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 28 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 29 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 30 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 31 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 32 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 33 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 34 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 35 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 36 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 37 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 38 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 39 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 40 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 41 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |
| 42 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 43 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 44 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 45 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 46 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 47 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 48 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 49 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 50 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 51 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 52 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 53 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 54 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 55 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 56 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 57 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 58 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 59 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 60 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

