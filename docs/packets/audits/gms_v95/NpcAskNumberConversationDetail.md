# NpcAskNumberConversationDetail (← `CScriptMan::OnAskNumber#AskNumber`)

- **IDA:** 0x6dcc00
- **Atlas file:** `libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message text` | ✅ |  |
| 1 | int32 | int32 `nDef (default)` | ✅ |  |
| 2 | int32 | int32 `nMin` | ✅ |  |
| 3 | int32 | int32 `nMax` | ✅ |  |


Ack: world-audit sub-phase 2f on 2026-05-28
