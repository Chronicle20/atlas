# Hello (← `CClientSocket::OnConnect#Hello`)

- **IDA:** 0x4aef10
- **Atlas file:** `../../libs/atlas-packet/socket/clientbound/hello.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `packet length (0x0E = 14)` | ✅ |  |
| 1 | int16 | int16 `majorVersion uint16` | ✅ |  |
| 2 | string | string `minorVersion string` | ✅ |  |
| 3 | bytes | int32 `recvIv (→ client m_uSeqSnd) 4 bytes` | ✅ |  |
| 4 | bytes | int32 `sendIv (→ client m_uSeqRcv) 4 bytes` | ✅ |  |
| 5 | byte | byte `locale byte (nVersionHeader; must be 8)` | ✅ |  |

