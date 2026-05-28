# GuildInviteReject (← `CFadeWnd::SendCloseMessage#DenyGuildRequest`)

- **IDA:** 0x557267
- **Atlas file:** `libs/atlas-packet/guild/serverbound/invite_reject.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `sub-op (deny/close)` | ✅ |  |
| 1 | string | byte `` | ❌ | atlas: extra — client never reads this field |

