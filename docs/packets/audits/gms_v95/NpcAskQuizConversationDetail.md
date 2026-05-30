# NpcAskQuizConversationDetail (← `CScriptMan::OnAskQuiz#AskQuiz`)

- **IDA:** 0x9ffad0
- **Atlas file:** `libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `flag (0=show quiz / Fail==false; nonzero=close)` | ✅ |  |
| 1 | string | string `title -- guarded (flag==0)` | ✅ |  |
| 2 | string | string `problem -- guarded` | ✅ |  |
| 3 | string | string `hint -- guarded` | ✅ |  |
| 4 | int32 | int32 `min -- guarded` | ✅ |  |
| 5 | int32 | int32 `max -- guarded` | ✅ |  |
| 6 | int32 | int32 `remaining time (sec) -- guarded` | ✅ |  |


Ack: world-audit sub-phase 2f on 2026-05-28
