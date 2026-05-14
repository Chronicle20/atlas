# Attack (← `CUserRemote::OnAttack`)

- **IDA:** 0x95a670
- **Atlas file:** `libs/atlas-packet/character/clientbound/attack.go`
- **Variant:** GMS/v95
- **Branch depth:** 3
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `packed: high nibble=nDamagePerMob, low nibble=hits` | ❌ | width mismatch |
| 1 | byte | byte `level (m_nLevel)` | ✅ |  |
| 2 | byte | byte `nSLV (skill level; 0 means no skill)` | ✅ |  |
| 3 | byte | int32 `skillId (only if nSLV != 0)` | ❌ | width mismatch |
| 4 | int32 | byte `passive SLV byte (only if skillId==3211006 and SLV!=0)` | ❌ | width mismatch |
| 5 | byte | int32 `passive skill ID (only if skillId==3211006 and passive SLV!=0)` | ❌ | width mismatch |
| 6 | byte | byte `option / bSerialAttack (& 0x20)` | ✅ |  |
| 7 | byte | int16 `packed: bit15=bLeft, low15=nAction (attackAction)` | ❌ | width mismatch |
| 8 | int16 | byte `nActionSpeed (only if nAction <= 0x110)` | ❌ | width mismatch |
| 9 | byte | byte `nMastery (only if nAction <= 0x110)` | ✅ |  |
| 10 | byte | int32 `nBulletItemID (only if nAction <= 0x110)` | ❌ | width mismatch |
| 11 | int32 | int32 `monsterId per damage target (loop nDamagePerMob times)` | ✅ |  |
| 12 | int32 | byte `hitAction per target (if monsterId != 0)` | ❌ | width mismatch |
| 13 | byte | byte `damage count (only if meso explosion skill 4211006, per target)` | ✅ |  |
| 14 | byte | int32 `damage value per hit (loop nHits times, or damage-count for meso explosion)` | ❌ | width mismatch |
| 15 | int32 | int16 `ptBallStart.x (only if nType==212 ranged)` | ❌ | width mismatch |
| 16 | int16 | int16 `ptBallStart.y (only if nType==212 ranged)` | ✅ |  |
| 17 | int16 | int32 `tKeyDown (only for keydown skills: 2121001/2221001/2321001/22121000/22151001)` | ❌ | width mismatch |
| 18 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

---

ack: dispatcher-layer offset — CUserPool::OnUserRemotePacket@0x94b390 reads characterId (Decode4) before calling CUserRemote::OnAttack@0x95a670; all tool ❌s are due to atlas including characterId at position 0 while IDA function starts one field later. After accounting for the +1 offset the remaining fields match IDA. Atlas Attack.Encode round-trips correctly across all variants including v95 strafe passive-SLV byte; no wire bug detected.

