# FieldCoconutScore (← `CField_Coconut::OnCoconutScore`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/field/clientbound/coconut_score.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 1 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |

