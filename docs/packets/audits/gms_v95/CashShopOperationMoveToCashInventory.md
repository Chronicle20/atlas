# CashShopOperationMoveToCashInventory (← `CCashShop::OnMoveCashItemStoL`)

- **IDA:** 0x482b50
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_move_to_cash_inventory.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int64 | bytes `liSN 8 bytes (serialNumber uint64)` | ❌ | width mismatch |
| 1 | byte | byte `nTI (inventoryType)` | ✅ |  |

