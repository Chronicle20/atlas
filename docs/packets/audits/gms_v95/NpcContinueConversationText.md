# NpcContinueConversationText (← `CScriptMan::OnAskText#Reply`)

- **IDA:** 0x6dc790
- **Atlas file:** `libs/atlas-packet/npc/serverbound/continue_conversation_text.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `input text (atlas ContinueConversationText.text)` | ✅ |  |


Ack: world-audit Phase 2g on 2026-05-28
