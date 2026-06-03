# ChannelChange (← `CClientSocket::OnMigrateCommand`)

- **IDA:** 0x496701
- **Atlas file:** `../../libs/atlas-packet/buddy/clientbound/channel_change.go`
- **Variant:** GMS/v83
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

**v83 IDA:** `CClientSocket::OnMigrateCommand` @ 0x496701 — Decode1(success), Decode4(IP), Decode2(port). Identical layout to v95 @ 0x4add50.

**Static-diff ❌ is a known false positive** (same tool artifact as v95): wrong atlas file analyzed (buddy/clientbound vs channel/clientbound). The correct atlas file is `libs/atlas-packet/channel/clientbound/change.go` which writes 7 bytes matching v83 exactly.

**Gate:** None needed — version-agnostic. Gate confirmed correct (✅).


Ack: misc-audit Phase 3 v83 on 2026-06-03
