# FieldAffectedAreaCreated (← `CAffectedAreaPool::OnAffectedAreaCreated`)

- **IDA:** 0x437ec0
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/affected_area_created.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwId (mist/affected-area object id)` | ✅ |  |
| 1 | int32 | int32 `nType (mist type; ==3 is item-area-buff branch)` | ✅ |  |
| 2 | int32 | int32 `dwOwnerId (owner character id)` | ✅ |  |
| 3 | int32 | int32 `nSkillID (skill id)` | ✅ |  |
| 4 | byte | byte `nSLV (skill level)` | ✅ |  |
| 5 | int16 | int16 `phase/delay (v6; layer-time multiplier)` | ✅ |  |
| 6 | int32 | bytes `rcArea RECT (16 bytes = ltX,ltY,rbX,rbY as 4x int32)` | ✅ |  |
| 7 | int32 | int32 `tStart (start time)` | ✅ |  |
| 8 | int32 | int32 `tEnd/nPhase (end time)` | ✅ |  |
| 9 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 10 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 11 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

