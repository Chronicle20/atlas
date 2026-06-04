# Hello (← `CClientSocket::OnConnect#Hello`)

- **IDA:** 0x4b0066
- **Atlas file:** `../../libs/atlas-packet/socket/clientbound/hello.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `packet length prefix` | ✅ |  |
| 1 | int16 | int16 `majorVersion uint16 (must be 185)` | ✅ |  |
| 2 | string | string `minorVersion string` | ✅ |  |
| 3 | bytes | int32 `recvIv (m_uSeqSnd) 4 bytes` | ✅ |  |
| 4 | bytes | int32 `sendIv (m_uSeqRcv) 4 bytes` | ✅ |  |
| 5 | byte | byte `locale byte (must be 3 for JMS)` | ✅ |  |

