# CashSetWishFailed (← `CCashShop::OnCashItemResult#SET_WISH_FAILED`)

- **IDA:** 0x484fce
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x5b SET_WISH_FAILED; op-byte consumed by dispatcher before OnCashItemResSetWishFailed)` | ✅ |  |
| 1 | byte | byte `errorCode (NoticeFailReason reason byte)` | ✅ |  |

