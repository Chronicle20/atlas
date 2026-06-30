# FieldMultiChat (← `CField::OnGroupMessage`)

- **IDA:** 0x51d328
- **Atlas file:** `libs/atlas-packet/field/clientbound/multi.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | string | string `` | ✅ |  |
| 2 | string | string `` | ✅ |  |
| 3 | byte | unresolved `function not found in IDB` | ❌ | atlas: short — missing trailing field |

