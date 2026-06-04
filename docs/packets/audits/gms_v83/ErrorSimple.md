# ErrorSimple (← `CWvsContext::OnEntrustedShopCheckResult#ErrorSimple`)

- **IDA:** 0xa27d75
- **Atlas file:** `../../libs/atlas-packet/merchant/clientbound/operation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (cases 9/10/11/15 — client shows fixed string-pool notice, no further reads; v83 also has case 11 unlike v95)` | ✅ |  |

