# AuthSuccess (← `CLogin::OnCheckPasswordResult`)

- **IDA:** 0x60d368
- **Atlas file:** `libs/atlas-packet/login/clientbound/auth_success.go`
- **Variant:** GMS/v84
- **Branch depth:** 2
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
| 5 | byte | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 6 | byte | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 7 | byte | int32 `` | ❌ | width mismatch |
| 8 | string | byte `` | ❌ | width mismatch |
| 9 | byte | byte `` | ✅ |  |
| 10 | byte | byte `` | ✅ |  |
| 11 | int64 | byte `` | ❌ | width mismatch |
| 12 | int64 | string `` | ❌ | width mismatch |
| 13 | int32 | byte `` | ❌ | width mismatch |
| 14 | byte | byte `` | ✅ |  |
| 15 | byte | bytes `` | ✅ |  |
| 16 | int64 | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 17 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 18 | byte | unresolved `function not found in IDB` | ❌ | atlas: short — missing trailing field |
| 19 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 20 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 21 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 22 | byte | bytes `` | ❌ | atlas: short — missing trailing field |

