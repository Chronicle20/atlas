# FieldItcOperationChangedCategorySub (← `CITC::OnChangedCategorySub`)

- **IDA:** 0x60489d
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (5) @0x6048cf` | ✅ |  |
| 1 | int32 | int32 `category (this+26) @0x6048da` | ✅ |  |
| 2 | int32 | int32 `categorySub @0x6048e6` | ✅ |  |
| 3 | int32 | int32 `page (const 0) @0x6048ef` | ✅ |  |
| 4 | byte | byte `sortType @0x6048fa` | ✅ |  |
| 5 | byte | byte `sortColumn @0x604905` | ✅ |  |
| 6 | int32 | int32 `searchOption (else-branch const 1) @0x604914` | ✅ |  |
| 7 | string | string `searchCondition @0x604954` | ✅ |  |

