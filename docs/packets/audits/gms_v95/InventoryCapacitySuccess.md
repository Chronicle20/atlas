# InventoryCapacitySuccess (← `CCashShop::OnCashItemResult#InventoryCapacitySuccess`)

- **IDA:** 0x497270
- **Atlas file:** `../../libs/atlas-packet/cash/clientbound/shop_operation_result.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x6D INC_SLOT_COUNT_DONE; op-byte consumed by dispatcher)` | ✅ |  |
| 1 | byte | byte `nTI (inventoryType)` | ✅ |  |
| 2 | int16 | int16 `newCount (capacity)` | ✅ |  |

