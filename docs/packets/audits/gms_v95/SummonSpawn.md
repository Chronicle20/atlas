# SummonSpawn (← `CSummonedPool::OnCreated`)

- **IDA:** 0x75a9a0
- **Atlas file:** `../../libs/atlas-packet/summon/clientbound/spawn.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `charId — read by CSummonedPool::OnPacket@0x75ac70 before dispatch` | ✅ |  |
| 1 | int32 | int32 `oid (dwSummonedID) — CSummonedPool::OnCreated@0x75a9e6` | ✅ |  |
| 2 | int32 | int32 `skillId (nSkillID) — OnCreated@0x75a9ef` | ✅ |  |
| 3 | byte | byte `charLevel (nCharLevel) — OnCreated@0x75a9fa; atlas writes fixed 0x0A (visual-only)` | ✅ |  |
| 4 | byte | byte `SLV skill level (nSLV) — OnCreated@0x75aa08; atlas 'level'` | ✅ |  |
| 5 | int16 | int16 `nX — CSummoned::Init@0x7557a3` | ✅ |  |
| 6 | int16 | int16 `nY — CSummoned::Init@0x7557b3` | ✅ |  |
| 7 | byte | byte `nMoveAction (stance) — CSummoned::Init@0x7557c1` | ✅ |  |
| 8 | int16 | int16 `nCurFoothold — CSummoned::Init@0x7557d7; atlas writes fixed 0 (visual-only)` | ✅ |  |
| 9 | byte | byte `nMoveAbility (movementType) — CSummoned::Init@0x7557ec` | ✅ |  |
| 10 | byte | byte `nAssistType (!puppet attack flag) — CSummoned::Init@0x7557fa` | ✅ |  |
| 11 | byte | byte `nEnterType (!animated flag) — CSummoned::Init@0x755806` | ✅ |  |
| 12 | byte | byte `bAvatarLook present — CSummoned::Init@0x755816; if !=0 AvatarLook::Decode follows; atlas v95+GMS writes 0. (NEW in v95)` | ✅ |  |

