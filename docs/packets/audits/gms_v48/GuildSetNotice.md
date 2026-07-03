# GuildSetNotice (← `CField::SendSetGuildNoticeMsg`)

- **IDA:** 0x4c63d8
- **Atlas file:** `libs/atlas-packet/guild/serverbound/operation_set_notice.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `GUILD_OPERATION mode = 0x10 (SET_NOTICE) @0x4c63fe` | ✅ |  |
| 1 | string | string `guild notice @0x4c6418` | ✅ |  |

