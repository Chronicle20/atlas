# CashEnableEquipSlotExtSuccess (← `CCashShop::OnCashItemResult#ENABLE_EQUIP_SLOT_EXT_SUCCESS`)

- **IDA:** 0x473c2c
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_slots.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x5E ENABLE_EQUIP_SLOT_EXT_SUCCESS; op-byte consumed by dispatcher before OnCashItemResEnableEquipSlotExtDone)` | ✅ |  |
| 1 | int16 | int16 `slotIndex (indexes aEquipExtExpire[v3]/aEquipped2[v3+59+13])` | ✅ |  |
| 2 | int16 | int16 `days (passed to Util::FTAddDay as day-count)` | ✅ |  |

