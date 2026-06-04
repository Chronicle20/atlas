# CashShopOperationBuyFriendship (← `CCashShop::OnBuyFriendship`)

- **IDA:** 0x481184
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_buy_friendship.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | string `s (SPW). op-byte 0x24 (NOT GMS 0x25)` | ❌ | atlas: short — missing trailing field |
| 1 | byte | int32 `nCommSN (serialNumber). JMS has NO option int` | ❌ | atlas: short — missing trailing field |
| 2 | byte | string `v36 (recipient name)` | ❌ | atlas: short — missing trailing field |
| 3 | byte | string `v33 (message)` | ❌ | atlas: short — missing trailing field |

