# CharacterDamage (← `CUserRemote::OnHit`)

- **IDA:** 0x954c50
- **Atlas file:** `libs/atlas-packet/character/clientbound/damage.go`
- **Variant:** GMS/v95
- **Branch depth:** 3
- **Verdict:** ❌

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
| 7 | byte | byte `stance source (only if mob found for power guard)` | ✅ |  |
| 8 | int32 | int16 `power guard hit x (only if mob found)` | ❌ | width mismatch |
| 9 | int32 | int16 `power guard hit y (only if mob found)` | ❌ | width mismatch |
| 10 | byte | byte `v4: stance action (always, after power guard block, if nAttackIdx > -2)` | ❌ | atlas: short — missing trailing field |
| 11 | byte | byte `v33: stance flags, bit0=bStance, bit1=bStanceSkillOverride (if nAttackIdx > -2)` | ❌ | atlas: short — missing trailing field |
| 12 | byte | int32 `nDamage repeated (always)` | ❌ | atlas: short — missing trailing field |
| 13 | byte | int32 `misdirection skill ID (only if nDamage == -1)` | ❌ | atlas: short — missing trailing field |

---

ack: Two tool-limitation causes — (1) dispatcher-layer +1 offset: CUserPool::OnUserRemotePacket consumes characterId before dispatching to OnHit, while atlas writes characterId at offset 0; (2) the analyzer linearizes conditionally-emitted fields (mob hit type, hit coordinates, power-guard, knockback) against IDA's guarded entries, producing width mismatches the diff cannot reconcile. No structural wire bug — atlas only encodes the hit-type subset wired through the service layer.
