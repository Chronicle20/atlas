# GuildInviteRequest (← `CField::SendInviteGuildMsg`)

- **IDA:** 0x56dab9
- **Atlas file:** `../../libs/atlas-packet/guild/serverbound/operation_invite.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `sub-op = 5 (INVITE)` | ✅ |  |
| 1 | string | string `target character name` | ✅ |  |

