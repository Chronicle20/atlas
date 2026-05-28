# Attack (← `CUserRemote::OnAttack`)

- **IDA:** 0xa05a50
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/attack.go`
- **Variant:** GMS/v87
- **Branch depth:** 3
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `packed: high nibble=nDamagePerMob, low nibble=hits` | ❌ | width mismatch |
| 1 | byte | byte `level (m_nLevel)` | ✅ |  |
| 2 | byte | byte `nSLV (skill level; 0 means no skill)` | ✅ |  |
| 3 | byte | int32 `skillId (only if nSLV != 0)` | ❌ | width mismatch |
| 4 | int32 | byte `passive SLV byte (only if skillId==3211006)` | ❌ | width mismatch |
| 5 | byte | int32 `passive skill ID (only if skillId==3211006 and passive SLV!=0)` | ❌ | width mismatch |
| 6 | byte | byte `option / bSerialAttack` | ✅ |  |
| 7 | int16 | int16 `packed: bit15=bLeft, low15=nAction (attackAction)` | ✅ |  |
| 8 | byte | byte `nActionSpeed (only if nAction <= 0x110)` | ✅ |  |
| 9 | byte | byte `nMastery (only if nAction <= 0x110)` | ✅ |  |
| 10 | int32 | int32 `nBulletItemID (only if nAction <= 0x110)` | ✅ |  |
| 11 | int32 | int32 `monsterId per damage target (loop nDamagePerMob times)` | ✅ |  |
| 12 | byte | byte `hitAction per target (if monsterId != 0)` | ✅ |  |
| 13 | byte | byte `damage count (only if meso explosion skill)` | ✅ |  |
| 14 | int32 | int32 `damage value per hit` | ✅ |  |
| 15 | int16 | int16 `ptBallStart.x (only if ranged)` | ✅ |  |
| 16 | int16 | int16 `ptBallStart.y (only if ranged)` | ✅ |  |
| 17 | int32 | int32 `tKeyDown (only for keydown skills)` | ✅ |  |

