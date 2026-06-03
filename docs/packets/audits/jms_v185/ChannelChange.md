# ChannelChange (← `CClientSocket::OnMigrateCommand`)

- **IDA:** 0x4b1924
- **Atlas file:** `libs/atlas-packet/buddy/clientbound/channel_change.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `success flag (if 0 → CDisconnectException thrown)` | ✅ |  |
| 1 | int32 | int32 `IP address as raw uint32 (stored into pAddr.sin_addr.S_un.S_addr)` | ✅ |  |
| 2 | byte | int16 `port (host byte order; sub_4AFC56 builds sockaddr)` | ❌ | width mismatch |
| 3 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |


## Manual analysis

**Same tool collision as GMS v95** — `locateAtlasFile` resolves to `buddy/clientbound/channel_change.go` instead of `channel/clientbound/change.go`. See `gms_v95/ChannelChange.md` for full explanation.

**Correct atlas file:** `libs/atlas-packet/channel/clientbound/change.go`

JMS v185 `CClientSocket::OnMigrateCommand` (@ 0x4b1924):
```
Decode1  → success flag  (if 0 → CDisconnectException thrown)
Decode4  → IP address    (v6, stored as sin_addr.S_un.S_addr)
Decode2  → port          (v7, passed to sub_4AFC56 which builds sockaddr)
```

Atlas `channel/clientbound/change.go` writes: `WriteByte(1) + WriteByteArray(ip) + WriteShort(port)` = 1+4+2 = 7 bytes. JMS reads 1+4+2 = 7 bytes. **Wire-correct ✅.**

Port is `Decode2` (uint16) in JMS — same as GMS v95. No port-width difference. No gate change needed.

**JMS vs GMS: gate confirmed ✅.** ChannelChange (channel migrate) layout is identical in JMS v185 and GMS v95.

Ack: misc-audit Phase 3 JMS185 on 2026-06-03
