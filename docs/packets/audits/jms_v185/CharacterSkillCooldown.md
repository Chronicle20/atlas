# CharacterSkillCooldown (← `CUserLocal::OnSkillCooltimeSet`)

- **IDA:** 0xa2747f
- **Atlas file:** `libs/atlas-packet/character/clientbound/skill_cooldown.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `nSkillID` | ✅ |  |
| 1 | int16 | int16 `usRemainSec cooldown in seconds` | ✅ |  |

