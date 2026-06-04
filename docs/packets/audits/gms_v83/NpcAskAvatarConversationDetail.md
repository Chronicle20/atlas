# NpcAskAvatarConversationDetail (← `CScriptMan::OnAskAvatar#AskAvatar`)

- **IDA:** 0x74713d
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message text (v26)` | ✅ |  |
| 1 | byte | byte `style count (v5)` | ✅ |  |
| 2 | int32 | int32 `style id -- loop body (count iterations; analyzer flattens)` | ✅ |  |

