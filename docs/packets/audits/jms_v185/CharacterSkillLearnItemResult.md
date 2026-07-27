# CharacterSkillLearnItemResult (← `CWvsContext::OnSkillLearnItemResult`)

- **IDA:** 0xb05116
- **Atlas file:** `libs/atlas-packet/character/clientbound/skill_learn_item_result.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bOnExclRequest (jms_v185 leading byte, consumed-and-discarded by the codec; clears the requester's exclusive-request lock when true) @0xb05132` | ✅ |  |
| 1 | int32 | int32 `characterId (CUserPool::GetUser lookup) @0xb05151` | ✅ |  |
| 2 | byte | byte `isMasteryBook @0xb05199` | ✅ |  |
| 3 | int32 | int32 `skillId (decoded, discarded) @0xb0519c` | ✅ |  |
| 4 | int32 | int32 `masterLevel (decoded, discarded) @0xb051a3` | ✅ |  |
| 5 | byte | byte `canUse @0xb051b4` | ✅ |  |
| 6 | byte | byte `success @0xb051b7` | ✅ |  |

