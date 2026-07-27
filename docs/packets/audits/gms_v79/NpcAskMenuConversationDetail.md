# NpcAskMenuConversationDetail (← `CScriptMan::OnAskMenu#AskMenu`)

- **IDA:** 0x6c8863
- **Atlas file:** `libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v79
- **Branch depth:** 3
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message text` | ✅ |  |
| 1 | byte | byte `count (avatar look ids; 0 for plain #L# text menus)` | ✅ |  |

