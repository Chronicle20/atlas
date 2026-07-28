# Lint & Format Enforcement Tooling — Design

Task: task-171-lint-format-enforcement
Status: Proposed
Created: 2026-07-15
PRD: docs/tasks/task-171-lint-format-enforcement/prd.md

---

## 1. Summary

One new guard script — `tools/lint.sh` — following the established guard pattern
(`redis-key-guard.sh`, `goroutine-guard.sh`, `gen-lb-ports.sh --check`): a single
`tools/` script that is the source of truth, run in fix mode locally and `--check`
mode in CI. golangci-lint v2 is the **single authority** for all Go formatting
(gofumpt + goimports via its `formatters` config) and linting (its `standard`
default linter group). atlas-ui gains Prettier alongside the existing ESLint.
Formatting is enforced **tree-wide** (cleared by a one-time baseline reformat);
lint *findings* are gated to **new code** via `--new-from-rev` diff mode, with a
follow-up burn-down task.

## 2. Decisions on PRD Open Questions

| # | Question (PRD §9) | Decision | Rationale |
|---|---|---|---|
| 1 | CI-green mechanism (FR-4.4) | **(b) Baseline-suppress** via `golangci-lint run --new-from-rev <merge-base>` | Enables the broad set on all new code without an unbounded remediation sprint. Formatters are still enforced whole-tree. See §5 for the reformat-PR interaction and mitigation. |
| 2 | gofumpt/goimports ownership (FR-2.4) | **golangci-lint owns both** via its v2 `formatters` section; fix = `golangci-lint fmt`, check = `golangci-lint fmt --diff` | One tool to pin, zero version-skew between the formatter and the `run` layer's format checks. gofumpt/goimports versions come embedded in the golangci-lint release — no separate pins to drift. |
| 3 | Local hook shape (FR-6.1) | **PostToolUse format-on-write only** (no Stop-time check) | Format-on-write is a no-op when the file is already clean (no mtime churn), so friction only occurs when a fix was actually needed. A Stop hook running golangci-lint on every stop is too slow/noisy; CI + the CLAUDE.md checklist are the backstop. |
| 4 | `go vet` de-duplication (FR-2.5) | **Retain** the standalone `go vet` checklist item | Zero-config, fast, and full-module — whereas the guard's `govet` runs under `--new-from-rev` (changed lines only). Dropping it would weaken an existing guarantee. Overlap is harmless. |
| 5 | golangci-lint version | **v2.12.2** (latest stable, verified 2026-07-15) | Pinned in `tools/lint.versions`; the `standard` default group membership is fixed by this pin. |
| 6 | Git pre-commit hook (FR-6.3) | **Not shipped** | YAGNI: the Claude hook covers agents, CI covers everyone. Nothing in the design blocks adding one later. |
| 7 | `tsc --noEmit` in the guard | **Excluded** — type-checking stays owned by the `build` job (`tsc -b` via `npm run build`) | **Deviation from FR-3.3.** Including it would duplicate the type-check in every UI PR (test-ui already builds). The guard covers format + lint; the existing build gate covers types. |

Deviations from the PRD: decision 7 (tsc excluded from `lint.sh`) and decision 4
(FR-2.5 resolved as "retain"). Both are within the PRD's own stated design
latitude.

## 3. Components

### 3.1 `tools/lint.sh` — the shared guard

```
tools/lint.sh [--check] [--fmt] [--go|--ui] [--base <rev>] [path ...]
```

- **Default (fix mode):** `golangci-lint fmt` + `golangci-lint run --fix
  --new-from-rev <base>` per Go module; `prettier --write` + `eslint --fix` for
  atlas-ui. Exit 0 unless a tool errors.
- **`--check`:** `golangci-lint fmt --diff` (non-empty diff → fail, tree-wide,
  NOT rev-gated) + `golangci-lint run --new-from-rev <base>` per module;
  `prettier --check` + `eslint` (no `--fix`) for atlas-ui. Any violation →
  non-zero exit with the offending files listed.
- **`--fmt`:** restricts to the formatter layer only (`golangci-lint fmt` /
  `prettier`). This is what produces the baseline reformat commit (FR-5.2:
  formatter output only, no lint autofixes mixed in).
- **`--go` / `--ui`:** ecosystem selection; default is both.
- **`--base <rev>`:** override for the lint diff base. Default:
  `git merge-base HEAD origin/main` (falling back to `main`); if neither ref
  exists, run un-gated with a loud warning (never silently skip the linter).
- **Path arguments:** zero paths = whole tree (baseline + local whole-tree
  runs). CI passes the changed module paths. Go fan-out discovers modules the
  same way `tools/test-all-go.sh` does: `find ./services ./libs -name go.mod`,
  filtered to the given paths when present.
- **Workspace discipline (FR-1.5):** `GOWORK=off` is set *scoped to each
  golangci-lint invocation*, never exported globally — per the repo's
  documented workspace/guard footguns. Each module is linted from its own
  directory with the root config passed explicitly
  (`golangci-lint run -c "$ROOT/.golangci.yml"`).
- **Bootstrap (FR-1.7):** if `golangci-lint` at the pinned version is not in
  the cache dir, install it with
  `GOBIN=$ROOT/.cache/tools/bin go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$GOLANGCI_LINT_VERSION`.
  Requires only a Go toolchain — same bootstrap philosophy as
  `redis-key-guard.sh` building its analyzer. The binary is version-suffixed
  in the cache so a pin bump forces reinstall. UI tooling bootstraps via
  `npm ci` in `services/atlas-ui` (prettier/eslint are devDependencies).
  Missing Go or Node toolchain → clear error, non-zero exit (fail closed);
  `--go`-only runs don't require Node and vice versa.
- **Node version:** the script asserts `node` major version 22 before UI work
  and errors with `nvm use 22` guidance otherwise (per the atlas-ui toolchain
  memory). It does not source nvm itself.

### 3.2 `tools/lint.versions` — the pin file

A flat `KEY=VALUE` file sourced by `lint.sh` and read by CI:

```
GOLANGCI_LINT_VERSION=v2.12.2
```

That is the only entry. gofumpt/goimports are embedded in the golangci-lint
release (decision 2). Prettier (`3.9.5`) and `eslint-config-prettier`
(`10.1.8`) are pinned **exactly** (no `^`) in `services/atlas-ui/package.json`
— package.json + lockfile are already the single source of truth for Node
tooling, duplicating them in the pin file would create a second place to
drift. Go and Node runtime versions stay where they already live
(`pr-validation.yml` `env`, `go.mod` toolchain directives).

### 3.3 Root `.golangci.yml`

golangci-lint v2 config (`version: "2"`):

- `linters`: the **`standard` default group** (errcheck, govet, ineffassign,
  staticcheck, unused) — exactly the PRD's "broad default set", pinned in
  meaning by the version pin. No additions, no removals.
- `formatters`: enable `gofumpt` and `goimports`;
  `goimports.local-prefixes: github.com/Chronicle20/atlas` (FR-2.2). No
  `gofumpt.module-path` setting — it must stay unset because one shared config
  serves 80 modules; golangci-lint derives it per module from each `go.mod`.
- `exclusions`: generated-file exclusion at its default (`lazy`), plus
  path excludes for test fixtures/testdata if the baseline run surfaces any.
  Any per-path exclusion added under §5's escape hatch carries a comment
  naming the follow-up burn-down task.

### 3.4 atlas-ui: Prettier + ESLint reconciliation

- `prettier@3.9.5` and `eslint-config-prettier@10.1.8` added as exact-version
  devDependencies.
- Checked-in `.prettierrc` (minimal — house style defaults, decided at
  implementation; the config file existing is what matters for determinism)
  and `.prettierignore` mirroring `eslint.config.js`'s `globalIgnores` list
  (`dist`, `node_modules`, and the stale root directories), plus `coverage`.
- `eslint.config.js`: `eslint-config-prettier` appended **last** in the
  `extends` chain so it wins over any formatting-adjacent rules (FR-3.4).
- New scripts: `"format": "prettier --write ."`,
  `"format:check": "prettier --check ."` (scope controlled by
  `.prettierignore`).
- `lint.sh` UI layer calls the npm scripts (`format`/`format:check`, `lint`)
  rather than raw binaries, so package.json stays the UI command authority.

### 3.5 CI: two jobs in `pr-validation.yml`

Two jobs, matching the per-ecosystem gating the matrices already provide
(FR-4.2). **No composite action** (FR-4.5 declined — the job bodies are 3–4
steps, thinner than the `go-test` action; a wrapper would add indirection for
nothing).

- **`lint-go`** ("Lint & Format Guard (Go)") — `needs: detect-changes`, runs
  when `go-services-matrix != '[]' || go-libraries-matrix != '[]'`. A single
  job (not a matrix fan-out): it extracts `module_path` from both matrices
  with jq and runs `tools/lint.sh --check --go <paths...>`. Steps: checkout
  with `fetch-depth: 0` (required for the merge-base computation),
  `setup-go`, `actions/cache` on `~/.cache/golangci-lint` + the tool bin dir,
  then the guard. A single job is chosen over a matrix because golangci-lint
  over a typical PR's changed modules is seconds; the worst case (go.work
  change → all 80 modules) is rare and made tolerable by the lint cache.
- **`lint-ui`** ("Lint & Format Guard (UI)") — `needs: detect-changes`, gated
  on `has-ui-changes == 'true' || has-workflow-changes == 'true' ||
  force-all`. Steps: checkout, `setup-node` (22, npm cache), `npm ci`, then
  `tools/lint.sh --check --ui`.
- Both are added to `pr-validation-complete`'s `needs` list and its failure
  check, alongside the existing guards. Both fail closed (FR-4.6): `lint.sh`
  is `set -euo pipefail` throughout and a missing tool is an error, never a
  skip.
- The base ref for `--new-from-rev` in CI: `git merge-base
  origin/$GITHUB_BASE_REF HEAD`, passed via `--base`.
- Housekeeping: the existing `node-test` action's `continue-on-error: true`
  lint step becomes redundant for PRs (lint-ui now gates properly) and is
  left untouched — removing it is out of scope.

### 3.6 Local enforcement

- **Claude hook:** a new `.claude/hooks/format-on-write.sh`, registered as a
  `PostToolUse` hook on `Write|Edit` in `.claude/settings.json` (alongside the
  existing `PreToolUse` block-home-paths hook, same jq-driven pattern). It
  reads the tool JSON, and for `*.go` files runs `golangci-lint fmt` on the
  file from its module directory (via the same bootstrap/cache as `lint.sh`);
  for `*.ts`/`*.tsx` under `services/atlas-ui` it runs `npx prettier --write`
  on the file. Non-matching files, missing toolchains, or tool errors exit 0
  silently — a *local convenience* hook must never block edits (CI is the
  enforcement point; fail-closed applies to CI, fail-open to the hook).
- **CLAUDE.md:** the Build & Verification checklist gains item 7:
  `tools/lint.sh --check` clean from the repo root. Item 2 (`go vet`) is
  retained unchanged (decision 4).

## 4. Alternatives Considered

- **Remediate-to-green (PRD option a):** fix every finding tree-wide.
  Rejected: unbounded scope on a never-linted 80-module tree; turns a tooling
  task into an open-ended refactor with real regression risk (errcheck fixes
  change behavior).
- **Standalone gofumpt + goimports binaries next to golangci-lint:** rejected
  — three pins instead of one, and two independent format authorities that can
  disagree with `golangci-lint run`'s formatter checks (exactly the
  double-tooling FR-2.4 warns about).
- **Per-module CI matrix for lint-go:** rejected — job-count explosion (a
  go.work change would spawn 80 lint jobs) for no verdict benefit; the
  single-job fan-out inside `lint.sh` is what every existing guard already
  does.
- **Committed issue-baseline file instead of `--new-from-rev`:** golangci-lint
  v2 has no native baseline-file mechanism; simulating one via generated
  exclusion rules is fragile (line-number drift invalidates it on every
  touch). Rejected in favor of git-diff gating, which needs no artifact.
- **Pinning a fixed post-reformat SHA as the permanent diff base:** rejected —
  the branch SHA does not survive a squash-merge, so CI on later PRs could not
  resolve it. Merge-base against the PR's target branch is self-maintaining.

## 5. The CI-Green Mechanism, Honestly (FR-4.4)

The chosen gate is `--new-from-rev <merge-base with main>`. This has one sharp
interaction **on this task's own PR**: the baseline reformat touches lines
tree-wide, and `--new-from-rev` flags issues on *changed* lines — so
pre-existing lint findings that happen to sit on reformatted lines will
surface on this PR, and only on this PR (after merge, every later PR's
merge-base includes the reformat).

Plan for reaching green on this branch, in order:

1. Run `tools/lint.sh --fmt` → the isolated, formatter-only baseline commit
   (FR-5.1/5.2). After it, `lint.sh --check`'s formatter layer is zero-diff
   tree-wide (FR-5.3).
2. Run `tools/lint.sh --check` and collect the linter residue: findings from
   the `standard` group on lines the reformat touched. The `standard` group is
   five linters, not the 100-linter `all` set, and the tree is currently
   `go vet`-clean by checklist discipline — so the residue is expected to be
   modest, but its size is **empirically unknown until run**.
3. Remediate the residue in ordinary, reviewed commits (separate from the
   baseline commit). These are real findings on lines this PR touches; fixing
   them is in-scope.
4. **Escape hatch** if a residue cluster is pathological (e.g. a module with
   hundreds of errcheck hits on reformatted lines): add a scoped path
   exclusion to `.golangci.yml` with a comment naming the burn-down task, and
   move on. The exclusion is visible, reviewed, and tracked — not silent.
5. **Follow-up task filed** (acceptance-criteria requirement): burn down to a
   fully lint-clean tree, remove any §5.4 exclusions, and ultimately drop
   `--new-from-rev` so the linter layer enforces whole-tree like the
   formatters do.

Side effect worth stating: the reformat touches source in every service, so
this PR's `docker-services-matrix` will be *all services* — CI builds every
image once. Expected and acceptable.

## 6. Error Handling

- `lint.sh`: `set -euo pipefail`; per-module failures accumulate into a
  non-zero exit with a per-module FAIL summary (same shape as
  `redis-key-guard.sh`), rather than aborting on the first module.
- Missing toolchain: explicit error naming the missing binary and the install
  command; exit non-zero (CI fail-closed per FR-4.6).
- Un-resolvable diff base: warn loudly and run the linter un-gated (strictly
  more findings, never fewer — fails closed, not open).
- The PostToolUse hook is the one deliberate fail-open component (§3.6).

## 7. Testing & Verification

- **Idempotence:** run fix mode twice; second run must produce
  `git diff --exit-code` clean.
- **Check-mode correctness:** on the finished branch, `tools/lint.sh --check`
  exits 0; then a scratch commit with a deliberately misformatted `.go` file
  and a `.tsx` file must make it exit non-zero naming both files — and the
  same commit pushed to the PR must show `lint-go`/`lint-ui` failing
  (acceptance criterion "deliberately-broken test commit shown failing").
  The scratch commit is then dropped.
- **Determinism:** CI and local both resolve the golangci-lint version from
  `tools/lint.versions` and prettier from the lockfile; there is no second
  version declaration anywhere to drift.
- **Existing gates still pass:** `go test -race`, `go build`,
  `docker buildx bake` for touched services (all of them, per §5), and the
  three existing guards — per the CLAUDE.md checklist.
- The guard scripts themselves get a smoke test in the same style as
  `gen-lb-ports_test.sh` only if cheap; otherwise the deliberate-failure
  exercise above is the functional test (the guard has no logic worth a
  fixture harness — it is orchestration).

## 8. Commit Sequencing on This Branch

1. Tooling: `tools/lint.sh`, `tools/lint.versions`, `.golangci.yml`,
   atlas-ui prettier config + deps + scripts, eslint-config-prettier wiring.
2. Baseline reformat commit — machine-generated only (`lint.sh --fmt`),
   Go + UI, isolated for reviewability.
3. Linter-residue remediation commits (§5.3), separately reviewable.
4. Enforcement wiring: CI jobs, Claude hook, CLAUDE.md checklist item.
5. Deliberate-failure verification (pushed, observed red, dropped).

## 9. Out of Scope (confirmed from PRD non-goals)

Behavioral refactors beyond the §5 residue, editor/IDE config, custom linters
(existing bespoke analyzers stay separate guards), lint/format of generated or
vendored code and docs, pre-commit-framework adoption, and removal of the
`node-test` action's redundant lint step.
