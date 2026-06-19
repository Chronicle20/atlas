# StatusMessageQuestRecordEx (← `CWvsContext::OnMessage#QuestRecordEx`)

- **IDA:** 0xa2160b
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `outer mode (QUEST_RECORD_EX)` | ✅ |  |
| 1 | int16 | int16 `questId` | ✅ |  |
| 2 | string | string `info string` | ✅ |  |

