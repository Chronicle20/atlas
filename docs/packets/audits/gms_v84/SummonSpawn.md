# SummonSpawn (← `CSummonedPool::OnCreated`)

- **IDA:** 0x97038b
- **Atlas file:** `libs/atlas-packet/summon/clientbound/spawn.go`
- **Variant:** GMS/v84
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `cid — read by sub_97038B@0x9703ad (consumed by spawn reader; pool is cid-keyed)` | ✅ |  |
| 1 | int32 | int32 `skillId (nSkillID) — sub_97038B@0x9703b7; passed to CSummoned ctor sub_7C7CC7@0x970440. (NO oid on v84 — the int after cid is the skillId)` | ✅ |  |
| 2 | int32 | byte `charLevel (nCharLevel) — sub_97038B@0x9703c1; atlas writes fixed 0x0A (visual-only)` | ❌ | width mismatch |
| 3 | byte | byte `SLV skill level (nSLV) — sub_97038B@0x9703d0; atlas 'level'` | ✅ |  |
| 4 | byte | int16 `nX — CSummoned Init blob sub_7C83D7@0x7c83ee` | ❌ | width mismatch |
| 5 | int16 | int16 `nY — sub_7C83D7@0x7c83fb` | ✅ |  |
| 6 | int16 | byte `nMoveAction (stance) — sub_7C83D7@0x7c8408` | ❌ | width mismatch |
| 7 | byte | int16 `nCurFoothold — sub_7C83D7@0x7c8412; atlas writes fixed 0 (visual-only)` | ❌ | width mismatch |
| 8 | int16 | byte `nMoveAbility (movementType) — sub_7C83D7@0x7c841f` | ❌ | width mismatch |
| 9 | byte | byte `nAssistType (!puppet attack flag) — sub_7C83D7@0x7c8436` | ✅ |  |
| 10 | byte | byte `nEnterType (!animated flag; read only if GetFoothold(foothold)!=0) — sub_7C83D7@0x7c845d` | ✅ |  |
| 11 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 12 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

