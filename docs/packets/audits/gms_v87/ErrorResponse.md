# ErrorResponse (← `CWvsContext::OnGivePopularityResult#ErrorResponse`)

- **IDA:** 0xab9c24
- **Atlas file:** `libs/atlas-packet/fame/clientbound/response.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (switch dispatch; cases 1-4 = error codes — no additional fields)` | ✅ |  |


## Manual analysis

v87 vs v95/v83: gate confirmed ✅. `OnGivePopularityResult` @ 0xab9c24, cases 1-4: mode byte only, no additional fields. Atlas matches.

Ack: misc-audit Phase 3 v87 on 2026-06-03
