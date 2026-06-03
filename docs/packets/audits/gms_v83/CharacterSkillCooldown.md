# CharacterSkillCooldown (← `CUserLocal::OnSkillCooltimeSet`)

- **IDA:** 0x95be66
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/skill_cooldown.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `nSkillID (v2 = Decode4)` | ✅ |  |
| 1 | int16 | int16 `usRemainSec cooldown in seconds (Decode2)` | ✅ |  |

