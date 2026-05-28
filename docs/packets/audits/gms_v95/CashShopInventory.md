# CashShopInventory (← `CCashShop::OnCashItemResult#CashShopInventory`)

- **IDA:** 0x494cb0
- **Atlas file:** `../../libs/atlas-packet/cash/clientbound/shop_inventory.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x58 LOAD_LOCKER_DONE; op-byte consumed by OnCashItemResult dispatcher)` | ✅ |  |
| 1 | int16 | int16 `count (m_aCashItemInfo size)` | ✅ |  |
| 2 | bytes | bytes `55 * count bytes (per GW_CashItemInfo = CashInventoryItem.EncodeBytes, 55 bytes each)` | ✅ |  |
| 3 | int16 | int16 `m_nTrunkCount (storageSlots)` | ✅ |  |
| 4 | int16 | int16 `m_nCharacterSlotCount (characterSlots)` | ✅ |  |
| 5 | byte | int16 `m_nBuyCharacterCount (NOT in atlas - missing trailing short)` | ❌ | atlas: short — missing trailing field |
| 6 | byte | int16 `m_nCharacterCount (NOT in atlas - missing trailing short)` | ❌ | atlas: short — missing trailing field |


> defer: version-gated — missing 2 trailing slot-counter shorts (buyCharacterCount, characterCount); later-GMS additions. See _pending.md "CashShopInventory — missing 2 trailing slot-counter shorts".
