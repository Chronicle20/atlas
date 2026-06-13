# Plan Audit — task-085-packet-audit-coverage-matrix

**Plan Path:** docs/tasks/task-085-packet-audit-coverage-matrix/plan.md
**Audit Date:** 2026-06-12
**Branch:** task-085-packet-audit-coverage-matrix
**Base Branch:** main

## Executive Summary

All 28 plan tasks (Phases 1–6 plus the verification sweep) have corresponding commits and artifacts on the branch (63 commits, 702 files changed). 27 tasks are DONE — five of them via controller-approved deviations that are documented in-tree exactly as agreed; Task 7.1 is PARTIAL only because its final step (code review + PR) is the step this audit executes. Builds, vet, and race tests pass clean in both changed modules (`tools/packet-audit`, `libs/atlas-packet`); `redis-key-guard` is clean; `matrix --check` exits 1 with exactly the grandfathered conflict class (1092 conflicts, 0 fatal-class findings), which is the documented pass condition of the Task 6.1 CI gate.

Note: the plan's checkboxes were never ticked (113 `- [ ]`, 0 `- [x]`) — completion was verified entirely from git history and on-disk artifacts.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1.1 | yaml.v3 dependency | DONE | c4b542514; `tools/packet-audit/go.mod:5` (`gopkg.in/yaml.v3 v3.0.1`) |
| 1.2 | `internal/opregistry` | DONE | f19106fa6; `tools/packet-audit/internal/opregistry/{opregistry.go,opregistry_test.go}` + 5 testdata YAMLs (plan asked 2; extras are negative cases) |
| 1.3 | `internal/seedcsv` | DONE | f2b8fb9fa + review-hardening 19c33d19c; `tools/packet-audit/internal/seedcsv/` with both excerpt CSVs |
| 1.4 | `registry seed` subcommand + real registry | DONE | bd4baec13 + d12bd60b4; `cmd/registry.go`, `cmd/root.go:41` dispatch; `docs/packets/registry/{gms_v83,gms_v84,gms_v87,gms_v95,jms_v185}.yaml` + `README.md` |
| 1.5 | matrix model + input loading | DONE | 3d4af4206; `internal/matrix/{model.go,load.go,load_test.go}` + testdata audits/templates |
| 1.6 | grading engine | DONE | ec1893e9d; `internal/matrix/{grade.go,grade_test.go}`; D3/D5 refinements visible in conflict-class messages (see `matrix --check` output) |
| 1.7 | assembly + rendering | DONE | c87413b31 + 6c6f139cd, 564fed943; `internal/matrix/{build.go,render.go,render_test.go}`, `testdata/golden_STATUS.md` |
| 1.8 | `matrix` subcommand + first STATUS.md | DONE | ab37d1e62 + 1b3ce7d5f, f9e7a9ab9; `cmd/matrix.go`, `cmd/root.go:44`; `docs/packets/audits/STATUS.md` (stamp = tool tree SHA per D2, line 6) + `status.json` |
| 2.1 | `internal/evidence` schema/loader/hash | DONE | 245542aaf; `internal/evidence/{evidence.go,hash.go,evidence_test.go}` + both testdata files (D1 `FunctionHash`) |
| 2.2 | `evidence pin` subcommand | DONE | 7ae59eb46; `cmd/evidence.go`, `cmd/root.go:47` |
| 2.3 | drift + `--check` wiring | DONE | 4716fa543 + deterministic ordering 139f451d2; `internal/matrix/evidence_input.go`(+test) |
| 2.4 | `tiers.yaml` + tier loader | DONE | 3acbe82a3; `docs/packets/evidence/tiers.yaml`, `internal/matrix/{tiers.go,tiers_test.go}`, `cmd/matrix_opaque_tier_test.go` |
| 2.5 | migrate prose acceptances + freeze | DONE (approved deviation) | 9b1f3c5d9 + regen 9a0a1fe4e; 29 records migrated, scoped to non-Match cells per design §6.1 (controller-approved); not-recoverable entries listed in commit body; FROZEN banners present in all three files (`docs/packets/ida-exports/_pending.md:1`, `docs/packets/audits/gms_v95/_pending.md:1`, `docs/packets/audits/OPAQUE_LEDGER.md:1`); 79 evidence YAMLs now under `docs/packets/evidence/{gms_v83(23),gms_v87(11),gms_v95(23),jms_v185(22)}` |
| 3.1 | `internal/marker` scanner | DONE | 31627bd9a; `internal/marker/{marker.go,marker_test.go}`, `testdata/invite_test.go.txt` |
| 3.2 | marker wiring: verified promotion + orphan check | DONE (approved deviation) | f07803e5a + 433b57944, ffb161231; `cmd/matrix_markers_test.go`. Planned tier-prefix removal NOT done: TypeRegistry recursion gaps (short-name misresolution, WriterName mismatches) documented in `docs/packets/evidence/tiers.yaml:8-24` with corrected D4 rationale at `tiers.yaml:36-43` (f5ada28ad, 3eb5c8daf) |
| 3.3 | retrofit markers onto byte-fixture tests | DONE | 7d034f987; 58 `packet-audit:verify` markers across 40+ files in `libs/atlas-packet/**/*_test.go` (e.g. `party/clientbound/invite_test.go`); TIER1-FIXTURE evidence records added; STATUS.md ✅ count now 23 (was 0) |
| 4.1 | VERIFYING_A_PACKET.md playbook | DONE | 85c43215a + feedback fold-in 454a16905; `docs/packets/audits/VERIFYING_A_PACKET.md` — 8 numbered steps (§1–§8) + failure-modes section |
| 4.2 | `/verify-packet` command | DONE | 4c450bba9 + f18bb49b6; `.claude/commands/verify-packet.md` (frontmatter: description + argument-hint) |
| 4.3 | `packet-verifier` agent | DONE | 30940ae1a; `.claude/agents/packet-verifier.md` |
| 4.4 | e2e promotion-loop validation (3 cells) | DONE | 60d216435; tier-0 `ui/clientbound/Disable×v83`, tier-1 `character/clientbound/CharacterSitResult×v83`, opaque `monster/clientbound/MonsterStatSet×v83` all promoted to ✅ (visible in STATUS.md); evidence record example `docs/packets/evidence/gms_v83/character.clientbound.CharacterSitResult.yaml` with `verifies:` filled |
| 5.1 | clientbound dispatch-walk parser | DONE | 2a0dc5285 + hardening 5421ad3e7, c510a38a0, 59cb63880, 55fadbeba; `internal/discover/{discover.go,discover_test.go}` + 3 real-Hex-Rays testdata fixtures |
| 5.2 | reconciliation (`Reconcile`) | DONE | 3618c4ca4; `internal/discover/reconcile_test.go` |
| 5.3 | `discover-ops` subcommand | DONE | 482eba710; `cmd/discover_ops.go`(+test), `cmd/root.go:56` |
| 5.4 | OPERATOR-GATED discovery ×5 IDBs | DONE (approved deviation) | One reconcile commit per version (88db3d2f9 v83, 3bcb21c8e v84, d0815ae66 v87, 11c041ea0 v95, 631b0cb80 jms185) + regen c03916c56; worklists `docs/packets/registry/discover_{gms_v83,gms_v84,gms_v87,gms_v95,jms_v185}.md` all present. Deviations as approved: multi-dispatcher union (593f016ee — ProcessPacket is a shim); v84 SHIFTED opcode table ≥0x3D → registry bannered UNVERIFIED (`docs/packets/registry/gms_v84.yaml:5-19`) per the plan's "large delta = investigate" rule; conflicts rose 964→1092 because the registry grew — documented in c03916c56. D7 serverbound pass deferred, documented at `cmd/discover_ops.go:85-89` and in every worklist header (`discover_gms_v83.md:4`) |
| 5.5 | OPERATOR-GATED v84 harvest + first audit pass | DONE (approved deviation) | 47ca13ca5; `docs/packets/ida-exports/gms_v84.json` (271 roster entries, 12 resolved — v84 IDB handlers unnamed, per commit body); `docs/packets/audits/gms_v84/` (507 files); v84 column populated (STATUS.md Totals line 914: 0✅/6🟡/559❌). Plan's `validate --version` invocation didn't exist; real first-audit-pass invocation used and recorded in commit |
| 6.1 | CI `matrix --check` gate | DONE (approved deviation) | 5016c17c7 + 6f641075a; `.github/workflows/packet-matrix.yml` — escape hatch implemented as hardened two-step gate: exit-1 grandfathered with tracking note + follow-up pointer (lines 30-38), runtime-failure guard (exit≠0,1 hard-fails, lines 55-59) and fatal-class grep (orphan/dangling/stale/drift/unresolv/malformed/no-export never grandfathered, lines 60-66) |
| 6.2 | STARTING_A_NEW_VERSION_PASS rewrite + task-close gate | DONE | 327a1bc46 + doc fixes 6f641075a; full rewrite as matrix-workflow orchestration doc; task-close gate at `docs/packets/audits/STARTING_A_NEW_VERSION_PASS.md:434` |
| 7.1 | full verification sweep | PARTIAL | Steps 1–2 re-verified by this audit (all green — see Build & Test Results; reseed invariant holds: 115-line diff vs committed `gms_v83.yaml`, all post-seed discovery/manual entries as plan expects). Step 3 done: retrospective pointer appended (025deecaf, `retrospective.md` final line). Step 4 (code review + PR) in progress — this audit is the plan-adherence leg; PR not yet opened (correct ordering per CLAUDE.md: review before PR) |

**Completion Rate:** 27/28 tasks DONE, 1 PARTIAL (96%)
**Skipped without approval:** 0
**Partial implementations:** 1 (Task 7.1 — final step is the in-flight review/PR itself)

## Skipped / Deferred Tasks

No task was skipped. Deferred/deviating items, all controller-approved and documented in-tree:

- **D7 serverbound verification pass** (within Task 5.4) — deferred to follow-up; documented at `tools/packet-audit/cmd/discover_ops.go:85-89` and in each `docs/packets/registry/discover_*.md` header. Impact: serverbound registry rows remain CSV-provenance until the live pass.
- **v84 clientbound opcodes ≥ 0x3F** — UNVERIFIED v83-inherited values, bannered at `docs/packets/registry/gms_v84.yaml:5-19` with per-range shift map in `discover_gms_v84.md`. Honest deviation: reconciliation would have required guessing unnamed sub_XXXX handlers.
- **Conflict burn-down (~1092 rows)** — grandfathered via the Task 6.1 escape hatch; tracking note + follow-up pointer in `.github/workflows/packet-matrix.yml:30-38`. Trajectory went up (964→1092) because discovery grew the registry; explained in commit c03916c56.
- **Tier-prefix cleanup** (Task 3.2's planned removal) — prefixes retained per D4 because of two verified TypeRegistry recursion gaps, documented with corrected rationale at `docs/packets/evidence/tiers.yaml:8-24,36-43`. Removing them would silently drop tier-1 for MonsterMovement/PetMovement/InventoryAdd/CharacterSpawn.

## Build & Test Results

| Module / Gate | Build | Tests | Notes |
|---------------|-------|-------|-------|
| tools/packet-audit | PASS | PASS | `go build ./...`, `go vet ./...`, `go test -race ./...` — 13 packages ok (incl. all 6 new: opregistry, seedcsv, evidence, marker, discover, matrix) |
| libs/atlas-packet | PASS | PASS | `go build ./...`, `go vet ./...`, `go test -race ./...` — 58 packages ok, 0 FAIL |
| tools/redis-key-guard.sh | — | PASS | exit 0, all 56 modules scanned |
| `matrix --check` | — | PASS (gate semantics) | exit 1, 1092 conflict lines (the grandfathered class), 0 fatal-class hits (`grep -ciE 'orphan|dangling|stale|drift|unresolv|malformed|no export for version'` = 0) — exactly the packet-matrix.yml pass condition |
| `registry seed` reseed invariant | — | PASS | `/tmp/reseed085/gms_v83.yaml` vs committed: 115 diff lines, all post-seed discovery/manual provenance (plan 7.1 step 2 expectation) |

No service `go.mod` was touched, so no `docker buildx bake` requirement triggers (plan execution constraint, confirmed via `git diff --stat main...HEAD`).

Working-tree note: one uncommitted hunk in `go.work.sum` (a single `golang.org/x/term v0.44.0 h1:` hash line) — local `go` invocation artifact, pre-existing before this audit; harmless but should be committed or discarded before the PR.

## Overall Assessment

- **Plan Adherence:** MOSTLY_COMPLETE (functionally FULL — every deliverable exists; the only open item is the review/PR step itself)
- **Recommendation:** NEEDS_REVIEW (complete the backend-guidelines review leg, then READY_TO_MERGE)

## Action Items

1. Complete the remaining code-review leg (backend-guidelines-reviewer — Go changed) before opening the PR (Task 7.1 step 4).
2. Commit or discard the stray `go.work.sum` hunk so the PR branch is clean.
3. PR body must include (per plan 7.1 step 4): the honesty-shock expectation (first STATUS.md mostly 🟡/❌), the conflict trajectory 964→1092 with the registry-growth explanation, and the three e2e-promoted cells (ui/clientbound/Disable, character/clientbound/CharacterSitResult, monster/clientbound/MonsterStatSet — all ×gms_v83).
4. Follow-up tasks already pointed to in-tree (no new action needed, listed for traceability): conflict burn-down (packet-matrix.yml note), D7 serverbound pass, v84 ≥0x3F re-derivation, TypeRegistry recursion gaps.
