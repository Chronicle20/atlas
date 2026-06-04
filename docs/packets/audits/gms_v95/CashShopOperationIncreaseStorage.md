# CashShopOperationIncreaseStorage (← `CCashShop::OnIncTrunkCount`)

- **IDA:** 0x48dc70
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_increase_storage.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `isMaplePoint bool (dwOption==2; isPoints)` | ✅ |  |
| 1 | int32 | int32 `dwOption (currency)` | ✅ |  |
| 2 | byte | byte `item bool (0 in this path -> no serialNumber)` | ✅ |  |
| 3 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

