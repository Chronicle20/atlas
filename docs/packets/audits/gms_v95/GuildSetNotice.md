# GuildSetNotice (← `CField::SendSetGuildNoticeMsg`)

- **IDA:** 0x0
- **Atlas file:** `libs/atlas-packet/guild/serverbound/operation_set_notice.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `notice text` | ✅ |  |

