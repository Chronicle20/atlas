# FieldMtsResultSaleCurrentItemToWishFailed (← `CITC::OnNormalItemResult#SaleCurrentItemToWishFailed`)

- **IDA:** 0x5b4be0
- **Atlas file:** `libs/atlas-packet/field/clientbound/mts_operation.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `MTS result mode byte (0x20 SaleCurrentItemToWishFailed)` | ✅ |  |
| 1 | byte | byte `Decode1 fail reason -> reason-keyed StringPool notice` | ✅ |  |

