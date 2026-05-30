# NpcContinueConversation (← `CScriptMan::OnSay#Reply`)

- **IDA:** 0x6dc110
- **Atlas file:** `libs/atlas-packet/npc/serverbound/continue_conversation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `msgType / lastMessageType (dialog-type discriminator echoed back; 0=SAY here)` | ✅ |  |
| 1 | byte | byte `action (button result: -1/0/1)` | ✅ |  |


Ack: world-audit Phase 2g on 2026-05-28
