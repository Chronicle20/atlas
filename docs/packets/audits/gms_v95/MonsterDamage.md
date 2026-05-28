# MonsterDamage (← `CMob::OnDamaged`)

- **IDA:** 0x64ecb0
- **Atlas file:** `libs/atlas-packet/monster/clientbound/damage.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwMobId — read by CMobPool::OnMobPacket before dispatch` | ✅ |  |
| 1 | byte | byte `damageType (v3 — 2 = friendly/no-show)` | ✅ |  |
| 2 | int32 | int32 `damage amount (v4/v5)` | ✅ |  |
| 3 | int32 | int32 `hp current — conditional: only if mobTemplate.bDamagedByMob (friendly-mob field)` | ✅ |  |
| 4 | int32 | int32 `hp max — conditional: only if mobTemplate.bDamagedByMob` | ✅ |  |

