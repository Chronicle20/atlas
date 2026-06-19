# CharacterSkillPrepare (← `CUserLocal::DoActiveSkill_Prepare`)

- **IDA:** 0x941710
- **Atlas file:** `libs/atlas-packet/character/serverbound/skill_prepare.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `nSkillID (skill to prepare)` | ✅ |  |
| 1 | byte | byte `nSLV (skill level, nSLV)` | ✅ |  |
| 2 | int16 | int16 `action ((m_nMoveAction << 15) \| (m_nOneTimeAction & 0x7FFF))` | ✅ |  |
| 3 | byte | byte `nActionSpeed (attack_speed_degree)` | ✅ |  |
| 4 | int32 | int32 `dwSwallowMobID (only when nSkillID == 33101005)` | ✅ |  |

