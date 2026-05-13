# Request (← `CLogin::SendCheckPasswordPacket`)

- **IDA:** 0x5db9d0
- **Atlas file:** `../../libs/atlas-packet/login/serverbound/request.go`
- **Variant:** GMS/v95/modified
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `name (modified) — stock sends password here` | ✅ |  |
| 1 | string | string `password (modified) — stock sends passport here` | ✅ |  |
| 2 | bytes | bytes `hwid/machineId (16 bytes)` | ✅ |  |
| 3 | int32 | int32 `gameRoomClient` | ✅ |  |
| 4 | byte | byte `gameStartMode` | ✅ |  |
| 5 | byte | byte `unknown1 (const 0)` | ✅ |  |
| 6 | byte | byte `unknown2 (v95 extra byte)` | ✅ |  |

