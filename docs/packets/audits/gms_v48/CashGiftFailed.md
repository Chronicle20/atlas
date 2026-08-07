# CashGiftFailed (← `CCashShop::OnCashItemResult#GIFT_FAILED`)

- **IDA:** 0x454def
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x3C GIFT_FAILED; op-byte consumed by dispatcher before OnCashItemResGiftFailed)` | ✅ |  |
| 1 | byte | byte `errorCode (reason byte)` | ✅ |  |

