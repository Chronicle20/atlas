# CashCashShopPurchaseSuccess (← `CCashShop::OnCashItemResult#CashShopPurchaseSuccess`)

- **IDA:** 0x47ca64
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_inventory.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x5a BUY_DONE; op-byte consumed by dispatcher before OnCashItemResBuyDone)` | ✅ |  |
| 1 | bytes | bytes `55 bytes GW_CashItemInfo (CashInventoryItem.EncodeBytes); DecodeBuffer(pItem, 0x37)` | ✅ |  |

