# NpcSayConversationDetail (← `CScriptMan::OnSay#Say`)

- **IDA:** 0x7b7315
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message (@0x7b7340)` | ✅ |  |
| 1 | byte | byte `bPrev / previous (@0x7b7356)` | ✅ |  |
| 2 | byte | byte `bNext / next (@0x7b7368)` | ✅ |  |


Ack: world-audit Phase 3 JMS185 npc domain on 2026-05-28
