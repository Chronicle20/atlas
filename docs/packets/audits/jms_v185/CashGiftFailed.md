# CashGiftFailed (← `CCashShop::OnCashItemResult#GIFT_FAILED`)

- **IDA:** 0x48c2f3
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0xa3 GIFT_FAILED; op-byte consumed by dispatcher before OnCashItemResGiftFailed)` | ✅ |  |
| 1 | byte | byte `errorCode (NoticeFailReason reason byte)` | ✅ |  |

