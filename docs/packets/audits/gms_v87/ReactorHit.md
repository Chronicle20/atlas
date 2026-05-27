# ReactorHit (← `CReactorPool::OnReactorChangeState`)

- **IDA:** 0x77aea2
- **Atlas file:** `libs/atlas-packet/reactor/clientbound/hit.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `reactorId` | ✅ |  |
| 1 | byte | byte `newState` | ✅ |  |
| 2 | int16 | int16 `ptPos.x` | ✅ |  |
| 3 | int16 | int16 `ptPos.y` | ✅ |  |
| 4 | int16 | int16 `tDelay` | ✅ |  |
| 5 | byte | byte `frameDelay` | ✅ |  |
| 6 | byte | byte `stance` | ✅ |  |

