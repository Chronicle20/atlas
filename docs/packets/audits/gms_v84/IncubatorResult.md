# IncubatorResult (← `CWvsContext::OnIncubatorResult`)

- **IDA:** 0xa73a5b
- **Atlas file:** `libs/atlas-packet/incubator/clientbound/result.go`
- **Variant:** GMS/v84
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `itemId (v1)` | ✅ |  |
| 1 | int16 | int16 `count (v32)` | ✅ |  |

