# NpcAskAvatarConversationDetail (← `CScriptMan::OnAskAvatar#AskAvatar`)

- **IDA:** 0x6dcff0
- **Atlas file:** `libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message text` | ✅ |  |
| 1 | byte | byte `style count` | ✅ |  |
| 2 | int32 | int32 `style id -- loop body (count iterations; analyzer flattens)` | ✅ |  |


Ack: world-audit sub-phase 2f on 2026-05-28
