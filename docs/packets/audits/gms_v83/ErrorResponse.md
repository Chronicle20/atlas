# ErrorResponse (← `CWvsContext::OnGivePopularityResult#ErrorResponse`)

- **IDA:** 0xa223dc
- **Atlas file:** `../../libs/atlas-packet/fame/clientbound/response.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (switch dispatch; cases 1-4 = error codes — no additional fields)` | ✅ |  |


## Manual analysis

**v83 IDA:** `CWvsContext::OnGivePopularityResult` @ 0xa223dc, cases 1-4 subtraction chain — no additional fields after mode byte. Matches v95 exactly.

**Gate:** None needed — version-agnostic. Gate confirmed correct (✅).


Ack: misc-audit Phase 3 v83 on 2026-06-03
