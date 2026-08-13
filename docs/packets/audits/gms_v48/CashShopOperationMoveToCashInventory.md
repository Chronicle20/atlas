# CashShopOperationMoveToCashInventory (← `CCashShop::OnMoveCashItemStoL`)

- **IDA:** 0x44ee3a
- **Atlas file:** `libs/atlas-packet/cash/serverbound/shop_operation_move_to_cash_inventory.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |
| 1 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

