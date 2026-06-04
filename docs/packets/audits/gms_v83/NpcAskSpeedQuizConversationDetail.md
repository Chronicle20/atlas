# NpcAskSpeedQuizConversationDetail (← `CScriptMan::OnAskSpeedQuiz#AskSpeedQuiz`)

- **IDA:** 0xa26c66
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `fail flag (v4)` | ✅ |  |
| 1 | int32 | int32 `type (v5)` | ✅ |  |
| 2 | int32 | int32 `answer (v9)` | ✅ |  |
| 3 | int32 | int32 `correct (v10)` | ✅ |  |
| 4 | int32 | int32 `remain (a2a)` | ✅ |  |
| 5 | int32 | int32 `timeRemaining (*1000)` | ✅ |  |

