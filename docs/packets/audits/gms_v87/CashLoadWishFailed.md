# CashLoadWishFailed (← `CCashShop::OnCashItemResult#LOAD_WISH_FAILED`)

- **IDA:** 0x484f49
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x55 LOAD_WISH_FAILED; op-byte consumed by dispatcher before OnCashItemResLoadWishFailed)` | ✅ |  |
| 1 | byte | byte `errorCode (NoticeFailReason reason byte)` | ✅ |  |

