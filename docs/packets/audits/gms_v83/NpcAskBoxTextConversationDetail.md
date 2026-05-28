# NpcAskBoxTextConversationDetail (← `CScriptMan::OnAskBoxText#AskBoxText`)

- **IDA:** 0x746c46
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message text (v28)` | ✅ |  |
| 1 | string | string `default text (a4)` | ✅ |  |
| 2 | int16 | int16 `col (v27)` | ✅ |  |
| 3 | int16 | int16 `line (v7)` | ✅ |  |


Ack: world-audit Phase 3 v83 (12b npc) on 2026-05-28
