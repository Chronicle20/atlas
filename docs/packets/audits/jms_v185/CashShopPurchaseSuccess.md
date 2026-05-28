# CashShopPurchaseSuccess (← `CCashShop::OnCashItemResult#CashShopPurchaseSuccess`)

- **IDA:** 0x48c0f0
- **Atlas file:** `../../libs/atlas-packet/cash/clientbound/shop_inventory.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x64 BUY_DONE; op-byte consumed by dispatcher)` | ✅ |  |
| 1 | bytes | bytes `0x37 = 55 bytes (per CashInventoryItem). Matches atlas` | ✅ |  |

