# SummonSpawn (← `CSummonedPool::OnCreated`)

- **IDA:** 0x95adec
- **Atlas file:** `libs/atlas-packet/summon/clientbound/spawn.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ⚠️

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `cid (character id) — read upstream by CUserPool::OnUserCommonPacket@0x97240c (consumed by the dispatch; pool is cid-keyed). Atlas writes it as ownerId.` | ✅ |  |
| 1 | int32 | int32 `oid (summon object id) — OnCreated@0x95ADEC Decode4@0x95ae0e; passed to the CSummoned ctor (sub_7A30A9) as the object id. (The active path's first Decode4 after the upstream cid IS the oid.)` | ✅ |  |
| 2 | int32 | int32 `skillId (nSkillID) — OnCreated@0x95ADEC Decode4@0x95ae17; consumed by GetSkill@CSkillInfo.` | ✅ |  |
| 3 | byte | byte `charLevel (nCharLevel) — OnCreated@0x95ADEC Decode1@0x95ae21; atlas writes fixed 0x0A (visual-only).` | ✅ |  |
| 4 | byte | byte `SLV skill level (nSLV) — OnCreated@0x95ADEC Decode1@0x95ae30; atlas 'level'.` | ✅ |  |
| 5 | int16 | int16 `nX — CSummoned::Init blob @0x7a379b Decode2@0x7a37b2.` | ✅ |  |
| 6 | int16 | int16 `nY — CSummoned::Init @0x7a379b Decode2@0x7a37bf.` | ✅ |  |
| 7 | byte | byte `nMoveAction (stance) — CSummoned::Init @0x7a379b Decode1@0x7a37cc.` | ✅ |  |
| 8 | int16 | int16 `nCurFoothold — CSummoned::Init @0x7a379b Decode2@0x7a37cf; atlas writes fixed 0 (visual-only).` | ✅ |  |
| 9 | byte | byte `nMoveAbility (movementType) — CSummoned::Init @0x7a379b Decode1@0x7a37e3.` | ✅ |  |
| 10 | byte | byte `nAssistType (!puppet attack flag) — CSummoned::Init @0x7a379b Decode1@0x7a37fa.` | ✅ |  |
| 11 | byte | byte `nEnterType (!animated flag; read after the Skill guard) — CSummoned::Init @0x7a379b Decode1@0x7a3821.` | ✅ |  |
| 12 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

