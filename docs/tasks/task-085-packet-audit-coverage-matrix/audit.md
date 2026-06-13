# Plan Audit — task-085-packet-audit-coverage-matrix

**Plan Path:** docs/tasks/task-085-packet-audit-coverage-matrix/plan.md
**Audit Date:** 2026-06-13 (re-audit; supersedes the 2026-06-12 snapshot below the line)
**Branch:** task-085-packet-audit-coverage-matrix
**Base Branch:** main (origin/main @ afb6224e)
**HEAD:** 2f0073af ("close gms_v84 gap → verified 145 → 185; fix 3 cash registry opcodes")
**Diff basis:** `git diff origin/main...HEAD` (three-dot; 111 commits, 3920 files)

## Executive Summary

All 28 plan tasks (Phases 1–7) are DONE. The branch has advanced materially since the
2026-06-12 audit (63 → 111 commits): the three items that audit flagged as
deferred/grandfathered have since been **completed**, not merely tracked. Specifically:
(1) the conflict backlog was burned down **1092 → 0** (registry re-derivation + template
wiring across all five versions); (2) the CI gate was flipped to **fully blocking** — any
non-zero `matrix --check` exit now fails CI, the escape-hatch is gone (commit ed5ac8016);
(3) the D7 **serverbound** verification pass was implemented (`verify-serverbound`
subcommand + per-version worklists), not deferred; and (4) **gms_v84 is fully populated**
(145 → 185 verified, parity with v83). `matrix --check` exits **0** on a clean tree;
committed STATUS.md/status.json are **fresh** (regeneration produced an empty diff).
Both changed Go modules (`tools/packet-audit`, `libs/atlas-packet`) build, vet, and
`go test -race` clean. Recommendation: **READY_TO_MERGE** pending the backend-guidelines
review leg (Go changed).

The plan's checkboxes were never ticked (112 `- [ ]`, 0 `- [x]`); completion verified
entirely from git history and on-disk artifacts.

## Matrix State (status.json @ toolSha a15e4d45)

| Metric | Value |
|--------|-------|
| Total cells | 4400 (880 rows × 5 versions) |
| ✅ verified | **1154** |
| ❌ incomplete | 2426 |
| ⬜ n-a | 820 |
| 🟥 conflict | **0** |
| `matrix --check` | exit **0** |

`exportHashes` present for all five versions (gms_v83/v84/v87/v95, jms_v185). `toolSha`
is a tools/packet-audit tree hash, not a commit SHA — consistent with design D2.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1.1 | yaml.v3 dependency | DONE | `tools/packet-audit/go.mod` lists `gopkg.in/yaml.v3 v3.0.1`; only go.mod touched on branch |
| 1.2 | `internal/opregistry` | DONE | `internal/opregistry/{opregistry.go,opregistry_test.go}` + testdata |
| 1.3 | `internal/seedcsv` | DONE | `internal/seedcsv/` with both excerpt CSVs |
| 1.4 | `registry seed` + real registry | DONE | `cmd/registry.go`; `cmd/root.go:41` dispatch; 5 `docs/packets/registry/*.yaml` + README |
| 1.5 | matrix model + input loading | DONE | `internal/matrix/{model.go,load.go,load_test.go}` + testdata |
| 1.6 | grading engine (all §5 rules) | DONE | `internal/matrix/{grade.go,grade_test.go}`; rules since refined (conflict content-gating, demux worst-of) |
| 1.7 | assembly + STATUS.md/status.json render | DONE | `internal/matrix/{build.go,render.go,render_test.go}` + golden |
| 1.8 | `matrix` subcommand + STATUS.md | DONE | `cmd/matrix.go`; `cmd/root.go:44`; `docs/packets/audits/STATUS.md` + `status.json` (both fresh) |
| 2.1 | `internal/evidence` schema/loader/hash | DONE | `internal/evidence/{evidence.go,hash.go,evidence_test.go}` |
| 2.2 | `evidence pin` subcommand | DONE | `cmd/evidence.go`; `cmd/root.go:47` |
| 2.3 | drift + `--check` wiring | DONE | `internal/matrix/evidence_input.go` (+test) |
| 2.4 | `tiers.yaml` + tier loader | DONE | `docs/packets/evidence/tiers.yaml`; `internal/matrix/{tiers.go,tiers_test.go}` |
| 2.5 | migrate prose acceptances + freeze | DONE | Prose files FROZEN; **1192** evidence YAMLs now under `docs/packets/evidence/{gms_v83,gms_v84,gms_v87,gms_v95,jms_v185}/` (was 79 / 4 dirs at prior audit) |
| 3.1 | `internal/marker` scanner | DONE | `internal/marker/{marker.go,marker_test.go}` + testdata |
| 3.2 | marker wiring: promotion + orphan check | DONE | `cmd/matrix_markers_test.go`; tier resolution refinements landed (TypeRegistry recursion gaps documented in `tiers.yaml`) |
| 3.3 | retrofit markers onto byte-fixtures | DONE | **1438** `packet-audit:verify` markers across **275** `*_test.go` files in `libs/atlas-packet` (was 58 markers at prior audit) |
| 4.1 | VERIFYING_A_PACKET.md playbook | DONE | `docs/packets/audits/VERIFYING_A_PACKET.md` |
| 4.2 | `/verify-packet` command | DONE | `.claude/commands/verify-packet.md` |
| 4.3 | `packet-verifier` agent | DONE | `.claude/agents/packet-verifier.md` |
| 4.4 | e2e promotion-loop validation (3 cells) | DONE | Promoted cells visible in STATUS.md; evidence records with `verifies:` filled |
| 5.1 | clientbound dispatch-walk parser | DONE | `internal/discover/{discover.go,discover_test.go}` + real-Hex-Rays fixtures |
| 5.2 | serverbound send-site enum + reconcile | **DONE** (was DEFERRED) | `cmd/verify_serverbound.go` (+test); `cmd/root.go:59` dispatch; worklists `docs/packets/registry/verify_serverbound_{gms_v87,gms_v95,jms_v185}.md`; commits d74448bad, cc9c5180e, 37241b873, fe0834586, e99b098d1 |
| 5.3 | `discover-ops` subcommand | DONE | `cmd/discover_ops.go` (+test); `cmd/root.go:56` |
| 5.4 | discovery ×5 IDBs | DONE | Per-version reconcile commits + worklists `docs/packets/registry/discover_{gms_v83,gms_v84,gms_v87,gms_v95,jms_v185}.md`; v84 gap since closed |
| 5.5 | v84 export harvest + audit pass | DONE | `docs/packets/ida-exports/gms_v84.json`; `docs/packets/audits/gms_v84/` (587 files); v84 verified **185** (parity with v83), HEAD commit 2f0073af |
| 6.1 | CI `matrix --check` gate | **DONE** (now fully blocking) | `.github/workflows/packet-matrix.yml` — any non-zero exit fails CI; grandfather/escape-hatch removed (commit ed5ac8016 "flip … to fully blocking; conflict backlog now zero") |
| 6.2 | STARTING_A_NEW_VERSION_PASS rewrite + gate | DONE | `docs/packets/audits/STARTING_A_NEW_VERSION_PASS.md` |
| 7.1 | full verification sweep | DONE | All gates re-verified by this audit (see below); reseed invariant holds |

**Completion Rate:** 28/28 tasks DONE (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Changes vs the prior (2026-06-12) audit

The prior audit was an accurate snapshot of HEAD-as-of-63-commits but is now stale on four
points, all resolved in the branch's favor:

1. **Conflicts 1092 → 0.** The grandfathered conflict class no longer exists; the registry
   was re-derived and templates wired across v87/v95/jms/v84. `matrix --check` exits 0.
2. **CI gate is fully blocking.** The two-step "grandfather exit 1 + fatal-class grep"
   escape hatch in packet-matrix.yml was replaced by an unconditional "non-zero = fail"
   gate (ed5ac8016). The categorized message step remains only to print a helpful error.
3. **D7 serverbound pass: implemented, not deferred.** `verify-serverbound` subcommand,
   send-site opcode adjudication, and per-version worklists landed; serverbound rows are
   IDB-verified where resolved, not left CSV-provenance.
4. **gms_v84 fully populated.** Prior audit noted v84 ≥0x3F as UNVERIFIED with the column
   mostly ❌. The v84 IDB was named out and re-exported; v84 now reads 185 verified (v83
   parity) per the HEAD commit.

## Build & Test Results

| Module / Gate | Build | Vet | Tests | Notes |
|---------------|-------|-----|-------|-------|
| tools/packet-audit | PASS | PASS | PASS | `go test -race ./...` — 12 internal packages (incl. opregistry, seedcsv, evidence, marker, discover, matrix) + cmd all `ok` |
| libs/atlas-packet | PASS | PASS | PASS | `go test -race ./...` — 0 FAIL |
| `matrix --check` | — | — | PASS | exit **0**, no output (no conflicts, no fatal-class findings, outputs fresh) |
| STATUS.md/status.json freshness | — | — | PASS | `go run ./tools/packet-audit matrix` regen → empty git diff |
| `registry seed` reseed invariant | — | — | PASS | Reseed differs from committed only by added `ida-discovered`/`manual` provenance + re-classified rows; pure csv-import seed reproducible |

Only `tools/packet-audit/go.mod` is touched on the branch (yaml.v3, Task 1.1). No service
`go.mod` changed, so the `docker buildx bake` requirement does not trigger. The four
`services/atlas-configurations/seed-data/templates/template_*.json` changes are opcode
wiring (data, not Go).

### redis-key-guard — not a task-085 finding

`tools/redis-key-guard.sh` exits 1, but the failure is in
`services/atlas-party-quests/atlas.com/party-quests` ("./... matched no packages"). That
directory **exists on origin/main and is NOT in this branch's three-dot diff** — it is a
pre-existing main-side condition (an incomplete module that breaks the guard's package
walk), unrelated to task-085. task-085 introduces no Redis code (only go.mod touched is
tools/packet-audit). The prior audit ran the guard before this main-side directory was
present. Recommend filing/handling the party-quests guard breakage separately; it does not
block this PR.

## Overall Assessment

- **Plan Adherence:** FULL — every plan deliverable exists and is wired; the items the
  prior audit listed as deferred/grandfathered have since been completed.
- **Recommendation:** READY_TO_MERGE (pending the backend-guidelines-reviewer leg, since
  Go changed — standard pre-PR step, not a gap in the work).

## Action Items

1. Run the backend-guidelines-reviewer (Go changed) before opening the PR — the one
   remaining standard pre-PR step.
2. PR body should state the expectation-setting facts: matrix now reads 1154 verified /
   0 conflict / `--check` exit 0; conflict trajectory 1092 → 0; v84 brought to v83 parity.
3. The pre-existing `services/atlas-party-quests` redis-key-guard breakage is a main-side
   issue surfaced incidentally; track it separately — it is out of scope for task-085.

---

_(Prior audit, 2026-06-12, retained for history. Its 27/28 + 1 PARTIAL tally and the
1092-conflict / grandfathered-CI / deferred-D7 / unverified-v84 findings reflect an
earlier HEAD (63 commits) and are superseded by the re-audit above.)_
