# CashShopOperationMoveFromCashInventory (← `CCashShop::OnMoveCashItemLtoS`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/cash/serverbound/shop_operation_move_from_cash_inventory.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int64 | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 1 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |

