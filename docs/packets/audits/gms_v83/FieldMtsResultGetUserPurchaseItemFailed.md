# FieldMtsResultGetUserPurchaseItemFailed (← `CITC::OnNormalItemResult#GetUserPurchaseItemFailed`)

- **IDA:** 0x5a4c2a
- **Atlas file:** `libs/atlas-packet/field/clientbound/mts_operation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `MTS result mode byte (0x22 GetUserPurchaseItemFailed)` | ✅ |  |
| 1 | byte | byte `Decode1 fail reason -> NoticeFailReason` | ✅ |  |

