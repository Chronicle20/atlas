# NpcAskTextConversationDetail (← `CScriptMan::OnAskText#AskText`)

- **IDA:** 0x7b77bd
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message (@0x7b77d8)` | ✅ |  |
| 1 | string | string `default text (@0x7b77e8)` | ✅ |  |
| 2 | int16 | int16 `nStrMin (@0x7b77fd)` | ✅ |  |
| 3 | int16 | int16 `nStrMax (@0x7b780f)` | ✅ |  |


Ack: world-audit Phase 3 JMS185 npc domain on 2026-05-28
