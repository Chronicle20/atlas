# CashCashShopPurchaseSuccess (← `CCashShop::OnCashItemResult#CashShopPurchaseSuccess`)

- **IDA:** 0x484fed
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_inventory.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x5c BUY_DONE; op-byte consumed by dispatcher before OnCashItemResBuyDone)` | ✅ |  |
| 1 | bytes | bytes `55 bytes GW_CashItemInfo (CashInventoryItem.EncodeBytes); DecodeBuffer(pItem, 0x37)` | ✅ |  |

