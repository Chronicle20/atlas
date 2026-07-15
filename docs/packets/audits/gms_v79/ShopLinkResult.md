# ShopLinkResult (← `CWvsContext::OnShopLinkResult`)

- **IDA:** 0x973035
- **Atlas file:** `libs/atlas-packet/merchant/clientbound/shop_link_result.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ⚠️

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

