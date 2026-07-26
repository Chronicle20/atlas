# CharacterUseSkillBook (← `CWvsContext::SendSkillLearnItemUseRequest`)

- **IDA:** 0x70e3e7
- **Atlas file:** `libs/atlas-packet/character/serverbound/use_skill_book.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `updateTime (sub_4A2518(this,200,0) result)` | ✅ |  |
| 1 | int16 | int16 `slot (a2)` | ✅ |  |
| 2 | int32 | int32 `itemId (a3); gated a3/10000 in {228,229} (skill-book item prefix)` | ✅ |  |

