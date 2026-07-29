# CashIncCharacterSlotCountSuccess (← `CCashShop::OnCashItemResult#INC_CHARACTER_SLOT_COUNT_SUCCESS`)

- **IDA:** 0x47ac0a
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_slots.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x64 INC_CHARACTER_SLOT_COUNT_SUCCESS; op-byte consumed by dispatcher before OnCashItemResIncCharacterSlotCountDone)` | ✅ |  |
| 1 | int16 | int16 `slotCount (new absolute m_nCharacterSlotCount); Decode2@0x47ac29` | ✅ |  |

