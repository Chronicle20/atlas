# NpcStartConversation (← `CUserLocal::TalkToNpc`)

- **IDA:** 0x8b7e10
- **Atlas file:** `libs/atlas-packet/npc/serverbound/start_conversation.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | int16 | int16 `` | ✅ |  |
| 2 | int16 | int16 `` | ✅ |  |

