# NpcContinueConversation (← `CScriptMan::OnSay#Reply`)

- **IDA:** 0x6a0d23
- **Atlas file:** `libs/atlas-packet/npc/serverbound/continue_conversation.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `lastMessageType` | ✅ |  |
| 1 | byte | byte `action` | ✅ |  |

