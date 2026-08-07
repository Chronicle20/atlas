# CharacterUseSkillBook (← `CWvsContext::SendSkillLearnItemUseRequest`)

- **IDA:** 0xa5459c
- **Atlas file:** `libs/atlas-packet/character/serverbound/use_skill_book.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `updateTime (sub_9C7771(v6,v5) -- get_update_time() twin)` | ✅ |  |
| 1 | int16 | int16 `slot (a2)` | ✅ |  |
| 2 | int32 | int32 `itemId (a3); gated a3/10000 == 228 \|\| sub_4F959A(a3) (skill-book/mastery-book class check)` | ✅ |  |

