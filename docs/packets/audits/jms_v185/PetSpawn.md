# PetSpawn (← `CWvsContext::SendActivatePetRequest`)

- **IDA:** 0xb0b40b
- **Atlas file:** `libs/atlas-packet/pet/serverbound/spawn.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `get_update_time()` | ✅ |  |
| 1 | int16 | int16 `nPos` | ✅ |  |
| 2 | byte | byte `bBossPet` | ✅ |  |

