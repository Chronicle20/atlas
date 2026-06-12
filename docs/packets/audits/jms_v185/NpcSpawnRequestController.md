# NpcSpawnRequestController (← `CNpcPool::OnNpcChangeController`)

- **IDA:** 0x720782
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/spawn_request_controller.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `localFlag (@0x720794) -- atlas WriteByte(1)` | ✅ |  |
| 1 | int32 | int32 `npcId / id (@0x720797)` | ✅ |  |
| 2 | int32 | int32 `templateId (SetLocalNpc @0x720294)` | ✅ |  |
| 3 | int16 | int16 `x (CNpc::Init @0x716dd6)` | ✅ |  |
| 4 | int16 | int16 `cy (CNpc::Init @0x716de4)` | ✅ |  |
| 5 | byte | byte `stance/moveAction (CNpc::Init @0x716e0c)` | ✅ |  |
| 6 | int16 | int16 `fh foothold (CNpc::Init @0x716e1c)` | ✅ |  |
| 7 | int16 | int16 `rx0 (CNpc::Init @0x716e3c)` | ✅ |  |
| 8 | int16 | int16 `rx1 (CNpc::Init @0x716e4a)` | ✅ |  |
| 9 | byte | byte `miniMap (CNpc::Init @0x716edf)` | ✅ |  |

