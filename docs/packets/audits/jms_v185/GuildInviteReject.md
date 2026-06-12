# GuildInviteReject (← `CFadeWnd::SendCloseMessage#DenyGuildRequest`)

- **IDA:** 0x557267
- **Atlas file:** `../../libs/atlas-packet/guild/serverbound/invite_reject.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `sub-op=0xD` | ✅ |  |
| 1 | string | string `grade1` | ✅ |  |
| 2 | byte | string `grade2` | ❌ | atlas: short — missing trailing field |
| 3 | byte | string `grade3` | ❌ | atlas: short — missing trailing field |
| 4 | byte | string `grade4` | ❌ | atlas: short — missing trailing field |
| 5 | byte | string `grade5` | ❌ | atlas: short — missing trailing field |

