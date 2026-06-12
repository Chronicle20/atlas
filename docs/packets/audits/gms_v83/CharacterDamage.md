# CharacterDamage (← `CUserRemote::OnHit`)

- **IDA:** 0x9832e3
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/damage.go`
- **Variant:** GMS/v83
- **Branch depth:** 3
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `nAttackIdx (signed; -1=miss/magic, -2=obstacle, >=0=mob attack index)` | ❌ | width mismatch |
| 1 | byte | int32 `nDamage (primary damage or miss sentinel)` | ❌ | width mismatch |
| 2 | int32 | int32 `ptHit=monsterTemplateId (only if nAttackIdx > -2)` | ✅ |  |
| 3 | int32 | byte `bLeft (only if nAttackIdx > -2)` | ❌ | width mismatch |
| 4 | byte | byte `v20: power guard flag (only if nAttackIdx > -2)` | ✅ |  |
| 5 | byte | byte `bPowerGuard% (only if v20 != 0)` | ✅ |  |
| 6 | byte | int32 `mobId for power guard (only if v20 != 0)` | ❌ | width mismatch |
| 7 | int32 | byte `stance source (only if mob found for power guard)` | ❌ | width mismatch |
| 8 | int32 | int16 `power guard hit x (only if mob found)` | ❌ | width mismatch |
| 9 | byte | int16 `power guard hit y (only if mob found)` | ❌ | atlas: short — missing trailing field |
| 10 | byte | byte `v4: stance action (always, after power guard block, if nAttackIdx > -2)` | ❌ | atlas: short — missing trailing field |
| 11 | byte | int32 `nDamage repeated (always)` | ❌ | atlas: short — missing trailing field |
| 12 | byte | int32 `misdirection skill ID (only if nDamage == -1)` | ❌ | atlas: short — missing trailing field |

