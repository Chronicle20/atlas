# Hello (← `CClientSocket::OnConnect#Hello`)

- **IDA:** 0x4a6e5a
- **Atlas file:** `libs/atlas-packet/socket/clientbound/hello.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `packet length (raw socket read; majorVersion parsed inline)` | ✅ |  |
| 1 | int16 | int16 `majorVersion uint16 (== 87, verified @0x4a71a4 TSingleton::m_nTargetVersion = 87)` | ✅ |  |
| 2 | string | string `minorVersion string (len-prefixed; atoi→namelen)` | ✅ |  |
| 3 | bytes | int32 `recvIv (→ client m_uSeqSnd) 4 bytes` | ❌ | width mismatch |
| 4 | bytes | int32 `sendIv (→ client m_uSeqRcv) 4 bytes` | ❌ | width mismatch |
| 5 | byte | byte `locale byte (nVersionHeader; must be 8, checked @0x4a712f)` | ✅ |  |


## Manual analysis

**v87 IDA:** `CClientSocket::OnConnect` @ 0x4a6e5a — reads raw socket bytes (not via CInPacket). The function verifies majorVersion == 87 at 0x4a71a4 (`m_nTargetVersion = 87`). Wire layout: 2-byte len, 2-byte version, len-prefixed minorVersion string, 4-byte recvIv, 4-byte sendIv, 1-byte locale (must be 8). The ❌ reflects a type mismatch in the audit tool comparing `bytes` vs `int32` for the IV fields — the wire bytes are identical (4 bytes each). **v87 Hello structure matches v83/v95 exactly. Gate confirmed N/A (not version-gated).**

Ack: misc-audit Phase 3 v87 on 2026-06-03
