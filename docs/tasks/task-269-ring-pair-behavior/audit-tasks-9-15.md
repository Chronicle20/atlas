# Plan Audit — task-269-ring-pair-behavior (Tasks 9-15)

**Plan Path:** docs/tasks/task-269-ring-pair-behavior/plan.md
**Audit Date:** 2026-08-27
**Branch:** task-269-ring-pair-behavior
**Base Branch:** main (branch itself is rebased onto task-240-cash-shop-stub-operations per plan's Global Constraints; merge-base with current HEAD used for scope is `32d55cb21`)
**Scope:** Tasks 9 through 15 only (a companion shard covers Tasks 1-8)

## Executive Summary

All seven tasks in this range (9-15) are fully implemented and verified against the final tree state at HEAD `e5f7cf0`. Task 9's REST consumer, Task 10's cache/processor, Task 11's four encoder sites, and Task 12's population/invalidation wiring (including the FR-4 session-destroy fix landed after the original Task 12 commit) all have passing unit tests that exercise every case the plan's brief tables specify. Tasks 13 and 14's packet-audit evidence is re-pinned/added with `verifies:` back-references to real test functions, and `packet-audit matrix --check` exits 0 with no degraded cells. Task 15's coverage manifest and FR-12 residual documentation are present and match the corrected (not the PRD-original) description of the residual limitation. `go build ./...` and `go test ./... -count=1` are green for both `atlas-channel` (152 packages, no failures) and `libs/atlas-packet` (no failures). No skipped, stubbed, or silently deferred work was found in this range.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 9 | `atlas-channel/ring` — REST consumer and model | DONE | `services/atlas-channel/atlas.com/channel/ring/{rest,model,requests}.go` all present; `rest_test.go` implements all four brief cases (full couple half, invalid uuid, unknown ringType, large cashId) — `TestExtract`, `TestExtractInvalidId`, `TestExtractUnknownRingType`, `TestExtractLargeCashIdSurvives` all pass. `requestByCharacterId` at `ring/requests.go:23` is a bare-URL builder consumed via `requests.DrainProvider`, matching the paginated-list requirement. |
| 10 | `atlas-channel/ring` — cache and `GetRingSet` | DONE | `ring/cache.go` mirrors `monster/information/cache.go`'s `perTenant map[uuid.UUID]map[uint32]cacheEntry` shape exactly (`cache.go:22-24`). `EvictTenant` (`cache.go:80-85`) is registered centrally from `main.go:307` inside the shared tenant-evictor closure rather than from a package `init()` — this deviates from the brief's suggested wiring point but matches the codebase's actual convention (every other tenant-scoped cache in `atlas-channel` — account registry, monster status/live mirrors, auto-aggro gate, monster inbox — registers the same way from `main.go`, not from its own `init()`), so it is `DONE` with a note, not a deviation of substance. `processor.go`'s `selectPair` implements the "highest slot / lowest cashId tie-break" rule exactly as specified; `TestRingCacheTenantIsolation` (4 subtests) and `TestGetRingSet` (10 subtests, covering every row of the brief's table including FR-3, FR-5, FR-14, FR-15) all pass. |
| 11 | Feed the four encoder sites from the ring processor | DONE | `character_spawn.go:60` (`ring.NewProcessor(l, ctx).GetRingSet(...)`), `character_info.go:54` (`rings.Marriage != nil`), `character_data.go:60` (`cd.Rings = ring.NewProcessor(l, ctx).GetRingRecords(...)`), and `kafka/consumer/asset/consumer.go:419` (`ringProcessorFn(l, ctx).GetRingSet(...)`, resolved once outside the per-session `updateAppearance` closure — confirmed by reading the closure at `consumer.go:411-425`) are all wired. `TestCharacterSpawnBodyCarriesRings`, `TestCharacterInfoBodySetsMarriageFlag`, and `TestUpdateAppearanceResolvesRingsOnce` (the PRD §8 hot-path guard) all exist and pass. |
| 12 | Cache population and `RING_PURCHASED` invalidation | DONE | `handleStatusEventRingPurchased` (`kafka/consumer/cashshop/consumer.go:508`) invalidates both the buyer (`e.CharacterId`) and the resolved partner (via `character.NewProcessor(...).GetByName(e.Body.PartnerName)`) rather than gating partner invalidation on a live session, which is a defensible and slightly broader implementation than the brief's "partner has a live session on this channel" framing — the cache is per-character, not per-session, so invalidating regardless of live presence is correct and is still exercised by `TestRingPurchasedInvalidatesCache`'s four subtests (buyer invalidated, partner invalidated when present, partner absent is not an error, wrong tenant untouched — all pass). Population is wired into the character-load bootstrap at `kafka/consumer/session/consumer.go:221` (`ring.NewProcessor(l, ctx).Populate(c.Id())`), the single point every login and channel-enter runs through, confirmed by `TestRingCachePopulatedOnCharacterLoad`. A later fix-round commit (`b6f255f71`, FR-4) added `clearRingsOnDestroy` at `session/processor.go` wired into `Destroy`, closing the other half of FR-4 (map/channel/logout invalidation) that the original Task 12 commit alone did not cover; `TestClearRingsOnDestroy_NonZeroCharacter_ClearsState` and `_ZeroCharacter_NoOp` both pass. |
| 13 | Packet coverage — `CharacterSpawn` and `CharacterInfo` | DONE | All ten `packet-audit:verify` markers exist for both ops (v48/v61/v72/v79/v83/v84/v87/v92/v95/jms_v185), split across `spawn_test.go`/`info_test.go` plus `v61_test.go`/`v72_test.go`/`v79_test.go` as Task 14's Ruling 35 note anticipates. `STATUS.md:86` (CHAR_INFO) and `:197` (SPAWN_PLAYER) both read ✅ across all ten columns including the newly-claimed gms_v92. `docs/packets/evidence/gms_v92/character.clientbound.{CharacterSpawn,CharacterInfo}.yaml` both exist with `verifies:` back-references (`TestCharacterSpawnV92Golden`/`TestCharacterSpawnRingBlocks`, `TestCharacterInfoV92Golden`), and those tests pass. `go run ./tools/packet-audit matrix --check` exits 0. |
| 14 | Packet coverage — `CharacterAppearanceUpdate` and `CharacterData` | DONE | All eight evidence records (`gms_v61`/`v72`/`v79`/`v83`/`v84`/`v87`/`v95`/`jms_v185`) for `character.clientbound.CharacterAppearanceUpdate` were re-pinned with fresh `decompile_sha256` values and `verifies:` fields pointing at real, passing byte-fixture tests (`TestCharacterAppearanceUpdateByteOutputV61` through `...JMS`). `STATUS.md:180` and `:264` correctly show `⬜` (n-a) at gms_v48 for both `SET_FIELD` and `UPDATE_CHAR_LOOK`, matching the plan's "nine columns, not ten" correction to design.md §7. `set_field_test.go` adds `TestSetFieldRoundTripPopulatedRings` beside the existing empty-Rings case, exercising the opaque `CharacterData` span in both states while preserving the OPAQUE_LEDGER discipline (comment at `set_field_test.go:306-315` cites the §5 caveat rather than inventing per-field citations). `matrix --check` exits 0 with no cell regression; v92 remains ❌ for both ops, which is a pre-existing, branch-wide global export defect (v92 fails broadly across unrelated ops like `GUEST_ID_LOGIN`/`LOGIN_STATUS`), not something introduced or left behind by this task, and is correctly declared out of scope in the coverage manifest. |
| 15 | Coverage manifest, service docs, and the full gate | DONE | `docs/tasks/task-269-ring-pair-behavior/coverage-manifest.yaml` exists with `ops:`/`versions:`/`fields:`/`out_of_scope:`/`residual:` sections matching the `task-252` schema precedent, including the disclosed schema deviation (per-op version lists rather than one flat list) and an accurate accounting of the two pre-existing IDA-export defects (Ruling 4, Ruling 36) discovered incidentally. `services/atlas-channel/docs/kafka.md:21` documents the corrected FR-12 residual limitation (a still-present partner's ring update resolves on their own next EQUIP, not on RING_PURCHASED itself), matching the plan's instruction to record the corrected residual rather than the PRD's original "requires a map change" assumption. This audit could not itself run the flagless `tools/verify.sh` (explicitly forbidden by the dispatch brief because a concurrent process is running it against this same tree); build/test verification below is the module-local substitute. |

**Completion Rate:** 7/7 tasks in range (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None in this range. Two pre-existing, out-of-scope defects were discovered and honestly declared rather than silently fixed or silently ignored:
- Ruling 4 (coverage-manifest.yaml): four checked-in IDA exports (`gms_v83`/`gms_v87`/`gms_v95`/`gms_jms_185.json`) misname the CharacterAppearanceUpdate trailing field as a character id rather than the correct item template id; fixing requires an IDB comment change and export regeneration, correctly declared out of this branch's scope (plan's Task 1 Step 6 forbids hand-editing a checked-in export).
- Ruling 36 (coverage-manifest.yaml): a pre-existing (pre-dating this branch, from PR #971) citation defect in the gms_v48 CharacterSpawn marker/evidence, naming the wrong IDA function; the v48 ✅ disposition itself is not affected, only the citation.

Both are documented in `coverage-manifest.yaml`'s `out_of_scope:` section with file:line references, not silently absorbed into this task's claimed coverage.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-channel (`services/atlas-channel/atlas.com/channel`) | PASS | PASS | `go build ./...` exit 0; `go test ./... -count=1` — 152 `ok` package results, zero `FAIL` lines, exit 0. |
| libs/atlas-packet | PASS | PASS | `go build ./...` exit 0; `go test ./... -count=1` exit 0, zero `FAIL` lines. |
| packet-audit matrix | N/A | PASS | `go run ./tools/packet-audit matrix --check` exits 0 with only informational `note` lines for pre-existing n-a evidence entries unrelated to this range. |

Flagless `tools/verify.sh` was not run by this audit (a concurrent process was already running it against the same working tree per the dispatch brief's explicit read-only constraint); the module-local build/test/matrix results above stand in as the audit's own verification for the files in scope.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (pending the companion Tasks 1-8 shard's own verdict, and pending the currently-running `tools/verify.sh` result)

## Action Items

None required for Tasks 9-15. The working tree was left byte-identical to how it was found; `git status --short` shows only untracked artifacts from concurrent/other-shard processes (`agent-ledger.tsv`, `audit-tasks-1-8.md`, `completeness-critic.md`, `frontend-audit.md`, `reviews/`), none of which this audit created or modified.
