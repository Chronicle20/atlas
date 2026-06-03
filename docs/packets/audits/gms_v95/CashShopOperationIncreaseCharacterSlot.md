# CashShopOperationIncreaseCharacterSlot (← `CCashShop::OnIncCharacterSlotCount`)

- **IDA:** 0x48dec0
- **Atlas file:** `libs/atlas-packet/cash/serverbound/shop_operation_increase_character_slot.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `isMaplePoint bool (dwOption==2; isPoints)` | ✅ |  |
| 1 | int32 | int32 `dwOption (currency)` | ✅ |  |
| 2 | int32 | int32 `nCommSN (serialNumber)` | ✅ |  |

