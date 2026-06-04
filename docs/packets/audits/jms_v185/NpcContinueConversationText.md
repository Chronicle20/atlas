# NpcContinueConversationText (← `CScriptMan::OnAskText#Reply`)

- **IDA:** 0x7b77bd
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/continue_conversation_text.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | byte `msgType = 3 (AskText @0x7b78d4)` | ❌ | width mismatch |
| 1 | byte | byte `action (@0x7b78e2)` | ❌ | atlas: short — missing trailing field |
| 2 | byte | string `answer text (only if action==1 @0x7b7900)` | ❌ | atlas: short — missing trailing field |

