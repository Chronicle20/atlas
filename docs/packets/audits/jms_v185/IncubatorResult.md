# IncubatorResult (← `CWvsContext::OnIncubatorResult`)

- **IDA:** 0xb0f30b
- **Atlas file:** `libs/atlas-packet/incubator/clientbound/result.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `itemId` | ✅ |  |
| 1 | int16 | int16 `count` | ✅ |  |

