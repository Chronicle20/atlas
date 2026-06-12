# NpcContinueConversationSelection (← `CScriptMan::OnAskMenu#Selection`)

- **IDA:** 0x746fad
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/continue_conversation_selection.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ⚠️

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `selection / m_nSelect (AskMenu = 4-byte int; atlas ContinueConversationSelection wide path)` | ✅ |  |
| 1 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

