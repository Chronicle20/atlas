# CashCashItemMovedToInventory (← `CCashShop::OnCashItemResult#CashItemMovedToInventory`)

- **IDA:** 0x4866b4
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_item_moved.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x6d MOVE_L_TO_S_DONE = item moved locker->slot=player inventory; op-byte consumed by dispatcher before OnCashItemResMoveLtoSDone)` | ✅ |  |
| 1 | int16 | int16 `nPOS (inventory slot); Decode2` | ✅ |  |
| 2 | byte | bytes `GW_ItemSlotBase::Decode(&pItem) = item slot payload (model.Asset)` | 🔍 | opaque type: model.Asset — register boundary (see opaque registry) |

