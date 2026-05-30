# NpcAskSpeedQuizConversationDetail (← `CScriptMan::OnAskSpeedQuiz#AskSpeedQuiz`)

- **IDA:** 0x792ba2
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `flag (0=show / Fail==false)` | ✅ |  |
| 1 | int32 | int32 `type -- guarded (flag==0)` | ✅ |  |
| 2 | int32 | int32 `answer -- guarded` | ✅ |  |
| 3 | int32 | int32 `correct -- guarded` | ✅ |  |
| 4 | int32 | int32 `remain -- guarded` | ✅ |  |
| 5 | int32 | int32 `remaining time (sec) -- guarded` | ✅ |  |


Ack: world-audit Phase 3 v87 cross-version on 2026-05-28
