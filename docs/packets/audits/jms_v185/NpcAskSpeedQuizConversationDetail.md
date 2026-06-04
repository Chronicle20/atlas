# NpcAskSpeedQuizConversationDetail (← `CScriptMan::OnAskSpeedQuiz#AskSpeedQuiz`)

- **IDA:** 0x7b8501
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `flag (0=show / Fail==false @0xb0e4a2)` | ✅ |  |
| 1 | int32 | int32 `type -- guarded (flag==0 @0xb0e4ef)` | ✅ |  |
| 2 | int32 | int32 `answer -- guarded (@0xb0e4f8)` | ✅ |  |
| 3 | int32 | int32 `correct -- guarded (@0xb0e502)` | ✅ |  |
| 4 | int32 | int32 `remain -- guarded (@0xb0e50c)` | ✅ |  |
| 5 | int32 | int32 `remaining time (1000x sec) -- guarded (@0xb0e520)` | ✅ |  |

