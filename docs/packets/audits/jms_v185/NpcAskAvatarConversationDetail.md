# NpcAskAvatarConversationDetail (← `CScriptMan::OnAskAvatar#AskAvatar`)

- **IDA:** 0x7b7e1d
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message (@0x7b7e3e)` | ✅ |  |
| 1 | byte | byte `style count (@0x7b7e52)` | ✅ |  |
| 2 | int32 | int32 `style id -- loop body (count iterations; analyzer flattens @0x7b7e6e)` | ✅ |  |


Ack: world-audit Phase 3 JMS185 npc domain on 2026-05-28
