# StartError (← `CClientSocket::OnConnect#StartError`)

- **IDA:** 0x4aef10
- **Atlas file:** `libs/atlas-packet/socket/serverbound/start_error.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `length uint16 (byte count of exception log data)` | ✅ |  |
| 1 | bytes | bytes `bytes variable-length exception log data` | ✅ |  |

## Manual analysis

**IDA function:** `CClientSocket::OnConnect` @ 0x4aef10 (login-connect branch, `bLogin == true`)

`StartError` is sent by the client on login-server connect when there is crash/exception log data from a previous session. The client reads an exception file and sends its contents to the login server.

### IDA encode sequence (lines 245–260 of decompile)

```c
COutPacket::COutPacket(&oPacket, 26)          // opcode 26 (0x1A)
COutPacket::Encode2(&oPacket, v29)            // length of exception data (uint16)
COutPacket::EncodeBuffer(&oPacket, a, v30)    // raw exception bytes (v30 = length)
CClientSocket::SendPacket(v52, &oPacket, ...);
```

Only sent when `!ZArray<unsigned char>::IsEmpty(&buf)` — i.e., only if exception log data exists. When no crash log is present, this packet is not sent at all.

### Wire comparison

| Field | IDA op | IDA width | Atlas decode | Atlas width | Match? |
|---|---|---|---|---|---|
| length | `Encode2` | 2 | `ReadUint16()` | 2 | ✅ |
| bytes | `EncodeBuffer(length)` | variable | `ReadBytes(length)` | variable | ✅ |

### No version gate needed

The login-branch exception-log packet layout is the same in v95 as in earlier versions. The atlas `StartError` struct is version-agnostic.

### No bug — already correct

The atlas `StartError` decoder reads a 2-byte length followed by that many raw bytes. This matches the v95 IDA exactly. ✅

### SUMMARY path verification

`locateAtlasFile` resolves to `libs/atlas-packet/socket/serverbound/start_error.go` — correct socket/serverbound path. ✅

Ack: misc-audit Phase 2i on 2026-06-03
