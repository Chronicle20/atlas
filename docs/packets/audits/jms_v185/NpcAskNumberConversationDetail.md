# NpcAskNumberConversationDetail (← `CScriptMan::OnAskNumber#AskNumber`)

- **IDA:** 0x7b7b0d
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message (@0x7b7b28)` | ✅ |  |
| 1 | int32 | int32 `default value (@0x7b7b3b)` | ✅ |  |
| 2 | int32 | int32 `min (@0x7b7b45)` | ✅ |  |
| 3 | int32 | int32 `max (@0x7b7b57)` | ✅ |  |

