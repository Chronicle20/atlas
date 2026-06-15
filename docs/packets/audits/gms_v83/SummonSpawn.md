# SummonSpawn (← `CSummonedPool::OnCreated`)

- **IDA:** 0x938f61
- **Atlas file:** `libs/atlas-packet/summon/clientbound/spawn.go`
- **Variant:** GMS/v83
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `cid — read by sub_938F61@0x938f7c (consumed by CSummonedPool::OnPacket dispatch; pool is cid-keyed)` | ✅ |  |
| 1 | int32 | int32 `skillId (nSkillID) — sub_938F61@0x938f86; consumed by GetSkill@CSkillInfo in sub_7A379B@0x7a37fd. (NO oid on v83 — the int after cid is the skillId)` | ✅ |  |
| 2 | int32 | byte `charLevel (nCharLevel) — sub_938F61@0x938f90; atlas writes fixed 0x0A (visual-only)` | ❌ | width mismatch |
| 3 | byte | byte `SLV skill level (nSLV) — sub_938F61@0x938f9a; atlas 'level'` | ✅ |  |
| 4 | byte | int16 `nX — CSummoned Init blob sub_7A379B@0x7a37ab` | ❌ | width mismatch |
| 5 | int16 | int16 `nY — sub_7A379B@0x7a37b5` | ✅ |  |
| 6 | int16 | byte `nMoveAction (stance) — sub_7A379B@0x7a37c2` | ❌ | width mismatch |
| 7 | byte | int16 `nCurFoothold — sub_7A379B@0x7a37cf; atlas writes fixed 0 (visual-only)` | ❌ | width mismatch |
| 8 | int16 | byte `nMoveAbility (movementType) — sub_7A379B@0x7a37d9` | ❌ | width mismatch |
| 9 | byte | byte `nAssistType (!puppet attack flag) — sub_7A379B@0x7a37e6` | ✅ |  |
| 10 | byte | byte `nEnterType (!animated flag; read only if GetFoothold(foothold)!=0) — sub_7A379B@0x7a3821` | ✅ |  |
| 11 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 12 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

