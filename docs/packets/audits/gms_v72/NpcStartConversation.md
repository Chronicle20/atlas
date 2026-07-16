# NpcStartConversation (← `CUserLocal::TalkToNpc`)

- **IDA:** 0x63fd91
- **Atlas file:** `libs/atlas-packet/npc/serverbound/start_conversation.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `oid` | ✅ |  |
| 1 | int16 | int16 `userX` | ✅ |  |
| 2 | int16 | int16 `userY` | ✅ |  |
