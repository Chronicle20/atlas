# ServerStatusRequest (← `CLogin::SendCheckUserLimitPacket`)

- **IDA:** 0x62f80a
- **Atlas file:** `libs/atlas-packet/login/serverbound/server_status_request.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `nWorldID (Encode2 same as v95; v87@0x62f857 confirmed COutPacket::Encode2)` | ✅ |  |

