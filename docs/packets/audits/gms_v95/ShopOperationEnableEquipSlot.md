# ShopOperationEnableEquipSlot (← `CCashShop::OnEnableEquipSlotExt`)

- **IDA:** 0x48e130
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_enable_equip_slot.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `isMaplePoint bool (dwOption==2; pointType)` | ✅ |  |
| 1 | int32 | int32 `nCommSN (serialNumber)` | ✅ |  |

