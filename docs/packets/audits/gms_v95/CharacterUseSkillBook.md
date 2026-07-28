# CharacterUseSkillBook (← `CWvsContext::SendSkillLearnItemUseRequest`)

- **IDA:** 0x9d65e0
- **Atlas file:** `libs/atlas-packet/character/serverbound/use_skill_book.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `updateTime (get_update_time()) @0x9d668e` | ✅ |  |
| 1 | int16 | int16 `slot (nPOS) @0x9d669c` | ✅ |  |
| 2 | int32 | int32 `itemId (nItemID); gated nItemID/10000 == 228 \|\| is_masterybook_item(nItemID) @0x9d66a6` | ✅ |  |

