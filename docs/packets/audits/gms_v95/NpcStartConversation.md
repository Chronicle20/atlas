# NpcStartConversation (← `CUserLocal::TalkToNpc`)

- **IDA:** 0x9321f0
- **Atlas file:** `libs/atlas-packet/npc/serverbound/start_conversation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `npcId (m_dwNpcId; atlas oid)` | ✅ |  |
| 1 | int16 | int16 `x (player GetPos x)` | ✅ |  |
| 2 | int16 | int16 `y (player GetPos y)` | ✅ |  |


Ack: world-audit Phase 2g on 2026-05-28
