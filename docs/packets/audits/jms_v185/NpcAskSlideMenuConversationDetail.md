# NpcAskSlideMenuConversationDetail (← `CScriptMan::OnAskSlideMenu#AskSlideMenu`)

- **IDA:** 0x7b8513
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `slideDlgType (this[37] @0x7e2ab4) -- JMS reads this unconditionally; atlas gates it GMS>83` | ✅ |  |
| 1 | int32 | int32 `menuType (v13; compared in selection loop @0x7e2ac2)` | ✅ |  |
| 2 | string | string `button info / message text (@0x7e2ac9)` | ✅ |  |


Ack: world-audit Phase 3 JMS185 npc domain on 2026-05-28
