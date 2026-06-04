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


## Manual analysis

**v83 IDA:** `CWvsContext::OnEntrustedShopCheckResult` @ 0xa27d75, cases 9/10/11/15 — no additional reads after mode byte. v83 has case 11 (string-pool 3474: "unable to open store") not present in v95, but atlas encodes the mode byte only, so there is no wire difference.

**Gate:** None needed — version-agnostic. Gate confirmed correct (✅).


Ack: misc-audit Phase 3 v83 on 2026-06-03
