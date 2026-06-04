# ShopRename (← `CWvsContext::OnEntrustedShopCheckResult#ShopRename`)

- **IDA:** 0xa27d75
- **Atlas file:** `../../libs/atlas-packet/merchant/clientbound/operation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (case 0xE = SHOP_RENAME)` | ✅ |  |
| 1 | byte | byte `success flag (if 0 return early; if 1 show chat-log success message)` | ✅ |  |

