# CashWishListLoad (← `CCashShop::OnCashItemResult#LOAD_WISHLIST`)

- **IDA:** 0x47c980
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x52 LOAD_WISH_DONE; op-byte consumed by dispatcher before OnCashItemResLoadWishDone)` | ✅ |  |
| 1 | int32 | bytes `40 bytes = 10 x int32 wishlist SNs (DecodeBuffer(this+wishbuf, 40))` | ✅ |  |

