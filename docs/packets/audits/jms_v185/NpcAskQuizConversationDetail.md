# NpcAskQuizConversationDetail (← `CScriptMan::OnAskQuiz#AskQuiz`)

- **IDA:** 0x7b84ef
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `flag (0=show quiz / Fail==false; nonzero=close @0xb0e34d)` | ✅ |  |
| 1 | string | string `title -- guarded (flag==0)` | ✅ |  |
| 2 | string | string `problem -- guarded` | ✅ |  |
| 3 | string | string `hint -- guarded` | ✅ |  |
| 4 | int32 | int32 `min -- guarded (@0xb0e3c3)` | ✅ |  |
| 5 | int32 | int32 `max -- guarded (@0xb0e3cd)` | ✅ |  |
| 6 | int32 | int32 `remaining time (1000x sec) -- guarded (@0xb0e3e1)` | ✅ |  |


Ack: world-audit Phase 3 JMS185 npc domain on 2026-05-28
