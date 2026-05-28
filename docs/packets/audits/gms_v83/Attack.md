# Attack (← `CUserRemote::OnAttack`)

- **IDA:** 0x9803ab
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/attack.go`
- **Variant:** GMS/v83
- **Branch depth:** 3
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `packed: high nibble=nDamagePerMob, low nibble=hits` | ❌ | width mismatch |
| 1 | byte | byte `level (m_nLevel)` | ✅ |  |
| 2 | byte | byte `nSLV (skill level; 0 means no skill)` | ✅ |  |
| 3 | byte | int32 `skillId (only if nSLV != 0)` | ❌ | width mismatch |
| 4 | int32 | byte `option / bSerialAttack (& 0x20)` | ❌ | width mismatch |
| 5 | byte | int16 `packed: bit15=bLeft, low15=nAction (attackAction)` | ❌ | width mismatch |
| 6 | byte | byte `nActionSpeed (unconditional in v83 — no nAction <= 0x110 guard)` | ✅ |  |
| 7 | int16 | byte `nMastery (unconditional in v83)` | ❌ | width mismatch |
| 8 | byte | int32 `nBulletItemID (unconditional in v83)` | ❌ | width mismatch |
| 9 | byte | int32 `monsterId per damage target (loop nDamagePerMob times)` | ❌ | width mismatch |
| 10 | int32 | byte `hitAction per target (if monsterId != 0)` | ❌ | width mismatch |
| 11 | int32 | byte `damage count (only if meso explosion skill 4211006, per target)` | ❌ | width mismatch |
| 12 | byte | int32 `damage value per hit (loop nHits times, or damage-count for meso explosion)` | ❌ | width mismatch |
| 13 | byte | int16 `ptBallStart.x (only if nType==212 ranged / opcode 0xBB)` | ❌ | width mismatch |
| 14 | int32 | int16 `ptBallStart.y (only if nType==212 ranged / opcode 0xBB)` | ❌ | width mismatch |
| 15 | int16 | int32 `tKeyDown (only for keydown skills: 2121001/2221001/2321001)` | ❌ | width mismatch |
| 16 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 17 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

