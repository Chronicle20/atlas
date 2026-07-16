# NpcStartConversation (← `CUserLocal::TalkToNpc`)

- **IDA:** 0x7b1403
- **Atlas file:** `libs/atlas-packet/npc/serverbound/start_conversation.go`
- **Variant:** GMS/v61
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `npcOid Encode4 @0x7b143a (COutPacket(54) ctor @0x7b142d)` | ✅ |  |
| 1 | int16 | int16 `userX Encode2 @0x7b1450` | ✅ |  |
| 2 | int16 | int16 `userY Encode2 @0x7b1461` | ✅ |  |

