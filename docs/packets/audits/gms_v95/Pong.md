# Pong (← `CClientSocket::OnAliveReq#PongSend`)

- **IDA:** 0x4afc90
- **Atlas file:** `libs/atlas-packet/socket/serverbound/pong.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|

## Manual analysis

**IDA function:** `CClientSocket::OnAliveReq` @ 0x4afc90

`OnAliveReq` is called when the client receives a server Ping (opcode 0x11). The function constructs and sends a Pong reply:

```c
COutPacket::COutPacket(&v9, 25);       // opcode 25, empty packet
CClientSocket::SendPacket(this, ...);  // send immediately
```

No additional fields are encoded. The pong is opcode-only.

### Wire comparison

| Field | IDA | Atlas | Match? |
|---|---|---|---|
| (none) | no Encode* calls after opcode | empty body | ✅ |

### No tick-count payload

The client does NOT send any tick-count or timestamp in the pong. The atlas `Pong` struct has an empty body — correct. No version gate needed.

### SUMMARY path verification

`locateAtlasFile` resolves to `libs/atlas-packet/socket/serverbound/pong.go` — correct socket/serverbound path. ✅

Ack: misc-audit Phase 2i on 2026-06-03
