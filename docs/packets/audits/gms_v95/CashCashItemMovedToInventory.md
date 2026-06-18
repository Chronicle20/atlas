# CashCashItemMovedToInventory (← `CCashShop::OnCashItemResult#CashItemMovedToInventory`)

- **IDA:** 0x495050
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_item_moved.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x77 MOVE_L_TO_S_DONE; op-byte consumed by OnCashItemResult dispatcher)` | ✅ |  |
| 1 | int16 | int16 `nPOS (inventory slot); nPOS = CInPacket::Decode2(iPacket)` | ✅ |  |
| 2 | byte | bytes `GW_ItemSlotBase::Decode(&pItem, iPacket) = item slot payload (model.Asset)` | 🔍 | opaque type: model.Asset — register boundary (see opaque registry) |

