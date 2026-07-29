# CashSetWishFailed (← `CCashShop::OnCashItemResult#SET_WISH_FAILED`)

- **IDA:** 0x453df4
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x33 SET_WISH_FAILED; op-byte consumed by dispatcher before OnCashItemResSetWishFailed)` | ✅ |  |
| 1 | byte | byte `errorCode (reason byte)` | ✅ |  |

