# Request (← `CLogin::SendCheckPasswordPacket`)

- **IDA:** 0x5d2190
- **Atlas file:** `libs/atlas-packet/login/serverbound/request.go`
- **Variant:** GMS/v92
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | byte `` | ❌ | width mismatch |
| 1 | string | int32 `` | ❌ | width mismatch |
| 2 | bytes | string `` | ❌ | width mismatch |
| 3 | int32 | string `` | ❌ | width mismatch |
| 4 | byte | bytes `` | ✅ |  |
| 5 | byte | int32 `` | ❌ | width mismatch |
| 6 | byte | byte `` | ✅ |  |
| 7 | int32 | byte `` | ❌ | width mismatch |
| 8 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 9 | byte | int32 `` | ❌ | atlas: short — missing trailing field |

