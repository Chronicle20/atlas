# NpcSpawnRequestController (← `CNpcPool::OnNpcChangeController`)

- **IDA:** 0x679730
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/spawn_request_controller.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `localFlag (controller assigned to local user; atlas leading 1)` | ✅ |  |
| 1 | int32 | int32 `npcId` | ✅ |  |
| 2 | int32 | int32 `templateId (SetLocalNpc)` | ✅ |  |
| 3 | int16 | int16 `x` | ✅ |  |
| 4 | int16 | int16 `cy (y)` | ✅ |  |
| 5 | byte | byte `moveAction (atlas f flag)` | ✅ |  |
| 6 | int16 | int16 `fh (foothold)` | ✅ |  |
| 7 | int16 | int16 `rx0` | ✅ |  |
| 8 | int16 | int16 `rx1` | ✅ |  |
| 9 | byte | byte `enabled (m_bEnabled; atlas miniMap bool)` | ✅ |  |

