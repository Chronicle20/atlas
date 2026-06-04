# NpcSpawnRequestController (← `CNpcPool::OnNpcChangeController`)

- **IDA:** 0x7170c8
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/spawn_request_controller.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bLocal / v3 (1=local controller w/ spawn body, 0=remote)` | ✅ |  |
| 1 | int32 | int32 `npcId / v4 (object id)` | ✅ |  |
| 2 | int32 | int32 `templateId (SetLocalNpc@0x716be2 CNpcTemplate id)` | ✅ |  |
| 3 | int16 | int16 `x (this+86)` | ✅ |  |
| 4 | int16 | int16 `cy (y, this+87)` | ✅ |  |
| 5 | byte | byte `moveAction (atlas f flag, this+43)` | ✅ |  |
| 6 | int16 | int16 `fh (foothold, GetFoothold arg)` | ✅ |  |
| 7 | int16 | int16 `rx0 (this+39)` | ✅ |  |
| 8 | int16 | int16 `rx1 (this+40)` | ✅ |  |
| 9 | byte | byte `enabled (this+77; atlas trailing flag)` | ✅ |  |

