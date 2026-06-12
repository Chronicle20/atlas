# AddCharacterEntry (← `CLogin::OnCreateNewCharacterResult`)

- **IDA:** 0x60f268
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/add_entry.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | int32 | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 2 | bytes | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 3 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 7 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |
| 8 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 9 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 10 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 11 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 12 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 13 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 14 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 15 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 16 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 17 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 18 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 19 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 20 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 21 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 22 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 23 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 24 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 25 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 26 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 27 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 28 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 29 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 30 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 31 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 32 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 33 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 34 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 35 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 36 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 37 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 38 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 39 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 40 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 41 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 42 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 43 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 44 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 45 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

