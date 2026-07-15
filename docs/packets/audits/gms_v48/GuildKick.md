# GuildKick (← `CField::SendKickGuildMsg`)

- **IDA:** 0x4c5e06
- **Atlas file:** `libs/atlas-packet/guild/serverbound/operation_kick.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | int32 | int32 `` | ✅ |  |
| 2 | string | string `` | ✅ |  |

