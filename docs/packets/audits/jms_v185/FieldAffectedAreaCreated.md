# FieldAffectedAreaCreated (← `CAffectedAreaPool::OnAffectedAreaCreated`)

- **IDA:** 0x436572
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/affected_area_created.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwId (mist/affected-area object id @line103)` | ✅ |  |
| 1 | int32 | int32 `nType (mist type; ==3 item-area-buff branch @line104)` | ✅ |  |
| 2 | int16 | int32 `dwOwnerId (owner character id @line105)` | ❌ | width mismatch |
| 3 | int16 | int32 `nSkillID (skill id @line106)` | ❌ | width mismatch |
| 4 | int16 | byte `nSLV (skill level @line107)` | ❌ | width mismatch |
| 5 | int16 | int16 `phase/delay (layer-time multiplier @line108)` | ✅ |  |
| 6 | int16 | bytes `rcArea RECT (16 bytes = lt/rb as 4x int32 @line109)` | ❌ | width mismatch |
| 7 | int16 | int32 `tEnd (end time @line110) — NO leading tStart (v95-only; absent in JMS like v83)` | ❌ | width mismatch |
| 8 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 9 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |


## Triage: ❌ — structural deferral carried forward (JMS185 confirms v83 shape, NO tStart)

JMS185 `CAffectedAreaPool::OnAffectedAreaCreated` @0x436572 reads 8 fields:
`Decode4 dwId` + `Decode4 nType` + `Decode4 dwOwnerId` + `Decode4 nSkillID` +
`Decode1 nSLV` + `Decode2 phase` + `DecodeBuffer(16) rcArea RECT` + `Decode4 tEnd`
(@lines 103-110). This is the **SAME 8-field layout as GMS v83** — JMS185 does **NOT**
have the v95-only leading `tStart` int (v95 = 9 fields). Atlas
`affected_area_created.go` writes a different 10-field shape (mistKey, ownerId, 6×
origin/lt/rb int16, duration int32, skillLevel int32) that matches NEITHER JMS185 nor
GMS v83/v95. The fix is a structural re-encode (add nType/nSkillID, pack the RECT as a
16-byte buffer, drop origin), version-gated on the v95-only tStart. **Still DEFERRED** —
see `_pending.md` AFFECTEDAREA-create-shape. The sibling REMOVE_MIST
(`AffectedAreaRemoved`, single Decode4) matches JMS185 cleanly (✅).

Ack: world-audit Phase 3 JMS185 field+portal on 2026-05-28
