# ServerStatusRequest (← `CLogin::SendCheckUserLimitPacket`)

- **IDA:** 0x5cd1a2
- **Atlas file:** `libs/atlas-packet/login/serverbound/server_status_request.go`
- **Variant:** GMS/v79
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `nWorldID (v79 sub_5CD1A2@0x5cd1a2 Encode2)` | ✅ |  |

