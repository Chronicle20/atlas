# NpcAskPetAllConversationDetail (← `CScriptMan::OnAskPetAll#AskPetAll`)

- **IDA:** 0x7b8250
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message (@0x7b826a)` | ✅ |  |
| 1 | byte | byte `pet count (@0x7b8281)` | ✅ |  |
| 2 | byte | byte `exceptionExists (@0x7b8290)` | ✅ |  |
| 3 | int64 | bytes `cashId (8 bytes) -- loop body (@0x7b82d7)` | ✅ |  |
| 4 | byte | byte `unused -- loop body (@0x7b82df)` | ✅ |  |

