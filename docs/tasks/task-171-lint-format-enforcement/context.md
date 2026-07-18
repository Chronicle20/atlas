# Task 171 — Lint & Format Enforcement: Execution Context

Companion to `plan.md`. Everything here was verified on 2026-07-15 unless noted.

## Key Files

| File | Role |
|---|---|
| `docs/tasks/task-171-lint-format-enforcement/prd.md`, `design.md` | Requirements + design (the spec this plan implements) |
| `tools/redis-key-guard.sh` | The guard pattern being followed: per-module fan-out via `find ... -name go.mod`, rc accumulation, FAIL summary |
| `tools/test-all-go.sh` | The module-discovery idiom (`find ./services ./libs -name go.mod`) — 80 modules currently |
| `.github/workflows/pr-validation.yml` | Guard jobs live here; `detect-changes` outputs consumed by the new jobs; `pr-validation-complete` at the bottom must gain both new jobs |
| `.github/actions/detect-changes/action.yml` | Matrix entries: services `{name, path, module_path, docker_image}`, libraries `{name, path, module_path, coverage_threshold}`; `module_path` is like `services/atlas-account/atlas.com/account` or `libs/atlas-constants` |
| `.claude/settings.json` + `.claude/hooks/block-home-paths-in-docs.sh` | Hook registration shape + the jq-driven stdin-JSON hook pattern to mirror |
| `services/atlas-ui/package.json`, `eslint.config.js` | UI wiring targets; eslint 10 flat config with `defineConfig` + `globalIgnores` |

## Decisions (from design.md, plus plan-time resolutions)

1. **CI-green mechanism:** baseline-suppress via `golangci-lint run --new-from-rev <merge-base>`; formatters enforced tree-wide. Burn-down follow-up filed in `docs/TODO.md`.
2. **Single Go authority:** golangci-lint v2 owns gofumpt + goimports (`formatters` config); fix = `golangci-lint fmt`, check = `fmt --diff`. No standalone formatter binaries.
3. **Pin:** `GOLANGCI_LINT_VERSION=v2.12.2` in `tools/lint.versions` (verified to exist via `go list -m -versions`). npm: `prettier@3.9.5`, `eslint-config-prettier@10.1.8`, exact (both verified on the registry).
4. **⚠ DEVIATION from design §3.1 — workspace mode, not GOWORK=off.** Verified: `GOWORK=off go build ./...` fails in `services/atlas-account`, `atlas-buddies`, `atlas-notes` ("updates to go.mod needed") because service go.mod/go.sum are not standalone-consistent; libs (`atlas-model`, `atlas-constants`) pass. CI's `go-test` runs per-module with root `go.work` active. So `lint.sh` runs golangci-lint per-module with the workspace ON — matching how `redis-key-guard.sh` runs its analyzer. FR-1.5's substance (never require `go work sync`; per-module iteration) is preserved.
5. **UI ESLint is a hard gate and is currently RED:** 52 errors / 7 warnings measured in `services/atlas-ui` (`npm run lint`, Node 22). Distribution: `react-refresh/only-export-components` ≈30 (colocated columns/forms exports — fix via scoped config override extending the existing `providers/ui/context` off-block), `@typescript-eslint/no-unused-vars` 12, `no-useless-escape` 4, `no-useless-assignment` 2, `preserve-caught-error` 1. Plan Task 3 remediates to zero errors BEFORE the baseline reformat so the reformat commit stays formatter-only.
6. **Hook is fail-open and never bootstraps:** the PostToolUse hook uses the cached golangci-lint binary only if `tools/lint.sh` already installed it (avoids a multi-minute `go install` stall on first Write). CI is the fail-closed enforcement point.
7. **`tsc --noEmit` excluded from the guard** (design decision 7 — the `build` job owns type-checking); standalone `go vet` checklist item retained (decision 4); no git pre-commit hook (decision 6); no composite action (design §3.5).
8. **`.prettierrc` is `{}`** — pure defaults; the checked-in file is what pins determinism. `.prettierignore` entries are root-anchored (gitignore semantics — unanchored `services` would ignore `src/services`), and additionally exclude `/docs`, `/public`, `*.md`, `package-lock.json` per the PRD non-goal on docs/generated content.

## Dependencies & Ordering

- Task 1 (guard) and Task 2 (prettier wiring) are independent; both precede everything else.
- Task 3 (UI eslint remediation) must precede Task 4 (baseline reformat) so the reformat commit contains zero hand edits.
- Task 4 must precede Task 5 (residue): `--new-from-rev` findings only materialize once the reformat has touched the lines.
- Task 6 (CI) requires the branch to be green under `--check` (Tasks 4–5) or its own jobs would fail on this PR.
- Task 9 Step 4 (CI half of the deliberate-failure exercise) can only run once the PR exists — it is the closing action of the PR checklist, not a pre-PR step.

## Environment Notes

- Node: `source ~/.nvm/nvm.sh && nvm use 22` before any atlas-ui command; `lint.sh` asserts major 22 and does not source nvm.
- golangci-lint bootstrap compiles from source (`go install …@v2.12.2`) — several minutes on first run; cached at `.cache/tools/bin/golangci-lint-v2.12.2` (dir gitignored via `/.cache/`; CI caches it keyed on `hashFiles('tools/lint.versions')`).
- Whole-tree runs iterate 80 modules — expect minutes. CI passes only changed `module_path`s.
- The baseline reformat touches source in every service, so the PR's docker matrix builds ALL images once — expected (design §5); no local bake is mandated because no `go.mod` is touched.
- Uncertainties an executor may hit (handled inline in plan steps): `golangci-lint fmt` arg handling for `./...` (Task 1 Step 5 fallback: drop the arg), `eslint-config-prettier/flat` import path (Task 2 Step 5 fallback: package default export).

## Risks

- **Unknown residue size** (design §5): the `standard`-group findings on reformatted lines are unmeasured until Task 5 Step 1. Escape hatch = scoped `.golangci.yml` exclusion with a "task-171 burn-down" comment.
- **Merge-base drift on a long-lived branch:** local `--check` uses merge-base with `origin/main`; if main moves substantially, re-run after a rebase before claiming green.
- **eslint remediation behavior risk:** unused-var deletions and `preserve-caught-error` fixes are behavior-adjacent; Task 3 Step 5 gates on `npm test` + `npm run build` (tsc) passing.
