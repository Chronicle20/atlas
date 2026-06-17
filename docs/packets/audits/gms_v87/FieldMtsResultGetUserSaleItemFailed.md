# FieldMtsResultGetUserSaleItemFailed (← `CITC::OnNormalItemResult#GetUserSaleItemFailed`)

- **IDA:** 0x5d4dd7
- **Atlas file:** `libs/atlas-packet/field/clientbound/mts_result_reason_modes.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `MTS result mode byte (0x24 GetUserSaleItemFailed)` | ✅ |  |
| 1 | byte | byte `Decode1 fail reason -> NoticeFailReason` | ✅ |  |

