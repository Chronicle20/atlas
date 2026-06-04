# NpcAskPetConversationDetail (← `CScriptMan::OnAskPet#AskPet`)

- **IDA:** 0x7474a2
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message text (v26)` | ✅ |  |
| 1 | byte | byte `pet count (v4)` | ✅ |  |
| 2 | int64 | bytes `cashId (8-byte buffer) -- loop body` | ✅ |  |
| 3 | byte | byte `unused byte per pet -- loop body` | ✅ |  |

