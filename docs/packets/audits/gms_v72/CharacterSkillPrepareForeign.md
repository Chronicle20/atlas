# CharacterSkillPrepareForeign (← `CUserRemote::OnSkillPrepare`)

- **IDA:** 0x889e3d
- **Atlas file:** `libs/atlas-packet/character/clientbound/skill_prepare_foreign.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | int32 | byte `` | ❌ | width mismatch |
| 2 | byte | byte `` | ✅ |  |
| 3 | int16 | byte `` | ❌ | width mismatch |
| 4 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

