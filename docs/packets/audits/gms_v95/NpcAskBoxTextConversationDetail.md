# NpcAskBoxTextConversationDetail (← `CScriptMan::OnAskBoxText#AskBoxText`)

- **IDA:** 0x6dc9c0
- **Atlas file:** `libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message text` | ✅ |  |
| 1 | string | string `default text` | ✅ |  |
| 2 | int16 | int16 `nCol (columns)` | ✅ |  |
| 3 | int16 | int16 `nLine (lines)` | ✅ |  |


Ack: world-audit sub-phase 2f on 2026-05-28
