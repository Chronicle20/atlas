# CashShopOperationBuyFriendship (← `CCashShop::OnBuyFriendship`)

- **IDA:** 0x491b30
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_buy_friendship.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `sSPW secondary-password string (atlas models leading int birthday - MISMATCH)` | ✅ |  |
| 1 | int32 | int32 `dwOption (option)` | ✅ |  |
| 2 | int32 | int32 `nCommSN (serialNumber)` | ✅ |  |
| 3 | string | string `sGiveTo (name)` | ✅ |  |
| 4 | string | string `sText (message)` | ✅ |  |

