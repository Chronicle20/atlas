# FieldMtsResultSaleCurrentItemToWishFailed (← `CITC::OnNormalItemResult#SaleCurrentItemToWishFailed`)

- **IDA:** 0x5a46f0
- **Atlas file:** `libs/atlas-packet/field/clientbound/mts_result_reason_modes.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `MTS result mode byte (0x20 SaleCurrentItemToWishFailed)` | ✅ |  |
| 1 | byte | byte `Decode1 fail reason -> reason-keyed StringPool notice` | ✅ |  |

