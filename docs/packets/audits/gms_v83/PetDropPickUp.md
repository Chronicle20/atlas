# PetDropPickUp (← `CPet::SendDropPickUpRequest`)

- **IDA:** 0x705c7c
- **Atlas file:** `../../libs/atlas-packet/pet/serverbound/drop_pick_up.go`
- **Variant:** GMS/v83
- **Branch depth:** 3
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int64 | bytes `m_liPetLockerSN (8 bytes)` | ✅ |  |
| 1 | byte | byte `bFieldKey` | ✅ |  |
| 2 | int32 | int32 `get_update_time() (tick)` | ✅ |  |
| 3 | int16 | int16 `pt.x` | ✅ |  |
| 4 | int16 | int16 `pt.y` | ✅ |  |
| 5 | int32 | int32 `dwID` | ✅ |  |
| 6 | int32 | int32 `dwCliCrc` | ✅ |  |
| 7 | byte | byte `m_bPickupOthers` | ✅ |  |
| 8 | byte | byte `m_bSweepForDrop` | ✅ |  |
| 9 | byte | byte `m_bLongRange` | ✅ |  |
| 10 | byte | int16 `pet pos.x — gated dwID % 13 == 0` | ❌ | atlas: short — missing trailing field |
| 11 | byte | int16 `pet pos.y` | ❌ | atlas: short — missing trailing field |
| 12 | byte | int32 `m_dwPosCRC` | ❌ | atlas: short — missing trailing field |
| 13 | byte | int32 `dwRectCrc` | ❌ | atlas: short — missing trailing field |

