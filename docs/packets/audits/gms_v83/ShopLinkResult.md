# ShopLinkResult (← `CWvsContext::OnShopLinkResult`)

- **IDA:** 0x8a4e7a
- **Atlas file:** `libs/atlas-packet/merchant/clientbound/shop_link_result.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `code (switch: 0 success/1 closed/2 full/3 busy/4 dead/7 no-trade/17 denied/18 maintenance/23 fm-only/default unable). IDB fn CUIShopScanResult::OnShopLinkResult` | ✅ |  |

