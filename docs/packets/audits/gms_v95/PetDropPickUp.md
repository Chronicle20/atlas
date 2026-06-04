# PetDropPickUp (← `CPet::SendDropPickUpRequest`)

- **IDA:** 0x6a0820
- **Atlas file:** `../../libs/atlas-packet/pet/serverbound/drop_pick_up.go`
- **Variant:** GMS/v95
- **Branch depth:** 3
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int64 | bytes `m_liPetLockerSN (8 bytes — _LARGE_INTEGER)` | ✅ |  |
| 1 | byte | byte `bFieldKey (or 0 if no field)` | ✅ |  |
| 2 | int32 | int32 `get_update_time() (tick)` | ✅ |  |
| 3 | int16 | int16 `pt.x` | ✅ |  |
| 4 | int16 | int16 `pt.y` | ✅ |  |
| 5 | int32 | int32 `dwID (drop id)` | ✅ |  |
| 6 | int32 | int32 `dwCliCrc` | ✅ |  |
| 7 | byte | byte `m_bPickupOthers` | ✅ |  |
| 8 | byte | byte `m_bSweepForDrop` | ✅ |  |
| 9 | byte | byte `m_bLongRange` | ✅ |  |
| 10 | int16 | int16 `pet pos.x — conditional: only when dwID % 13 == 0` | ✅ |  |
| 11 | int16 | int16 `pet pos.y — conditional with above` | ✅ |  |
| 12 | int32 | int32 `m_dwPosCRC — conditional with above` | ✅ |  |
| 13 | int32 | int32 `dwRectCrc — conditional with above` | ✅ |  |

