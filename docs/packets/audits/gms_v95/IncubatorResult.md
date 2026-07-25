# IncubatorResult (← `CWvsContext::OnIncubatorResult`)

- **IDA:** 0xa00380
- **Atlas file:** `libs/atlas-packet/incubator/clientbound/result.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `itemId (v3)` | ✅ |  |
| 1 | int16 | int16 `count (v4)` | ✅ |  |
| 2 | int32 | int32 `nGachaponItemID` | ✅ |  |
| 3 | int32 | int32 `nBonusItemID` | ✅ |  |
| 4 | int32 | int32 `nBonusCount` | ✅ |  |

