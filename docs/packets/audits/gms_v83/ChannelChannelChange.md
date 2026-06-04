# ChannelChannelChange (← `CClientSocket::OnMigrateCommand`)

- **IDA:** 0x496701
- **Atlas file:** `../../libs/atlas-packet/channel/clientbound/change.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `success flag (if 0 → client disconnects)` | ✅ |  |
| 1 | bytes | int32 `IP address as raw uint32 (stored directly into sin_addr.s_addr — network byte order preserved)` | ✅ |  |
| 2 | int16 | int16 `port (host byte order; htons applied when building sockaddr)` | ✅ |  |

