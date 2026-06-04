# NpcAskBoxTextConversationDetail (← `CScriptMan::OnAskBoxText#AskBoxText`)

- **IDA:** 0x7b7966
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message (@0x7b7981)` | ✅ |  |
| 1 | string | string `default text (@0x7b7991)` | ✅ |  |
| 2 | int16 | int16 `col (@0x7b79a6)` | ✅ |  |
| 3 | int16 | int16 `line (@0x7b79b8)` | ✅ |  |

