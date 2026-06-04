# ShopSearch (← `CWvsContext::OnEntrustedShopCheckResult#ShopSearch`)

- **IDA:** 0xabf9ea
- **Atlas file:** `../../libs/atlas-packet/merchant/clientbound/operation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (case 0xD = SHOP_SEARCH)` | ✅ |  |
| 1 | int32 | int32 `dwSearchedShop (stored into CUIMiniMap::m_dwSearchedShop)` | ✅ |  |

