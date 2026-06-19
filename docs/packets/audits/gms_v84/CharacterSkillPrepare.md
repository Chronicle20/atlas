# CharacterSkillPrepare (← `CUserLocal::DoActiveSkill_Prepare`)

- **IDA:** 0x9a9761
- **Atlas file:** `libs/atlas-packet/character/serverbound/skill_prepare.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `nSkillID (skill to prepare)` | ✅ |  |
| 1 | byte | byte `nSLV (skill level / action arg, Value)` | ✅ |  |
| 2 | int16 | int16 `action ((bMoveAction << 15) \| (nOneTimeAction & 0x7FFF))` | ✅ |  |
| 3 | byte | byte `nActionSpeed (v46)` | ✅ |  |
| 4 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

