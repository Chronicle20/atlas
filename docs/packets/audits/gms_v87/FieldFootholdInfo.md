# FieldFootholdInfo (← `CField::OnFootHoldInfo`)

- **IDA:** 0x560fec
- **Atlas file:** `libs/atlas-packet/field/clientbound/foothold_info.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | string `` | ❌ | width mismatch |
| 1 | string | int32 `` | ❌ | width mismatch |
| 2 | int32 | int32 `` | ✅ |  |
| 3 | int32 | int32 `` | ✅ |  |
| 4 | int32 | int32 `` | ✅ |  |
| 5 | byte | int32 `` | ❌ | width mismatch |
| 6 | byte | int32 `` | ❌ | width mismatch |
| 7 | string | int32 `` | ❌ | width mismatch |
| 8 | int32 | int32 `` | ✅ |  |
| 9 | int32 | int32 `` | ✅ |  |
| 10 | byte | byte `` | ✅ |  |
| 11 | byte | byte `` | ✅ |  |

