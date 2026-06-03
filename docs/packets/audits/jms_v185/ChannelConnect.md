# ChannelConnect (← `CClientSocket::OnConnect#ChannelConnect`)

- **IDA:** 0x4b0066
- **Atlas file:** `libs/atlas-packet/socket/serverbound/channel_connect.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId uint32` | ✅ |  |
| 1 | bytes | bytes `machineId 16 bytes (CSystemInfo::GetMachineId)` | ✅ |  |
| 2 | int16 | int16 `gm/dummy1 uint16 (CConfig::dummy1) — JMS sends 2 bytes; GMS sends 1 byte` | ✅ |  |
| 3 | byte | byte `unknown1 (literal 0)` | ✅ |  |
| 4 | int64 | bytes `unknown2 8 bytes` | ❌ | width mismatch |


## Manual analysis

**The auto-generated ❌ at row 4 (`int64` vs `bytes`) is a type-representation artifact.** Atlas writes `WriteLong(uint64)` = 8 bytes; the IDA JSON has `EncodeBuf` (8 bytes). Both are 8 bytes on the wire.

Rows 0-3 are all ✅ after the gate fix for the `gm/dummy1` field (row 2).

**Real JMS gate fix applied:** JMS v185 `CClientSocket::OnConnect` non-login branch (@ 0x4b051f):
```c
COutPacket::Encode4(characterId)       // 4 bytes
COutPacket::EncodeBuffer(MachineId, 16) // 16 bytes
COutPacket::Encode2(dummy1)            // 2 bytes ← JMS gm field is uint16
COutPacket::Encode1(0)                 // 1 byte (unknown1)
COutPacket::EncodeBuffer(unknown2, 8)  // 8 bytes
```

GMS v95 sends `Encode1` (1 byte) for the gm field; JMS sends `Encode2` (2 bytes). This was a real wire mismatch. **Fix:** `ChannelConnect.Encode/Decode` now reads `ReadUint16() / WriteShort()` when `t.Region() == "JMS"` and `ReadBool() / WriteBool()` for all other regions. Wire shape verified by `TestChannelConnectWireShape`: GMS=30 bytes, JMS=31 bytes.

**JMS vs GMS: WIDENED to include JMS** — gm field gate added. Row 4 ❌ is a static-tool artifact (both 8 bytes).

Ack: misc-audit Phase 3 JMS185 on 2026-06-03
