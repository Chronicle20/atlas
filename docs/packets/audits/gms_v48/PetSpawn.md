# PetSpawn (← `CWvsContext::SendActivatePetRequest`)

- **IDA:** 0x71d118
- **Atlas file:** `libs/atlas-packet/pet/serverbound/spawn.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ⚠️

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | int16 | int16 `` | ✅ |  |
| 2 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

