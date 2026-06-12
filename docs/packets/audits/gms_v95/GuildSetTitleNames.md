# GuildSetTitleNames (← `CWvsContext::SendSetGuildTitleNames`)

- **IDA:** 0x0
- **Atlas file:** `../../libs/atlas-packet/guild/serverbound/operation_set_title_names.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `title[0]` | ✅ |  |
| 1 | string | string `title[1]` | ✅ |  |
| 2 | string | string `title[2]` | ✅ |  |
| 3 | string | string `title[3]` | ✅ |  |
| 4 | string | string `title[4]` | ✅ |  |

