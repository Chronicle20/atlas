# FieldMtsResultCancelSaleItemFailed (← `CITC::OnNormalItemResult#CancelSaleItemFailed`)

- **IDA:** 0x5a4d49
- **Atlas file:** `libs/atlas-packet/field/clientbound/mts_result_reason_modes.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `MTS result mode byte (0x26 CancelSaleItemFailed)` | ✅ |  |
| 1 | byte | byte `Decode1 fail reason -> NoticeFailReason` | ✅ |  |

