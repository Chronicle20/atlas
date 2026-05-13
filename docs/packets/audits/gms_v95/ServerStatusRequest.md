# ServerStatusRequest (← `CLogin::SendCheckUserLimitPacket`)

- **IDA:** 0x5d43d0
- **Atlas file:** `../../libs/atlas-packet/login/serverbound/server_status_request.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | int16 `nWorldID` | ❌ | width mismatch |

