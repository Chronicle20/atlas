# FieldItcOperationChangedCategorySub (← `CITC::OnChangedCategorySub`)

- **IDA:** 0x5af6d9
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (5) @0x5af70b` | ✅ |  |
| 1 | int32 | int32 `category @0x5af716` | ✅ |  |
| 2 | int32 | int32 `categorySub @0x5af722` | ✅ |  |
| 3 | int32 | int32 `page (const 0) @0x5af72b` | ✅ |  |
| 4 | byte | byte `sortType @0x5af736` | ✅ |  |
| 5 | byte | byte `sortColumn @0x5af741` | ✅ |  |
| 6 | int32 | int32 `searchOption @0x5af750` | ✅ |  |
| 7 | string | string `searchCondition @0x5af790` | ✅ |  |

