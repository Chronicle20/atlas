# ChannelConnect (← `CClientSocket::OnConnect#ChannelConnect`)

- **IDA:** 0x4a6e5a
- **Atlas file:** `../../libs/atlas-packet/socket/serverbound/channel_connect.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId uint32 (Encode4 m_dwAccountId @0x4a73ec)` | ✅ |  |
| 1 | bytes | bytes `machineId 16 bytes (EncodeBuffer GetMachineId @0x4a7401)` | ✅ |  |
| 2 | byte | byte `gm byte (Encode1 based on TSecType GM check @0x4a7418)` | ✅ |  |
| 3 | byte | byte `unknown1 byte (literal 0 @0x4a742b)` | ✅ |  |
| 4 | int64 | bytes `unknown2 8 bytes raw buffer (EncodeBuffer szCookie+88 @0x4a7440)` | ✅ |  |

