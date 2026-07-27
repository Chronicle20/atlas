# CashEnableEquipSlotExtSuccess (← `CCashShop::OnCashItemResult#ENABLE_EQUIP_SLOT_EXT_SUCCESS`)

- **IDA:** 0x48d8b1
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_slots.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x69 ENABLE_EQUIP_SLOT_EXT_SUCCESS; op-byte consumed by dispatcher before OnCashItemResEnableEquipSlotExtDone)` | ✅ |  |
| 1 | int16 | int16 `slotIndex; Decode2@0x48d8ce` | ✅ |  |
| 2 | int16 | int16 `days; Decode2@0x48d8df` | ✅ |  |

