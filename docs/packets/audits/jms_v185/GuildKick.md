# GuildKick (← `CField::SendKickGuildMsg`)

- **IDA:** 0x56ddf7
- **Atlas file:** `libs/atlas-packet/guild/serverbound/operation_kick.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `sub-op = 8 (KICK)` | ❌ | width mismatch |
| 1 | string | string `target character name` | ✅ |  |

