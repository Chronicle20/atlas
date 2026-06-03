# Hello (← `CClientSocket::OnConnect#Hello`)

- **IDA:** 0x4aef10
- **Atlas file:** `libs/atlas-packet/socket/clientbound/hello.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `packet length (0x0E = 14)` | ✅ |  |
| 1 | int16 | int16 `majorVersion uint16` | ✅ |  |
| 2 | string | string `minorVersion string` | ✅ |  |
| 3 | bytes | int32 `recvIv (→ client m_uSeqSnd) 4 bytes` | ❌ | width mismatch |
| 4 | bytes | int32 `sendIv (→ client m_uSeqRcv) 4 bytes` | ❌ | width mismatch |
| 5 | byte | byte `locale byte (nVersionHeader; must be 8)` | ✅ |  |

## Manual analysis

**IDA function:** `CClientSocket::OnConnect` @ 0x4aef10

The `Hello` packet is the unencrypted handshake sent by the server immediately on TCP connect, before AES key exchange. The client parses it in `CClientSocket::OnConnect` before any encryption context is established.

### IDA decode sequence (lines 113–154 of decompile)

```
Decode2(&buf, v6, ...)           → majorVersion (uint16)  → buf.a
DecodeStr(&bLenRead, v13, ...)   → minorVersion string    → bLenRead (atoi'd)
Decode4(&uSeqSnd, v14, ...)      → first IV               → uSeqSnd
Decode4(&uSeqRcv, v15, ...)      → second IV              → uSeqRcv
Decode1(&nVersionHeader, v16, ...) → locale byte          → nVersionHeader
```

Assignment after decode:
```
v52->m_uSeqSnd = uSeqSnd;   // first Decode4 → client's send IV
v17->m_uSeqRcv = v18;       // second Decode4 → client's receive IV
```

Total body: 2 (length) + 2 (majorVersion) + (2+N) (minorVersion string) + 4 (recvIv) + 4 (sendIv) + 1 (locale) bytes.

### Wire comparison

| Field | IDA op | IDA width | Atlas write | Atlas width | Match? |
|---|---|---|---|---|---|
| length header | (framing, not decoded) | 2 | `WriteShort(0x0E)` | 2 | ✅ |
| majorVersion | `Decode2` | 2 | `WriteShort(majorVersion)` | 2 | ✅ |
| minorVersion | `DecodeStr` | 2+N | `WriteAsciiString` | 2+N | ✅ |
| recvIv | `Decode4` | 4 | `WriteByteArray(recvIv)` | 4 | ✅ (see note) |
| sendIv | `Decode4` | 4 | `WriteByteArray(sendIv)` | 4 | ✅ (see note) |
| locale | `Decode1` | 1 | `WriteByte(locale)` | 1 | ✅ |

### Static-diff false positive explanation

Rows 3 and 4 show ❌ because the static diff labels `WriteByteArray` as type `bytes` and `Decode4` as `int32`, triggering a "width mismatch" flag. Both represent **exactly 4 bytes** on the wire — the IDA `Decode4` is called with a 4-byte destination and `WriteByteArray` is called with a 4-element slice. The tool cannot statically resolve slice lengths; this is a known audit-tool limitation for fixed-size byte arrays vs. numeric Decode4. No real wire discrepancy exists.

### IV ordering — correct

Atlas writes `recvIv` first, then `sendIv`. The IDA reads first IV into `m_uSeqSnd` (client's send key) and second into `m_uSeqRcv` (client's receive key). From the server perspective:
- `recvIv` = IV the server uses to decrypt incoming bytes = IV the client uses to encrypt its sends = client's `m_uSeqSnd` ✅
- `sendIv` = IV the server uses to encrypt outgoing bytes = IV the client uses to decrypt its receives = client's `m_uSeqRcv` ✅

The field ordering in atlas `Hello` is **correct** and matches v95 exactly.

### Locale byte

`nVersionHeader` is checked for value `8` (GMS locale). Atlas passes the locale byte through without version-gating — all versions use the same layout. No version gate is needed; the atlas `Hello` struct is version-agnostic.

### No code bug — static-diff artifact only

The ❌ in the wire-level diff table is a static-analysis false positive. The atlas `Hello` encoder and decoder are correct for v95. `TestHelloWireShape` (added in this audit) verifies the exact byte layout against the IDA field order.

### SUMMARY path verification

`locateAtlasFile` resolves to `libs/atlas-packet/socket/clientbound/hello.go` — correct socket/clientbound path. ✅

Ack: misc-audit Phase 2i on 2026-06-03
