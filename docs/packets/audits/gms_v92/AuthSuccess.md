# AuthSuccess (← `CLogin::OnCheckPasswordResult`)

- **IDA:** 0x5d2dc0
- **Atlas file:** `libs/atlas-packet/login/clientbound/auth_success.go`
- **Variant:** GMS/v92
- **Branch depth:** 3
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | byte | byte `` | ✅ |  |
| 2 | int32 | int32 `` | ✅ |  |
| 3 | int32 | byte `` | ❌ | width mismatch |
| 4 | byte | bytes `` | ✅ |  |
| 5 | byte | int32 `` | ❌ | width mismatch |
| 6 | byte | byte `` | ✅ |  |
| 7 | byte | byte `` | ✅ |  |
| 8 | string | byte `` | ❌ | width mismatch |
| 9 | byte | byte `` | ✅ |  |
| 10 | byte | string `` | ❌ | width mismatch |
| 11 | int64 | byte `` | ❌ | width mismatch |
| 12 | int64 | byte `` | ❌ | width mismatch |
| 13 | int32 | bytes `` | ✅ |  |
| 14 | byte | bytes `` | ✅ |  |
| 15 | byte | int32 `` | ❌ | width mismatch |
| 16 | int64 | byte `` | ❌ | width mismatch |
| 17 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 18 | byte | bytes `` | ❌ | atlas: short — missing trailing field |

