# CashCashItemMovedToCashInventory (← `CCashShop::OnCashItemResult#CashItemMovedToCashInventory`)

- **IDA:** 0x4948d0
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_item_moved.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x79 MOVE_S_TO_L_DONE; op-byte consumed by OnCashItemResult dispatcher)` | ✅ |  |
| 1 | bytes | bytes `55 bytes GW_CashItemInfo (CashInventoryItem.EncodeBytes); CInPacket::DecodeBuffer(iPacket, v6, 0x37u)` | ✅ |  |

