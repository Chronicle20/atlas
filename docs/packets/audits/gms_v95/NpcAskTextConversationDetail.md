# NpcAskTextConversationDetail (← `CScriptMan::OnAskText#AskText`)

- **IDA:** 0x6dc790
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message text` | ✅ |  |
| 1 | string | string `default text` | ✅ |  |
| 2 | int16 | int16 `nLenMin (min length)` | ✅ |  |
| 3 | int16 | int16 `nLenMax (max length)` | ✅ |  |

