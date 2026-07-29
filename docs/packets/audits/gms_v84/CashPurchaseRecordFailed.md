# CashPurchaseRecordFailed (← `CCashShop::OnCashItemResult#PURCHASE_RECORD_FAILED`)

- **IDA:** 0x47f29a
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x9e PURCHASE_RECORD_FAILED; op-byte consumed by dispatcher before OnCashItemResPurchaseRecordFailed)` | ✅ |  |
| 1 | byte | byte `errorCode (NoticeFailReason reason byte)` | ✅ |  |

