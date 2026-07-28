# CharacterSkillLearnItemResult (← `CWvsContext::OnSkillLearnItemResult`)

- **IDA:** 0xab58e8
- **Atlas file:** `libs/atlas-packet/character/clientbound/skill_learn_item_result.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bOnExclRequest (v87 leading byte, consumed-and-discarded by the codec; clears the requester's exclusive-request lock when true -- this[2093]=0, this[2094]=get_update_time()) @0xab5904` | ✅ |  |
| 1 | int32 | int32 `characterId (CUserPool::GetUser lookup) @0xab5923` | ✅ |  |
| 2 | byte | byte `isMasteryBook @0xab596b` | ✅ |  |
| 3 | int32 | int32 `skillId (decoded, discarded) @0xab596e` | ✅ |  |
| 4 | int32 | int32 `masterLevel (decoded, discarded) @0xab5975` | ✅ |  |
| 5 | byte | byte `canUse @0xab5986` | ✅ |  |
| 6 | byte | byte `success @0xab5989` | ✅ |  |

