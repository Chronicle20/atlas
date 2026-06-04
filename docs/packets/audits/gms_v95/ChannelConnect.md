# ChannelConnect (← `CClientSocket::OnConnect#ChannelConnect`)

- **IDA:** 0x4aef10
- **Atlas file:** `libs/atlas-packet/socket/serverbound/channel_connect.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId uint32` | ✅ |  |
| 1 | bytes | bytes `machineId 16 bytes` | ✅ |  |
| 2 | byte | byte `gm byte (1 if GM, 0 otherwise)` | ✅ |  |
| 3 | byte | byte `unknown1 byte (literal 0)` | ✅ |  |
| 4 | int64 | bytes `unknown2 8 bytes raw buffer` | ❌ | width mismatch |

## Manual analysis

**IDA function:** `CClientSocket::OnConnect` @ 0x4aef10 (channel-connect branch, `bLogin == false`)

The `ChannelConnect` packet is sent by the client after receiving and parsing the `Hello` handshake, when connecting to a channel server (not the login server). This is the first packet the server receives from the client on a channel connection.

### IDA encode sequence (lines 267–288 of decompile)

```
COutPacket::COutPacket(&oPacket, 20)                              // opcode 20
COutPacket::Encode4(&oPacket, v33)                               // characterId uint32
COutPacket::EncodeBuffer(&oPacket, MachineId, 0x10u)             // machineId 16 bytes
COutPacket::Encode1(&oPacket, IsUserGM ? 1 : 0)                  // gm byte (1 or 0)
COutPacket::Encode1(&oPacket, 0)                                 // unknown1 byte (literal 0)
COutPacket::EncodeBuffer(&oPacket, ms_pInstance + 8360, 8u)      // unknown2 8 raw bytes
```

Total payload: 4 + 16 + 1 + 1 + 8 = 30 bytes.

### Wire comparison

| Field | IDA op | IDA width | Atlas decode | Atlas width | Match? |
|---|---|---|---|---|---|
| characterId | `Encode4` | 4 | `ReadUint32()` | 4 | ✅ |
| machineId | `EncodeBuffer(16)` | 16 | `ReadBytes(16)` | 16 | ✅ |
| gm | `Encode1` | 1 | `ReadBool()` | 1 | ✅ |
| unknown1 | `Encode1(0)` | 1 | `ReadBool()` | 1 | ✅ |
| unknown2 | `EncodeBuffer(8)` | 8 | `ReadUint64()` | 8 | ✅ (see note) |

### Static-diff false positive explanation

Row 4 shows ❌ because the static diff labels `WriteLong` (int64) and `EncodeBuffer(8)` (bytes) as a "width mismatch." Both produce exactly 8 bytes on the wire. `WriteLong` writes an 8-byte little-endian integer; `EncodeBuffer(8)` writes 8 raw bytes. For an `unknown2` field (opaque session context data), the interpretation is irrelevant — only the 8-byte width matters. The tool cannot statically resolve `DecodeBuffer(8)` into a byte count distinct from `int64`. No real wire discrepancy exists.

### No version gate needed

The v95 `OnConnect` channel branch uses this exact layout. No prior version exported for comparison; the atlas struct is version-agnostic. No guard is required.

### No code bug — static-diff artifact only

The ❌ in the wire-level diff is a static-analysis false positive. The atlas `ChannelConnect` decoder reads the correct 30-byte layout matching the v95 IDA. All field widths and ordering are correct.

### SUMMARY path verification

`locateAtlasFile` resolves to `libs/atlas-packet/socket/serverbound/channel_connect.go` — correct socket/serverbound path. ✅

Ack: misc-audit Phase 2i on 2026-06-03
