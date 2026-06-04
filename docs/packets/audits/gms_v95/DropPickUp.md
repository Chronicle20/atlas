# DropPickUp (← `CWvsContext::SendDropPickUpRequest`)

- **IDA:** 0x9d5d50
- **Atlas file:** `../../libs/atlas-packet/drop/serverbound/pick_up.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `field->m_bFieldKey` | ✅ |  |
| 1 | int32 | int32 `get_update_time() (tick)` | ✅ |  |
| 2 | int16 | int16 `pt.x (pickup point x)` | ✅ |  |
| 3 | int16 | int16 `pt.y (pickup point y)` | ✅ |  |
| 4 | int32 | int32 `dwDropID` | ✅ |  |
| 5 | int32 | int32 `dwCliCrc` | ✅ |  |

