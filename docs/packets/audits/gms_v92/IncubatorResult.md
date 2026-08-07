# IncubatorResult (← `CWvsContext::OnIncubatorResult`)

- **IDA:** 0x9d4d10
- **Atlas file:** `libs/atlas-packet/incubator/clientbound/result.go`
- **Variant:** GMS/v92
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | int16 | int16 `` | ✅ |  |
| 2 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 3 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 4 | byte | int32 `` | ❌ | atlas: short — missing trailing field |

