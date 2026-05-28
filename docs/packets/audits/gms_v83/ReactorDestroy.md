# ReactorDestroy (← `CReactorPool::OnReactorLeaveField`)

- **IDA:** 0x73551f
- **Atlas file:** `../../libs/atlas-packet/reactor/clientbound/destroy.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `reactorId` | ✅ |  |
| 1 | byte | byte `finalState` | ✅ |  |
| 2 | int16 | int16 `ptPos.x` | ✅ |  |
| 3 | int16 | int16 `ptPos.y` | ✅ |  |

