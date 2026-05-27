# MonsterSpawn (← `CMobPool::OnMobEnterField`)

- **IDA:** 0x6f885c
- **Atlas file:** `libs/atlas-packet/monster/clientbound/spawn.go`
- **Variant:** JMS/v185
- **Branch depth:** 3
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwMobId` | ✅ |  |
| 1 | byte | byte `nCalcDamageIndex (controlled — atlas writes for JMS too)` | ✅ |  |
| 2 | byte | int32 `dwTemplateID` | ❌ | width mismatch |
| 3 | int32 | bytes `MonsterModel body` | ❌ | width mismatch |
| 4 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

