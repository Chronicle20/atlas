# Attack (← `CUserRemote::OnAttack`)

- **IDA:** 0xa05a50
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/attack.go`
- **Variant:** GMS/v87
- **Branch depth:** 3
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `packed: high nibble=nDamagePerMob, low nibble=hits` | ❌ | width mismatch |
| 1 | byte | byte `level (m_nLevel)` | ✅ |  |
| 2 | byte | byte `nSLV (skill level; 0 means no skill)` | ✅ |  |
| 3 | byte | int32 `skillId (only if nSLV != 0)` | ❌ | width mismatch |
| 4 | int32 | byte `passive SLV byte (only if skillId==3211006)` | ❌ | width mismatch |
| 5 | int32 | int32 `passive skill ID (only if skillId==3211006 and passive SLV!=0)` | ✅ |  |
| 6 | byte | byte `option / bSerialAttack` | ✅ |  |
| 7 | byte | int16 `packed: bit15=bLeft, low15=nAction (attackAction)` | ❌ | width mismatch |
| 8 | int32 | byte `nActionSpeed (only if nAction <= 0x110)` | ❌ | width mismatch |
| 9 | int32 | byte `nMastery (only if nAction <= 0x110)` | ❌ | width mismatch |
| 10 | byte | int32 `nBulletItemID (only if nAction <= 0x110)` | ❌ | width mismatch |
| 11 | byte | int32 `monsterId per damage target (loop nDamagePerMob times)` | ❌ | width mismatch |
| 12 | int32 | byte `hitAction per target (if monsterId != 0)` | ❌ | width mismatch |
| 13 | int16 | byte `damage count (only if meso explosion skill)` | ❌ | width mismatch |
| 14 | int16 | int32 `damage value per hit` | ❌ | width mismatch |
| 15 | int32 | int16 `ptBallStart.x (only if ranged)` | ❌ | width mismatch |
| 16 | byte | int16 `ptBallStart.y (only if ranged)` | ❌ | atlas: short — missing trailing field |
| 17 | byte | int32 `tKeyDown (only for keydown skills)` | ❌ | atlas: short — missing trailing field |

