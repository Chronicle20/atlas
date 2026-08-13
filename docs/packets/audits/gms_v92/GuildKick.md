# GuildKick (← `CField::SendKickGuildMsg`)

- **IDA:** 0x52f0b0
- **Atlas file:** `libs/atlas-packet/guild/serverbound/operation_kick.go`
- **Variant:** GMS/v92
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | int32 | int32 `` | ✅ |  |
| 2 | string | string `` | ✅ |  |

