# CharacterSkillCooldown (← `CUserLocal::OnSkillCooltimeSet`)

- **IDA:** 0x86851a
- **Atlas file:** `libs/atlas-packet/character/clientbound/skill_cooldown.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | int16 | int16 `` | ✅ |  |

