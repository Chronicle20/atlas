# CashCashShopInventory (← `CCashShop::OnCashItemResult#CashShopInventory`)

- **IDA:** 0x4794f6
- **Atlas file:** `../../libs/atlas-packet/cash/clientbound/shop_inventory.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x58 LOAD_LOCKER_DONE; op-byte consumed by OnCashItemResult dispatcher)` | ✅ |  |
| 1 | int16 | int16 `count (m_aCashItemInfo size)` | ✅ |  |
| 2 | bytes | bytes `55 * count bytes (per GW_CashItemInfo = CashInventoryItem.EncodeBytes, 55 bytes each)` | ✅ |  |
| 3 | int16 | int16 `m_nTrunkCount (storageSlots; *(this+291))` | ✅ |  |
| 4 | int16 | int16 `m_nCharacterSlotCount (characterSlots; *(this+292))` | ✅ |  |
| 5 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |

