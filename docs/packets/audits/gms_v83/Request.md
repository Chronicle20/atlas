# Request (← `CLogin::SendCheckPasswordPacket`)

- **IDA:** 0x5f6952
- **Atlas file:** `../../libs/atlas-packet/login/serverbound/request.go`
- **Variant:** GMS/v83
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `name` | ✅ |  |
| 1 | string | string `password` | ✅ |  |
| 2 | bytes | bytes `hwid (16 bytes)` | ✅ |  |
| 3 | int32 | int32 `gameRoomClient` | ✅ |  |
| 4 | byte | byte `gameStartMode` | ✅ |  |
| 5 | byte | byte `unknown1` | ✅ |  |

