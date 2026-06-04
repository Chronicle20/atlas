# OpenShop (← `CWvsContext::OnEntrustedShopCheckResult#OpenShop`)

- **IDA:** 0xb0ee59
- **Atlas file:** `../../libs/atlas-packet/merchant/clientbound/operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (case 7 = OPEN_SHOP; calls SendOpenShopRequest — no further reads)` | ✅ |  |

