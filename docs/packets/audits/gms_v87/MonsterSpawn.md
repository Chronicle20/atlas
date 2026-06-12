# MonsterSpawn (← `CMobPool::OnMobEnterField`)

- **IDA:** 0x6b4fa6
- **Atlas file:** `../../libs/atlas-packet/monster/clientbound/spawn.go`
- **Variant:** GMS/v87
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwMobId (uniqueId)` | ✅ |  |
| 1 | byte | byte `nCalcDamageIndex (controlled)` | ✅ |  |
| 2 | int32 | int32 `dwTemplateID (monsterId)` | ✅ |  |
| 3 | bytes | bytes `MonsterModel body via CMob::SetTemporaryStat + CMob::Init` | ✅ |  |

