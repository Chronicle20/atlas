# SummonSpawn (← `CSummonedPool::OnCreated`)

- **IDA:** 0x9f80f8
- **Atlas file:** `libs/atlas-packet/summon/clientbound/spawn.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `cid — read by sub_9F80F8@0x9f811a (also consumed by CSummonedPool::OnPacket dispatch; pool is cid-keyed)` | ✅ |  |
| 1 | int32 | int32 `skillId (nSkillID) — sub_9F80F8@0x9f8124; consumed by GetSkill@CSkillInfo in sub_823AED@0x823b6b. (NO oid on jms185 — the int after cid is the skillId)` | ✅ |  |
| 2 | int32 | byte `charLevel (nCharLevel) — sub_9F80F8@0x9f812e; atlas writes fixed 0x0A (visual-only)` | ❌ | width mismatch |
| 3 | byte | byte `SLV skill level (nSLV) — sub_9F80F8@0x9f813d; atlas 'level'` | ✅ |  |
| 4 | byte | int16 `nX — CSummoned Init blob sub_823AED@0x823b15` | ❌ | width mismatch |
| 5 | int16 | int16 `nY — sub_823AED@0x823b22` | ✅ |  |
| 6 | int16 | byte `nMoveAction (stance) — sub_823AED@0x823b2f` | ❌ | width mismatch |
| 7 | byte | int16 `nCurFoothold — sub_823AED@0x823b39; atlas writes fixed 0 (visual-only)` | ❌ | width mismatch |
| 8 | int16 | byte `nMoveAbility (movementType) — sub_823AED@0x823b46` | ❌ | width mismatch |
| 9 | byte | byte `nAssistType (!puppet attack flag) — sub_823AED@0x823b49` | ✅ |  |
| 10 | byte | byte `nEnterType (!animated flag) — sub_823AED@0x823b8b (read unconditionally on jms185)` | ✅ |  |
| 11 | byte | byte `bAvatarLook present byte — sub_823AED@0x823b99; atlas writes fixed 0 (no AvatarLook blob / Tesla tail for the v83 roster). PRESENT on jms185 (avatar-look field GMS gained at v95); ABSENT on GMS v83/v84/v87.` | ✅ |  |
| 12 | byte | bytes `AvatarLook blob — sub_823AED@0x823bb0, read ONLY when bAvatarLook!=0 (AvatarLook::Decode). Absent for the v83 roster (present byte = 0).` | ✅ |  |

