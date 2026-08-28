# NpcSpawn (← `CNpcPool::OnNpcEnterField`)

- **IDA:** 0x72068f
- **Atlas file:** `libs/atlas-packet/npc/clientbound/spawn.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `npcId (object id) (@0x7206a1)` | ✅ |  |
| 1 | int32 | int32 `templateId (@0x7206d8)` | ✅ |  |
| 2 | int16 | int16 `x (CNpc::Init @0x716dd6)` | ✅ |  |
| 3 | int16 | int16 `cy (CNpc::Init @0x716de4)` | ✅ |  |
| 4 | byte | byte `stance/moveAction (CNpc::Init @0x716e0c)` | ✅ |  |
| 5 | int16 | int16 `fh foothold (CNpc::Init @0x716e1c)` | ✅ |  |
| 6 | int16 | int16 `rx0 (CNpc::Init @0x716e3c)` | ✅ |  |
| 7 | int16 | int16 `rx1 (CNpc::Init @0x716e4a)` | ✅ |  |
| 8 | byte | byte `miniMap / bEnabled (CNpc::Init @0x716edf)` | ✅ |  |

