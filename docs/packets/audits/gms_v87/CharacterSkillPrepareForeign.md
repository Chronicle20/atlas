# CharacterSkillPrepareForeign (← `CUserRemote::OnSkillPrepare`)

- **IDA:** 0xa06135
- **Atlas file:** `libs/atlas-packet/character/clientbound/skill_prepare_foreign.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `nSkillID` | ✅ |  |
| 1 | int32 | byte `nSLV (skill level)` | ❌ | width mismatch |
| 2 | byte | int16 `action (bit15 = move-action, low15 = one-time-action)` | ❌ | width mismatch |
| 3 | int16 | byte `nActionSpeed` | ❌ | width mismatch |
| 4 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

