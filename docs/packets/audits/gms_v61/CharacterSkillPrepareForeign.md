# CharacterSkillPrepareForeign (← `CUserRemote::OnSkillPrepare`)

- **IDA:** 0x7c9963
- **Atlas file:** `libs/atlas-packet/character/clientbound/skill_prepare_foreign.go`
- **Variant:** GMS/v61
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | int32 | byte `` | ❌ | width mismatch |
| 2 | byte | byte `` | ✅ |  |
| 3 | byte | byte `` | ✅ |  |
| 4 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

