# NpcAskPetAllConversationDetail (← `CScriptMan::OnAskPetAll#AskPetAll`)

- **IDA:** 0x7928f1
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message text` | ✅ |  |
| 1 | byte | byte `pet count (nCount)` | ✅ |  |
| 2 | byte | byte `bExceptionExist (exception list present)` | ✅ |  |
| 3 | int64 | int64 `cashItemSN (8-byte SN; client DecodeBuffer(8), atlas WriteLong) -- loop body` | ✅ |  |
| 4 | byte | byte `unused trailing byte -- loop body` | ✅ |  |


Ack: world-audit Phase 3 v87 cross-version on 2026-05-28
