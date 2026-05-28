# PetSpawn (← `CWvsContext::SendActivatePetRequest`)

- **IDA:** 0x9f6980
- **Atlas file:** `../../libs/atlas-packet/pet/serverbound/spawn.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `get_update_time() (tick)` | ✅ |  |
| 1 | int16 | int16 `nPos (pet inventory slot)` | ✅ |  |
| 2 | byte | byte `bBossPet flag` | ✅ |  |

