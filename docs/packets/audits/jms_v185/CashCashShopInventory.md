# CashCashShopInventory (← `CCashShop::OnCashItemResult#CashShopInventory`)

- **IDA:** 0x48bcff
- **Atlas file:** `../../libs/atlas-packet/cash/clientbound/shop_inventory.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x58 LOAD_LOCKER_DONE; op-byte consumed by OnCashItemResult dispatcher)` | ✅ |  |
| 1 | int16 | int16 `count (locker item count)` | ✅ |  |
| 2 | bytes | bytes `55 * count bytes (per CashInventoryItem.EncodeBytes)` | ✅ |  |
| 3 | int16 | int16 `m_nTrunkCount (storageSlots; *(this+288))` | ✅ |  |
| 4 | int16 | int16 `m_nCharacterSlotCount (characterSlots; *(this+289))` | ✅ |  |
| 5 | int16 | int16 `m_nBuyCharacterCount (*(this+290)). JMS PRESENT — atlas else-branch (2 shorts) is WRONG; JMS reads 4 shorts like GMS v95` | ✅ |  |
| 6 | int16 | int16 `m_nCharacterCount (*(this+291))` | ✅ |  |

