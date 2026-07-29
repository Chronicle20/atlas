# CashLoadWishFailed (← `CCashShop::OnCashItemResult#LOAD_WISH_FAILED`)

- **IDA:** 0x4714b2
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x3C LOAD_WISH_FAILED; op-byte consumed by dispatcher before OnCashItemResLoadWishFailed)` | ✅ |  |
| 1 | byte | byte `errorCode (reason byte)` | ✅ |  |

