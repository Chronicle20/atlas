# CashShopOperationMoveFromCashInventory (← `CCashShop::OnMoveCashItemLtoS`)

- **IDA:** 0x48411b
- **Atlas file:** `libs/atlas-packet/cash/serverbound/shop_operation_move_from_cash_inventory.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int64 | byte `` | ❌ | width mismatch |
| 1 | byte | bytes `` | ✅ |  |
| 2 | int16 | byte `` | ❌ | width mismatch |
| 3 | byte | int16 `` | ❌ | atlas: short — missing trailing field |

