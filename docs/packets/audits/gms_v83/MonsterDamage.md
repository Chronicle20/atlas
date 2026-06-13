# MonsterDamage (← `CMob::OnDamaged`)

- **IDA:** 0x66c6c2
- **Atlas file:** `libs/atlas-packet/monster/clientbound/damage.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwMobId — read by CMobPool::OnMobPacket before dispatch` | ✅ |  |
| 1 | byte | byte `damageType` | ✅ |  |
| 2 | int32 | int32 `damage amount` | ✅ |  |
| 3 | int32 | int32 `hp current — gated mob.bDamagedByMob` | ✅ |  |
| 4 | int32 | int32 `hp max — gated mob.bDamagedByMob` | ✅ |  |

