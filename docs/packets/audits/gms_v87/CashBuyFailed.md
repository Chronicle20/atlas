# CashBuyFailed (← `CCashShop::OnCashItemResult#BUY_FAILED`)

- **IDA:** 0x48515e
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x5d BUY_FAILED; op-byte consumed by dispatcher before OnCashItemResBuyFailed)` | ✅ |  |
| 1 | byte | byte `errorCode (NoticeFailReason reason byte)` | ✅ |  |

