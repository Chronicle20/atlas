# GuildTitleChange (← `CWvsContext::OnGuildResult#TitleChange`)

- **IDA:** 0xa0e239
- **Atlas file:** `../../libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (64)` | ✅ |  |
| 1 | int32 | int32 `guildId` | ✅ |  |
| 2 | string | string `title[0]` | ✅ |  |
| 3 | string | string `title[1]` | ✅ |  |
| 4 | string | string `title[2]` | ✅ |  |
| 5 | string | string `title[3]` | ✅ |  |
| 6 | string | string `title[4]` | ✅ |  |

