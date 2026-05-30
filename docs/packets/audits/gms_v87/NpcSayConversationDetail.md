# NpcSayConversationDetail (← `CScriptMan::OnSay#Say`)

- **IDA:** 0x791828
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message text` | ✅ |  |
| 1 | byte | byte `bPrev (Previous button)` | ✅ |  |
| 2 | byte | byte `bNext (Next button)` | ✅ |  |


Ack: world-audit Phase 3 v87 cross-version on 2026-05-28
