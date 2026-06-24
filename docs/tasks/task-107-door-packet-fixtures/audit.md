# Plan Audit — task-107-door-packet-fixtures

**Plan Path:** docs/tasks/task-107-door-packet-fixtures/plan.md
**Audit Date:** 2026-06-23
**Branch:** task-107-door-packet-fixtures
**Base Branch:** main (9a1656e46)
**HEAD:** 2225110664a2d9650012633140529c5a382e86bc

## Executive Summary

The campaign faithfully implemented its plan. All 12 incomplete door/clientbound cells (3 packets × gms_v84/gms_v87/gms_v95/jms_v185) were promoted to `verified`, landed as 12 coupled per-cell commits (Tasks 2–13), with the final sweep (Task 14) gates passing. For every cell the four coupled artifacts — export splice, audit report, marker, and (tier-1 only) evidence record — agree on the receiver address. No production Go was modified. `go test -race`/`vet`/`build` are clean in `libs/atlas-packet`, and `matrix --check` exits **0** with zero door-related problem lines. **Recommendation: READY_TO_MERGE.**

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Preflight (no commit) | DONE | Reshaping finding re-confirmed: receivers absent on `main` for all 4 exports (0 occurrences each). |
| 2 | gms_v84 SpawnDoor (T0) | DONE | export `0x7e3740`; report `Address 0x7e3740`, FlatInvalid:false, rows [0,0,0,0]; marker spawn_test.go:26; commit e2171d1c7 |
| 3 | gms_v84 RemoveDoor (T0) | DONE | export `0x7e40de`; report `0x7e40de`, FlatInvalid:false, rows [0,0]; marker remove_test.go:30; commit 02bc9e34c |
| 4 | gms_v84 RemoveTownDoor (T1) | DONE | export `0xa6dbb8`; report `0xa6dbb8`, FlatInvalid:true, rows [0,0,2,2]; marker remove_town_test.go:31; evidence gms_v84/...RemoveTownDoor.yaml w/ verifies:; commit 8ce0c5f07 |
| 5 | gms_v87 SpawnDoor (T0) | DONE | export/report/marker `0x810af2`; rows [0,0,0,0]; commit 9c04d5804 |
| 6 | gms_v87 RemoveDoor (T0) | DONE | export/report/marker `0x811487`; rows [0,0]; commit 29a5f5c1c |
| 7 | gms_v87 RemoveTownDoor (T1) | DONE | export/report/marker `0xab9ef6`; FlatInvalid:true rows [0,0,2,2]; evidence present w/ verifies:; commit 2f3a33866 |
| 8 | gms_v95 SpawnDoor (T0) | DONE | export/report/marker `0x762c00`; rows [0,0,0,0]; commit b27e0b366 |
| 9 | gms_v95 RemoveDoor (T0) | DONE | export/report/marker `0x761920`; rows [0,0]; commit 164087ff6 |
| 10 | gms_v95 RemoveTownDoor (T1) | DONE | export/report/marker `0x9f1330`; FlatInvalid:true rows [0,0,2,2]; evidence present w/ verifies:; commit 191f93547 |
| 11 | jms_v185 SpawnDoor (T0) | DONE | export/report/marker `0x840fc6`; rows [0,0,0,0]; commit 44167d84b |
| 12 | jms_v185 RemoveDoor (T0) | DONE | export/report/marker `0x84195b`; rows [0,0]; commit 4bfe4f44e |
| 13 | jms_v185 RemoveTownDoor (T1) | DONE | export/report/marker `0xb0977c`; FlatInvalid:true rows [0,0,2,2]; evidence present w/ verifies:; commit 222511066 |
| 14 | Final sweep / gates | DONE | 15/15 door cells verified; matrix --check exit 0, no door lines; module test/vet/build clean |

**Completion Rate:** 14/14 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Per-Cell Coupled-Artifact Cross-Check

Address agreement (marker `ida=` == report `Address` == spliced export address) holds for **all 12 cells** — verified directly:

| Version | SpawnDoor | RemoveDoor | RemoveTownDoor |
|---|---|---|---|
| gms_v84 | 0x7e3740 | 0x7e40de | 0xa6dbb8 |
| gms_v87 | 0x810af2 | 0x811487 | 0xab9ef6 |
| gms_v95 | 0x762c00 | 0x761920 | 0x9f1330 |
| jms_v185 | 0x840fc6 | 0x84195b | 0xb0977c |

- **Exports ADD-only:** `git diff --numstat main...HEAD` shows `+N / -0` for all 4 export files (v84 +60, v87 +60, v95 +60, jms +64; 0 removed). Each receiver absent on `main`, present exactly once on HEAD — pure additive splice, no overwrite of any pre-existing key.
- **Markers:** each of the 3 test files has exactly 5 markers (v83 + 4 new); addresses match exports.
- **Tier-0 reports:** SpawnDoor 4 rows all Verdict 0; RemoveDoor 2 rows all Verdict 0; FlatInvalid:false — matches verified gms_v83 reference exactly.
- **Tier-1 reports:** RemoveTownDoor FlatInvalid:true, rows [0,0,2,2] — matches the verified gms_v83 reference (the two guarded position reads are V2; this is the CORRECT expected shape, not a defect).
- **Tier-1 evidence:** all 4 records carry that version's own address (no cross-version copy), 4 distinct `decompile_sha256` values, and the `verifies: - libs/atlas-packet/door/clientbound/remove_town_test.go#TestRemoveTownDoor` block.
- **Tier-0 evidence:** none exists (correct — SpawnDoor/RemoveDoor must not be pinned).
- **status.json:** all 15 cells (3 packets × gms_v83/v84/v87/v95/jms_v185) read `"state":"verified"`; no `"note":"no audit report"` remains on any door cell.
- **Commit coupling:** tier-1 commit (8ce0c5f07) bundles export + report json/md + evidence yaml + marker + STATUS.md/status.json; tier-0 commit (b27e0b366) bundles the same minus evidence. Per-cell granularity as planned.

## Skipped / Deferred Tasks

None. `SpawnPortal` (live-portal, 12 bytes) was correctly left out of scope per design §9 — no marker, report, or evidence added on any target version; it has no status.json op row.

## Build & Test Results

| Service / Module | Build | Tests | Notes |
|---|---|---|---|
| libs/atlas-packet | PASS (`go build ./...`) | PASS (`go test -race ./door/...`) | `go vet ./...` exit 0; no go.mod touched → no `docker buildx bake` required (correct) |
| tools/packet-audit `matrix --check` | n/a | PASS (exit 0) | Zero door orphan/dangling/stale/drift lines |
| tools/packet-audit `fname-doc --check` | n/a | PASS | No door lines |
| tools/packet-audit `operations --check` | n/a | PASS | No door lines |

`tools/redis-key-guard.sh` not re-run as a blocker: the diff contains zero production Go, so it cannot be affected (fails identically on main — pre-existing, unrelated).

## Notable Observations (non-blocking)

1. **jms binary name vs plan text.** Commits record `MapleStory_dump_SCY.exe@13338`; the plan (Task 11/13) instructed using the clean `*_U_DEVM` build "NOT the SMC retail dump." This is NOT a defect: the committed `docs/packets/ida-exports/gms_jms_185.json` metadata itself declares `binary: MapleStory_dump_SCY.exe (JMS v185.1)` (md5 af6652ff…). The worker used the binary that matches the canonical committed jms export, and the resulting report verdicts are clean (FlatInvalid:false on tier-0, the expected 0/0/2/2 on tier-1). The plan's `*_U_DEVM` reference was a planner assumption that diverged from the on-disk export's actual source binary; the implementation is internally consistent.
2. **matrix --check exits 0**, not 1. The plan budgeted for a pre-existing conflict backlog ("no new problems" bar rather than exit 0). The actual run is fully clean — strictly better than the bar.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None required. Optional, before/with the PR:
1. In the PR description, note the jms binary-name reconciliation (Observation 1) so a reviewer isn't surprised the commit says `MapleStory_dump_SCY.exe` while the plan text said `*_U_DEVM` — the committed export is the source of truth and they agree.
2. Per plan Task 14 Step 6, flag `SpawnPortal` in the PR description as the untracked-but-evidenced writer (v83 evidence + report exist, no status.json op row) — a candidate future matrix row, deliberately out of scope here.

---

# Backend Audit — atlas-packet (door/clientbound)

- **Auditor:** backend-guidelines-reviewer
- **Date:** 2026-06-23
- **Scope:** Go delta of `task-107-door-packet-fixtures` vs `main`
- **Module:** `libs/atlas-packet`
- **Build:** PASS
- **Tests:** door/... PASS (-race, -count=1); vet clean; build clean
- **Overall:** PASS

## Scope Confirmation

`git diff --stat main...HEAD -- '*.go'` returns exactly three files, +12 lines:

| File | Lines added |
|------|-------------|
| libs/atlas-packet/door/clientbound/remove_test.go | +4 |
| libs/atlas-packet/door/clientbound/remove_town_test.go | +4 |
| libs/atlas-packet/door/clientbound/spawn_test.go | +4 |

Every added line is a `// packet-audit:verify packet=... version=... ida=0x...`
comment placed directly above an existing `Test*` function. No statements, no
identifiers, no encoder/decoder logic. Confirmed by reading the full
`git diff main...HEAD -- '*.go'`.

## Mechanical Checks

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SCOPE-01 | No production (non-test) `.go` file modified | PASS | `git diff --name-only main...HEAD -- '*.go' \| grep -v '_test\.go$'` → empty |
| SCOPE-02 | No new `*_testhelpers.go` / test-only constructor added | PASS | `git diff --name-status main...HEAD \| grep testhelpers` → empty; all added Go files are `_test.go` edits only (no `A` Go files) |
| SCOPE-03 | Added lines are comment markers only | PASS | All +12 lines match `^// packet-audit:verify ` above existing `Test*` funcs (remove_test.go:30-33, remove_town_test.go:31-34, spawn_test.go:26-29) |
| GATE-01 | `go test -race -count=1 ./door/...` passes | PASS | `clientbound` 1.014s ok, `serverbound` 1.012s ok, exit 0 |
| GATE-02 | `go vet ./...` clean | PASS | exit 0, no output |
| GATE-03 | `go build ./...` clean | PASS | exit 0, no output |

## DOM-* / SUB-* Applicability

The DOM-* and SUB-* checklists target service **domain packages** (those with
`model.go` / `resource.go` under a `services/atlas-<svc>/.../internal/`).
This change touches `libs/atlas-packet/door/clientbound`, a packet-codec
library with no domain model, processor, resource, administrator, provider, or
REST layer. No DOM-* or SUB-* item is in scope. The Test Helper Pattern rule
(no `*_testhelpers.go`) is the only project-wide rule that could apply here, and
it passes (SCOPE-02).

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- None.

The only Go change on this branch is comment-line `packet-audit:verify` markers
added above three pre-existing door packet tests. The tests still build and pass
under `-race`, `go vet` and `go build` are clean, and no production code, new
types, new test bodies, or test-helper files were introduced. No DOM-*
violation is possible from comment additions. Backend-guidelines verdict: PASS.
