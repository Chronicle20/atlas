# GuildBBSEntryNotFound (← `CUIGuildBBS::OnGuildBBSPacket#BBSEntryNotFound`)

- **IDA:** ABSENT
- **Atlas file:** `libs/atlas-packet/guild/clientbound/bbs.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ⚠️

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

