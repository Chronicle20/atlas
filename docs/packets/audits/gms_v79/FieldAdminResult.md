# FieldAdminResult (← `CField::OnAdminResult`)

- **IDA:** 0x52075c
- **Atlas file:** `libs/atlas-packet/field/clientbound/admin_result.go`
- **Variant:** GMS/v79
- **Branch depth:** 3
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | string | byte `` | ❌ | width mismatch |
| 2 | byte | byte `` | ✅ |  |
| 3 | byte | byte `` | ✅ |  |
| 4 | byte | int32 `` | ❌ | width mismatch |
| 5 | int32 | byte `` | ❌ | width mismatch |
| 6 | byte | byte `` | ✅ |  |
| 7 | byte | string `` | ❌ | width mismatch |
| 8 | string | string `` | ✅ |  |
| 9 | string | string `` | ✅ |  |
| 10 | string | byte `` | ❌ | width mismatch |
| 11 | byte | byte `` | ✅ |  |
| 12 | byte | byte `` | ✅ |  |
| 13 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

