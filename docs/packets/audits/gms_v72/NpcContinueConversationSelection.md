# NpcContinueConversationSelection (← `CScriptMan::OnAskMenu#Selection`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/npc/serverbound/continue_conversation_selection.go`
- **Variant:** GMS/v72
- **Branch depth:** 1
- **Verdict:** ⚠️

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 1 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

