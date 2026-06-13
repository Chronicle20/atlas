# PetDropPickUp (← `CPet::SendDropPickUpRequest`)

- **IDA:** 0x722672
- **Atlas file:** `libs/atlas-packet/pet/serverbound/drop_pick_up.go`
- **Variant:** GMS/v84
- **Branch depth:** 3
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int64 | bytes `` | ✅ |  |
| 1 | byte | byte `` | ✅ |  |
| 2 | int32 | int32 `` | ✅ |  |
| 3 | int16 | int16 `` | ✅ |  |
| 4 | int16 | int16 `` | ✅ |  |
| 5 | int32 | int32 `` | ✅ |  |
| 6 | int32 | int32 `` | ✅ |  |
| 7 | byte | byte `` | ✅ |  |
| 8 | byte | byte `` | ✅ |  |
| 9 | byte | byte `` | ✅ |  |
| 10 | int16 | int16 `` | ✅ |  |
| 11 | int16 | int16 `` | ✅ |  |
| 12 | int32 | int32 `` | ✅ |  |
| 13 | int32 | int32 `` | ✅ |  |

