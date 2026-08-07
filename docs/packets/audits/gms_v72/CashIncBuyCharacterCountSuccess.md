# CashIncBuyCharacterCountSuccess (← `CCashShop::OnCashItemResult#INC_BUY_CHARACTER_COUNT_SUCCESS`)

- **IDA:** 0x472967
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_slots.go`
- **Variant:** GMS/v72
- **Branch depth:** 1
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x52 INC_BUY_CHARACTER_COUNT_SUCCESS; op-byte consumed by dispatcher before OnCashItemResIncBuyCharacterCountDone)` | ✅ |  |
| 1 | int16 | int16 `slotIndex; Decode2 (v72 LEGACY shape, materially different from MODERN bare-counter arm — task-183 Wave 3 task-3.4-legacy-misc-report.md)` | ✅ |  |
| 2 | byte | bytes `GW_ItemSlotBase::Decode(&pItem) = item slot payload (model.Asset) — consumed to find-and-remove the matching locker entry` | 🔍 | opaque type: model.Asset — register boundary (see opaque registry) |
| 3 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |

