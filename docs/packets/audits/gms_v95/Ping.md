# Ping (← `CClientSocket::OnAliveReq#PingReceive`)

- **IDA:** 0x4afc90
- **Atlas file:** `libs/atlas-packet/socket/clientbound/ping.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|

## Manual analysis

**IDA function:** `CClientSocket::OnAliveReq` @ 0x4afc90 (called by `ProcessPacket` case 0x11)

The server sends `Ping` (opcode dispatched as 0x11 in the client's `ProcessPacket`) as a keepalive. The client processes the incoming ping in `CClientSocket::OnAliveReq`.

### IDA analysis

```c
// ProcessPacket @ 0x4b00f0:
case 0x11u:
    CClientSocket::OnAliveReq(this, iPacket);   // dispatches to OnAliveReq

// OnAliveReq @ 0x4afc90:
COutPacket::COutPacket(&v9, 25);    // construct pong with opcode 25
CClientSocket::SendPacket(this, ...); // send pong — no iPacket decode calls
```

`OnAliveReq` receives a `CInPacket&` but performs **zero `Decode*` operations** on it. The incoming ping packet carries no payload beyond the opcode. This is consistent with the atlas `Ping` struct having an empty body.

### Wire comparison

| Field | IDA | Atlas | Match? |
|---|---|---|---|
| (none) | no Decode* calls | empty body | ✅ |

### No bug — already correct

The atlas `Ping` encoder writes an empty body (`[]byte{}`). The client reads nothing after the opcode. Wire is opcode-only. ✅

### SUMMARY path verification

`locateAtlasFile` resolves to `libs/atlas-packet/socket/clientbound/ping.go` — correct socket/clientbound path. ✅

Ack: misc-audit Phase 2i on 2026-06-03
