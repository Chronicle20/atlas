# FieldClock (← `CField::OnClock`)

- **IDA:** 0x52b920
- **Atlas file:** `libs/atlas-packet/field/clientbound/clock.go`
- **Variant:** GMS/v92
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | int32 | int32 `` | ✅ |  |
| 2 | byte | byte `` | ✅ |  |
| 3 | byte | byte `` | ✅ |  |
| 4 | byte | byte `` | ✅ |  |
| 5 | byte | int32 `` | ❌ | width mismatch |
| 6 | int32 | byte `` | ❌ | width mismatch |
| 7 | byte | int32 `` | ❌ | width mismatch |
| 8 | byte | byte `` | ✅ |  |
| 9 | int32 | byte `` | ❌ | width mismatch |
| 10 | byte | int32 `` | ❌ | atlas: short — missing trailing field |

