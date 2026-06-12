# FieldChange (← `CField::SendTransferFieldRequest`)

- **IDA:** 0x56d75a
- **Atlas file:** `../../libs/atlas-packet/field/serverbound/change.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `fieldKey (get_field()->m_bFieldKey @line33)` | ✅ |  |
| 1 | int32 | int32 `dwTargetField / targetId (@line34)` | ✅ |  |
| 2 | string | string `sPortal / portalName (@line40; empty ZXString when NULL)` | ✅ |  |
| 3 | int16 | int16 `x — only when sPortal!=NULL (@line44)` | ✅ |  |
| 4 | int16 | int16 `y — only when sPortal!=NULL (@line46)` | ✅ |  |
| 5 | byte | byte `unused (constant 0 @line48)` | ✅ |  |
| 6 | byte | byte `bPremium / premium (@line49)` | ✅ |  |
| 7 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 8 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

