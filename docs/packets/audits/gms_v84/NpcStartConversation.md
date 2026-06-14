# NpcStartConversation (← `CUserLocal::TalkToNpc`)

- **IDA:** 0x99ec4e
- **Atlas file:** `libs/atlas-packet/npc/serverbound/start_conversation.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | int16 | int16 `` | ✅ |  |
| 2 | int16 | int16 `` | ✅ |  |

