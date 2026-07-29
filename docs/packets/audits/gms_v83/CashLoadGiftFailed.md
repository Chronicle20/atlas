# CashLoadGiftFailed (← `CCashShop::OnCashItemResult#LOAD_GIFT_FAILED`)

- **IDA:** 0x4797c0
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x4e LOAD_GIFT_FAILED; op-byte consumed by dispatcher before OnCashItemResLoadGiftFailed)` | ✅ |  |
| 1 | byte | byte `errorCode (NoticeFailReason reason byte)` | ✅ |  |

