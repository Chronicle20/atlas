# Use (← `CWvsContext::SendMapTransferItemUseRequest`)

- **IDA:** 0xa0a3bb
- **Atlas file:** `libs/atlas-packet/teleportrock/serverbound/use.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `` | ✅ |  |
| 1 | int32 | int32 `` | ✅ |  |
| 2 | byte | int32 `` | ❌ | width mismatch |
| 3 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

