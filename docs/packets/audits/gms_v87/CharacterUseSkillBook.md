# CharacterUseSkillBook (← `CWvsContext::SendSkillLearnItemUseRequest`)

- **IDA:** 0xa9fa66
- **Atlas file:** `libs/atlas-packet/character/serverbound/use_skill_book.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `updateTime (get_update_time()) @0xa9fabe` | ✅ |  |
| 1 | int16 | int16 `slot (a2) @0xa9fac9` | ✅ |  |
| 2 | int32 | int32 `itemId (arg4); gated arg4/10000 == 228 \|\| sub_512125(arg4) (skill-book/mastery-book class check) @0xa9fad4` | ✅ |  |

