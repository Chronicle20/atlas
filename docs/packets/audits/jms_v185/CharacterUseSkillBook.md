# CharacterUseSkillBook (← `CWvsContext::SendSkillLearnItemUseRequest`)

- **IDA:** 0xaeee61
- **Atlas file:** `libs/atlas-packet/character/serverbound/use_skill_book.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `updateTime (get_update_time()) @0xaeeeba` | ✅ |  |
| 1 | int16 | int16 `slot (nPOS) @0xaeeec5` | ✅ |  |
| 2 | int32 | int32 `itemId (nItemID); gated nItemID/10000 == 228 \|\| v4==229 \|\| v4==562 @0xaeeed0` | ✅ |  |

