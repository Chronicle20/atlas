# GuildNoticeChange (← `CWvsContext::OnGuildResult#NoticeChange`)

- **IDA:** 0xb22518
- **Atlas file:** `../../libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (NoticeChange/'D')` | ✅ |  |
| 1 | int32 | int32 `guildId` | ✅ |  |
| 2 | string | string `new notice text` | ✅ |  |

