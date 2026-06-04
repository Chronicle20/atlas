# ShopSearch (← `CWvsContext::OnEntrustedShopCheckResult#ShopSearch`)

- **IDA:** 0xabf9ea
- **Atlas file:** `libs/atlas-packet/merchant/clientbound/operation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (case 0xD = SHOP_SEARCH)` | ✅ |  |
| 1 | int32 | int32 `dwSearchedShop (stored into CUIMiniMap::m_dwSearchedShop)` | ✅ |  |


## Manual analysis

v87 vs v95/v83: gate confirmed ✅. `OnEntrustedShopCheckResult` @ 0xabf9ea case 0xD: mode + Decode4(dwSearchedShop). Atlas matches.

Ack: misc-audit Phase 3 v87 on 2026-06-03
