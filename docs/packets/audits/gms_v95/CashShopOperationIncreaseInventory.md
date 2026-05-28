# CashShopOperationIncreaseInventory (← `CCashShop::OnBuySlotInc`)

- **IDA:** 0x491710
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_increase_inventory.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `isMaplePoint bool (dwOption==2; isPoints)` | ✅ |  |
| 1 | int32 | int32 `dwOption (currency)` | ✅ |  |
| 2 | byte | byte `item bool (always 1 in this path)` | ✅ |  |
| 3 | int32 | int32 `nCommSN (serialNumber; item branch)` | ✅ |  |
| 4 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

