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


## Manual analysis

**v83 IDA:** `CWvsContext::OnEntrustedShopCheckResult` @ 0xa27d75, case 7 — returns immediately after mode byte. Matches v95 exactly.

**Gate:** None needed — version-agnostic. Gate confirmed correct (✅).


Ack: misc-audit Phase 3 v83 on 2026-06-03
