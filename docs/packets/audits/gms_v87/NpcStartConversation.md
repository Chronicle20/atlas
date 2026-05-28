# NpcStartConversation (← `CUserLocal::TalkToNpc`)

- **IDA:** 0x9e3066
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/start_conversation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `npcObjectId (arg4+164)` | ✅ |  |
| 1 | int16 | int16 `x (cursor/char x)` | ✅ |  |
| 2 | int16 | int16 `y (cursor/char y)` | ✅ |  |


Ack: world-audit Phase 3 v87 cross-version on 2026-05-28
