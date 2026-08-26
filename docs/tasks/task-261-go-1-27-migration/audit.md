# Plan Audit — task-261-go-1-27-migration

**Plan Path:** docs/tasks/task-261-go-1-27-migration/plan.md
**Audit Date:** 2026-08-25
**Branch:** task-261-go-1-27-migration
**Base Branch:** main (merge base `855fef4d1`)

## Executive Summary

All 10 plan tasks are implemented and verified against live code, not just against
the plan's own evidence docs. 103 non-fixture `go.mod` files plus `go.work` are
pinned to `go 1.27.0` exactly (confirmed by direct grep sweep, not by trusting the
guard's own count); the 8 `tools/cideps/testdata` fixtures remain untouched at
`go 1.25.5`; no `toolchain` directive exists anywhere; `go.sum` has zero diff.
`tools/toolchain-pin-guard.sh` runs clean locally (`GUARD_EXIT=0`, 103 go.mod + 7 CI
sites) and its `--selftest` passes. The guard, `verify.sh` wiring, and the CI job are
all genuinely connected — confirmed by reading `tools/verify.sh:443-446` and the
`toolchain-pin-guard` job in `pr-validation.yml`, which is also present in the
`pr-validation-complete` `needs:` gate list. The implementer found and fixed a real
7th CI pin site (`.github/config/services.json`) that the plan's inventory missed,
and recorded it transparently. The one `.go` source change outside the plan's
allowance (`summon/builder.go` and 24 other files) is exactly the gofumpt
formatting sweep FR-5.3/AC-16(a) permits, with one staticcheck finding correctly
routed to a `.golangci.yml` exclusion + `docs/TODO.md` burn-down entry rather than a
silent behavioral rewrite. AC-13's original wording was replaced with a substitute
assertion because the underlying cideps fixture tests are pre-existing-red on `main`
(independently reproduced here via `git archive 855fef4d1`) — a scope-appropriate
deviation, documented, not silent. AC-17 (code review) is explicitly PENDING in
`acceptance.md`, matching the task instructions that this audit's verdict is the
open item, not a finding.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Rename pin file, make it source of truth | DONE | `tools/toolchain.versions` exists with all three values (`GO_VERSION=1.27.0`, `ALPINE_VERSION=3.24`, `GOLANGCI_LINT_VERSION=v2.13.1`); `tools/lint.sh:19-20,38`, `.claude/hooks/format-on-write.sh:28-29`, `.golangci.yml:12`, `.github/workflows/pr-validation.yml:500` all updated. `grep -rln "lint\.versions"` sweep shows matches only in `tools/toolchain.versions`'s own historical-note comment, `docs/tasks/task-171-*`, `docs/tasks/task-261-*`, and `.superpowers/sdd/plan/*` agent artifacts — zero live references. |
| 2 | Bump 103 module directives + workspace | DONE | Direct sweep: `go 1.27.0` × 103, `go 1.25.5` × 8 (exactly the `tools/cideps/testdata` fixtures). `go.work:1` = `go 1.27.0`. `grep -n '^toolchain '` over all 111 tracked go.mod files: zero matches. `git diff --stat 855fef4d1..HEAD -- '*go.sum'` is empty (no unexpected churn). `go.work.sum` shows only stale-checksum pruning (benign, matches design's documented allowance). |
| 3 | Measure/resolve lint + vet blast radius | DONE | `evidence-lint.md` present: golangci-lint v2.13.1 confirmed running; one real staticcheck QF1011 finding routed to `.golangci.yml:16-27` exclusion + `docs/TODO.md:44-51` burn-down entry (verified present in both files); 13-module gofumpt formatting sweep is the entire non-guard `.go` diff (`git diff --stat *.go` = 25 files, all mechanical blank-line/paren formatting, spot-checked at `summon/builder.go`). `go vet` reported 0 findings across 91 modules. |
| 4 | Container build pins | DONE | `Dockerfile:17-18` = `ARG GO_VERSION=1.27.0` / `ARG ALPINE_VERSION=3.24`; stage-local `ARG GO_VERSION` re-declared after `FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS build-env`; `Dockerfile`'s synthesized-workspace `printf 'go %s\n\nuse (\n' "${GO_VERSION}"` derives correctly. `services/atlas-kafka-precreate/Dockerfile:9-10` matches. `docker-bake.hcl:25-31` variable defaults at `1.27.0`/`3.24`. `services/atlas-renders/Dockerfile` confirmed deleted (`test -e` → DELETED). |
| 5 | CI workflow and composite action pins | DONE | All six plan-listed sites confirmed at `1.27.0`: `pr-validation.yml:41`, `main-publish.yml:33`, `catalog-lint.yml:17`, `packet-matrix.yml:13`, `go-test/action.yml:11`, `detect-changes/action.yml:280`. Indirections (`${{ env.GO_VERSION }}`, `${{ inputs.go-version }}`) untouched as required. |
| 6 | README prerequisites + GOTOOLCHAIN callout | DONE | `README.md:79` = `\| Go \| 1.27.0+ \| All backend services \|`; `GOTOOLCHAIN` callout block present at README.md:86-93 naming `tools/toolchain.versions` and the guard. Surrounding rows (Node/Docker/Kafka/PostgreSQL/Redis) unchanged. |
| 7 | Write `tools/toolchain-pin-guard.sh` | DONE | File exists, executable, pure bash, 355 lines. Header comment states the ~110-site rationale and the pre-guard partial-bump failure mode (design §4.1 requirement). `EXEMPT_GOMOD` array present with required rationale comment. Seven checked classes present: gomod, workspace, Dockerfile ARGs, bake vars, CI pins (now 7, see below), lint-pin presence check, README. Vacuous-pass protection present (`CHECKED_GOMOD -lt 1` → exit 2). `--selftest` present and independently re-run here: `SELFTEST_EXIT=0`. Live guard run here: `GUARD_EXIT=0`, "clean (103 go.mod + go.work + 4 Dockerfile ARGs + 2 bake vars + 7 CI pins + README checked)". `evidence-guard.md` contains the required Step 4/5/7 verbatim outputs plus AC-11 mutation/restore evidence. |
| 8 | Wire the guard into verify.sh, CI, docs | DONE | `tools/verify.sh:443-446` — path-gated `if touched '...'; then step "toolchain pin guard" ./tools/toolchain-pin-guard.sh; else skip ...; fi`, placed exactly where the plan specifies. `.github/workflows/pr-validation.yml:250-261` — `toolchain-pin-guard` job, no `setup-go` step (by design), gated on `deploy-only != 'true'`; also present in the `pr-validation-complete` job's `needs:` list at line 1368, so a red guard blocks the aggregate gate. `docs/verification.md:204-209` documents it in the Path-gated section. |
| 9 | Stop Renovate auto-merging Go bumps | DONE | `jq` sweep confirms 13 `packageRules` entries, last two are exactly `gomod/go/automerge=false` and `dockerfile/golang/automerge=false` — position-as-mechanism requirement met. `renovate-trace.md` present with the reasoning trace. |
| 10 | Full acceptance sweep and evidence | DONE | `acceptance.md` present with one section per AC-1..AC-17. AC-17 explicitly marked PENDING pending code review — correct per task instructions (this audit is that review input, not itself the AC-17 verdict). AC-13 substitutes a fixture-immutability assertion for the originally-planned "fixtures still pass their tests" claim, with the pre-existing-red condition on `main` independently reproduced in this audit (see Build & Test section) rather than merely asserted. `go.sum` diffstat empty; `.go` diffstat matches the Task 3 formatting sweep exactly. |

**Completion Rate:** 10/10 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. One item is worth flagging as a documented, reasoned deviation rather than a
gap: AC-13 as originally worded in the plan ("verify the excluded fixtures still
pass their own tests") is not achievable, because `tools/cideps`' 9 fixture-driven
unit tests are red on `main` independent of this branch — reproduced here directly:

```
$ git archive 855fef4d1 tools/cideps | tar -x -C /tmp/cideps-check
$ (cd /tmp/cideps-check/tools/cideps && GOWORK=off go test ./...)
--- FAIL: TestBuildGraph_Simple ... (identical 9 failures)
```

and identically on the branch tip:

```
$ (cd tools/cideps && go test ./...)
--- FAIL: TestBuildGraph_Simple ... (same 9 failures)
```

`acceptance.md` documents this and substitutes a fixture-byte-identity assertion,
which this audit independently confirms (`grep -m1 '^go ' tools/cideps/testdata/...`
= `go 1.25.5`, unchanged; `tools/cideps/graph_test.go:14` and
`tools/plan-context_test.sh:50` unchanged). This is a scope-appropriate deviation
that the plan's own Step 8 anticipated a burn-down escalation path for, is called
out rather than hidden, and does not implicate any task-261 source change (the
`realrepo_test.go` production-path test — not fixture-driven — passes).

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| Workspace root | N/A (expected) | — | `go build ./...` from repo root correctly fails with `directory prefix . does not contain modules listed in go.work` — this is documented pre-existing behavior (design §1.3/§5, context.md "Landmines"), not a task-261 regression; the repo root is not itself a module. |
| services/atlas-account/atlas.com/account | PASS | — | `go build ./...` from the module directory: exit 0, no output. |
| libs/atlas-retry | PASS | — | `go build ./...`: exit 0. Local toolchain confirmed `go1.27.0` via `go env GOVERSION`. |
| libs/atlas-packet | — | PASS | `go vet ./...`: no output, exit 0. |
| tools/cideps | PASS (build) | FAIL (9 pre-existing) | `go test ./...` fails identically at `855fef4d1` and at HEAD — pre-existing on `main`, not introduced by this branch (see above). Matches `acceptance.md`'s documented AC-13 substitution. |
| Flagless `tools/verify.sh` (full sweep, 91 modules + 67 docker bakes) | PASS | PASS | Not re-run in this audit (would require a full `-race` rebuild across 91 modules and 67 `docker buildx bake` service builds, explicitly out of scope for a code-review-time re-verification per the plan's own Step 3 note that this is "out of scope inside an implementer context"). Evidence log `.superpowers/sdd/plan/gate-final-flagless.log` inspected directly: 664 lines, ends in `All checks passed.`, contains a `✓ toolchain pin guard` line, `grep -cE '✓'` = 171, `grep -cE 'go build/vet/test -race'` = 91, `grep -cE 'docker buildx bake'` = 67 — matches `acceptance.md`'s AC-15 claims exactly. |
| `tools/toolchain-pin-guard.sh` | N/A (bash) | PASS | Re-run directly in this audit: clean run `GUARD_EXIT=0` ("clean (103 go.mod + go.work + 4 Dockerfile ARGs + 2 bake vars + 7 CI pins + README checked)"); `--selftest` → `SELFTEST_EXIT=0`. |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (pending AC-17 code review, which is this
  review itself and any prior/subsequent per-task reviews already referenced in
  `evidence-guard.md` — e.g. the Task 7 fix round addressing absent-key reporting
  and structural `go-test` scanning)

## Action Items

None required to close out the plan's own tasks. One informational note for the
code reviewer completing AC-17: the guard's `ci_check` deletion-case output for
`docker-bake.hcl`'s `ALPINE_VERSION` block renders an empty `file::` line/value
pair (`docker-bake.hcl:: expected default = "3.24", got`) rather than a clean
line number — `evidence-guard.md` §"Finding 3" already flags this as a known
cosmetic rendering quirk, not a functional defect (the guard still detects and
fails on the deletion). No action required unless the reviewer wants the
cosmetic output tightened.
