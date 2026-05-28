# NpcContinueConversationText (← `CScriptMan::OnAskText#Reply`)

- **IDA:** 0x746a8b
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/continue_conversation_text.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `input text (atlas ContinueConversationText.text; only emitted when action==1)` | ✅ |  |


Ack: world-audit Phase 3 v83 (12b npc) on 2026-05-28
