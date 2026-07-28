# CashPurchaseRecordFailed (← `CCashShop::OnCashItemResult#PURCHASE_RECORD_FAILED`)

- **IDA:** 0x4878dd
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0xa4 PURCHASE_RECORD_FAILED; op-byte consumed by dispatcher before OnCashItemResPurchaseRecordFailed)` | ✅ |  |
| 1 | byte | byte `errorCode (NoticeFailReason reason byte)` | ✅ |  |

