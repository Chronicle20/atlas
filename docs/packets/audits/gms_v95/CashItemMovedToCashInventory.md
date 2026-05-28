# CashItemMovedToCashInventory (← `CCashShop::OnCashItemResult#CashItemMovedToCashInventory`)

- **IDA:** 0x495050
- **Atlas file:** `../../libs/atlas-packet/cash/clientbound/shop_item_moved.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x77 MOVE_L_TO_S_DONE; op-byte consumed by dispatcher)` | ✅ |  |
| 1 | bytes | bytes `55 bytes GW_CashItemInfo (CashInventoryItem.EncodeBytes)` | ✅ |  |

