# CashCashItemMovedToCashInventory (← `CCashShop::OnCashItemResult#CashItemMovedToCashInventory`)

- **IDA:** 0x48ded7
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_item_moved.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x6d MOVE_S_TO_L_DONE = item moved storage->locker=cash inventory; op-byte consumed by dispatcher before OnCashItemResMoveStoLDone)` | ✅ |  |
| 1 | bytes | bytes `55 bytes GW_CashItemInfo (CashInventoryItem.EncodeBytes); DecodeBuffer(pItem, 0x37)` | ✅ |  |

