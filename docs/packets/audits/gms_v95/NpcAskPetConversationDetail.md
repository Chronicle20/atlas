# NpcAskPetConversationDetail (← `CScriptMan::OnAskPet#AskPet`)

- **IDA:** 0x6dd6e0
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message text` | ✅ |  |
| 1 | byte | byte `pet count (nCount)` | ✅ |  |
| 2 | int64 | int64 `cashItemSN (8-byte SN; client DecodeBuffer(8), atlas WriteLong) -- loop body` | ✅ |  |
| 3 | byte | byte `unused trailing byte -- loop body` | ✅ |  |

