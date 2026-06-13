# CashShopOperationMoveToCashInventory (← `CCashShop::OnMoveCashItemStoL`)

- **IDA:** 0x4842f9
- **Atlas file:** `libs/atlas-packet/cash/serverbound/shop_operation_move_to_cash_inventory.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int64 | byte `` | ❌ | width mismatch |
| 1 | byte | bytes `` | ✅ |  |
| 2 | byte | byte `` | ❌ | atlas: short — missing trailing field |

