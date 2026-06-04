# Hello (← `CClientSocket::OnConnect#Hello`)

- **IDA:** 0x494ed1
- **Atlas file:** `../../libs/atlas-packet/socket/clientbound/hello.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `packet length (0x0E = 14) — v83 OnConnect is anti-tamper obfuscated; layout inferred from v95 @0x4aef10 which is stable across versions` | ✅ |  |
| 1 | int16 | int16 `majorVersion uint16` | ✅ |  |
| 2 | string | string `minorVersion string` | ✅ |  |
| 3 | bytes | int32 `recvIv (→ client m_uSeqSnd) 4 bytes` | ✅ |  |
| 4 | bytes | int32 `sendIv (→ client m_uSeqRcv) 4 bytes` | ✅ |  |
| 5 | byte | byte `locale byte (nVersionHeader; must be 8)` | ✅ |  |

