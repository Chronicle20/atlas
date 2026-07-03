# FieldMtsResultRegisterSaleEntryFailed (← `CITC::OnNormalItemResult#RegisterSaleEntryFailed`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/field/clientbound/mts_operation.go`
- **Variant:** GMS/v48
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 1 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |

