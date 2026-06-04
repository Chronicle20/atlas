# NpcAskTextConversationDetail (← `CScriptMan::OnAskText#AskText`)

- **IDA:** 0x746a8b
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message text (v29)` | ✅ |  |
| 1 | string | string `default text (a4)` | ✅ |  |
| 2 | int16 | int16 `min (v28)` | ✅ |  |
| 3 | int16 | int16 `max (v7)` | ✅ |  |

