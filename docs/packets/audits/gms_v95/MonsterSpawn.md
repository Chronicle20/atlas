# MonsterSpawn (← `CMobPool::OnMobEnterField`)

- **IDA:** 0x6589e0
- **Atlas file:** `libs/atlas-packet/monster/clientbound/spawn.go`
- **Variant:** GMS/v95
- **Branch depth:** 3
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwMobId (uniqueId)` | ✅ |  |
| 1 | byte | byte `nCalcDamageIndex (atlas: controlled — region/version gated >v12 GMS \|\| JMS)` | ✅ |  |
| 2 | byte | int32 `dwTemplateID (monsterId)` | ❌ | width mismatch |
| 3 | int32 | bytes `MonsterModel body (atlas delegates to m.monster.Encode; IDA delegates to CMob::SetTemporaryStat + CMob::Init — variable-length sub-struct)` | ❌ | width mismatch |
| 4 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

