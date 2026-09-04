# Plan Audit — task-295-mob-escort-full-path-fix

**Plan Path:** docs/tasks/task-295-mob-escort-full-path-fix/plan.md
**Audit Date:** 2026-09-04
**Branch:** task-295-mob-escort-full-path-fix
**Base Branch:** main (range audited: 25ba8d1cb..7ca937148)

## Executive Summary

All four planned tasks (plus one controller-approved out-of-plan fixup, Task 5/commit a02152042) were executed faithfully and match the plan's prescribed content byte-for-byte in the load-bearing artifacts: the codec rewrite, the golden fixture, the channel writer signature, the seed-template row, and the evidence/matrix promotion. Both flagged deviations were verified against the described explanations and are accepted as non-blocking. `libs/atlas-packet` and `services/atlas-channel/atlas.com/channel` build, vet, and test clean, and all six packet-audit CI gates (Task 4 Step 5) exit 0 with no unexpected drift. The only gap is that `tools/verify.sh` was not run by this audit (per the controller's ruling, it is being run by a separately-dispatched task-verifier at branch end); this audit did not duplicate that run.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Correct the codec and its byte fixture | DONE | `libs/atlas-packet/monster/clientbound/mob_escort_full_path.go` matches the plan's struct/constructor/accessor/Encode/Decode text verbatim, including `oldDestX`/`oldDestY`, deleted `mode`, waypoint `attr`/`stopDuration`. Doc comment reads "10×Decode4" (post a02152042 fix). Test file `mob_escort_full_path_test.go` matches the plan's three-marker block, golden input (attr=1/attr=2 waypoints), and golden byte table exactly. Commit b1a0bd331. |
| 2 | Update the atlas-channel writer to the corrected signature | DONE | `services/atlas-channel/atlas.com/channel/socket/writer/mob_escort_full_path.go:12-20` matches the plan's doc comment and signature verbatim. `grep -rn "MobEscortFullPathBody" --include="*.go" .` → single hit, the definition itself (no other callers), confirming Step 1's expected result. Commit 6073f7ee3. |
| 3 | Route MOB_ESCORT_FULL_PATH in the gms_v92 seed template | DONE | `git show cd2d70316` shows exactly one inserted object (`+8/-0`) at the `0x128` slot between `0x124` (CatchMonsterWithItem) and `0x12F` (SpawnNPC). Python sort/order check confirms `count 153 sorted True` and the row lands correctly. Commit cd2d70316. |
| 4 | Re-pin evidence and promote the gms_v92 matrix cell | DONE | `git show abae6efea` shows only the new `gms_v92` evidence YAML (`decompile_sha256: 028ca4bd…`, `category: TIER1-FIXTURE`, `verifies:` link present), and `STATUS.md`/`status.json` diff moving exactly the `MOB_ESCORT_FULL_PATH` × `gms_v92` cell (`incomplete` → `verified`, ❌→✅) plus the v92 summary row (`74→75`, `8.4%→8.5%`). No `gms_v95`/`jms_v185` evidence files modified (expected — unchanged export hash, per design §6). Step 5's six gates were re-run in this audit: `cmd` test suite `ok`, `fname-doc --check` OK, `operations --check` OK (0 absent-writer notes), `dispatcher-lint` clean, `doc-freshness --check` OK, `gate-check --check` OK (21 gates, 1 partial-by-design), `matrix --check` exit 0 (only pre-existing unrelated n-a notes, no MOB_ESCORT_FULL_PATH drift). Step 7 (flagless `tools/verify.sh`) was deliberately moved out of the implementer's scope to a branch-end task-verifier per controller ruling — confirmed via `agent-ledger.tsv` line `branch-end gate  task-verifier  ...  dispatched  7ca937148`. Commit abae6efea. |

**Completion Rate:** 4/4 plan tasks (100%), plus 1 approved out-of-plan follow-up (Task 5, commit a02152042, fixing the design.md/code doc-comment arithmetic error, reviewed APPROVED per ledger).
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. No task was skipped or left partial. The one deferred item — Task 1's review flagged the doc-comment's `9×Decode4` arithmetic error as a non-blocking finding (ledger: Task 1 review `APPROVED_WITH_FINDINGS`) — was picked up and closed by the controller-dispatched Task 5 (commit a02152042), which corrected the code comment and design.md to `10×Decode4`. Verified arithmetic: `count(1) + oldDestX(1) + oldDestY(1) + 2×(x,y,attr)(6) + currentDestIndex(1) = 10`, matching the struct's actual `Encode`/`Decode` field order in `mob_escort_full_path.go:113-146`. `plan.md:222` still reads `9×Decode4` — per the controller's explicit ruling this is an accepted historical artifact of the pre-fix doc comment quoted into the plan, not a live divergence, and is not treated as a blocking finding here.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| libs/atlas-packet | PASS | PASS | `gofmt -l ./monster/clientbound/` empty; `go build ./...` clean; `go test ./... -count=1` all `ok` |
| services/atlas-channel/atlas.com/channel | PASS | PASS | `gofmt -l ./socket/writer/` empty; `go build ./...` clean; `go vet ./...` clean; `go test ./... -count=1` all `ok`, including `atlas-channel/socket/writer` |
| tools/packet-audit (CI gate set) | PASS | PASS | `cd tools/packet-audit && go test ./...` all `ok`; the six Task 4 Step 5 gates (`fname-doc`, `operations`, `dispatcher-lint`, `doc-freshness`, `gate-check`, `matrix --check`) all exit 0 |
| tools/verify.sh (flagless) | NOT RUN BY THIS AUDIT | — | Deliberately deferred to a separately-dispatched branch-end `task-verifier` per controller ruling (confirmed in `agent-ledger.tsv`); this audit did not duplicate that run |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (contingent on the in-flight branch-end `tools/verify.sh` task-verifier run reporting exit 0 — not yet confirmed by this audit)

## Action Items

1. None required from this audit. Confirm the branch-end `task-verifier`'s flagless `tools/verify.sh` run (ledger entry, dispatched at commit `7ca937148`) completes with exit 0 before merge, per "Done means verified."
