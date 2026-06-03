# ShopSearch (← `CWvsContext::OnEntrustedShopCheckResult#ShopSearch`)

- **IDA:** 0xa27d75
- **Atlas file:** `../../libs/atlas-packet/merchant/clientbound/operation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (case 0xD = SHOP_SEARCH)` | ✅ |  |
| 1 | int32 | int32 `dwSearchedShop (stored into CUIMiniMap::m_dwSearchedShop @0xa2812d)` | ✅ |  |


## Manual analysis

**v83 IDA:** `CWvsContext::OnEntrustedShopCheckResult` @ 0xa27d75, case 13 — Decode4(dwSearchedShop). Matches v95 exactly.

**Gate:** None needed — version-agnostic. Gate confirmed correct (✅).


Ack: misc-audit Phase 3 v83 on 2026-06-03
