# Plan Audit — task-290-cosmic-map-action-parity (Plan C3, tasks C16–C23 + BC2, C22b, C23b)

**Plan Path:** docs/tasks/task-290-cosmic-map-action-parity/plan-c3.md
**Audit Date:** 2026-09-03
**Branch:** task-290-cosmic-map-action-parity
**Base Branch:** main (merge base 9613e7259)
**Shard range:** C16–C23, plus out-of-plan tasks BC2, C22b, C23b (commits c26bdf00a..9c2b9be13, restricted to BC2 onward — 65b60f7f9..9c2b9be13)

## Executive Summary

All 8 plan-c3 tasks in this shard (C16–C23) plus the three controller-created out-of-plan
tasks (BC2, C22b, C23b) landed, were independently code-reviewed by `task-reviewer`, and are
recorded complete in `.superpowers/sdd/plan-c3/progress.md`. Two tasks (C20, C21) had a
review cycle surface a genuine BLOCKING defect (an unreachable `areaInfo` condition, and a
wrong `questStatus` threshold value) — both were fixed in a follow-up commit within the same
task and re-verified; neither survives at HEAD. This audit independently re-derived the two
fixes, the QUEST_URL routing fix, the infoEx exposure, and the schema/catalog-lint state
directly against the working tree rather than trusting the ledger's prose. All Go modules
touched by this shard (`atlas-drops`, `atlas-reactors`, `atlas-maps`, `libs/atlas-saga`,
`atlas-saga-orchestrator`, `atlas-map-actions`, `atlas-character`, `atlas-query-aggregator`,
`libs/atlas-script-core`, `atlas-channel`, `atlas-quest`, `atlas-data`, `atlas-monsters`)
build and test clean, and the flagless `tools/verify.sh` passed at HEAD (9c2b9be13,
`/tmp/task290/gate-final.log`, "All checks passed."). One task (C22) is intentionally partial
relative to Cosmic — the explorer-quest completion packet (`getShowQuestCompletion`/
`earnTitleMessage`) is not sent because no client codec exists for it — disclosed by the
implementer, reviewed, and accepted rather than silently dropped; C22b closed the larger gap
(the progress write) per an explicit user ruling extending scope. No task in this shard was
skipped or landed with an unresolved blocking finding.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| C16 | `clear_drops` field-scoped drop clear in atlas-drops (G5) | DONE | `drop.ClearForField` (drop/processor.go:318-343), DELETE `.../drops` route (map/resource.go), `libs/atlas-saga` ClearDrops action/payload, orchestrator + map-actions executor wiring, schema block. Commits c26bdf00a..bb682ec67. Review APPROVED_WITH_FINDINGS, 0 blocking (`task-C16-review.md`). Reserved/mid-pickup drops correctly cleared (no owner filter), matching Cosmic. |
| C17 | `reset_reactors` / `shuffle_reactors` in atlas-reactors (G5) | DONE | `ResetInField(f, minState *int8)` + `ShuffleInField` (reactor/processor.go:358,388-416), PRD's discredited "state-filtered overload" premise deliberately not followed (single optional-filter reset is correct). Commits 19ffc9a4b..60992787c. Review APPROVED_WITH_FINDINGS, 0 blocking (`task-C17-review.md`). Non-blocking, unfixed: ShuffleInField aborts on first mid-loop Update failure (partial non-permutation) — carried to final review, not a live bug against an in-memory registry. |
| C18 | `reset_field` (Cosmic `resetPQ`) (G5) | DONE | New saga action + orchestrator `fieldclient` processor + POST `.../reset` route composing clear/reset/shuffle/monster-restore at the map-script level (not internally, per controller correction). Commit a8fb2207d. Review APPROVED_WITH_FINDINGS, 0 blocking (`task-C18-review.md`). |
| C19 | The four G5 seeds | DONE | 44 seed files across 11 roots, byte-identical (verified by reviewer md5sum). Commit 692e3f3a3. A genuine cross-saga ordering race (reset_field followed by spawn_monster in the same rule) was found and fixed by combining the pair into one two-step saga (`executeResetFieldThenSpawnMonster`). Review APPROVED_WITH_FINDINGS, 0 blocking (`task-C19-review.md`). Non-blocking, carried forward: the combine is pair-specific, not a general same-rule ordering guarantee (map-922000000.json's three-op rule still emits three independent sagas) — disclosed, not silently dropped. |
| C20 | Area-info persistence + `areaInfo` condition (G12) | DONE (after fix round) | `libs/atlas-saga` AreaInfoCondition, `atlas-character` area_info package (tenant-scoped GORM upsert-replace), orchestrator persist-before-announce, query-aggregator `EvaluateWithContext` case. Commits 692e3f3a3..3b02d51b4 (d2ca80bb0, 9ce335e21, 3b02d51b4). **First review CHANGES_REQUIRED**: `AreaInfoCondition` was absent from `requiresContextPath` (validation/processor.go), so the condition was unreachable via the only production entry point (`ValidateStructured`) — verified independently in this audit: `grep -n "AreaInfoCondition" validation/processor.go` shows the case present at line 179 in the current tree. Fix round 1 (3b02d51b4) closed it with a new regression test entering via `ValidateStructured`. Also found and fixed in the same commit range: `libs/atlas-script-core` condition builder never copied `valueString` into the returned Model (silent no-op). |
| C21 | The two area-info seeds (`rienArrow`, `rien`) (G12) | DONE (after fix round) | 22 seed files. Commits 3b02d51b4..bdb833f72 (4b9669bed, bdb833f72). **First review CHANGES_REQUIRED (blocking)**: map-rien.json encoded `questStatus` 21101 as value "3" (retracted "+1" PRD premise); correct value is "2" (Completed). Fix round 1 corrected all 11 copies. Independently re-verified in this audit: `grep -n questStatus` across all 11 `deploy/seed/*/map-actions/onUserEnter/map-rien.json` copies shows `"value": "2"` in every one, no "3" remaining. Review of the fix round confirmed 11 files collapse to one sha256. |
| C22 | `explorer_quest` (G14) | DONE (base scope) | atlas-quest medal-map dedup, orchestrator `RequestExplorerQuest`/`handleExplorerQuest` routing (via `GetHandler`, not a direct call — closes the C20 defect class), `QUEST_URL` → `QUESTS` constant fix. Commits bdb833f72..519c8335e. Independently re-verified: `quest/medal_map_requests.go:16` reads `const medalMapBaseUrl = "QUESTS"` in the current tree. Review APPROVED, 0 blocking (`task-C22-review.md`). Disclosed, accepted gap: the quest-progress write (Cosmic step 4) was deferred because atlas-data did not yet expose a per-status infoNumber — this became C22b per an explicit user ruling (#5), not a silently dropped requirement. |
| C22b | (out-of-plan) expose infoEx / write explorer-quest progress | DONE | atlas-data `reader.go`/`rest.go` now serve `InfoEx`; `RequestExplorerQuest` extended to return Count/NewlyRecorded/InfoNumber/Threshold; `handleExplorerQuest` writes the count via `RequestUpdateProgress`. Commit 7830166f9. Independently re-verified: `quest/rest.go:84` has `InfoEx []string`, `reader.go:159-164` parses the infoEx node. Review APPROVED_WITH_FINDINGS, 0 blocking (`task-C22b-review.md`). Disclosed, accepted gap: the completion packet (`getShowQuestCompletion`/`earnTitleMessage`) is still not sent — no client codec exists, out of scope per brief, not silently invented. |
| C23 | `explorationPoint` (G14) | DONE | Ten-rule, first-match-wins seed document (map_id=104000000 carries both explorer_quest + field_effect, correctly ordered ahead of beginner_range). 11 byte-identical seed files (single sha256, confirmed by reviewer). Commit fc3dbd313. Review APPROVED, 0 blocking (`task-C23-review.md`) — boundary arithmetic (105040300 vs 110000000) and byte-exact area-name text independently hand-checked by the reviewer. |
| C23b | (out-of-plan) correct stale explorer-quest infoEx doc claims | DONE | Corrected 3 surviving copies of the C22b-retracted "infoEx unavailable" claim: `libs/atlas-saga/payloads.go` doc comment (via C23), the hand-maintained `definitions.operation.allOf` region of `map_script_schema.json`, and `medal_map/processor.go`'s doc comment. Commit 9c2b9be13. Review APPROVED, 0 blocking (`task-C23b-review.md`) — confirmed the schema edit sits in the verified hand-written region (`tools/gen-map-action-schema/main.go:271-283` `existingAllOf`), independently re-run `--check` exits 0 in this audit. |
| BC2 | (out-of-plan) resync scripted NPCs to a character entering the field | DONE | Single-module atlas-channel fix closing a real gap found reviewing task BC (2nd+ field entrant saw no NPC). Commit 19ffc9a4b. Review APPROVED, 0 blocking (`task-BC2-review.md`) — byte-equality with the broadcast path pinned by a dedicated test. |

**Completion Rate:** 11/11 shard tasks (100%) — the plan's 8 (C16–C23) plus the 3 controller-created follow-ons (BC2, C22b, C23b), all tracked to completion in the same ledger.
**Skipped without approval:** 0
**Partial implementations:** 0 at HEAD (C20 and C21 each had a review-cycle PARTIAL/blocking finding that was fixed within the same task before being marked complete — see below)

## Skipped / Deferred Tasks

None skipped. Two tasks passed through a genuine blocking-finding-then-fix cycle before landing; both are resolved at HEAD and independently re-verified above:

- **C20** — `AreaInfoCondition` was unreachable in production (absent from `requiresContextPath`); fixed in commit 3b02d51b4, confirmed present in the current tree.
- **C21** — `map-rien.json`'s `questStatus` value was wrong ("3" instead of "2"), which would have made the `rien` tutorial's checkpoint guard permanently unsatisfiable; fixed in commit bdb833f72, confirmed corrected across all 11 seed roots.

Two intentional, disclosed scope narrowings exist and are not "skipped" work — they were surfaced to the user and accepted rather than silently dropped:

- **C22/C22b** — the explorer-quest completion packet (`getShowQuestCompletion`/`earnTitleMessage`) is not sent to the client; no codec for it exists in this codebase, and inventing one was out of scope for this shard. The progress write itself (Cosmic step 4) IS implemented (C22b).
- **C19** — the `reset_field → spawn_monster` saga-combining fix is pair-specific, not a general same-rule ordering guarantee. No other pair in the current seed corpus needs it (swept by the reviewer), so this is a documented residual risk, not a missing implementation.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-drops | PASS | PASS | `go build ./...`, `go test ./... -count=1` clean |
| atlas-reactors | PASS | PASS | clean |
| atlas-maps | PASS | PASS | clean |
| libs/atlas-saga | PASS | PASS | clean |
| atlas-saga-orchestrator | PASS | PASS | clean |
| atlas-map-actions | PASS | PASS | clean |
| atlas-character | PASS | PASS | full suite green incl. `area_info` package (10.5s `character` pkg, 11.6s `kafka/consumer/character` — DB-backed, both green) |
| atlas-query-aggregator | PASS | PASS | clean, incl. `validation` package (AreaInfoCondition routing) |
| libs/atlas-script-core | PASS | PASS | clean |
| atlas-channel | PASS | PASS | clean |
| atlas-quest | PASS | PASS | clean, incl. `medal_map` |
| atlas-data | PASS | PASS | clean, incl. `quest` (infoEx) |
| atlas-monsters | PASS | PASS | clean |
| Whole-branch flagless `tools/verify.sh` | PASS | PASS | Confirmed at HEAD 9c2b9be13 via `/tmp/task290/gate-final.log`: ends with the GREEN "All checks passed.", including "catalog lint", "map-action schema drift", "map-action schema gen tests", and "lint & format guard (93 module(s))" all ✓. Re-ran `./tools/gen-map-action-schema.sh --check` directly in this audit — exits 0, "up to date". |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (for this shard's scope)

## Action Items

None required for this shard. The following items are pre-existing or cross-cutting and were
explicitly carried to the final whole-branch review by the controller — they are not gaps in
C16–C23/BC2/C22b/C23b's own implementation and are noted here only so this audit does not
silently drop them:

1. Task BC's `ScriptedNpcSpawn` rx0/rx1/facing/cy substitution (outside this shard's task
   range but touches the same NPC-resync mechanism BC2 extends) still needs explicit product
   sign-off per the ledger — raised four times, unanswered as of this shard's completion.
2. `reactor.ShuffleInField` (C17) aborts on the first mid-loop `Update` failure, which can
   leave positions as a non-permutation of the original set. Untested, unspecified by the
   brief, harmless while the registry is in-memory.
3. `reset_field → spawn_monster` saga combining (C19) is pair-specific; a future three-op
   ordering-sensitive rule would still race.
4. Pre-existing, not introduced by this shard: orchestrator `BaseUrl` constants
   (`CHARACTER_URL`/`GACHAPONS_URL`/`RPS_URL`/`TRANSPORTS_URL`) likely share the same
   silent-fallback shape as the `QUEST_URL` bug C22 fixed.
5. Pre-existing, not introduced by this shard: `query-aggregator` `TransportInTransitCondition`
   is in neither `Evaluate` nor `requiresContextPath` — same unreachability class as C20's bug.
