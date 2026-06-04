# NpcContinueConversationSelection (← `CScriptMan::OnAskMenu#Selection`)

- **IDA:** 0x6dce00
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/continue_conversation_selection.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `selection (m_nSelect; AskMenu = 4-byte int; atlas ContinueConversationSelection wide path)` | ✅ |  |
| 1 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

