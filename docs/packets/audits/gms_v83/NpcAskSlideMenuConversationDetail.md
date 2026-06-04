# NpcAskSlideMenuConversationDetail (← `CScriptMan::OnAskSlideMenu#AskSlideMenu`)

- **IDA:** 0x76b5c8
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v83
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `menuType / dwType (this+33)` | ✅ |  |
| 1 | string | string `message text (v11)` | ✅ |  |

