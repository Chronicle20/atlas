# ErrorSimple (← `CWvsContext::OnEntrustedShopCheckResult#ErrorSimple`)

- **IDA:** 0xabf9ea
- **Atlas file:** `libs/atlas-packet/merchant/clientbound/operation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (cases 9/10/11/15 — client shows fixed string-pool notice, no further reads)` | ✅ |  |


## Manual analysis

v87 vs v95/v83: gate confirmed ✅. `OnEntrustedShopCheckResult` @ 0xabf9ea cases 9/10/11/15: mode byte + string-pool notice, no wire payload. Atlas matches.

Ack: misc-audit Phase 3 v87 on 2026-06-03
