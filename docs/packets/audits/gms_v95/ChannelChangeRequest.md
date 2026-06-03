# ChannelChangeRequest (← `CField::SendTransferChannelRequest`)

- **IDA:** 0x52efa0
- **Atlas file:** `libs/atlas-packet/channel/serverbound/channel_change.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `nTargetChannel (target channel ID, 0-based byte)` | ✅ |  |
| 1 | int32 | int32 `get_update_time() (client tick / update time, uint32)` | ✅ |  |

## Manual analysis

### IDA evidence — `CField::SendTransferChannelRequest` @ 0x52efa0

The function constructs an outbound packet (opcode 42 / `0x2A`) before validation guards pass:

```cpp
COutPacket::COutPacket(&oPacket, 42);      // opcode header
COutPacket::Encode1(&oPacket, nTargetChannel);   // 1 byte
update_time = get_update_time();
COutPacket::Encode4(&oPacket, update_time);      // 4 bytes
CClientSocket::SendPacket(..., &oPacket);
```

Total client payload: 5 bytes.

### Atlas decoder (`channel/serverbound/channel_change.go`)

```
ReadByte()    → channelId  (1 byte)
ReadUint32()  → updateTime (4 bytes LE)
```

Total: 5 bytes.

### Wire comparison

| Field | IDA width | Atlas width | Match? |
|---|---|---|---|
| channelId | 1 byte (Encode1) | 1 byte (ReadByte) | ✅ |
| updateTime | 4 bytes (Encode4, uint32) | 4 bytes (ReadUint32) | ✅ |

### No bug — already correct

The `ChannelChangeRequest` decoder matches v95 exactly. No fix needed. The ✅ auto-verdict is correct.

The byte-level wire shape is verified by `TestChannelChangeRequestWireShape` in `libs/atlas-packet/channel/serverbound/channel_change_test.go`:
- All four variants produce exactly 5 bytes.
- Byte 0 = channelId.
- Bytes 1–4 = updateTime in little-endian order.

Ack: misc-audit Phase 2c on 2026-06-03