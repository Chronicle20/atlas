# CharacterSkillLearnItemResult (← `CWvsContext::OnSkillLearnItemResult`)

- **IDA:** 0x9f7af0
- **Atlas file:** `libs/atlas-packet/character/clientbound/skill_learn_item_result.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bOnExclRequest (v95 leading byte, consumed-and-discarded by the codec; clears the requester's exclusive-request lock when true) @0x9f7b31` | ✅ |  |
| 1 | int32 | int32 `characterId (CUserPool::GetUser lookup) @0x9f7b35` | ✅ |  |
| 2 | byte | byte `isMasteryBook @0x9f7b86` | ✅ |  |
| 3 | int32 | int32 `skillId (decoded, discarded) @0x9f7b8a` | ✅ |  |
| 4 | int32 | int32 `masterLevel (decoded, discarded) @0x9f7b91` | ✅ |  |
| 5 | byte | byte `canUse @0x9f7ba2` | ✅ |  |
| 6 | byte | byte `success @0x9f7ba6` | ✅ |  |

