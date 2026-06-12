# Request (← `CLogin::SendCheckPasswordPacket`)

- **IDA:** 0x5db9d0
- **Atlas file:** `../../libs/atlas-packet/login/serverbound/request.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `name` | ✅ |  |
| 1 | string | string `password` | ✅ |  |
| 2 | bytes | bytes `hwid (16 bytes MachineId)` | ✅ |  |
| 3 | int32 | int32 `gameRoomClient` | ✅ |  |
| 4 | byte | byte `gameStartMode` | ✅ |  |
| 5 | byte | byte `unknown1` | ✅ |  |
| 6 | byte | byte `unknown2` | ✅ |  |
| 7 | int32 | int32 `partnerCode (GMS trailing field)` | ✅ |  |

