# NpcAskTextConversationDetail (← `CScriptMan::OnAskText#AskText`)

- **IDA:** 0x791cd0
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message text` | ✅ |  |
| 1 | string | string `default/preset text` | ✅ |  |
| 2 | int16 | int16 `minLength (v28)` | ✅ |  |
| 3 | int16 | int16 `maxLength (v7)` | ✅ |  |


Ack: world-audit Phase 3 v87 cross-version on 2026-05-28
