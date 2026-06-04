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

