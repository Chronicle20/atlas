# GuildQuestWaitingNotice (← `CWvsContext::OnGuildResult#QuestWaitingNotice`)

- **IDA:** 0xb22518
- **Atlas file:** `libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (QUEST_WAITING_NOTICE)` | ✅ |  |
| 1 | byte | byte `channel` | ✅ |  |
| 2 | int32 | int32 `state` | ✅ |  |

