# CashIncCharacterSlotCountSuccess (← `CCashShop::OnCashItemResult#INC_CHARACTER_SLOT_COUNT_SUCCESS`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_slots.go`
- **Variant:** GMS/v92
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | unresolved `dispatcher-family arm not harvested for gms_v92 (see notes)` | 🚫 | IDA read-order unresolved: dispatcher-family arm not harvested for gms_v92 (see notes) |
| 1 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |

