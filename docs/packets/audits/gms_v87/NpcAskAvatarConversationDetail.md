# NpcAskAvatarConversationDetail (← `CScriptMan::OnAskAvatar#AskAvatar`)

- **IDA:** 0x792330
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message text` | ✅ |  |
| 1 | byte | byte `avatar count (v5)` | ✅ |  |
| 2 | int32 | int32 `avatar look id -- loop body (count iterations; analyzer flattens)` | ✅ |  |

