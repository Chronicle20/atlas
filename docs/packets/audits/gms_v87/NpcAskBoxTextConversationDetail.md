# NpcAskBoxTextConversationDetail (← `CScriptMan::OnAskBoxText#AskBoxText`)

- **IDA:** 0x791e79
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message text` | ✅ |  |
| 1 | string | string `default/preset text` | ✅ |  |
| 2 | int16 | int16 `column / width (v27)` | ✅ |  |
| 3 | int16 | int16 `line / height (v7)` | ✅ |  |

