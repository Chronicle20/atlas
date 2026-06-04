# Hello (← `CClientSocket::OnConnect#Hello`)

- **IDA:** 0x494ed1
- **Atlas file:** `../../libs/atlas-packet/socket/clientbound/hello.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `packet length (0x0E = 14) — v83 OnConnect is anti-tamper obfuscated; layout inferred from v95 @0x4aef10 which is stable across versions` | ✅ |  |
| 1 | int16 | int16 `majorVersion uint16` | ✅ |  |
| 2 | string | string `minorVersion string` | ✅ |  |
| 3 | bytes | int32 `recvIv (→ client m_uSeqSnd) 4 bytes` | ❌ | width mismatch |
| 4 | bytes | int32 `sendIv (→ client m_uSeqRcv) 4 bytes` | ❌ | width mismatch |
| 5 | byte | byte `locale byte (nVersionHeader; must be 8)` | ✅ |  |


## Manual analysis

**v83 IDA:** `CClientSocket::OnConnect` @ 0x494ed1 is **anti-tamper obfuscated**. Layout (length/majorVersion/minorVersion/recvIv/sendIv/locale) is the fundamental MapleStory handshake invariant and matches v95 @ 0x4aef10.

**Static-diff ❌ is a known false positive** — `WriteByteArray`(bytes) vs `Decode4`(int32) for the two 4-byte IV fields; same artifact as v95.

**Gate:** None needed — version-agnostic. Gate confirmed correct (✅).


Ack: misc-audit Phase 3 v83 on 2026-06-03
