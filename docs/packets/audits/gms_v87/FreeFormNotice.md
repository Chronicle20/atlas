# FreeFormNotice (← `CWvsContext::OnEntrustedShopCheckResult#FreeFormNotice`)

- **IDA:** 0xabf9ea
- **Atlas file:** `../../libs/atlas-packet/merchant/clientbound/operation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (case 0x12 = FREE_FORM_NOTICE)` | ✅ |  |
| 1 | byte | byte `flag (if 0 return immediately; atlas always encodes 1)` | ✅ |  |
| 2 | string | string `sMsg — message string (only read when flag != 0)` | ✅ |  |

