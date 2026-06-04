# NpcStartConversation (← `CUserLocal::TalkToNpc`)

- **IDA:** 0xa2cc90
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/start_conversation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `npcId (m_dwNpcId @0xa2cd15)` | ✅ |  |
| 1 | int16 | int16 `x (GetPos.x @0xa2cd30)` | ✅ |  |
| 2 | int16 | int16 `y (GetPos.y @0xa2cd48)` | ✅ |  |

