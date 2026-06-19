# CharacterSkillCancelForeign (← `CUserRemote::OnSkillCancel`)

- **IDA:** 0x9c0dd3
- **Atlas file:** `libs/atlas-packet/character/clientbound/skill_cancel_foreign.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `nSkillID (skill to cancel)` | ✅ |  |
| 1 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

