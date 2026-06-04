# NpcAskPetConversationDetail (← `CScriptMan::OnAskPet#AskPet`)

- **IDA:** 0x7b7fc2
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message (@0x7b7fde)` | ✅ |  |
| 1 | byte | byte `pet count (@0x7b7ff8)` | ✅ |  |
| 2 | int64 | bytes `cashId (8 bytes) -- loop body (@0x7b8046)` | ✅ |  |
| 3 | byte | byte `unused -- loop body (@0x7b804e)` | ✅ |  |

