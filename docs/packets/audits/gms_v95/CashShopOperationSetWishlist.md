# CashShopOperationSetWishlist (← `CCashShop::OnSetWish`)

- **IDA:** 0x4837d0
- **Atlas file:** `libs/atlas-packet/cash/serverbound/shop_operation_set_wishlist.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `wishSN[0]` | ✅ |  |
| 1 | byte | int32 `wishSN[1]` | ❌ | atlas: short — missing trailing field |
| 2 | byte | int32 `wishSN[2]` | ❌ | atlas: short — missing trailing field |
| 3 | byte | int32 `wishSN[3]` | ❌ | atlas: short — missing trailing field |
| 4 | byte | int32 `wishSN[4]` | ❌ | atlas: short — missing trailing field |
| 5 | byte | int32 `wishSN[5]` | ❌ | atlas: short — missing trailing field |
| 6 | byte | int32 `wishSN[6]` | ❌ | atlas: short — missing trailing field |
| 7 | byte | int32 `wishSN[7]` | ❌ | atlas: short — missing trailing field |
| 8 | byte | int32 `wishSN[8]` | ❌ | atlas: short — missing trailing field |
| 9 | byte | int32 `wishSN[9]` | ❌ | atlas: short — missing trailing field |

