# CashShopOperationMoveFromCashInventory (← `CCashShop::OnMoveCashItemLtoS`)

- **IDA:** 0x44ec2c
- **Atlas file:** `libs/atlas-packet/cash/serverbound/shop_operation_move_from_cash_inventory.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |
| 1 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |

