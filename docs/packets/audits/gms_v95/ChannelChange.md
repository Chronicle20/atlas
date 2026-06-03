# ChannelChange (← `CClientSocket::OnMigrateCommand`)

- **IDA:** 0x4add50
- **Atlas file:** `libs/atlas-packet/buddy/clientbound/channel_change.go`
- **Variant:** GMS/v95
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

**The auto-generated table above and the ❌ verdict are a static-tool artifact — the wrong atlas file was analyzed.** The tool's `locateAtlasFile` performs a lexicographic `WalkDir` over `clientbound/` directories; it found `libs/atlas-packet/buddy/clientbound/channel_change.go` (the buddy-list channel-change notification) before `libs/atlas-packet/channel/clientbound/change.go` (the actual migrate/channel-change packet). Those are two distinct structs that share the name `ChannelChange`. The rows in the diff table above describe the buddy struct, not the channel-migrate struct.

**Correct atlas file: `libs/atlas-packet/channel/clientbound/change.go`**

### IDA evidence — `CClientSocket::OnMigrateCommand` @ 0x4add50

```
Decode1  → success flag  (if 0 → client posts QuitMessage / disconnects)
Decode4  → IP address    (4 bytes; stored as-is into sin_addr.s_addr — no htonl)
Decode2  → port          (2 bytes; htons applied when placing into HIWORD of sockaddr)
```

Total: 7 bytes.

### Atlas encoder (`channel/clientbound/change.go`)

```
WriteByte(1)                        → 1 byte  (success flag, hard-coded)
WriteByteArray(channelIpAsByteArray) → 4 bytes (IP octets in network order)
WriteShort(port)                    → 2 bytes
```

Total: 7 bytes.

### Wire comparison

| Field | IDA width | Atlas width | Match? |
|---|---|---|---|
| success flag | 1 byte (Decode1) | 1 byte (WriteByte) | ✅ |
| IP address | 4 bytes (Decode4, no byte-swap) | 4 bytes (WriteByteArray 4 octets) | ✅ |
| port | 2 bytes (Decode2 → htons) | 2 bytes (WriteShort LE) | ✅ |

The `WriteByteArray` / `Decode4` difference is wire-equivalent: atlas writes the 4 octets in declaration order; the client's `Decode4` reads them as a raw 4-byte LE word and stores directly into `sin_addr.s_addr` (no further byte-swap on the IP), so the octet order in memory is identical.

### No bug — already correct

The channel/clientbound `ChannelChange` encoder matches v95 exactly. No fix needed. The ❌ verdict in the auto-generated table is entirely due to the name collision (`buddy/clientbound/ChannelChange` being analyzed instead of `channel/clientbound/ChannelChange`).

The byte-level wire shape is verified by `TestChannelChangeWireShape` in `libs/atlas-packet/channel/clientbound/change_test.go`:
- All four variants produce exactly 7 bytes.
- Byte 0 = `0x01` (success flag).
- Bytes 1–4 = IP octets in network order.

Ack: misc-audit Phase 2c on 2026-06-03