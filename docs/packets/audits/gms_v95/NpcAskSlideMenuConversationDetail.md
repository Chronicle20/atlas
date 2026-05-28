# NpcAskSlideMenuConversationDetail (← `CScriptMan::OnAskSlideMenu#AskSlideMenu`)

- **IDA:** 0x6dbe50
- **Atlas file:** `libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `slideDlgType (atlas Unknown 0/1; selects dialog variant) -- GMS major>83` | ✅ |  |
| 1 | int32 | int32 `menuType` | ✅ |  |
| 2 | string | string `button info / message text` | ✅ |  |


Ack: world-audit sub-phase 2f on 2026-05-28
