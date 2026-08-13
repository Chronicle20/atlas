# FieldAdminResult (← `CField::OnAdminResult`)

- **IDA:** 0x536080
- **Atlas file:** `libs/atlas-packet/field/clientbound/admin_result.go`
- **Variant:** GMS/v92
- **Branch depth:** 3
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | byte | byte `` | ✅ |  |
| 2 | byte | byte `` | ✅ |  |
| 3 | byte | byte `` | ✅ |  |
| 4 | string | string `` | ✅ |  |
| 5 | string | string `` | ✅ |  |
| 6 | string | string `` | ✅ |  |
| 7 | byte | byte `` | ✅ |  |
| 8 | byte | byte `` | ✅ |  |
| 9 | byte | byte `` | ✅ |  |
| 10 | int32 | int32 `` | ✅ |  |
| 11 | byte | unresolved `packet var passed to unresolved/indirect call; hand-trace` | 🚫 | IDA read-order unresolved: packet var passed to unresolved/indirect call; hand-trace |
| 12 | byte | byte `` | ✅ |  |
| 13 | string | byte `` | ❌ | width mismatch |
| 14 | string | string `` | ✅ |  |
| 15 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 16 | byte | string `` | ❌ | atlas: short — missing trailing field |

