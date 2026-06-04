# NpcContinueConversationText (← `CScriptMan::OnAskText#Reply`)

- **IDA:** 0x791cd0
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/continue_conversation_text.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `entered text (atlas ContinueConversationText trailing string; only when accepted)` | ✅ |  |

