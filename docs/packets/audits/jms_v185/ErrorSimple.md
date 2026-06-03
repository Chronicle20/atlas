# ErrorSimple (← `CWvsContext::OnEntrustedShopCheckResult#ErrorSimple`)

- **IDA:** 0xb0ee59
- **Atlas file:** `libs/atlas-packet/merchant/clientbound/operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (cases 9/10/11/15 — string pool notice, no further reads; JMS adds case 0xB/0xF vs GMS 9/10/15)` | ✅ |  |

