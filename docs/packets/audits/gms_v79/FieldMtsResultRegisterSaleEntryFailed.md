# FieldMtsResultRegisterSaleEntryFailed (← `CITC::OnNormalItemResult#RegisterSaleEntryFailed`)

- **IDA:** 0x57f717
- **Atlas file:** `libs/atlas-packet/field/clientbound/mts_operation.go`
- **Variant:** GMS/v79
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `MTS result mode byte (0x1E RegisterSaleEntryFailed)` | ✅ |  |
| 1 | byte | byte `fail reason byte -> NoticeFailReason` | ✅ |  |
| 2 | int16 | int16 `sale-limit count (reason==0x48 branch only)` | ✅ |  |

