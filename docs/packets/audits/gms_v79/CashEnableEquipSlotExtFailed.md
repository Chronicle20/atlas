# CashEnableEquipSlotExtFailed (← `CCashShop::OnCashItemResult#ENABLE_EQUIP_SLOT_EXT_FAILED`)

- **IDA:** 0x473df3
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x5F ENABLE_EQUIP_SLOT_EXT_FAILED; op-byte consumed by dispatcher before OnCashItemResEnableEquipSlotExtFailed)` | ✅ |  |
| 1 | byte | byte `errorCode (reason byte)` | ✅ |  |

