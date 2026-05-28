# CharacterDamage (← `CUserRemote::OnHit`)

- **IDA:** 0xa08d57
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/damage.go`
- **Variant:** GMS/v87
- **Branch depth:** 3
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `nAttackIdx (signed; -1=miss/magic, -2=obstacle, >=0=mob attack index)` | ❌ | width mismatch |
| 1 | byte | int32 `nDamage (primary damage or miss sentinel)` | ❌ | width mismatch |
| 2 | int32 | int32 `ptHit=monsterTemplateId (only if nAttackIdx > -2)` | ✅ |  |
| 3 | int32 | byte `bLeft (only if nAttackIdx > -2)` | ❌ | width mismatch |
| 4 | byte | byte `power guard flag (only if nAttackIdx > -2)` | ✅ |  |
| 5 | byte | byte `bPowerGuard% (only if power guard flag != 0)` | ✅ |  |
| 6 | byte | int32 `mobId for power guard (only if power guard flag != 0)` | ❌ | width mismatch |
| 7 | int32 | byte `stance source (only if mob found)` | ❌ | width mismatch |
| 8 | int32 | int16 `power guard hit x` | ❌ | width mismatch |
| 9 | byte | int16 `power guard hit y` | ❌ | atlas: short — missing trailing field |
| 10 | byte | byte `stance action (always, if nAttackIdx > -2)` | ❌ | atlas: short — missing trailing field |
| 11 | byte | byte `stance flags` | ❌ | atlas: short — missing trailing field |
| 12 | byte | int32 `nDamage repeated (always)` | ❌ | atlas: short — missing trailing field |
| 13 | byte | int32 `misdirection skill ID (only if nDamage == -1)` | ❌ | atlas: short — missing trailing field |

