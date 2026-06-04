# OpenShop (← `CWvsContext::OnEntrustedShopCheckResult#OpenShop`)

- **IDA:** 0xa27d75
- **Atlas file:** `../../libs/atlas-packet/merchant/clientbound/operation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (case 7 = OPEN_SHOP; client calls SendOpenShopRequest — no further reads)` | ✅ |  |

