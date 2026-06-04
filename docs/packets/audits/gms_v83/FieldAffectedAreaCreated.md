# FieldAffectedAreaCreated (← `CAffectedAreaPool::OnAffectedAreaCreated`)

- **IDA:** 0x431a63
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/affected_area_created.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwId (mist/affected-area object id, v92)` | ✅ |  |
| 1 | int32 | int32 `nType (mist type; ==3 is item-area-buff branch, v95.m_Data)` | ✅ |  |
| 2 | int32 | int32 `dwOwnerId (owner character id, v4)` | ✅ |  |
| 3 | int32 | int32 `nSkillID (skill id, Unknown; ==130/131/2111003/... GetSkill)` | ✅ |  |
| 4 | byte | byte `nSLV (skill level, v99)` | ✅ |  |
| 5 | int16 | int16 `phase/delay (v88; layer-time multiplier)` | ✅ |  |
| 6 | int32 | bytes `rcArea RECT (16 bytes = ltX,ltY,rbX,rbY as 4x int32, v86)` | ✅ |  |
| 7 | int32 | int32 `tEnd/nPhase (end time, v90)` | ✅ |  |
| 8 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 9 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 10 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 11 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

