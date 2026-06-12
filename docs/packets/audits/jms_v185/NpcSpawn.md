# NpcSpawn (← `CNpcPool::OnNpcEnterField`)

- **IDA:** 0x72068f
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/spawn.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `npcId (object id)` | ✅ |  |
| 1 | int32 | int32 `templateId` | ✅ |  |
| 2 | int16 | bytes `CNpc::Init(iPacket) — opaque npc position/appearance block` | ✅ |  |
| 3 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 4 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 5 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 6 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 7 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 8 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |

