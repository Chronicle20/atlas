# FieldFootholdInfo (← `CField::OnFootHoldInfo`)

- **IDA:** 0x53a810
- **Atlas file:** `libs/atlas-packet/field/clientbound/foothold_info.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | string | string `` | ✅ |  |
| 2 | int32 | int32 `` | ✅ |  |
| 3 | int32 | int32 `` | ✅ |  |
| 4 | int32 | int32 `` | ✅ |  |
| 5 | byte | int32 `` | ❌ | width mismatch |
| 6 | byte | int32 `` | ❌ | width mismatch |
| 7 | string | int32 `` | ❌ | width mismatch |
| 8 | int32 | int32 `` | ✅ |  |
| 9 | int32 | int32 `` | ✅ |  |
| 10 | byte | int32 `` | ❌ | width mismatch |
| 11 | byte | int32 `` | ❌ | width mismatch |
| 12 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 13 | byte | byte `` | ❌ | atlas: short — missing trailing field |

