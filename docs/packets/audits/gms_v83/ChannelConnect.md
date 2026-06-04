# ChannelConnect (← `CClientSocket::OnConnect#ChannelConnect`)

- **IDA:** 0x494ed1
- **Atlas file:** `../../libs/atlas-packet/socket/serverbound/channel_connect.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId uint32 — v83 OnConnect obfuscated; layout inferred from v95 @0x4aef10 (stable across versions)` | ✅ |  |
| 1 | bytes | bytes `machineId 16 bytes` | ✅ |  |
| 2 | byte | byte `gm byte (1 if GM, 0 otherwise)` | ✅ |  |
| 3 | byte | byte `unknown1 byte (literal 0)` | ✅ |  |
| 4 | int64 | bytes `unknown2 8 bytes raw buffer` | ❌ | width mismatch |


## Manual analysis

**v83 IDA:** `CClientSocket::OnConnect` @ 0x494ed1 is **anti-tamper obfuscated** — cannot be cleanly decompiled. Layout inferred from v95 @ 0x4aef10 which is stable. The field sequence (charId/machineId/gm/unknown1/unknown2) is the first thing a channel server reads on connect and is invariant across GMS versions in this era.

**Static-diff ❌ is a known false positive** — `WriteLong`(int64) vs `EncodeBuffer(8)` width mismatch; both produce 8 bytes on the wire.

**Gate:** None needed — version-agnostic. Gate confirmed correct (✅).


Ack: misc-audit Phase 3 v83 on 2026-06-03
