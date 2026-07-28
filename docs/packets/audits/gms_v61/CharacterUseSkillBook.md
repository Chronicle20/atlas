# CharacterUseSkillBook (← `CWvsContext::SendSkillLearnItemUseRequest`)

- **IDA:** 0x8325d2
- **Atlas file:** `libs/atlas-packet/character/serverbound/use_skill_book.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `updateTime` | ✅ |  |
| 1 | int16 | int16 `slot (a2)` | ✅ |  |
| 2 | int32 | int32 `itemId (a3); gated a3/10000 in {228,229} (skill-book item prefix)` | ✅ |  |

