# FieldAffectedAreaCreated (← `CAffectedAreaPool::OnAffectedAreaCreated`)

- **IDA:** 0x437ec0
- **Atlas file:** `libs/atlas-packet/field/clientbound/affected_area_created.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwId (mist/affected-area object id)` | ✅ |  |
| 1 | int32 | int32 `nType (mist type; ==3 is item-area-buff branch)` | ✅ |  |
| 2 | int16 | int32 `dwOwnerId (owner character id)` | ❌ | width mismatch |
| 3 | int16 | int32 `nSkillID (skill id)` | ❌ | width mismatch |
| 4 | int16 | byte `nSLV (skill level)` | ❌ | width mismatch |
| 5 | int16 | int16 `phase/delay (v6; layer-time multiplier)` | ✅ |  |
| 6 | int16 | bytes `rcArea RECT (16 bytes = ltX,ltY,rbX,rbY as 4x int32)` | ❌ | width mismatch |
| 7 | int16 | int32 `tStart (start time)` | ❌ | width mismatch |
| 8 | int32 | int32 `tEnd/nPhase (end time)` | ✅ |  |
| 9 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

## Triage: ❌ — v83-vs-v95 protocol divergence (DEFERRED)

The atlas `AffectedAreaCreated` struct (`affected_area_created.go`) is explicitly
documented as the **v83** SPAWN_MIST wire format. The GMS **v95** client
(`CAffectedAreaPool::OnAffectedAreaCreated`@0x437ec0) decodes a structurally
different packet:

- **v95 reads (9 fields):** `Decode4 dwId`, `Decode4 nType`, `Decode4 dwOwnerId`,
  `Decode4 nSkillID`, `Decode1 nSLV`, `Decode2 phase`, `DecodeBuffer(16) rcArea
  RECT`, `Decode4 tStart`, `Decode4 tEnd`.
- **atlas writes (v83, 10 fields):** `int32 mistKey`, `int32 ownerId`,
  `int16 originX`, `int16 originY`, `int16 ltX/ltY/rbX/rbY`, `int32 duration`,
  `int32 skillLevel`.

Only positions 0/1/5/8 line up; the rest diverge in width and meaning. v95 adds
`nType` + `nSkillID` (4-byte) fields, drops `originX/originY`, and packs the
LT/RB rectangle as a 16-byte RECT buffer rather than four inline int16s. This is
not a single-field width bug — it is the entire packet being a different protocol
version.

Fixing it for v95 would require a full re-encode (add type/skillId, drop origin,
emit the RECT as a buffer) which would simultaneously **break the v83 client**
this struct is written for, so a correct fix must be version-guarded and
cross-version-verified against the v83/v87/v92 IDBs. That is out of scope for
this clientbound field bucket. **Deferred to `_pending.md`** (see
"Out of scope for GMS v95 audit (cross-region or cross-version)").

The sibling REMOVE_MIST packet (`AffectedAreaRemoved`, single int32 id) matches
v95 cleanly — only the *create* layout diverged between versions.

Ack: world-audit Phase 2c on 2026-05-28

## Correction (task-068 Phase 3 v83): atlas is NOT the v83 shape

The Phase 2c note above assumed atlas encodes the v83 SPAWN_MIST layout and that
a v95 fix "would break the v83 client." **The v83 audit disproves this.**
`CAffectedAreaPool::OnAffectedAreaCreated` v83 @0x431a63 reads `Decode4 dwId,
Decode4 nType, Decode4 dwOwnerId, Decode4 nSkillID, Decode1 nSLV, Decode2 phase,
DecodeBuffer(16) rcArea, Decode4 tEnd` — the SAME field set as v95 (v95 only adds
a leading `tStart` int32). Atlas matches NEITHER version. So a rewrite to the
client SPAWN_MIST shape would FIX v83 too, not break it — v83 and v95 share the
layout, and a single `tStart`-gated (`GMS>=95`) encode satisfies both. Still
deferred (structural rewrite, new model fields) — see `_pending.md`
AFFECTEDAREA-create-shape with v83-confirmed evidence.

Ack: world-audit Phase 3 v95-refresh on 2026-05-28
