# MonsterSpawn (← `CMobPool::OnMobEnterField`)

- **IDA:** 0x67945a
- **Atlas file:** `../../libs/atlas-packet/monster/clientbound/spawn.go`
- **Variant:** GMS/v83
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwMobId (uniqueId)` | ✅ |  |
| 1 | byte | byte `nCalcDamageIndex (controlled — gated >v12 GMS, on for v83)` | ✅ |  |
| 2 | int32 | int32 `dwTemplateID (monsterId)` | ✅ |  |
| 3 | bytes | bytes `MonsterModel body via CMob::SetTemporaryStat + CMob::Init` | ✅ |  |

