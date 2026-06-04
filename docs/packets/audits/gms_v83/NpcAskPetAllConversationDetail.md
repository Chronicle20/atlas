# NpcAskPetAllConversationDetail (← `CScriptMan::OnAskPetAll#AskPetAll`)

- **IDA:** 0x74775c
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message text (v27)` | ✅ |  |
| 1 | byte | byte `pet count (v4)` | ✅ |  |
| 2 | byte | byte `exceptionExists flag (v24)` | ✅ |  |
| 3 | int64 | bytes `cashId (8-byte buffer) -- loop body` | ✅ |  |
| 4 | byte | byte `unused byte per pet -- loop body` | ✅ |  |

