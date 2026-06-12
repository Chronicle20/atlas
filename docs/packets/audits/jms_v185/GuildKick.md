# GuildKick (← `CField::SendKickGuildMsg`)

- **IDA:** 0x56ddf7
- **Atlas file:** `../../libs/atlas-packet/guild/serverbound/operation_kick.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `sub-op=8 KICK` | ✅ |  |
| 1 | int32 | int32 `charId` | ✅ |  |
| 2 | string | string `name` | ✅ |  |

