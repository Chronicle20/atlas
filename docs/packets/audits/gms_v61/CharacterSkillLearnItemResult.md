# CharacterSkillLearnItemResult (← `CWvsContext::OnSkillLearnItemResult`)

- **IDA:** 0x841e5f
- **Atlas file:** `libs/atlas-packet/character/clientbound/skill_learn_item_result.go`
- **Variant:** GMS/v61
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | int32 `characterId` | ❌ | width mismatch |
| 1 | int32 | byte `isMasteryBook` | ❌ | width mismatch |
| 2 | byte | int32 `skillId (decoded, discarded)` | ❌ | width mismatch |
| 3 | int32 | int32 `masterLevel (decoded, discarded)` | ✅ |  |
| 4 | int32 | byte `canUse` | ❌ | width mismatch |
| 5 | byte | byte `success` | ✅ |  |
| 6 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

