# ChannelConnect (← `CClientSocket::OnConnect#ChannelConnect`)

- **IDA:** 0x4b0066
- **Atlas file:** `../../libs/atlas-packet/socket/serverbound/channel_connect.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId uint32` | ✅ |  |
| 1 | bytes | bytes `machineId 16 bytes (CSystemInfo::GetMachineId)` | ✅ |  |
| 2 | int16 | int16 `gm/dummy1 uint16 (CConfig::dummy1) — JMS sends 2 bytes; GMS sends 1 byte` | ✅ |  |
| 3 | byte | byte `unknown1 (literal 0)` | ✅ |  |
| 4 | int64 | bytes `unknown2 8 bytes` | ✅ |  |

