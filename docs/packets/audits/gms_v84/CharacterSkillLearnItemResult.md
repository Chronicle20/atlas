# CharacterSkillLearnItemResult (← `CWvsContext::OnSkillLearnItemResult`)

- **IDA:** 0xa6984e
- **Atlas file:** `libs/atlas-packet/character/clientbound/skill_learn_item_result.go`
- **Variant:** GMS/v84
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bOnExclRequest (v84+ leading byte, consumed-and-discarded by the codec; clears the requester's exclusive-request lock when true) @0xa6986a` | ✅ |  |
| 1 | int32 | int32 `characterId (CUserPool-style lookup via sub_9B1635) @0xa69889` | ✅ |  |
| 2 | byte | byte `isMasteryBook @0xa698d1` | ✅ |  |
| 3 | int32 | int32 `skillId (decoded, discarded) @0xa698d4` | ✅ |  |
| 4 | int32 | int32 `masterLevel (decoded, discarded) @0xa698db` | ✅ |  |
| 5 | byte | byte `canUse @0xa698ec` | ✅ |  |
| 6 | byte | byte `success @0xa698ef` | ✅ |  |

