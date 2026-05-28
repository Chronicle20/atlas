# GuildTitleChange (← `CWvsContext::OnGuildResult#TitleChange`)

- **IDA:** 0xb22518
- **Atlas file:** `../../libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (TitleChange/'>'}` | ✅ |  |
| 1 | int32 | int32 `guildId` | ✅ |  |
| 2 | string | string `title[0]` | ✅ |  |
| 3 | byte | string `title[1]` | ❌ | atlas: short — missing trailing field |
| 4 | byte | string `title[2]` | ❌ | atlas: short — missing trailing field |
| 5 | byte | string `title[3]` | ❌ | atlas: short — missing trailing field |
| 6 | byte | string `title[4]` | ❌ | atlas: short — missing trailing field |

