# GuildSetTitleNames (← `CWvsContext::SendSetGuildTitleNames`)

- **IDA:** 0x0
- **Atlas file:** `../../libs/atlas-packet/guild/serverbound/operation_set_title_names.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `title[0]` | ✅ |  |
| 1 | byte | string `title[1]` | ❌ | atlas: short — missing trailing field |
| 2 | byte | string `title[2]` | ❌ | atlas: short — missing trailing field |
| 3 | byte | string `title[3]` | ❌ | atlas: short — missing trailing field |
| 4 | byte | string `title[4]` | ❌ | atlas: short — missing trailing field |

