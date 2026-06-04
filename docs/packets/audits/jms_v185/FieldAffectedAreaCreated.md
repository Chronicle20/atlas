# FieldAffectedAreaCreated (← `CAffectedAreaPool::OnAffectedAreaCreated`)

- **IDA:** 0x436572
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/affected_area_created.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwId (mist/affected-area object id @line103)` | ✅ |  |
| 1 | int32 | int32 `nType (mist type; ==3 item-area-buff branch @line104)` | ✅ |  |
| 2 | int32 | int32 `dwOwnerId (owner character id @line105)` | ✅ |  |
| 3 | int32 | int32 `nSkillID (skill id @line106)` | ✅ |  |
| 4 | byte | byte `nSLV (skill level @line107)` | ✅ |  |
| 5 | int16 | int16 `phase/delay (layer-time multiplier @line108)` | ✅ |  |
| 6 | int32 | bytes `rcArea RECT (16 bytes = lt/rb as 4x int32 @line109)` | ✅ |  |
| 7 | int32 | int32 `tEnd (end time @line110) — NO leading tStart (v95-only; absent in JMS like v83)` | ✅ |  |
| 8 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 9 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 10 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 11 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

