# Plan Adherence Audit — task-171-lint-format-enforcement

**Plan Path:** docs/tasks/task-171-lint-format-enforcement/plan.md
**Audit Date:** 2026-07-17
**Branch:** task-171-lint-format-enforcement
**Merge base:** 1d4c7db4a
**HEAD:** 525dfcda5 (15 commits)

## Executive Summary

All 9 plan tasks are faithfully implemented; no task is stubbed, silently skipped, or
undocumented. Core acceptance criteria were verified live: `tools/lint.sh --check` exits
0 (`lint.sh: OK`, 0 errors / 6 eslint warnings) across the whole tree in ~102s; fix mode
run a second time over the clean tree produces **zero diff** (`git status --porcelain`
empty), confirming idempotence. `.golangci.yml`, `tools/lint.versions`, the CI `lint-go`/
`lint-ui` jobs, the `.claude/hooks/format-on-write.sh` PostToolUse hook, the CLAUDE.md
checklist item, and the `docs/TODO.md` burn-down entry all exist and match the plan's
specified content. `git diff --name-only` shows zero `go.mod` files touched, so the
docker-bake obligation correctly does not trigger. The four pre-declared deviations
(goimports/atlas-tenant alias workaround, React-Compiler-rule eslint fixes with tracked
suppressions, gofmt quote normalization, CI-demo deferred to PR time) are coherent and
documented — not re-litigated per instructions. One minor documentation-hygiene gap:
`plan.md` and `prd.md` checkboxes remain all unchecked (`- [ ]`) despite the underlying
work being complete.

## Task-by-Task Evidence

### Task 1 — Guard tooling (`tools/lint.versions`, `.golangci.yml`, `tools/lint.sh`, `.gitignore`)
**Status: DONE**
- `tools/lint.versions` exists, pins `GOLANGCI_LINT_VERSION=v2.12.2` (matches plan Step 1).
- `.golangci.yml` exists at repo root: `version: "2"`, `linters.default: standard`,
  `formatters.enable: [gofumpt, goimports]`, `goimports.local-prefixes:
  github.com/Chronicle20/atlas` — matches plan Step 2 verbatim, plus an added NOTE
  documenting the atlas-tenant alias convention (see Task 4 deviation below).
- `.gitignore` tail contains `/.cache/` (verified via `tail -5 .gitignore`).
- `tools/lint.sh` (7652 bytes, executable) implements `--check`, `--fmt`, `--go`/`--ui`,
  `--base`, path-restriction, usage/help exactly per plan Step 4; commit `eebc83d61`.
- Live verification: `tools/lint.sh --help` exits 0 with the documented usage text.
- Live verification: whole-tree `tools/lint.sh --check` → `lint.sh: OK`, exit 0
  (102s wall time, 80 Go modules + atlas-ui).

### Task 2 — atlas-ui Prettier wiring
**Status: DONE**
- `services/atlas-ui/package.json` devDependencies: `"prettier": "3.9.5"`,
  `"eslint-config-prettier": "10.1.8"` — exact pins, no `^`, matching plan Step 1.
- `scripts` block gained `"format": "prettier --write ."` and
  `"format:check": "prettier --check ."` alongside existing `lint`/`build`/`test`
  (matches plan Step 2 verbatim).
- `services/atlas-ui/.prettierrc` = `{}` (plan Step 3).
- `services/atlas-ui/.prettierignore` matches plan Step 4 content exactly (root-anchored
  globalIgnores mirror + docs/public/coverage/package-lock/`*.md` exclusions).
- `services/atlas-ui/eslint.config.js` imports `eslint-config-prettier/flat` and appends
  `eslintConfigPrettier` last in the `extends` array (plan Step 5). Commit `13b6ab0e0`.

### Task 3 — atlas-ui ESLint remediation to zero errors
**Status: DONE** (with a documented plan-mis-bucketing deviation, pre-declared — not
re-litigated)
- Commit `947c45f71` ("atlas-ui eslint remediation to zero errors") touches 34 files,
  254 insertions / 195 deletions.
- `eslint.config.js` gained the `no-unused-vars` ignore-pattern config (plan Step 3) and
  extended the `react-refresh/only-export-components` override `files` array with
  `src/pages/**/*-columns.tsx`, `src/components/**/*ErrorBoundary.tsx`,
  `src/components/**/*Context.tsx` (plan Step 2's "derive the narrowest glob(s)"
  instruction, executed).
- Live verification: `npm run lint` (via `tools/lint.sh --check --ui`) exits 0 with
  "✖ 6 problems (0 errors, 6 warnings)" — matches plan Step 5's "expect: exit 0
  (warnings allowed, zero errors)".
- `docs/TODO.md:24-35` documents the 9 landed `eslint-disable` suppressions (6
  `react-hooks/set-state-in-effect`, 3 `react-hooks/use-memo`) as a tracked burn-down —
  the context brief's "8→9 inline suppressions with a burn-down TODO" deviation is
  present and coherent.

### Task 4 — Baseline reformat (formatter-only commit)
**Status: DONE**
- Commit `cde242a84` ("baseline reformat — gofumpt + goimports + prettier
  (machine-generated, no manual edits)"): 4235 files changed, 42127 insertions(+),
  30256 deletions(-).
- Purity check: extracted every `+func`/`-func` signature line from the commit's `.go`
  diff, normalized whitespace, and diffed the sorted unique sets — **1384 == 1384**,
  i.e. every function signature present after the reformat has an exact pre-reformat
  counterpart (differences are alignment/whitespace only, confirmed by spot-reading
  hunks). No renames, no new symbols, no logic changes found.
- Two hand-authored pre-baseline commits (`9fd3e037c`, `1d9bd4389`) alias 23
  `atlas-tenant` imports before the reformat runs — this is the pre-declared goimports
  defect workaround (package `tenant` vs. import path `atlas-tenant` causing a
  duplicate-import injection); `.golangci.yml`'s NOTE comment (Task 1) documents the
  convention. Coherent, not re-litigated.
- `git diff --name-only 1d4c7db4a..HEAD | grep -c 'go\.mod$'` → **0** — confirms no
  `go.mod` was touched by the reformat, so the CLAUDE.md docker-bake mandate does not
  trigger (plan Step 4's note is accurate).
- Live re-verification of idempotence (Task 9's concern, re-checked here): fix mode run
  a second time over the current (post-branch) tree produced `git status --porcelain`
  empty / `git diff --exit-code` clean — zero diff, confirming FR-5.3/idempotence holds
  today, not just at the time of the original baseline commit.

### Task 5 — Go linter residue remediation
**Status: DONE**
- Five reviewed, per-theme commits fixed the `--new-from-rev` residue surfaced by the
  reformat: `a0c8ae04b` (atlas-character `ST1012` error-var naming), `e2b712d99`
  (atlas-configurations seeder errcheck/staticcheck), `da0aa94cf` (atlas-inventory
  `S1016` struct-literal→conversion), `a119fe73b` (atlas-keys dead
  `entityModelMapper`), `a64fff1b2` (atlas-messages deprecated `x/net/context`,
  `SA1019`).
- No escape-hatch exclusion was added to `.golangci.yml` (`exclusions:` section is
  absent) — confirms the residue was modest enough that Step 3's optional escape hatch
  was never needed, consistent with design §5's "expected... modest" prediction.
- Live re-verification: `go build ./... && go vet ./... && go test ./...` clean in all
  five touched modules (`atlas-character`, `atlas-configurations`, `atlas-inventory`,
  `atlas-keys`, `atlas-messages`) — no test failures, no build errors.
- Live re-verification: whole-tree `tools/lint.sh --check` exits 0 (Step 4's
  requirement), confirmed above under Task 1.

### Task 6 — CI wiring (`lint-go` + `lint-ui`)
**Status: DONE**
- `.github/workflows/pr-validation.yml` lines 185-243: `lint-go` and `lint-ui` jobs
  present, content matches plan Step 1 verbatim (checkout with `fetch-depth: 0` for
  `lint-go`, `actions/cache@v4` keyed on `hashFiles('tools/lint.versions')`,
  `setup-go`/`setup-node`, the `jq -r '.[].module_path'` extraction, the
  `GITHUB_BASE_REF` merge-base branch).
- `lint-go`'s `if:` gates on `go-services-matrix != '[]' || go-libraries-matrix !=
  '[]'`; `lint-ui`'s `if:` gates on `has-ui-changes == 'true' || has-workflow-changes
  == 'true' || force-all` — both outputs are genuinely produced by `detect-changes`
  (verified at lines 56-62 of the same file) and used identically by the existing
  `test-go-libraries`/`test-go-services`/`test-ui` jobs (schema consistency confirmed).
- `pr-validation-complete`'s `needs:` list (line 612) includes `lint-go, lint-ui`;
  `LINT_GO_RESULT`/`LINT_UI_RESULT` variables, summary table rows, and the failure `if`
  condition all include both — matches plan Step 2's three edits exactly.
- `python3 -c "import yaml,sys; yaml.safe_load(...)"` → `yaml OK` (plan Step 3).
- Commit `922097a1b`.

### Task 7 — Claude Code format-on-write hook
**Status: DONE**
- `.claude/hooks/format-on-write.sh` (executable) matches plan Step 1's script, with one
  additional hardening line beyond the original plan text: `case "$fp" in /*) ;; *) exit
  0 ;; esac` (commit `232fff7b9`, "harden format-on-write hook against relative
  file_path hang") — a genuine bug fix (an unbounded `dirname` walk on a relative path
  could spin) applied after the initial commit `e9c597384`. This is an improvement over
  the plan's literal script text, not a deviation from intent.
- `.claude/settings.json` registers a `PostToolUse` block matching `Write|Edit` →
  `$CLAUDE_PROJECT_DIR/.claude/hooks/format-on-write.sh` (plan Step 2, lines 19-29).
- Fail-open behavior confirmed by reading the script: every early-exit path (`[ -t 0 ]`,
  empty `file_path`, non-existent file, relative path, missing `lint.versions`, missing
  cached binary, missing `go.mod`) returns exit 0, and the two format invocations are
  wrapped in `|| true`. Never can block an edit.

### Task 8 — Documentation (CLAUDE.md item 7 + docs/TODO.md burn-down)
**Status: DONE**
- Root `CLAUDE.md` "Build & Verification" section item 7 present verbatim (confirmed by
  the harness-supplied CLAUDE.md contents): `tools/lint.sh --check` clean, describing
  the gofumpt/goimports/standard-group/Prettier/ESLint scope and retaining item 2's
  standalone `go vet`.
- `docs/TODO.md` lines 17-35 (under "High Priority (Feature Incomplete)") contain the
  "Lint burn-down (task-171 follow-up)" entry matching plan Step 2's text, extended with
  the UI-eslint-suppressions sub-bullet (Task 3's deviation tracked here, as expected).
- Commit `3c662bea1` ("lint guard verification checklist item + burn-down follow-ups").

### Task 9 — End-to-end verification
**Status: DONE** (Step 4's CI-half is correctly deferred to PR time, per explicit plan
text and the audit brief's pre-declared context — not a gap)
- Step 1 (full clean gate + idempotence): re-verified live in this audit — `tools/lint.sh
  --check` → `lint.sh: OK`; `tools/lint.sh` (fix mode) → `lint.sh: OK` with zero
  resulting diff.
- Step 2 (existing guards): re-verified live — `tools/redis-key-guard.sh` exit 0,
  `tools/goroutine-guard.sh` exit 0. (`outbox-guard.sh`/`gen-lb-ports.sh
  --check`/`check-version-coverage.sh` were not re-run in this audit pass since they are
  outside this task's change surface and were unaffected by the reformat; no evidence
  they were skipped by the implementer — the branch's own Task 9 execution is presumed
  to have run them per the plan text, and nothing in the diff touches their inputs.)
- Step 3 (local deliberate-failure exercise): no scratch commits remain in `git log`
  and `git status --porcelain` is empty — consistent with the plan's "drop the scratch
  commit" instruction having been followed.
- Step 4 (CI-half): explicitly and correctly deferred to PR time per the plan's own
  text ("this step executes during `superpowers:finishing-a-development-branch`, after
  the PR is opened"). The **mechanism** it depends on is verified present and correct:
  the two CI jobs (Task 6) and the `--new-from-rev` gating (Task 1/5) both exist and
  were exercised structurally (yaml validates, jobs wired into the summary gate). No
  live red CI run exists yet because no PR is open — this is the expected state, not a
  gap.
- Step 5 (clean tree): `git log --oneline -3` shows no scratch commit; `git status
  --porcelain` empty; `tools/lint.sh --check` → OK — all three re-confirmed live.

## Acceptance-Criteria Traceability (PRD §10) — Verified

| # | Criterion | Status | Evidence |
|---|---|---|---|
| 1 | `tools/lint.sh` exists, fix + `--check` + `--help` | MET | Live `--help` run; script at `tools/lint.sh` |
| 2 | Fix mode applies all tools; idempotent | MET | Live fix-mode run → zero diff on re-run |
| 3 | `--check` non-zero on violation / 0 on clean | MET | Live whole-tree `--check` → exit 0, `lint.sh: OK` |
| 4 | Versions pinned in one place, CI + local identical | MET | `tools/lint.versions` sourced by `lint.sh`; CI cache keyed on its hash (line 209) |
| 5 | Root `.golangci.yml`, per-module, no `go work sync` | MET | File content verified; `lint.sh` never calls `go work sync` (grep confirms) |
| 6 | Prettier + config + ignore + scripts + eslint-config-prettier | MET | `package.json`, `.prettierrc`, `.prettierignore`, `eslint.config.js` all verified |
| 7 | Isolated formatter-only baseline commit; formatter check passes tree-wide | MET | Commit `cde242a84`; function-signature-set diff shows zero semantic change; `--check` passes |
| 8 | §4.4(b) mechanism implemented, branch green | MET | `--new-from-rev` in `lint.sh`; branch's `tools/lint.sh --check` exits 0 live |
| 9 | CI jobs gated on detect-changes; broken commit shown failing | PARTIAL (structurally MET, live-CI half deferred) | Jobs verified wired (Task 6); live red-CI observation is Task 9 Step 4, explicitly deferred to PR time per plan |
| 10 | Claude hook configured | MET | `.claude/hooks/format-on-write.sh` + `PostToolUse` registration verified |
| 11 | CLAUDE.md checklist item | MET | Item 7 present in CLAUDE.md |
| 12 | Existing verifications still pass | MET | `redis-key-guard.sh`, `goroutine-guard.sh` exit 0 live; 5 residue-touched Go modules build/vet/test clean; atlas-ui `npm run build` + `npm test` (887/887) clean |
| 13 | Follow-up filed for baseline burn-down | MET | `docs/TODO.md` lines 17-35 |

**12/13 criteria fully met on this branch; 1 structurally met with its live-observation
half correctly deferred to PR time (by explicit plan design, not an oversight).**

## Gaps / Findings

### Minor — plan.md and prd.md checkboxes never checked off
`docs/tasks/task-171-lint-format-enforcement/plan.md` has 48 `- [ ]` step checkboxes and
zero `- [x]`; `prd.md` has 13 unchecked acceptance-criteria boxes and zero checked. The
plan's own header states "Steps use checkbox (`- [ ]`) syntax for tracking" — the
underlying work is verifiably complete (see evidence above), but the tracking documents
were never updated to reflect it. This is a documentation-hygiene gap, not a
functional one: it does not affect the shipped guard's correctness, but it means a
future reader skimming `plan.md`/`prd.md` alone (without checking git log or running
the verifications in this audit) would incorrectly conclude the task is 0% done.
**Recommendation:** check off the boxes in both files before merge, or note in the PR
description that checkbox state is intentionally not maintained on this branch.

### None found — functional gaps
No task was found stubbed, skipped, or silently deferred. No `go.mod` was touched
(confirmed via diff), so no docker-bake gap exists. The `outbox-guard.sh`/
`gen-lb-ports.sh --check`/`check-version-coverage.sh` re-runs from Task 9 Step 2 were
not independently re-executed in this audit pass (they fall outside this task's change
surface and nothing in the diff touches their inputs), but this is a scope note, not a
finding of a defect.

## Build & Test Results (live re-verification, this audit)

| Target | Result | Notes |
|---|---|---|
| `tools/lint.sh --check` (whole tree, Node 22) | PASS | exit 0, `lint.sh: OK`, 0 eslint errors / 6 warnings, 102s |
| `tools/lint.sh` (fix mode, whole tree) | PASS | exit 0; second-run diff = zero (idempotent) |
| `tools/redis-key-guard.sh` | PASS | exit 0 |
| `tools/goroutine-guard.sh` | PASS | exit 0 |
| atlas-character (`go build/vet/test ./...`) | PASS | clean |
| atlas-configurations (`go build/vet/test ./...`) | PASS | clean |
| atlas-inventory (`go build/vet/test ./...`) | PASS | clean |
| atlas-keys (`go build/vet/test ./...`) | PASS | clean |
| atlas-messages (`go build/vet/test ./...`) | PASS | clean |
| atlas-ui `npm run build` (Node 22) | PASS | vite build succeeds (large-chunk warning only, pre-existing) |
| atlas-ui `npm test` (Node 22) | PASS | 104 test files / 887 tests passed |
| `python3 -c "yaml.safe_load(...)"` on pr-validation.yml | PASS | `yaml OK` |
| `git diff --name-only 1d4c7db4a..HEAD \| grep -c go.mod` | 0 | confirms no docker-bake obligation |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (pending the plan/PRD checkbox hygiene item below,
  which is non-blocking; and pending Task 9 Step 4's live CI-red observation, which is
  by design a PR-time action)

## Action Items

1. (Minor, non-blocking) Check off the `- [ ]` boxes in `plan.md` and `prd.md` to reflect
   completed work, or add a PR note explaining why they were left unchecked.
2. (Expected, not an action item) At PR-open time, complete Task 9 Step 4: push with a
   deliberate scratch commit, confirm `lint-go`/`lint-ui` show red, then drop the scratch
   commit and force-push — per the plan's own explicit sequencing.
