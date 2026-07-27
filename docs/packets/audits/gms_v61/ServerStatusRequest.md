# ServerStatusRequest (← `CLogin::SendCheckUserLimitPacket`)

- **IDA:** 0x5655df
- **Atlas file:** `libs/atlas-packet/login/serverbound/server_status_request.go`
- **Variant:** GMS/v61
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `worldId @0x56562e (short)` | ✅ |  |

