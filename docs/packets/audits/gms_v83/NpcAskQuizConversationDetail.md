# NpcAskQuizConversationDetail (← `CScriptMan::OnAskQuiz#AskQuiz`)

- **IDA:** 0xa26b09
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `fail flag (v4; nonzero = clear quiz UI, no further fields)` | ✅ |  |
| 1 | string | string `title (v20; fail==0 branch)` | ✅ |  |
| 2 | string | string `problem (v21)` | ✅ |  |
| 3 | string | string `hint (a2)` | ✅ |  |
| 4 | int32 | int32 `min (v18)` | ✅ |  |
| 5 | int32 | int32 `max (v19)` | ✅ |  |
| 6 | int32 | int32 `timeRemaining (sec, *1000)` | ✅ |  |


Ack: world-audit Phase 3 v83 (12b npc) on 2026-05-28
