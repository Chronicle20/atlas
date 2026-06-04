# NpcStartConversation (← `CUserLocal::TalkToNpc`)

- **IDA:** 0x95fe9e
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/start_conversation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `npcId / oid (v9)` | ✅ |  |
| 1 | int16 | int16 `x` | ✅ |  |
| 2 | int16 | int16 `y` | ✅ |  |

