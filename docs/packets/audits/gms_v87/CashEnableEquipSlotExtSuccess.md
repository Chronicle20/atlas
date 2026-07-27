# CashEnableEquipSlotExtSuccess (← `CCashShop::OnCashItemResult#ENABLE_EQUIP_SLOT_EXT_SUCCESS`)

- **IDA:** 0x4864ad
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_slots.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x6B ENABLE_EQUIP_SLOT_EXT_SUCCESS; op-byte consumed by dispatcher before OnCashItemResEnableEquipSlotExtDone)` | ✅ |  |
| 1 | int16 | int16 `slotIndex; Decode2@0x4864ca` | ✅ |  |
| 2 | int16 | int16 `days; Decode2@0x4864db` | ✅ |  |

