# ReactorHitRequest (← `CReactorPool::FindHitReactor`)

- **IDA:** 0x7356c7
- **Atlas file:** `../../libs/atlas-packet/reactor/serverbound/hit.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `reactorId` | ✅ |  |
| 1 | int32 | int32 `reserved (0)` | ✅ |  |
| 2 | int32 | int32 `stance flag` | ✅ |  |
| 3 | int16 | int16 `tDelay` | ✅ |  |
| 4 | int32 | int32 `reserved (0)` | ✅ |  |

