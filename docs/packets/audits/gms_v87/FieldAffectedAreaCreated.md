# FieldAffectedAreaCreated (← `CAffectedAreaPool::OnAffectedAreaCreated`)

- **IDA:** 0x432f3f
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/affected_area_created.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwId (mist/affected-area object id, v89 @0x432f6e)` | ✅ |  |
| 1 | int32 | int32 `nType (mist type; ==3 is item-area-buff branch, v91[0] @0x432f78)` | ✅ |  |
| 2 | int32 | int32 `dwOwnerId (owner character id, v4 @0x432f82)` | ✅ |  |
| 3 | int32 | int32 `nSkillID (skill id, a2 @0x432f8b)` | ✅ |  |
| 4 | byte | byte `nSLV (skill level, Value @0x432f98)` | ✅ |  |
| 5 | int16 | int16 `phase/delay (v87; layer-time multiplier, @0x432fa3)` | ✅ |  |
| 6 | int32 | bytes `rcArea RECT (16 bytes = ltX,ltY,rbX,rbY as 4x int32, @0x432fae)` | ✅ |  |
| 7 | int32 | int32 `tEnd/nPhase (end time, Skill @0x432fba) — v87 has NO leading tStart (matches v83; v95 adds tStart)` | ✅ |  |
| 8 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 9 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 10 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 11 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

