# SpawnDoor (← `CTownPortalPool::OnTownPortalCreated`)

- **IDA:** 0x7e3740
- **Atlas file:** `libs/atlas-packet/door/clientbound/spawn.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `launched flag (writeBool launched)` | ✅ |  |
| 1 | int32 | int32 `ownerId (door owner character id)` | ✅ |  |
| 2 | int16 | int16 `x (door position, writeShort)` | ✅ |  |
| 3 | int16 | int16 `y (door position, writeShort)` | ✅ |  |

