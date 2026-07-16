# CharacterSkillPrepare (← `CUserLocal::DoActiveSkill_Prepare`)

- **IDA:** 0x7b8001
- **Atlas file:** `libs/atlas-packet/character/serverbound/skill_prepare.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | byte | byte `` | ✅ |  |
| 2 | byte | byte `` | ✅ |  |
| 3 | int16 | byte `` | ❌ | width mismatch |
| 4 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

