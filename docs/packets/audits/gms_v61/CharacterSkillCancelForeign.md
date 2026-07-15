# CharacterSkillCancelForeign (← `CUserRemote::OnSkillCancel`)

- **IDA:** 0x7c9b1f
- **Atlas file:** `libs/atlas-packet/character/clientbound/skill_cancel_foreign.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

