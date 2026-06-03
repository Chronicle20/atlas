# ChannelChange (← `CClientSocket::OnMigrateCommand`)

- **IDA:** 0x4a874b
- **Atlas file:** `libs/atlas-packet/buddy/clientbound/channel_change.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `success flag (if 0 → client disconnects)` | ✅ |  |
| 1 | int32 | int32 `IP address as raw uint32 (stored directly into sin_addr.s_addr — network byte order preserved)` | ✅ |  |
| 2 | byte | int16 `port (host byte order; htons applied when building sockaddr)` | ❌ | width mismatch |
| 3 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |


## Manual analysis

**v87 vs v95/v83:** Same pre-existing divergence seen in v83/v95. `CClientSocket::OnMigrateCommand` @ 0x4a874b confirmed: reads Decode1(success) + Decode4(IP) + Decode2(port). Atlas writes byte(success) + int32(IP) + byte(port) + int32(extra) — the port width (int16 vs byte) and extra int32 are pre-existing issues tracked in all passes. No v87-specific divergence. No new gate needed.

Ack: misc-audit Phase 3 v87 on 2026-06-03
