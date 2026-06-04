# NpcAction (← `CNpc::OnMove`)

- **IDA:** 0x6d2e07
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/action.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `npcId (consumed by CNpcPool::OnNpcPacket@0x6d98de dispatcher before OnMove; atlas Action.objectId)` | ✅ |  |
| 1 | byte | byte `action / v3 (atlas unk)` | ✅ |  |
| 2 | byte | byte `chatIdx / v4 (atlas unk2)` | ✅ |  |
| 3 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

