# GuildSetNotice (← `CField::SendSetGuildNoticeMsg`)

- **IDA:** 0x56e3a2
- **Atlas file:** `../../libs/atlas-packet/guild/serverbound/operation_set_notice.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `sub-op = 16 (SET_NOTICE)` | ✅ |  |
| 1 | string | string `new guild notice text` | ✅ |  |

