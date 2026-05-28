# GuildInviteReject (← `CFadeWnd::SendCloseMessage#DenyGuildRequest`)

- **IDA:** 0x0
- **Atlas file:** `../../libs/atlas-packet/guild/serverbound/invite_reject.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `unk/declineCode byte` | ✅ |  |
| 1 | string | string `fromCharacterName (inviter name)` | ✅ |  |

