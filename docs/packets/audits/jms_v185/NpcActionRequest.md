# NpcActionRequest (← `CNpc::GenerateMovePath`)

- **IDA:** 0x7199ce
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/action.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `npcId (m_dwNpcId @0x719a7e)` | ✅ |  |
| 1 | byte | byte `nAction (@0x719a89)` | ✅ |  |
| 2 | byte | byte `nChatIdx (@0x719a94)` | ✅ |  |
| 3 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

