# Plan Audit — task-156-gm-hide-heal-dispel

**Plan Path:** docs/tasks/task-156-gm-hide-heal-dispel/plan.md
**Audit Date:** 2026-07-17
**Branch:** task-156-gm-hide-heal-dispel
**Base Branch:** main (base commit `43894c25bfc368d743a5434c4537c112f8a85b14`, head `d5d2e104d`)
**Module audited:** `services/atlas-channel/atlas.com/channel` (Go)

## Executive Summary

All 8 plan tasks (46 checklist steps) were faithfully implemented; nothing was silently skipped. The two pre-approved deviations — switching Heal from flat+ratio-from-WZ to heal-to-full, and adding an ungated `spawnCharacterForSessionRevealed` for the reveal path — are both cleanly implemented, well-documented in source comments, and consistent with the ledger in `.superpowers/sdd/progress.md`. `go build ./...`, `go vet ./...`, and `go test -race ./...` are all clean in the changed module; `tools/redis-key-guard.sh` and `tools/goroutine-guard.sh` are both clean at the repo root. No changes leaked outside `atlas-channel` (verified `libs/`, `atlas-data`, `atlas-buffs` diffs are empty), and `go.mod`/`go.sum` are byte-identical to base, so the plan's "docker bake mandatory" gate does not actually apply (correctly noted as N/A in the progress ledger, verified by an empty go.mod diff). Two Minor gaps flagged mid-execution (missing regression tests for per-recipient isolation and the effective-max fallback) were closed in the final commit with two new passing tests.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Effect recovery accessors (`MP`/`HpR`/`MpR`) | DONE (superseded by approved deviation) | Added in commit `4b411a4f3`; removed in `d5d2e104d` per user-approved heal-to-full pivot. `data/skill/effect/model.go` diff base→head is empty (net no-op); `HP()` retained (still used by Cleric Heal, `skill/handler/heal/heal.go`). `model_test.go` deleted along with the dead accessors. Approved deviation #1 — not a defect. |
| 2 | `IsGmHidden` buff predicate | DONE | `character/buff/hidden.go` matches plan verbatim (keyed on `SourceId() == SuperGmHideId && !Expired()`); `character/buff/hidden_test.go` matches plan test verbatim; `go test -race ./character/buff/... -run TestIsGmHidden` PASS. |
| 3 | `CancelByTypes` producer + processor | DONE | `kafka/message/buff/kafka.go:16` `CommandTypeCancelByTypes = "CANCEL_BY_TYPES"`, `:46` `CancelByTypesCommandBody`; `character/buff/producer.go:57-71` `CancelByTypesCommandProvider`; `character/buff/processor.go:21,66-69` interface + impl. `producer_test.go` present and passing. |
| 4 | `SelectAllCharactersInMap` selector + MP fields | DONE | `skill/handler/recipients.go:26-27` (`mp`/`maxMp` fields), `:35-36` getters, `:50-51` builder setters, `:203-`  `SelectAllCharactersInMap`. `recipients_map_test.go` present; `go test ./skill/handler/ -run TestSelectAllCharactersInMap` covered by full-package PASS run. |
| 5 | Heal + Dispel handler | DONE (heal magnitude uses approved deviation) | `skill/handler/healdispel/healdispel.go` created; registered via `skill/handler/registrations/registrations.go`. Restore formula is `fullRestoreDelta` (heal-to-full), not the plan's flat+ratio — approved deviation #1, documented in the function's doc comment (healdispel.go:88-93). Disease list (`diseaseTypes`, lines 31-43) matches the 11-type authoritative set in `services/atlas-buffs/atlas.com/buffs/character/immunity.go:7-11` exactly. No `AwardExperience` call anywhere in the file (FR-9). Per-recipient isolation (log+continue) implemented at lines 120-132, and now covered by `TestPerRecipientIsolation`. `TestEffectiveMaxFallsBackToBase` covers the `effectiveMaxOrBase` zero-fallback branch. 5 tests, all PASS. |
| 6 | Spawn suppression gate + reveal/despawn helpers | DONE (extended by approved deviation #2) | Gate inserted into `spawnCharacterForSession` (`kafka/consumer/map/consumer.go:464-466`, `buff.IsGmHidden(bs)` short-circuits to `nil`). `DespawnCharacterInMap` (line 517) and `SpawnCharacterInMap` (line 533) both present. Per deviation #2, `SpawnCharacterInMap` was later repointed from the gated `spawnCharacterForSession` to a new ungated `spawnCharacterForSessionRevealed` (line 497) — see Task 6.5 below. |
| 6.5 | (unplanned, approved) Ungated reveal spawn fix | DONE | commit `cc253d511`. `spawnCharacterForSessionRevealed` (consumer.go:497-512) has zero references to `buff.IsGmHidden` — a structural guarantee documented in its own comment (line 495-496). The two production callers of the *gated* `spawnCharacterForSession` — `consumer.go:176` (SpawnForSelf-of-others) and `:372` (enterMap→others) — are unchanged and still call the gated function. Only `SpawnCharacterInMap` (line 541, the reveal path) calls the ungated variant. Confirms the plan's instruction: "the two normal spawn callers remain gated." |
| 7 | Hide handler | DONE | `skill/handler/hide/hide.go` matches plan verbatim: `HideBuffDuration = int32(math.MaxInt32)`, SuperGM gate, toggle logic (apply+despawn / cancel+spawn), self-only announce (no foreign seam exists in `hideDeps` — matches Global Constraints resolution of the design's §3.4 inconsistency). `hide_test.go` present with `TestNonSuperGmRejected`/`TestHideOn`/`TestHideOff`, all PASS. Registered in `registrations.go`. |
| 8 | Full-module verification | DONE | See Build & Test Results below. All CLAUDE.md gates green; docker-bake step correctly identified as N/A (go.mod/go.sum unchanged — verified independently, see Executive Summary). Execute-time gates (WZ recovery values for 9101000, DARK_SIGHT self-give byte) recorded in `.superpowers/sdd/progress.md` ("Execute-time Gate findings", lines 24-26) — Gate 1 (WZ=0 across all versions) directly motivated the approved heal-to-full pivot; Gate 2 (DARK_SIGHT non-zero self-give) verified at source level against `character_temporary_stat.go:89` and the mask/AddStat path. |

**Completion Rate:** 8/8 plan tasks (100%), plus 1 approved out-of-plan fix (reveal race) and 1 approved plan-deviation (heal-to-full), both explicitly pre-cleared by the user per the task brief.
**Skipped without approval:** 0
**Partial implementations:** 0 (the two Minor findings noted mid-execution in the progress ledger were closed before Task 8 was marked complete — see below)

## Skipped / Deferred Tasks

None. No task was skipped, and the two documented "Minor" findings from the ledger (`.superpowers/sdd/progress.md` lines 15-18) were resolved, not deferred:

- **Per-recipient isolation untested** → closed by `TestPerRecipientIsolation` in `healdispel_test.go` (verified passing).
- **`effectiveMaxOrBase` zero-fallback untested** → closed by `TestEffectiveMaxFallsBackToBase` in `healdispel_test.go` (verified passing).

The one CRITICAL finding from the whole-branch review (async-cancel vs. sync-gated-spawn race defeating Hide reveal) was fixed in commit `cc253d511` and independently re-verified in this audit — see Task 6.5 above.

## Build & Test Results

| Service | Build | Vet | Tests (`-race`) | Notes |
|---------|-------|-----|------------------|-------|
| atlas-channel | PASS | PASS (no output) | PASS (all packages `ok`, no `FAIL`) | Ran from `services/atlas-channel/atlas.com/channel`: `go build ./...`, `go vet ./...`, `go test -race ./...` — all clean, independently re-run during this audit (not just trusting the ledger). New/changed packages individually re-verified with `-v -count=1`: `character/buff` (`TestIsGmHidden`, `TestCancelByTypesCommandProvider` PASS), `skill/handler` (`TestSelectAllCharactersInMap` and siblings PASS), `skill/handler/healdispel` (5/5 PASS), `skill/handler/hide` (3/3 PASS), `kafka/consumer/map` (5/5 PASS incl. mysticdoor-adjacent tests unaffected). |

Additional mandatory gates (repo root):
- `tools/redis-key-guard.sh` — exit 0, no raw keyed go-redis usage flagged.
- `tools/goroutine-guard.sh` — exit 0, no bare `go` statements flagged.
- `docker buildx bake atlas-channel` — **not run**; independently confirmed N/A because `go.mod`/`go.sum` for `atlas-channel` are byte-identical between base and head (`git diff <base>..<head> -- .../go.mod .../go.sum` is empty). The CLAUDE.md gate is triggered by a touched `go.mod`, which did not occur on this branch — all new code lives in existing packages/dependencies already vendored. This matches the ledger's Task 8 note.
- `libs/`, `services/atlas-data`, `services/atlas-buffs` diffs confirmed empty (Global Constraint "only atlas-channel's go.mod is touched" — trivially true since go.mod itself is untouched, and no files under those trees changed).

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None. All plan tasks are implemented, tested, and verified; both approved deviations are cleanly documented in source and match the user's explicit direction; all mandatory build/test/guard gates pass; the working tree is clean with no stray files.
