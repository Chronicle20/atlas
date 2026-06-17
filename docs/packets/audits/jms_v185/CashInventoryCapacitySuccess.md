# CashInventoryCapacitySuccess (← `CCashShop::OnCashItemResult#InventoryCapacitySuccess`)

- **IDA:** 0x48d54e
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x61 INC_SLOT_COUNT_DONE; op-byte consumed by dispatcher before OnCashItemResIncSlotCountDone)` | ✅ |  |
| 1 | byte | byte `nTI (inventoryType); Decode1` | ✅ |  |
| 2 | int16 | int16 `newCount (capacity); Decode2` | ✅ |  |

