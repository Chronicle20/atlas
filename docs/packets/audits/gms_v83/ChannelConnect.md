# ChannelConnect (← `CClientSocket::OnConnect#ChannelConnect`)

- **IDA:** 0x494ed1
- **Atlas file:** `../../libs/atlas-packet/socket/serverbound/channel_connect.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId uint32 — v83 OnConnect obfuscated; layout inferred from v95 @0x4aef10 (stable across versions)` | ✅ |  |
| 1 | bytes | bytes `machineId 16 bytes` | ✅ |  |
| 2 | byte | byte `gm byte (1 if GM, 0 otherwise)` | ✅ |  |
| 3 | byte | byte `unknown1 byte (literal 0)` | ✅ |  |
| 4 | int64 | bytes `unknown2 8 bytes raw buffer` | ✅ |  |

