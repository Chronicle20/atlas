# NpcContinueConversation (← `CScriptMan::OnSay#Reply`)

- **IDA:** 0x7b7315
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/continue_conversation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `msgType = 0 (SAY @0x7b7425)` | ✅ |  |
| 1 | byte | byte `action (@0x7b7447)` | ✅ |  |


Ack: world-audit Phase 3 JMS185 npc domain on 2026-05-28
