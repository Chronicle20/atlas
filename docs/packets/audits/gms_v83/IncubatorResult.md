# IncubatorResult (← `CWvsContext::OnIncubatorResult`)

- **IDA:** 0xa28298
- **Atlas file:** `libs/atlas-packet/incubator/clientbound/result.go`
- **Variant:** GMS/v83
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `itemId (v2)` | ✅ |  |
| 1 | int16 | int16 `count (a2a)` | ✅ |  |

