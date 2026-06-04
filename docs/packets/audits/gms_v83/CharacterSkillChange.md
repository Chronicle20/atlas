# CharacterSkillChange (← `CWvsContext::OnChangeSkillRecordResult`)

- **IDA:** 0xa1e48c
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/skill_change.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `exclRequestSent flag (bExclRequestSent)` | ✅ |  |
| 1 | int16 | int16 `count of skill entries` | ✅ |  |
| 2 | int32 | int32 `nSkillID (per entry, loop count times)` | ✅ |  |
| 3 | int32 | int32 `nLevel (nInfo, per entry)` | ✅ |  |
| 4 | int32 | int32 `nMasterLevel (per entry, for skills needing master level)` | ✅ |  |
| 5 | int64 | bytes `dateExpire: 8-byte FILETIME (per entry)` | ✅ |  |
| 6 | byte | byte `sn / MovementAffectingStat (after loop)` | ✅ |  |

