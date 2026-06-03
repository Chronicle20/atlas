# Hello (← `CClientSocket::OnConnect#Hello`)

- **IDA:** 0x4b0066
- **Atlas file:** `libs/atlas-packet/socket/clientbound/hello.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `packet length prefix` | ✅ |  |
| 1 | int16 | int16 `majorVersion uint16 (must be 185)` | ✅ |  |
| 2 | string | string `minorVersion string` | ✅ |  |
| 3 | bytes | int32 `recvIv (m_uSeqSnd) 4 bytes` | ❌ | width mismatch |
| 4 | bytes | int32 `sendIv (m_uSeqRcv) 4 bytes` | ❌ | width mismatch |
| 5 | byte | byte `locale byte (must be 3 for JMS)` | ✅ |  |


## Manual analysis

**The auto-generated ❌ at rows 3 and 4 is a static-tool type-representation artifact.** The tool encodes `WriteByteArray(4 bytes)` as type `bytes` and `CInPacket::DecodeBuffer(4)` as `int32`, but both are exactly 4 bytes on the wire.

JMS v185 `CClientSocket::OnConnect` (@ 0x4b0066) parses the Hello packet via a raw socket recv loop + `CIOBufferManipulator::DecodeStr`, not via `CInPacket::Decode*`. The wire parsing:
```
2 bytes: length prefix (raw LE uint16)
2 bytes: majorVersion (raw LE uint16 — must equal 185)
DecodeStr: minorVersion string (2-byte LE length + content)
4 bytes: m_uSeqSnd (unsigned int, stored via DWORD ptr read = recvIv)
4 bytes: m_uSeqRcv (unsigned int = sendIv)
1 byte:  locale (must be 3 for JMS)
```

Atlas `Hello.Encode` writes:
```
WriteShort(0x0E)          → 2 bytes
WriteShort(majorVersion)  → 2 bytes
WriteAsciiString(minor)   → 2+len bytes
WriteByteArray(recvIv)    → 4 bytes  ← wire-identical to Decode4
WriteByteArray(sendIv)    → 4 bytes  ← wire-identical to Decode4
WriteByte(locale)         → 1 byte
```

All fields match JMS wire exactly. No fix needed.

**JMS vs GMS: gate confirmed ✅.** Hello has no version/region gate; the same layout works for all regions. The only behavioral difference is that JMS expects locale byte = 3 (not 8 as in GMS). Atlas passes `m.locale` through — the tenant template controls what locale value is written; no atlas-packet code change needed.

Ack: misc-audit Phase 3 JMS185 on 2026-06-03
