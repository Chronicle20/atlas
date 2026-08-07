# CashSetWishFailed (← `CCashShop::OnCashItemResult#SET_WISH_FAILED`)

- **IDA:** 0x471537
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x42 SET_WISH_FAILED; op-byte consumed by dispatcher before OnCashItemResSetWishFailed)` | ✅ |  |
| 1 | byte | byte `errorCode (reason byte)` | ✅ |  |

