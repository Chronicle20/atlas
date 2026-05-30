# FieldAffectedAreaCreated (← `CAffectedAreaPool::OnAffectedAreaCreated`)

- **IDA:** 0x432f3f
- **Atlas file:** `libs/atlas-packet/field/clientbound/affected_area_created.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwId (mist/affected-area object id, v89 @0x432f6e)` | ✅ |  |
| 1 | int32 | int32 `nType (mist type; ==3 is item-area-buff branch, v91[0] @0x432f78)` | ✅ |  |
| 2 | int16 | int32 `dwOwnerId (owner character id, v4 @0x432f82)` | ❌ | width mismatch |
| 3 | int16 | int32 `nSkillID (skill id, a2 @0x432f8b)` | ❌ | width mismatch |
| 4 | int16 | byte `nSLV (skill level, Value @0x432f98)` | ❌ | width mismatch |
| 5 | int16 | int16 `phase/delay (v87; layer-time multiplier, @0x432fa3)` | ✅ |  |
| 6 | int16 | bytes `rcArea RECT (16 bytes = ltX,ltY,rbX,rbY as 4x int32, @0x432fae)` | ❌ | width mismatch |
| 7 | int16 | int32 `tEnd/nPhase (end time, Skill @0x432fba) — v87 has NO leading tStart (matches v83; v95 adds tStart)` | ❌ | width mismatch |
| 8 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 9 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |


## Triage: ❌ — structural protocol divergence (DEFERRED, v87 confirms v83)

The atlas `AffectedAreaCreated` struct (`affected_area_created.go`) matches
NEITHER the v87 nor the v95 SPAWN_MIST client layout. The GMS **v87** client
(`CAffectedAreaPool::OnAffectedAreaCreated`@0x432f3f) decodes:

- **v87 reads (8 fields):** `Decode4 dwId`, `Decode4 nType`, `Decode4 dwOwnerId`,
  `Decode4 nSkillID`, `Decode1 nSLV`, `Decode2 phase`, `DecodeBuffer(16) rcArea
  RECT`, `Decode4 tEnd` (single trailing int32, NO leading tStart).
- **atlas writes (v83-era, 10 fields):** `int32 mistKey`, `int32 ownerId`,
  `int16 originX`, `int16 originY`, `int16 ltX/ltY/rbX/rbY`, `int32 duration`,
  `int32 skillLevel`.

**Cross-version finding (task-068 Phase 3 v87):** v87's field set is identical to
**v83** (@0x431a63) — both read 8 fields with a single trailing `Decode4` after
the 16-byte RECT and NO leading `tStart`. v95 (@0x437ec0) is the version that
ADDS a leading `tStart` int32 (giving it 9 fields with two trailing Decode4s).
So the version axis is: v83 == v87 (8 fields, tEnd only) → v95 (9 fields, adds
tStart). Atlas matches none of them (it carries the legacy origin int16 layout).

This is not a single-field width bug — it is the entire packet being a different
protocol shape. A correct fix is a structural re-encode (add type/skillId, drop
origin, emit the RECT as a 16-byte buffer, gate the `tStart` int32 behind
`GMS>=95`) requiring new model fields. **Still DEFERRED to `_pending.md`**
(AFFECTEDAREA-create-shape) — NOT rewritten in this pass per task scope.

The sibling REMOVE_MIST packet (`AffectedAreaRemoved`, single int32 id) matches
v87 cleanly — only the *create* layout diverged.

Ack: world-audit Phase 3 v87 cross-version on 2026-05-28
