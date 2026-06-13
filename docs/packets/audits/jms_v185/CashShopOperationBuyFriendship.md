# CashShopOperationBuyFriendship (← `CCashShop::OnBuyFriendship`)

- **IDA:** 0x481184
- **Atlas file:** `libs/atlas-packet/cash/serverbound/shop_operation_buy_friendship.go`
- **Variant:** JMS/v185
- **Branch depth:** 3
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | byte `leading sub-action byte (task-081 off-by-one remediation 2026-06-10)` | ❌ | width mismatch |
| 1 | int32 | string `s (SPW). op-byte 0x24 (NOT GMS 0x25)` | ❌ | width mismatch |
| 2 | string | int32 `nCommSN (serialNumber). JMS has NO option int` | ❌ | width mismatch |
| 3 | string | string `v36 (recipient name)` | ✅ |  |
| 4 | byte | string `v33 (message)` | ❌ | atlas: short — missing trailing field |

