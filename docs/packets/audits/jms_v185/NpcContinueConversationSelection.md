# NpcContinueConversationSelection (← `CScriptMan::OnAskMenu#Selection`)

- **IDA:** 0x7b7c95
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/continue_conversation_selection.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `msgType = 5 (AskMenu @0x7b7d82)` | ❌ | width mismatch |
| 1 | byte | byte `action (@0x7b7da4)` | ✅ |  |
| 2 | byte | int32 `selection (only if action==1 @0x7b7daf)` | ❌ | atlas: short — missing trailing field |

