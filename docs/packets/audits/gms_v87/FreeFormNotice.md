# FreeFormNotice (← `CWvsContext::OnEntrustedShopCheckResult#FreeFormNotice`)

- **IDA:** 0xabf9ea
- **Atlas file:** `libs/atlas-packet/merchant/clientbound/operation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (case 0x12 = FREE_FORM_NOTICE)` | ✅ |  |
| 1 | byte | byte `flag (if 0 return immediately; atlas always encodes 1)` | ✅ |  |
| 2 | string | string `sMsg — message string (only read when flag != 0)` | ✅ |  |


## Manual analysis

v87 vs v95/v83: gate confirmed ✅. `OnEntrustedShopCheckResult` @ 0xabf9ea case 0x12: mode + Decode1(flag) + DecodeStr(sMsg, when flag!=0). Atlas matches.

Ack: misc-audit Phase 3 v87 on 2026-06-03
