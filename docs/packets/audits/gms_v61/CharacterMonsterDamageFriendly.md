# CharacterMonsterDamageFriendly (← `CMob::Update`)

- **IDA:** 0x5c71b7
- **Atlas file:** `libs/atlas-packet/character/serverbound/monster_damage_friendly.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `attackerId this.m_dwMobID @0x5c78d9 (op159 site) / mobCrc @0x5c78d9 (op158)` | ✅ |  |
| 1 | int32 | int32 `observerId dwCharacterID g_pWvsContext+8328 @0x5c7ada` | ✅ |  |
| 2 | int32 | int32 `attackedId target.m_dwMobID @0x5c7af7` | ✅ |  |

