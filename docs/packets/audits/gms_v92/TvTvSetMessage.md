# TvTvSetMessage (← `CMapleTVMan::OnSetMessage`)

- **IDA:** 0x603d20
- **Atlas file:** `libs/atlas-packet/tv/clientbound/set_message.go`
- **Variant:** GMS/v92
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | byte | byte `` | ✅ |  |
| 2 | byte | string `` | ❌ | width mismatch |
| 3 | byte | string `` | ❌ | width mismatch |
| 4 | int32 | string `` | ❌ | width mismatch |
| 5 | byte | string `` | ❌ | width mismatch |
| 6 | int32 | string `` | ❌ | width mismatch |
| 7 | byte | string `` | ❌ | width mismatch |
| 8 | int32 | string `` | ❌ | width mismatch |
| 9 | byte | int32 `` | ❌ | width mismatch |
| 10 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 11 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 12 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 13 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 14 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 15 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 16 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 17 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 18 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 19 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 20 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 21 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 22 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 23 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 24 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 25 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 26 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 27 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 28 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 29 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 30 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 31 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 32 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 33 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

