# NpcContinueConversation (← `CScriptMan::OnSay#Reply`)

- **IDA:** 0x791828
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/continue_conversation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `msgType = 0 (Say) -- atlas ContinueConversation message type` | ✅ |  |
| 1 | byte | byte `action (-1 prev / 0 end / 1 next; atlas ContinueConversation action)` | ✅ |  |

