# FreeFormNotice (← `CWvsContext::OnEntrustedShopCheckResult#FreeFormNotice`)

- **IDA:** 0xa27d75
- **Atlas file:** `../../libs/atlas-packet/merchant/clientbound/operation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (case 0x12 = FREE_FORM_NOTICE)` | ✅ |  |
| 1 | byte | byte `flag (if 0 return immediately; atlas always encodes 1 @0xa2821b)` | ✅ |  |
| 2 | string | string `sMsg — message string (only read when flag != 0 @0xa28223)` | ✅ |  |


## Manual analysis

**v83 IDA:** `CWvsContext::OnEntrustedShopCheckResult` @ 0xa27d75, case 18 — Decode1(flag), if non-zero then DecodeStr(sMsg). Matches v95 exactly.

**Gate:** None needed — version-agnostic. Gate confirmed correct (✅).


Ack: misc-audit Phase 3 v83 on 2026-06-03
