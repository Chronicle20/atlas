# Hello (← `CClientSocket::OnConnect#Hello`)

- **IDA:** 0x4a6e5a
- **Atlas file:** `../../libs/atlas-packet/socket/clientbound/hello.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `packet length (raw socket read; majorVersion parsed inline)` | ✅ |  |
| 1 | int16 | int16 `majorVersion uint16 (== 87, verified @0x4a71a4 TSingleton::m_nTargetVersion = 87)` | ✅ |  |
| 2 | string | string `minorVersion string (len-prefixed; atoi→namelen)` | ✅ |  |
| 3 | bytes | int32 `recvIv (→ client m_uSeqSnd) 4 bytes` | ✅ |  |
| 4 | bytes | int32 `sendIv (→ client m_uSeqRcv) 4 bytes` | ✅ |  |
| 5 | byte | byte `locale byte (nVersionHeader; must be 8, checked @0x4a712f)` | ✅ |  |

