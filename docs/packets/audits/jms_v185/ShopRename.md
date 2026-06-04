# ShopRename (← `CWvsContext::OnEntrustedShopCheckResult#ShopRename`)

- **IDA:** 0xb0ee59
- **Atlas file:** `../../libs/atlas-packet/merchant/clientbound/operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (case 0xE = SHOP_RENAME)` | ✅ |  |
| 1 | byte | byte `success flag (if 0 shows fail msg; if 1 shows success)` | ✅ |  |

