# ReactorHit (← `CReactorPool::OnReactorChangeState`)

- **IDA:** 0x6ccd60
- **Atlas file:** `libs/atlas-packet/reactor/clientbound/hit.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `reactorId (dwID)` | ✅ |  |
| 1 | byte | byte `newState` | ✅ |  |
| 2 | int16 | int16 `ptPos.x` | ✅ |  |
| 3 | int16 | int16 `ptPos.y` | ✅ |  |
| 4 | int16 | int16 `tDelay (animation delay)` | ✅ |  |
| 5 | byte | byte `frameDelay (state-transition timing)` | ✅ |  |
| 6 | byte | byte `stance/proper-event index (multiplied by 100)` | ✅ |  |

