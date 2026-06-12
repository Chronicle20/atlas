# MonsterSpawn (← `CMobPool::OnMobEnterField`)

- **IDA:** 0x6589e0
- **Atlas file:** `../../libs/atlas-packet/monster/clientbound/spawn.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwMobId (uniqueId)` | ✅ |  |
| 1 | byte | byte `nCalcDamageIndex (atlas: controlled — region/version gated >v12 GMS \|\| JMS)` | ✅ |  |
| 2 | int32 | int32 `dwTemplateID (monsterId)` | ✅ |  |
| 3 | bytes | bytes `MonsterModel body (atlas delegates to m.monster.Encode; IDA delegates to CMob::SetTemporaryStat + CMob::Init — variable-length sub-struct)` | ✅ |  |

