# NpcAskMenuConversationDetail (← `CScriptMan::OnAskMenu#AskMenu`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v61
- **Branch depth:** 3
- **Verdict:** ⚠️

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 1 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

