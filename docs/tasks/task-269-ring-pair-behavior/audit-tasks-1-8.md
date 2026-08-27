# Plan Audit — task-269-ring-pair-behavior (Tasks 1-8)

**Plan Path:** docs/tasks/task-269-ring-pair-behavior/plan.md
**Audit Date:** 2026-08-27
**Branch:** task-269-ring-pair-behavior
**Base Branch:** main (task branch itself was rebased on
`origin/task-240-cash-shop-stub-operations`, per plan's Global Constraints;
audited against the final tree, HEAD `e5f7cf0`)

## Executive Summary

Tasks 1-8 are all fully implemented and match the plan's final (corrected)
text, with one legitimate deviation: Task 6's `CharacterAppearanceUpdate`
frame correction was refined in a post-review fix round (commit `00dc42a52`)
to gate the trailing `nCompletedSetItemID` write on `IsRegion("GMS") &&
MajorAtLeast(87)` rather than deleting it unconditionally as the plan's Task
6 prose still literally reads — this is a correctness improvement discovered
during review, not a skip, and the derivation/test evidence for it is present
in the diff. `libs/atlas-packet` (all packages), `atlas-cashshop`, and the
`atlas-channel` packages touched by Task 7 all build and test clean. 8/8
tasks DONE.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Pin the trailing 4-byte ring field from IDA | DONE | `docs/tasks/task-269-ring-pair-behavior/ring-field-derivation.md` (created, 34.9K, decompile-derived tables for couple/friendship/marriage blocks with IDA addresses and session ids); verdict consumed verbatim by Task 2's `PairRing.ItemId` / `MarriageRing.MarriageCharacterId` field names and doc comments in `libs/atlas-packet/model/ring.go:9-20,29-38`. Commit `c134a7fc9`. |
| 2 | Shared ring codec — field blocks (sites B, C) | DONE | `libs/atlas-packet/model/ring.go:42-127` (`PairRing`, `MarriageRing`, `RingSet`, `EncodeField`/`DecodeField`); `libs/atlas-packet/model/ring_test.go:66-260` (`TestRingSetEncodeField`, `TestRingSetFieldRoundTrip`) reproduce every fixture byte string from the plan's table exactly (verified GMS/JMS/empty/v48/v95 cases at `ring_test.go:82-120`). Commit `3a2cd247a`. `go test ./model/...` passes. |
| 3 | Shared ring codec — record blocks (site A) | DONE | `libs/atlas-packet/model/ring.go:129-330` (`CoupleRecord`, `FriendRecord`, `MarriageRecord`, `RingRecords`, `EncodeRecords`/`DecodeRecords`); derivation appended to `ring-field-derivation.md` (commit `961fb763d`); codec commit `720bee275`; `TestRingRecordsEncode`/`TestRingRecordsRoundTrip` at `ring_test.go:261-339+`. One documented, justified deviation from the plan's literal Step-1 table: `writeRecordName` reserves 1 byte for a NUL terminator (`recordNameWidth-1` = 12 usable bytes, not the full 13) per the task-3a IDA derivation recorded in the comment at `ring.go:129-142` — this is a derivation refinement the plan's own Step 1 explicitly authorized ("Confirm that split from IDA before writing the encoder... update them to the derived offsets if Step 1 lands elsewhere"), not a silent deviation. |
| 4 | Wire site A — `CharacterData` | DONE | `libs/atlas-packet/character/data.go:121` (`Rings model.RingRecords` field), `:770-776` (`encodeRings`/`decodeRings` delegate to `m.Rings.EncodeRecords`/`DecodeRecords`); `TestCharacterDataRingsRoundTrip` at `data_test.go:486`. Commit `a5f8ac146`. |
| 5 | Wire sites B and D — `CharacterSpawn`, `CharacterInfo` | DONE | `spawn.go:46,52,58,177,221,294` (`rings model.RingSet` field/param, `EncodeField`/`DecodeField` call sites, `Rings()` getter); `info.go:61,66,71,133,213,256` (`hasMarriageRing bool`, `WriteBool(m.hasMarriageRing)`, `HasMarriageRing()` getter). Tests present: `TestCharacterSpawnRingBlocks` (`spawn_test.go:168`), `TestCharacterInfoMarriageFlag` (`info_test.go:310`). Commit `940fd0ae4`. |
| 6 | Wire site C — `CharacterAppearanceUpdate`, frame correction | DONE (with a documented plan/code deviation resolved by a later fix round) | `appearance_update.go` — struct/constructor take `rings model.RingSet` (lines 18-26); `Encode`/`Decode` call `m.rings.EncodeField`/`DecodeField` (lines 43,64); trailing `w.WriteInt(0)` is retained but now version-gated by `hasTrailingCompletedSetItemId(t)` = `IsRegion("GMS") && MajorAtLeast(87)` (lines 44-51, 65-67, 74-83), added by commit `00dc42a52` after Task 15's review found the plan's unconditional-delete instruction wrong for gms_v87/v95 (IDA addresses cited: `gms_v87.json @0xa090f4`/Decode4 `@0xa092d5`; `gms_v95.json @0x954110`/Decode4 `@0x9542ec`). **This contradicts the plan's literal Task 6 Step 3 text** ("delete `w.WriteInt(0)`... and nothing else in the frame"), which fix-round commit `ac3c8bde5` did not amend (it amended six other rulings but left Task 6's text stale). The code is the corrected, IDA-grounded behavior; the plan prose is what's stale here — flagging per the audit brief's instruction to report plan/code disagreements rather than assume the plan wins. |
| 7 | Restore `atlas-channel` compilation at the four call sites | DONE | Commit `60f87d269` — `character_spawn.go:63` (`packetmodel.RingSet{}`), `character_info.go:64` (trailing `false`), `consumer.go:420` (`packetmodel.RingSet{}` in `NewCharacterAppearanceUpdate`) — exact zero-value plumbing the plan specifies. Note: by the current tree tip these call sites have since been overwritten by Task 11 (commit `9f65d8ac0`, out of this shard's range) to pass real `RingSet` values instead of zero values; Task 7's own commit is intact history and its stated goal (green build) was met at the time. |
| 8 | `atlas-cashshop` read model — `cashId`, `partnerCashId`, `partnerName` | DONE | `ring/model.go:39-43` (three new unexported fields + getters at 84-101); `ring/builder.go:89-105` (`SetCashId`/`SetPartnerCashId`/`SetPartnerName`); `ring/rest.go:15-29,49-64` (`RestModel` JSON fields + `Transform`); `ring/processor.go:32-46,64-79,112-146` (`NewProcessor` takes `character.Processor`, `GetByCharacterId`/`GetByCharacterIdPaged`/`GetById` all route through `enrich`, which fails soft on each of the three resolutions per the plan's fail-soft requirement). Tests: `TestTransform` (`rest_test.go:10`, extended with the three fields), `TestGetByCharacterIdEnrichesCashIdAndPartnerName` (`processor_test.go:88`). Commit `02b6eeb81`. |

**Completion Rate:** 8/8 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. All eight tasks in this shard's range have code, tests, and commit
evidence in the final tree.

## Build & Test Results

| Service / Module | Build | Tests | Notes |
|---|---|---|---|
| `libs/atlas-packet` | PASS | PASS | `go build ./...` and `go test ./... -count=1` both clean; `model`, `character`, `character/clientbound` packages (Tasks 2-6 scope) all `ok`. |
| `services/atlas-cashshop/atlas.com/cashshop` | PASS | PASS | `go build ./...` clean; `go test ./ring/... -count=1` → `ok`; `go test ./... -count=1` produced no `FAIL` lines. |
| `services/atlas-channel/atlas.com/channel` | PASS | PASS (scoped) | `go build ./...` clean. Ran the Task-7-scoped packages only (`./socket/...`, `./kafka/consumer/asset/...`) rather than the full suite, since the full suite also covers Tasks 9-15 (out of this shard's range) — all scoped packages `ok`. |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (for this task range; overall branch readiness also depends on the Tasks 9-15 shard's audit)

## Action Items

1. Non-blocking documentation gap: `docs/tasks/task-269-ring-pair-behavior/plan.md`'s Task 6 Step 3 text still describes an unconditional `WriteInt(0)` deletion; it should be updated to describe the `hasTrailingCompletedSetItemId` version gate added by commit `00dc42a52`, so a future reader of the plan doesn't reintroduce the gms_v87/v95 regression the fix round corrected. The code and tests are already correct — this is a plan-text staleness note only, not a code defect.
