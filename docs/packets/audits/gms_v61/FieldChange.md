# FieldChange (← `CField::SendTransferFieldRequest`)

- **IDA:** 0x4e8f58
- **Atlas file:** `libs/atlas-packet/field/serverbound/change.go`
- **Variant:** GMS/v61
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | int32 | int32 `` | ✅ |  |
| 2 | string | string `` | ✅ |  |
| 3 | int16 | int16 `` | ✅ |  |
| 4 | int16 | int16 `` | ✅ |  |
| 5 | byte | byte `` | ✅ |  |
| 6 | byte | byte `` | ✅ |  |
| 7 | int32 | byte `` | ❌ | width mismatch |
| 8 | int32 | int32 `` | ✅ |  |
| 9 | byte | int32 `` | ❌ | atlas: short — missing trailing field |

