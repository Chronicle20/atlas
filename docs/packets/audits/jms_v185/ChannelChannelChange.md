# ChannelChannelChange (← `CClientSocket::OnMigrateCommand`)

- **IDA:** 0x4b1924
- **Atlas file:** `../../libs/atlas-packet/channel/clientbound/change.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `success flag (if 0 → CDisconnectException thrown)` | ✅ |  |
| 1 | bytes | int32 `IP address as raw uint32 (stored into pAddr.sin_addr.S_un.S_addr)` | ✅ |  |
| 2 | int16 | int16 `port (host byte order; sub_4AFC56 builds sockaddr)` | ✅ |  |

