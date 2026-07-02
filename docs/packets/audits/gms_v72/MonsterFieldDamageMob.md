# MonsterFieldDamageMob (← `CMob::Update`)

- **IDA:** 0x616dd0
- **Atlas file:** `libs/atlas-packet/monster/serverbound/field_damage_mob.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `attackerId this.m_dwMobID @0x6176e1` | ✅ |  |
| 1 | int32 | int32 `observerId dwCharacterID @0x6176f4` | ✅ |  |
| 2 | byte | int32 `attackedId target.m_dwMobID @0x617711` | ❌ | atlas: short — missing trailing field |

