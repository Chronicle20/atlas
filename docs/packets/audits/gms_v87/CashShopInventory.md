# CashShopInventory (← `CCashShop::OnCashItemResult#CashShopInventory`)

- **IDA:** 0x484c1d
- **Atlas file:** `../../libs/atlas-packet/cash/clientbound/shop_inventory.go`
- **Variant:** GMS/v87
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x58 LOAD_LOCKER_DONE; op-byte consumed by OnCashItemResult dispatcher before OnCashItemResLoadLockerDone@0x484c1d)` | ✅ |  |
| 1 | int16 | int16 `count (m_aCashItemInfo size, v3)` | ✅ |  |
| 2 | bytes | bytes `55 * count bytes (per GW_CashItemInfo = CashInventoryItem.EncodeBytes, 55 bytes each)` | ✅ |  |
| 3 | int16 | int16 `m_nTrunkCount (storageSlots; *(this+1168))` | ✅ |  |
| 4 | int16 | int16 `m_nCharacterSlotCount (characterSlots; *(this+1172)). v87 reads ONLY these 2 trailing shorts — m_nBuyCharacterCount + m_nCharacterCount are v95-only; >=95 gate CORRECT.` | ✅ |  |

