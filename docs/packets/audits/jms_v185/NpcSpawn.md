# NpcSpawn (← `CNpcPool::OnNpcEnterField`)

- **IDA:** 0x72068f
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/spawn.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `npcId (object id @0x7206a1)` | ✅ |  |
| 1 | int32 | int32 `templateId (new-entry path @0x7206d8)` | ✅ |  |
| 2 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 7 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 8 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

