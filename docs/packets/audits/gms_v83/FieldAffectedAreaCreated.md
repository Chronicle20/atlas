# FieldAffectedAreaCreated (← `CAffectedAreaPool::OnAffectedAreaCreated`)

- **IDA:** 0x431a63
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/affected_area_created.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwId (mist/affected-area object id, v92)` | ✅ |  |
| 1 | int32 | int32 `nType (mist type; ==3 is item-area-buff branch, v95.m_Data)` | ✅ |  |
| 2 | int16 | int32 `dwOwnerId (owner character id, v4)` | ❌ | width mismatch |
| 3 | int16 | int32 `nSkillID (skill id, Unknown; ==130/131/2111003/... GetSkill)` | ❌ | width mismatch |
| 4 | int16 | byte `nSLV (skill level, v99)` | ❌ | width mismatch |
| 5 | int16 | int16 `phase/delay (v88; layer-time multiplier)` | ✅ |  |
| 6 | int16 | bytes `rcArea RECT (16 bytes = ltX,ltY,rbX,rbY as 4x int32, v86)` | ❌ | width mismatch |
| 7 | int16 | int32 `tEnd/nPhase (end time, v90)` | ❌ | width mismatch |
| 8 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 9 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |


## Triage: ❌ — atlas matches NEITHER v83 NOR v95 (NEW FINDING; DEFERRED)

The atlas `AffectedAreaCreated` struct (`affected_area_created.go`) is documented
in-code as the **v83** SPAWN_MIST wire format. **The v83 IDA disproves that
claim.** `CAffectedAreaPool::OnAffectedAreaCreated` v83 @0x431a63 decodes:

- `Decode4 dwId` (line 108)
- `Decode4 nType` (line 109; `== 3` is the item-area-buff branch)
- `Decode4 dwOwnerId` (line 110)
- `Decode4 nSkillID` (line 111; switched on 130/131/2111003/... → GetSkill)
- `Decode1 nSLV` (line 112)
- `Decode2 phase/delay` (line 113)
- `DecodeBuffer(16) rcArea` RECT = ltX,ltY,rbX,rbY as 4× int32 (line 115)
- `Decode4 tEnd` (line 116)

This is the **same shape as GMS v95**, differing only by v95's extra leading
`tStart` int32 (v95 reads two int32 after the RECT: tStart + tEnd; v83 reads
one). Atlas, by contrast, writes:

`int32 mistKey, int32 ownerId, int16 originX, int16 originY, int16 ltX, int16
ltY, int16 rbX, int16 rbY, int32 duration, int32 skillLevel`

— which matches **neither version**. Atlas omits `nType` and `nSkillID`
(4 bytes each), invents `originX/originY` int16 fields that no client reads,
and packs the LT/RB rectangle as four inline int16s instead of the client's
16-byte RECT buffer (4× int32). Only positions 0 (id), 1 (type vs ownerId —
coincidental width), and the phase/delay int16 partially line up.

**Conclusion:** the in-code "v83 SPAWN_MIST" comment is incorrect; atlas's mist
layout is a bespoke shape that no audited GMS client decodes correctly. A
correct fix is a **structural rewrite** (add `nType`+`nSkillID`, drop
`originX/originY`, emit a 16-byte RECT buffer, widen `nSLV` handling) and the
struct must carry the new fields end-to-end from atlas-maps. Because v83 and v95
share the same 8/9-field shape (differing only by v95's `tStart`), a single
version-gated rewrite could satisfy BOTH — but it requires new model data and
exceeds this bucket's scope (>2 nested guards, new struct fields, cross-service
plumbing).

**DEFERRED to `_pending.md`** with v83-confirmed evidence. This is now a stronger
finding than the v95 deferral: rather than "atlas serves the v83 shape and v95
diverges," the reality is "atlas serves a shape no client reads." Sibling-task
candidate: rewrite `affected_area_created.go` to the client SPAWN_MIST layout
(type+skillId+slv+phase+RECT+tStart?+tEnd), version-gating the `tStart` int32 on
`GMS>=95`.

Ack: world-audit Phase 3 v83 on 2026-05-28
