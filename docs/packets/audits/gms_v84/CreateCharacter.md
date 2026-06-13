# CreateCharacter (← `CLogin::SendNewCharPacket`)

- **IDA:** 0x60cdf0
- **Atlas file:** `libs/atlas-packet/character/serverbound/create.go`
- **Variant:** GMS/v84
- **Branch depth:** 3
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `` | ✅ |  |
| 1 | int32 | int32 `` | ✅ |  |
| 2 | int16 | int32 `` | ❌ | width mismatch |
| 3 | int32 | byte `` | ❌ | width mismatch |
| 4 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 7 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 8 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 9 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 10 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 11 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

