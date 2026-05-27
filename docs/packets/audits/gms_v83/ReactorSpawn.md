# ReactorSpawn (← `CReactorPool::OnReactorEnterField`)

- **IDA:** 0x735127
- **Atlas file:** `libs/atlas-packet/reactor/clientbound/spawn.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwID` | ✅ |  |
| 1 | int32 | int32 `dwTemplateID` | ✅ |  |
| 2 | byte | byte `nState` | ✅ |  |
| 3 | int16 | int16 `ptPos.x` | ✅ |  |
| 4 | int16 | int16 `ptPos.y` | ✅ |  |
| 5 | byte | byte `bFlip` | ✅ |  |
| 6 | string | string `sName` | ✅ |  |

