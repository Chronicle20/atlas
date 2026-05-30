# ReactorSpawn (← `CReactorPool::OnReactorEnterField`)

- **IDA:** 0x6cf490
- **Atlas file:** `libs/atlas-packet/reactor/clientbound/spawn.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwID (reactor unique id)` | ✅ |  |
| 1 | int32 | int32 `dwTemplateID (reactor template id)` | ✅ |  |
| 2 | byte | byte `nState (initial state)` | ✅ |  |
| 3 | int16 | int16 `ptPos.x` | ✅ |  |
| 4 | int16 | int16 `ptPos.y` | ✅ |  |
| 5 | byte | byte `bFlip (orientation)` | ✅ |  |
| 6 | string | string `sName (reactor name override)` | ✅ |  |

