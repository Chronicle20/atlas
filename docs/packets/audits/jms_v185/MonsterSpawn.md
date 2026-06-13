# MonsterSpawn (← `CMobPool::OnMobEnterField`)

- **IDA:** 0x6f885c
- **Atlas file:** `libs/atlas-packet/monster/clientbound/spawn.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwMobId` | ✅ |  |
| 1 | byte | byte `nCalcDamageIndex (controlled — atlas writes for JMS too)` | ✅ |  |
| 2 | int32 | int32 `dwTemplateID` | ✅ |  |
| 3 | bytes | bytes `MonsterModel body` | ✅ |  |

