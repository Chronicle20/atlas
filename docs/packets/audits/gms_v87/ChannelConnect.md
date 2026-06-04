# ChannelConnect (← `CClientSocket::OnConnect#ChannelConnect`)

- **IDA:** 0x4a6e5a
- **Atlas file:** `libs/atlas-packet/socket/serverbound/channel_connect.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId uint32 (Encode4 m_dwAccountId @0x4a73ec)` | ✅ |  |
| 1 | bytes | bytes `machineId 16 bytes (EncodeBuffer GetMachineId @0x4a7401)` | ✅ |  |
| 2 | byte | byte `gm byte (Encode1 based on TSecType GM check @0x4a7418)` | ✅ |  |
| 3 | byte | byte `unknown1 byte (literal 0 @0x4a742b)` | ✅ |  |
| 4 | int64 | bytes `unknown2 8 bytes raw buffer (EncodeBuffer szCookie+88 @0x4a7440)` | ❌ | width mismatch |


## Manual analysis

**v87 vs v95/v83:** Same pre-existing divergence seen in v83. `CClientSocket::OnConnect` @ 0x4a6e5a confirmed: channel-connect path sends Encode4(characterId) + EncodeBuffer(machineId 16) + Encode1(gm) + Encode1(0) + EncodeBuffer(8 bytes). The ❌ is a type mismatch: atlas writes int64 for the 8-byte buffer, audit tool expects `bytes`. Wire payload is identical. No v87-specific divergence. No new gate needed.

Ack: misc-audit Phase 3 v87 on 2026-06-03
