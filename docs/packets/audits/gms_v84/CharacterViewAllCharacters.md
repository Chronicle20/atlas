# CharacterViewAllCharacters (← `CLogin::OnViewAllCharResult#CharacterViewAllCharacters`)

- **IDA:** 0x60ffe8
- **Atlas file:** `libs/atlas-packet/character/clientbound/view_all.go`
- **Variant:** GMS/v84
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | byte | byte `` | ✅ |  |
| 2 | byte | byte `` | ✅ |  |
| 3 | int32 | byte `` | ❌ | width mismatch |
| 4 | bytes | bytes `` | ✅ |  |
| 5 | byte | int32 `` | ❌ | width mismatch |
| 6 | byte | int32 `` | ❌ | width mismatch |
| 7 | int32 | byte `` | ❌ | width mismatch |
| 8 | int32 | string `` | ❌ | width mismatch |
| 9 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |
| 10 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 11 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 12 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 13 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 14 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 15 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 16 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 17 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 18 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 19 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 20 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 21 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 22 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 23 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 24 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 25 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 26 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 27 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 28 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 29 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 30 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 31 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 32 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 33 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 34 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 35 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 36 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 37 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 38 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 39 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 40 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 41 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 42 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 43 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 44 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 45 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 46 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 47 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

