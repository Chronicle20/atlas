# ServerStatusRequest (← `CLogin::SendCheckUserLimitPacket`)

- **IDA:** 0x5f8078
- **Atlas file:** `../../libs/atlas-packet/login/serverbound/server_status_request.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `nWorldID` | ✅ |  |

