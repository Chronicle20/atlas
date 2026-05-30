# NpcContinueConversation (← `CScriptMan::OnSay#Reply`)

- **IDA:** 0x7467ab
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/continue_conversation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `msgType (v83 Say=0; dispatcher discriminator)` | ✅ |  |
| 1 | byte | byte `action / button result` | ✅ |  |


Ack: world-audit Phase 3 v83 (12b npc) on 2026-05-28
