# SpawnPortal (← `CWvsContext::OnTownPortal`)

- **IDA:** 0xa226a6
- **Atlas file:** `libs/atlas-packet/door/clientbound/spawn_portal.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `townId (SpawnPortal); MapId.NONE (999999999) for RemoveTownDoor` | ✅ |  |
| 1 | int32 | int32 `targetId; MapId.NONE (999999999) for RemoveTownDoor` | ✅ |  |
| 2 | int16 | int16 `x (door position)` | ✅ |  |
| 3 | int16 | int16 `y (door position)` | ✅ |  |

