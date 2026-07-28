# CharacterSkillChange (← `CWvsContext::OnChangeSkillRecordResult`)

- **IDA:** 0x9174dd
- **Atlas file:** `libs/atlas-packet/character/clientbound/skill_change.go`
- **Variant:** GMS/v72
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | int16 | int16 `` | ✅ |  |
| 2 | int32 | int32 `` | ✅ |  |
| 3 | int32 | int32 `` | ✅ |  |
| 4 | int32 | int32 `` | ✅ |  |
| 5 | int64 | byte `` | ❌ | width mismatch |
| 6 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

