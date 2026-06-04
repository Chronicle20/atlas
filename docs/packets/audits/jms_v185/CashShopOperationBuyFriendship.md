# CashShopOperationBuyFriendship (← `CCashShop::OnBuyFriendship`)

- **IDA:** 0x481184
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_buy_friendship.go`
- **Variant:** JMS/v185
- **Branch depth:** 3
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `s (SPW). op-byte 0x24 (NOT GMS 0x25)` | ✅ |  |
| 1 | int32 | int32 `nCommSN (serialNumber). JMS has NO option int` | ✅ |  |
| 2 | string | string `v36 (recipient name)` | ✅ |  |
| 3 | string | string `v33 (message)` | ✅ |  |

