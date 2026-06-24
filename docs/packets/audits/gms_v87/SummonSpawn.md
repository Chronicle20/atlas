# SummonSpawn (← `CSummonedPool::OnCreated`)

- **IDA:** 0x9b3749
- **Atlas file:** `libs/atlas-packet/summon/clientbound/spawn.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** ⚠️

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `cid (character id) — read upstream by CUserPool::OnUserCommonPacket@0x9f7392 (consumed by the dispatch; pool is cid-keyed). Atlas writes it as ownerId.` | ✅ |  |
| 1 | int32 | int32 `oid (summon object id) — OnCreated@0x9B3749 Decode4@0x9b376b; passed to the CSummoned ctor sub_7F489E as arg1 (stored at obj+172, the object id). (The active path's first Decode4 after the upstream cid IS the oid.)` | ✅ |  |
| 2 | int32 | int32 `skillId (nSkillID) — OnCreated@0x9B3749 Decode4@0x9b3775; passed to the CSummoned ctor as arg2 (stored obj+180); consumed by GetSkill@CSkillInfo in sub_7F504A@0x7f50b7.` | ✅ |  |
| 3 | byte | byte `charLevel (nCharLevel) — sub_9B3749@0x9b377f; atlas writes fixed 0x0A (visual-only)` | ✅ |  |
| 4 | byte | byte `SLV skill level (nSLV) — sub_9B3749@0x9b378e; atlas 'level'` | ✅ |  |
| 5 | int16 | int16 `nX — CSummoned Init blob sub_7F504A@0x7f5061` | ✅ |  |
| 6 | int16 | int16 `nY — sub_7F504A@0x7f506e` | ✅ |  |
| 7 | byte | byte `nMoveAction (stance) — sub_7F504A@0x7f507b` | ✅ |  |
| 8 | int16 | int16 `nCurFoothold — sub_7F504A@0x7f507e; atlas writes fixed 0 (visual-only)` | ✅ |  |
| 9 | byte | byte `nMoveAbility (movementType) — sub_7F504A@0x7f5092` | ✅ |  |
| 10 | byte | byte `nAssistType (!puppet attack flag) — sub_7F504A@0x7f50a9` | ✅ |  |
| 11 | byte | byte `nEnterType (!animated flag; read only if GetSkill(skillId)!=0) — sub_7F504A@0x7f50d0` | ✅ |  |
| 12 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

