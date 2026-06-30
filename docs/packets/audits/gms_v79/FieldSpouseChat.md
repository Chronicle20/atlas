# FieldSpouseChat (← `CField::OnCoupleMessage`)

- **IDA:** 0x51d566
- **Atlas file:** `libs/atlas-packet/field/clientbound/spouse_chat.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | string | string `` | ✅ |  |
| 2 | byte | byte `` | ✅ |  |
| 3 | string | string `` | ✅ |  |
| 4 | byte | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 5 | string | byte `` | ❌ | width mismatch |
| 6 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 7 | byte | unresolved `function not found in IDB` | ❌ | atlas: short — missing trailing field |

