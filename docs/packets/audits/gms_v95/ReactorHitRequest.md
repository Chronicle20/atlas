# ReactorHitRequest (← `CReactorPool::FindHitReactor`)

- **IDA:** 0x6cd4e0
- **Atlas file:** `libs/atlas-packet/reactor/serverbound/hit.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `reactorId (dwID)` | ✅ |  |
| 1 | int32 | int32 `reserved (0)` | ✅ |  |
| 2 | int32 | int32 `stance flag (v21[13] — combined user-state bits)` | ✅ |  |
| 3 | int32 | int16 `tDelay` | ❌ | width mismatch |
| 4 | int16 | int32 `reserved (0)` | ❌ | width mismatch |
| 5 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

