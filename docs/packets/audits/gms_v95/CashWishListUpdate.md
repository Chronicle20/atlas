# CashWishListUpdate (← `CCashShop::OnCashItemResult#UPDATE_WISHLIST`)

- **IDA:** 0x494d60
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x62 SET_WISH_DONE; op-byte consumed by dispatcher before OnCashItemResSetWishDone)` | ✅ |  |
| 1 | int32 | bytes `40 bytes = 10 x int32 wishlist SNs (DecodeBuffer(this+wishbuf, 40))` | ✅ |  |

