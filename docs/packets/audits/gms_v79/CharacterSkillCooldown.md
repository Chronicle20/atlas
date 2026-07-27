# CharacterSkillCooldown (← `CUserLocal::OnSkillCooltimeSet`)

- **IDA:** 0x8b3ec5
- **Atlas file:** `libs/atlas-packet/character/clientbound/skill_cooldown.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | int16 | int16 `` | ✅ |  |

