# GuildSetNotice (← `CField::SendSetGuildNoticeMsg`)

- **IDA:** 0x56e3a2
- **Atlas file:** `libs/atlas-packet/guild/serverbound/operation_set_notice.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | byte `sub-op = 16 (SET_NOTICE)` | ❌ | width mismatch |
| 1 | byte | string `new guild notice text` | ❌ | atlas: short — missing trailing field |

