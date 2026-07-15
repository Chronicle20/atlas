# Lint & Format Enforcement Tooling — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-15
---

## 1. Overview

Atlas has no automated formatting or lint enforcement. The only static gates today are
`go vet` (run manually per CLAUDE.md's verification checklist), `tools/redis-key-guard.sh`,
and `tools/gen-lb-ports.sh --check`. Go formatting (`gofmt`) is assumed but never verified;
golangci-lint has never been run (no `.golangci.yml` exists anywhere in the tree). The
frontend has ESLint (`services/atlas-ui/eslint.config.js`, `npm run lint`) and `tsc -b`
type-checking, but no Prettier and no CI gate that runs either on a PR.

The result is stylistic drift that surfaces as noise in code review and as inconsistent
output from agentic contributors (Claude and others), who currently have no deterministic
"format before you commit" step. Nothing catches an unformatted file until a human notices.

This task introduces a **single shared guard** — `tools/lint.sh` — with two modes:

- **fix mode** (default): rewrites files in place. Run by agents/humans before committing.
- **`--check` mode**: mutates nothing, exits non-zero if anything is unformatted or fails a
  lint rule. Run by CI.

Both modes invoke the **same tools with the same pinned versions and the same config files**,
so local and CI never disagree. The guard is wired into three enforcement points:

1. **Local (automatic):** a Claude Code hook that formats files as they are written / before
   the branch is finished.
2. **Local (documented):** a new line in the root `CLAUDE.md` Build & Verification checklist.
3. **CI (backstop):** a new `Lint & Format Guard` job in `.github/workflows/pr-validation.yml`,
   gated on the existing `detect-changes` matrices so it only runs against changed modules.

This is a direct extension of the repo's established guard pattern (`redis-key-guard`,
`gen-lb-ports --check`): one `tools/` script that is the single source of truth, invoked both
locally and as a CI job that mirrors it in `--check` mode.

Covered ecosystems:

- **Go** (80 `go.mod` modules across `services/` and `libs/`): gofumpt (formatting) +
  goimports (import grouping) + golangci-lint (broad default linter set).
- **atlas-ui** (TypeScript/React): Prettier (formatting, new) + ESLint (existing) +
  `tsc --noEmit` (type-check).

A **one-time baseline reformat** of the entire tree ships as part of this task so that CI's
`--check` can enforce the whole tree and future PRs only fail on drift they themselves introduce.

## 2. Goals

Primary goals:

- Provide `tools/lint.sh` with fix and `--check` modes covering both Go and atlas-ui, using
  pinned tool versions and checked-in config as the single source of truth.
- Enforce Go formatting via **gofumpt** + **goimports** across all 80 modules.
- Enforce Go linting via **golangci-lint** with the broad default linter set.
- Add **Prettier** to atlas-ui and enforce it alongside the existing ESLint + `tsc`.
- Wire local enforcement for agents: a Claude Code hook plus a CLAUDE.md verification-checklist line.
- Add a CI `Lint & Format Guard` job to `pr-validation.yml`, gated on `detect-changes`, that
  runs the guard in `--check` mode and fails the PR on any violation.
- Land a one-time baseline reformat so the tree is green under the new gate on day one.
- Define an explicit, honest mechanism for how CI reaches green given a broad linter set on a
  never-linted tree (see §4.4 and §9 — this is the central risk of the chosen scope).

Non-goals:

- Refactoring service logic or fixing behavioral bugs the linters surface beyond what is
  auto-fixable or trivially mechanical.
- IDE/editor on-save configuration (`.editorconfig`, VS Code settings) — may be a follow-up.
- Adding new custom linters or analyzers beyond golangci-lint's defaults (the repo's bespoke
  analyzers — rediskeyguard, etc. — remain separate guards, not folded into this one).
- Enforcing lint/format on generated code, vendored code, or `docs/` content.
- Pre-commit-framework adoption (e.g. `pre-commit` / husky) — the Claude Code hook + CLAUDE.md
  is the local mechanism; a git pre-commit hook is optional and called out as an open question.

## 3. User Stories

- As an **agentic contributor (Claude)**, I want a single command to format and lint my changes
  before committing, so that my PRs don't fail CI on mechanical style issues.
- As a **human maintainer**, I want CI to reject any PR containing unformatted or lint-failing
  code, so that style is never a review comment again.
- As a **reviewer**, I want diffs free of formatting noise, so that review focuses on logic.
- As a **new contributor**, I want the formatting/lint rules encoded in checked-in config and one
  script, so that I can reproduce CI's verdict locally without guessing tool versions.
- As a **maintainer**, I want the CI job to only run against changed modules, so that PR feedback
  stays fast on an 80-module monorepo.

## 4. Functional Requirements

### 4.1 The shared guard — `tools/lint.sh`

- FR-1.1 A new executable `tools/lint.sh` is the single entry point for both local and CI use.
- FR-1.2 Default (no `--check`) runs **fix mode**: applies gofumpt `-w`, goimports `-w`,
  `golangci-lint run --fix`, `prettier --write`, and `eslint --fix` in place. Exit 0 unless a
  tool errors internally.
- FR-1.3 `tools/lint.sh --check` runs **check mode**: `gofmt`/`gofumpt -l` (list, non-empty →
  fail), `golangci-lint run` (no `--fix`), `prettier --check`, `eslint` (no `--fix`),
  `tsc --noEmit`. Any violation → non-zero exit with a readable summary of offending files.
- FR-1.4 Scope selection: the script accepts an optional set of target modules/paths. With no
  targets it operates on the whole tree (used by the baseline reformat and by whole-tree local
  runs); CI passes only the changed modules (see §4.4). Support at minimum:
  `tools/lint.sh [--check] [--go|--ui] [path ...]`.
- FR-1.5 The script must run correctly with `GOWORK=off` semantics where the existing guards do
  (per repo memory on workspace/guard footguns) — i.e. it must not depend on the go workspace
  being synced, and must iterate per-module the way `tools/test-all-go.sh` does
  (`find ./services ./libs -name go.mod`).
- FR-1.6 Tool versions are pinned in one place the script reads (e.g. a `tools/lint.versions`
  file or constants at the top of the script) and the same versions are referenced by CI, so
  local and CI never run different tool versions. gofumpt, goimports, golangci-lint, prettier,
  and the Node/Go toolchain versions are all pinned.
- FR-1.7 The script installs/bootstraps missing Go tools deterministically (e.g. via
  `go run <tool>@<pinned>` or a documented `go install`), so a fresh checkout can run it without
  manual setup. It must degrade gracefully with a clear error if a required toolchain
  (Go, Node) is absent.

### 4.2 Go formatting & linting

- FR-2.1 **gofumpt** is the formatter (stricter superset of gofmt). Formatting is enforced
  tree-wide (all 80 modules) in both fix and check modes.
- FR-2.2 **goimports** enforces import grouping/ordering with the local-prefix set to the module
  path family (`github.com/Chronicle20/atlas/...`) so intra-repo imports group correctly.
- FR-2.3 A root **`.golangci.yml`** is added, enabling golangci-lint's **broad default linter
  set**. The config is the single source of truth for enabled linters, exclusions, and
  per-path skips (generated code, test fixtures).
- FR-2.4 golangci-lint runs per-module (each `go.mod` is its own root); the script handles the
  fan-out. Formatting linters inside golangci-lint (gofumpt/goimports integration) must not
  conflict with the standalone gofumpt/goimports invocation — pick one authority and document it
  (recommended: let golangci-lint own gofumpt+goimports via its `formatters`/`gofumpt` settings,
  and have fix mode call `golangci-lint fmt`/`--fix`, to avoid double-tooling). Final decision
  recorded in design phase.
- FR-2.5 The existing `go vet` obligation is subsumed by golangci-lint's `govet` linter; the
  CLAUDE.md checklist is updated so `go vet` is not run twice (or is explicitly retained — design
  decision).

### 4.3 atlas-ui formatting, linting & type-check

- FR-3.1 **Prettier** is added as a dev dependency to `services/atlas-ui` with a checked-in
  config (`.prettierrc` / `prettier.config.js`) and `.prettierignore`.
- FR-3.2 New `package.json` scripts: `format` (prettier --write), `format:check`
  (prettier --check). Existing `lint` (eslint) and `build`/`tsc` remain.
- FR-3.3 The guard's UI check runs prettier `--check` + `eslint` + `tsc --noEmit`. Fix mode runs
  prettier `--write` + `eslint --fix`.
- FR-3.4 ESLint and Prettier must not fight over the same rules — ESLint's formatting-related
  rules are disabled where Prettier owns them (e.g. via `eslint-config-prettier`) so the two
  tools are complementary, not contradictory.
- FR-3.5 The atlas-ui toolchain requires nvm/Node 22 (per repo memory); the script and CI must
  select the pinned Node version before running UI checks.

### 4.4 CI enforcement

- FR-4.1 A new job `lint-format` (display name "Lint & Format Guard") is added to
  `.github/workflows/pr-validation.yml`, structured like the existing `redis-key-guard` /
  `gen-lb-ports` guard jobs.
- FR-4.2 The job is **gated on `detect-changes`**: it consumes the existing outputs
  (`go-services-matrix`, `go-libraries-matrix`, `has-ui-changes`) and only lints changed
  modules. Split into per-ecosystem jobs if that maps more cleanly onto the matrices (e.g.
  `lint-go` fanned across the module matrix + a single `lint-ui` gated on `has-ui-changes`).
- FR-4.3 The CI job runs `tools/lint.sh --check` (or the composite action wrapping it) against
  the changed targets. Non-zero exit fails the PR.
- FR-4.4 **How CI reaches green (central mechanism).** Formatters (gofumpt/goimports/Prettier)
  are fully auto-fixable, so the §4.5 baseline reformat clears them tree-wide. golangci-lint's
  broad set will surface **non-auto-fixable** findings (errcheck, staticcheck, etc.) that a
  reformat cannot clear. The PRD requires exactly one of the following to be chosen in the
  design phase, and the chosen one implemented so CI is green on merge:
  - **(a) Remediate-to-green:** fix every finding as part of this task. Honest cost: potentially
    large and unbounded on a never-linted 80-module tree; risks scope blow-up.
  - **(b) Baseline-suppress:** commit a golangci-lint issue baseline (e.g. `--new-from-rev=<merge-base>`
    diff mode for the linter layer, or a generated `//nolint`-free exclusion baseline) so only
    **newly introduced** lint findings fail CI, while formatters are enforced whole-tree. A
    tracked follow-up task burns down the baseline. **Recommended** — it honors "broad set +
    reformat whole tree now" (broad set is enabled and catches all new issues; tree is formatted)
    without an unbounded remediation sprint.
  - This choice is the single most important open decision (see §9).
- FR-4.5 Consider a composite action `.github/actions/lint` mirroring the `go-test` / `node-test`
  action pattern, so the job body stays thin and reusable. Optional but recommended.
- FR-4.6 The CI job must fail closed: if the guard script errors or a tool is missing, the job
  fails rather than passing silently.

### 4.5 Baseline reformat

- FR-5.1 A one-time mechanical reformat of the entire tree (all Go modules via gofumpt+goimports,
  all atlas-ui TS/TSX via Prettier) lands as part of this task, ideally as an isolated commit
  separate from the tooling commits so reviewers can trust it is machine-generated.
- FR-5.2 The reformat commit must contain **only** formatter output — no manual edits, no logic
  changes. It is produced by running `tools/lint.sh` fix mode (formatters only) and committing.
- FR-5.3 After the reformat, `tools/lint.sh --check` (formatter portion) must pass on the whole
  tree with zero diffs.
- FR-5.4 The reformat must not touch generated code, vendored trees, or paths excluded by config.

### 4.6 Local enforcement (agentic + human)

- FR-6.1 A Claude Code hook is added to `.claude/settings.json` that runs the formatter. Options
  (design decision): a `PostToolUse` hook on `Edit`/`Write` that formats the just-written file,
  and/or a `Stop`/pre-finish hook that runs `tools/lint.sh --check` and surfaces failures. At
  minimum, agents must have an automatic, harness-enforced formatting reflex that does not rely
  on the model remembering.
- FR-6.2 The root `CLAUDE.md` "Build & Verification" section gains a numbered item requiring
  `tools/lint.sh --check` clean before a branch is claimed done, alongside the existing
  test/vet/bake/redis-key-guard steps.
- FR-6.3 (Optional, open question) an installable git `pre-commit` hook script under `tools/` that
  humans can opt into, mirroring the Claude hook for non-agent contributors.

## 5. API Surface

No service HTTP/REST endpoints are added or modified. The "interface" surface of this task is:

- **CLI:** `tools/lint.sh [--check] [--go|--ui] [path ...]` — documented in `--help` and in the
  script header.
- **npm scripts** (atlas-ui `package.json`): `format`, `format:check` (new); `lint`, `build`,
  `test` (existing, unchanged).
- **CI job contract:** `lint-format` (or `lint-go` + `lint-ui`) in `pr-validation.yml`, consuming
  `detect-changes` outputs, emitting standard job pass/fail status on the PR.
- **Config files (new, checked in):** root `.golangci.yml`; `services/atlas-ui/.prettierrc`
  (+ `.prettierignore`); tool-version pin file; optional `.github/actions/lint/action.yml`.

## 6. Data Model

Not applicable — this task introduces no entities, database tables, or migrations. No
`tenant_id`-scoped data is involved. State lives entirely in checked-in config and CI workflow
definitions.

## 7. Service Impact

| Area | Change |
|------|--------|
| `tools/` | New `tools/lint.sh` + version-pin file (+ optional pre-commit hook script). |
| Root `.golangci.yml` | New — golangci-lint broad-default config, exclusions, per-path skips. |
| `.github/workflows/pr-validation.yml` | New `lint-format` (or `lint-go`+`lint-ui`) job(s) gated on `detect-changes`. |
| `.github/actions/lint/` | Optional new composite action wrapping the guard. |
| Root `CLAUDE.md` | New verification-checklist item; possible `go vet` de-duplication. |
| `.claude/settings.json` | New formatting hook (PostToolUse and/or Stop). |
| `services/atlas-ui` | New Prettier dep + config + `.prettierignore` + package.json scripts; `eslint-config-prettier` wiring. |
| **All 80 Go modules** | Baseline reformat (gofumpt + goimports) — mechanical whitespace/import diffs. |
| **atlas-ui `src/`** | Baseline Prettier reformat — mechanical TS/TSX diffs. |
| golangci-lint remediation | Depends on §4.4 decision: either fixes across many modules (option a) or a baseline artifact (option b). |

No runtime service behavior changes. No packet, WZ, tenant-config, or Kafka surface is touched.

## 8. Non-Functional Requirements

- **Determinism:** local and CI must produce identical verdicts. Guaranteed by pinned tool
  versions (FR-1.6) and shared config. A version drift between local and CI is a defect.
- **Performance:** CI feedback must stay fast on an 80-module repo — hence the `detect-changes`
  gating (FR-4.2). Whole-tree runs (baseline, opt-in local) may be slow; that is acceptable.
- **Idempotence:** running fix mode twice produces no diff on the second run.
- **Fail-closed CI:** missing tools or script errors fail the job (FR-4.6), never a silent pass —
  consistent with the repo's "no false verified" discipline.
- **No workspace coupling:** the guard must not require `go work sync` and must run per-module
  (FR-1.5), per the repo's documented workspace/guard footguns.
- **Reviewability:** the baseline reformat is isolated to formatter-only commits (FR-5.1/5.2) so
  it can be trusted without line-by-line review.
- **Multi-tenancy / security / observability:** not applicable — no runtime, no data, no tenant
  context.

## 9. Open Questions

1. **CI-green mechanism for the broad linter set (FR-4.4) — highest priority.** Remediate-to-green
   (a) vs baseline-suppress (b). Recommendation: **(b)** — enable the broad set, enforce formatters
   tree-wide, gate linter *findings* to new code via a baseline, and file a follow-up to burn the
   baseline down. Needs explicit sign-off in design because it determines task size.
2. **gofumpt/goimports ownership vs golangci-lint (FR-2.4):** run them standalone, or let
   golangci-lint's formatter integration own them? Recommendation: single authority via
   golangci-lint `formatters` to avoid double-tooling and version skew.
3. **Local hook shape (FR-6.1):** PostToolUse format-on-write, Stop-time `--check`, or both?
   Trade-off: format-on-write is invisible and frictionless but can surprise; Stop-time check is
   explicit but relies on the run reaching Stop.
4. **`go vet` de-duplication (FR-2.5):** drop the standalone `go vet` checklist item now that
   `govet` runs under golangci-lint, or keep both?
5. **golangci-lint default set exact membership:** pin to a specific golangci-lint version and
   snapshot its default enabled linters, since "default set" changes across releases. Which
   version?
6. **Optional git pre-commit hook (FR-6.3):** ship an opt-in installer for human contributors, or
   rely solely on the Claude hook + CI?
7. **atlas-ui `tsc` in the guard:** is `tsc --noEmit` part of `lint.sh --check`, or left to the
   existing `build` job? (It overlaps with the build's `tsc -b`.)

## 10. Acceptance Criteria

- [ ] `tools/lint.sh` exists, is executable, supports fix (default) and `--check` modes, and
      `--help` documents usage.
- [ ] Fix mode applies gofumpt + goimports + golangci-lint `--fix` + Prettier + eslint `--fix`;
      running it twice is a no-op (idempotent).
- [ ] `--check` mode exits non-zero with a readable file list on any formatting or lint violation,
      and exits 0 on a clean tree.
- [ ] Tool versions are pinned in one place read by both the script and CI; local and CI run
      identical versions.
- [ ] Root `.golangci.yml` enables the broad default set and is the single config source; runs
      per-module across all 80 modules without requiring `go work sync`.
- [ ] atlas-ui has Prettier + config + `.prettierignore` + `format`/`format:check` scripts, with
      `eslint-config-prettier` preventing rule conflicts.
- [ ] The whole tree is reformatted in an isolated, formatter-only baseline commit; the formatter
      portion of `--check` passes tree-wide with zero diffs afterward.
- [ ] The chosen §4.4 CI-green mechanism is implemented such that `pr-validation.yml`'s new
      guard job passes on this branch.
- [ ] `pr-validation.yml` has a `lint-format` (or `lint-go` + `lint-ui`) job gated on
      `detect-changes` that runs the guard in `--check` mode and fails a PR on violations; a
      deliberately-broken test commit is shown failing the job.
- [ ] A Claude Code formatting hook is configured in `.claude/settings.json`.
- [ ] Root `CLAUDE.md` Build & Verification checklist includes the `tools/lint.sh --check` step.
- [ ] All existing verifications still pass: `go test -race ./...`, `go build ./...`, affected
      `docker buildx bake` targets, `tools/redis-key-guard.sh`, and the atlas-ui `build` + `test`.
- [ ] A follow-up task is filed if §4.4 option (b) is chosen (baseline burn-down).
