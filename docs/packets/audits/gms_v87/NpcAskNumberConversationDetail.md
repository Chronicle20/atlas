# NpcAskNumberConversationDetail (← `CScriptMan::OnAskNumber#AskNumber`)

- **IDA:** 0x792020
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message text` | ✅ |  |
| 1 | int32 | int32 `default (v26)` | ✅ |  |
| 2 | int32 | int32 `min (v27)` | ✅ |  |
| 3 | int32 | int32 `max (v7)` | ✅ |  |

