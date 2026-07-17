# Lint & Format Enforcement Tooling — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** One shared guard — `tools/lint.sh` — with fix and `--check` modes covering Go (golangci-lint v2 as single authority for gofumpt+goimports formatting and `standard`-group linting) and atlas-ui (Prettier + ESLint), enforced locally via a Claude Code hook + CLAUDE.md checklist and in CI via two new `pr-validation.yml` jobs, with a one-time baseline reformat landing the tree green.

**Architecture:** Follows the repo's established guard pattern (`redis-key-guard.sh`, `goroutine-guard.sh`): a single `tools/` script that is the source of truth, run in fix mode locally and `--check` mode in CI. Formatting is enforced tree-wide (cleared by the baseline reformat); Go lint *findings* are gated to new code via `--new-from-rev <merge-base>` with a tracked burn-down follow-up. Task docs: `docs/tasks/task-171-lint-format-enforcement/{prd.md,design.md}`.

**Tech Stack:** bash, golangci-lint v2.12.2 (bootstrapped via `go install`, cached under `.cache/tools/bin`), Prettier 3.9.5 + eslint-config-prettier 10.1.8 (exact-pinned npm devDependencies), GitHub Actions.

## Global Constraints

- golangci-lint pin: **v2.12.2** — the ONLY entry in `tools/lint.versions` (verified to exist on the Go module proxy, 2026-07-15).
- npm pins (exact, no `^`): **prettier@3.9.5**, **eslint-config-prettier@10.1.8** (both verified to exist on the npm registry, 2026-07-15).
- atlas-ui commands need **Node 22** (`nvm use 22`; `source ~/.nvm/nvm.sh` first). The script asserts major version 22 and errors otherwise — it never sources nvm itself.
- **DEVIATION from design §3.1 (verified 2026-07-15):** `GOWORK=off go build ./...` FAILS in service modules (`services/atlas-account`, `atlas-buddies`, `atlas-notes` all fail with "updates to go.mod needed"; libs pass). Service go.mod files are not standalone-consistent; CI's `go-test` action runs per-module with the root `go.work` active. Therefore `lint.sh` runs golangci-lint **in workspace mode (no `GOWORK=off`)** — same as `redis-key-guard.sh` runs its analyzer. This still satisfies FR-1.5's real requirement: the guard never runs `go work sync` and iterates per-module.
- The Go linter layer is always invoked with `-c <repo-root>/.golangci.yml` from each module's own directory; no per-module configs, ever.
- The baseline reformat commit (Task 4) must contain ONLY machine-generated formatter output — no manual edits mixed in.
- All new shell scripts: `set -euo pipefail` (except the Claude hook, which is deliberately fail-open), executable bit set.
- Commit messages reference `task-171`. Never commit to `main`; all work happens on branch `task-171-lint-format-enforcement` in this worktree.
- Files under `docs/` must never contain absolute home paths (a PreToolUse hook blocks them); use repo-relative paths.

## File Structure

| File | Responsibility |
|---|---|
| `tools/lint.versions` | Version pin file (single entry), sourced by `lint.sh` and hashed for the CI cache key |
| `tools/lint.sh` | The shared guard: arg parsing, golangci-lint bootstrap, per-module Go fan-out, atlas-ui npm-script layer, FAIL summary |
| `.golangci.yml` | Repo-root golangci-lint v2 config: `standard` linter group + gofumpt/goimports formatters |
| `.gitignore` | Gains `/.cache/` (tool bootstrap cache) |
| `services/atlas-ui/.prettierrc` | Prettier config (empty object — pure defaults; existence pins determinism) |
| `services/atlas-ui/.prettierignore` | Prettier scope control (root-anchored mirror of eslint ignores + docs/md/public exclusions) |
| `services/atlas-ui/package.json` | `format`/`format:check` scripts; exact-pinned prettier + eslint-config-prettier devDependencies |
| `services/atlas-ui/eslint.config.js` | `eslint-config-prettier` appended last; scoped rule overrides from the remediation task |
| `.github/workflows/pr-validation.yml` | New `lint-go` + `lint-ui` jobs; `pr-validation-complete` wiring |
| `.claude/hooks/format-on-write.sh` | PostToolUse format-on-write hook (fail-open) |
| `.claude/settings.json` | Registers the PostToolUse hook |
| `CLAUDE.md` | Build & Verification checklist item 7 |
| `docs/TODO.md` | Lint burn-down follow-up entry |

---

### Task 1: Guard tooling — pin file, golangci config, `tools/lint.sh`

**Files:**
- Create: `tools/lint.versions`
- Create: `.golangci.yml`
- Create: `tools/lint.sh`
- Modify: `.gitignore` (append one line)

**Interfaces:**
- Produces: `tools/lint.sh [--check] [--fmt] [--go|--ui] [--base <rev>] [path ...]` — exit 0 clean / 1 violations / 2 usage error. Consumed verbatim by Tasks 4, 5, 6, 9.
- Produces: `tools/lint.versions` defining `GOLANGCI_LINT_VERSION=v2.12.2` — sourced by `lint.sh` (Task 1) and the hook (Task 7); hashed by the CI cache key (Task 6).
- Produces: bootstrapped binary at `.cache/tools/bin/golangci-lint-v2.12.2` — reused by the hook (Task 7).
- Note: the UI layer of `lint.sh` calls npm scripts (`format`, `format:check`, `lint`) that do not exist until Task 2. That is fine — Task 1's verification exercises only the `--go` path.

- [x] **Step 1: Write `tools/lint.versions`**

```bash
# Tool version pins for tools/lint.sh — the single source of truth read by
# both local runs and CI (task-171). gofumpt/goimports versions are embedded
# in the golangci-lint release; Prettier is pinned exactly in
# services/atlas-ui/package.json (package.json + lockfile own Node tooling).
GOLANGCI_LINT_VERSION=v2.12.2
```

- [x] **Step 2: Write `.golangci.yml`**

```yaml
# Root golangci-lint v2 config — the single config source for every Go module
# in the repo (task-171). tools/lint.sh runs golangci-lint from each module's
# own directory with this file passed via -c; do not add per-module configs.
#
# Escape-hatch exclusions (design.md §5.4) must carry a comment naming the
# burn-down follow-up (docs/TODO.md "Lint burn-down").
version: "2"

linters:
  # The v2 `standard` default group: errcheck, govet, ineffassign,
  # staticcheck, unused. Membership is fixed by the version pin in
  # tools/lint.versions.
  default: standard

formatters:
  enable:
    - gofumpt
    - goimports
  settings:
    goimports:
      # Group intra-repo imports separately (FR-2.2). gofumpt's module-path
      # stays UNSET — one shared config serves 80 modules; golangci-lint
      # derives it per module from each go.mod.
      local-prefixes:
        - github.com/Chronicle20/atlas
```

- [x] **Step 3: Append the tool-cache dir to `.gitignore`**

Append this line to the end of `.gitignore`:

```
/.cache/
```

- [x] **Step 4: Write `tools/lint.sh`**

```bash
#!/usr/bin/env bash
# tools/lint.sh — shared lint & format guard (task-171).
#
# One entry point for both local use (fix mode) and CI (--check mode), so the
# two can never disagree. golangci-lint v2 is the single authority for Go
# formatting (gofumpt + goimports via .golangci.yml `formatters`) and linting
# (`standard` group). atlas-ui uses Prettier + ESLint via its npm scripts.
#
# Formatting is enforced TREE-WIDE. Linter findings are gated to NEW code via
# --new-from-rev (burn-down tracked in docs/TODO.md "Lint burn-down").
#
# golangci-lint runs per-module in WORKSPACE MODE (root go.work active):
# service go.mod files are not standalone-consistent, so GOWORK=off would
# fail type-loading (verified — see docs/tasks/task-171-lint-format-enforcement/context.md).
# The guard never requires `go work sync`.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
# shellcheck source=lint.versions
source "$ROOT/tools/lint.versions"

NODE_MAJOR_REQUIRED=22

usage() {
    cat <<'EOF'
Usage: tools/lint.sh [--check] [--fmt] [--go|--ui] [--base <rev>] [path ...]

  (no flags)    fix mode: rewrite files in place (formatters + lint --fix)
  --check       check mode: mutate nothing; non-zero exit on any violation
  --fmt         formatter layer only (produces the baseline reformat)
  --go / --ui   restrict to one ecosystem (default: both)
  --base <rev>  diff base for the Go *linter* layer (--new-from-rev).
                Default: merge-base of HEAD and origin/main (fallback: main).
                Formatting is never rev-gated — it is enforced tree-wide.
  path ...      restrict Go module discovery to modules under these paths
                (CI passes changed module paths). No paths = whole tree.

Versions are pinned in tools/lint.versions. Exit: 0 clean, 1 violations, 2 usage.
EOF
}

CHECK=0
FMT_ONLY=0
DO_GO=1
DO_UI=1
BASE=""
PATHS=()

while [ $# -gt 0 ]; do
    case "$1" in
        --check) CHECK=1 ;;
        --fmt)   FMT_ONLY=1 ;;
        --go)    DO_UI=0 ;;
        --ui)    DO_GO=0 ;;
        --base)  BASE="${2:?--base requires a revision}"; shift ;;
        -h|--help) usage; exit 0 ;;
        -*) echo "lint.sh: unknown flag: $1" >&2; usage >&2; exit 2 ;;
        *) PATHS+=("$1") ;;
    esac
    shift
done

TOOLS_BIN="$ROOT/.cache/tools/bin"
GOLANGCI="$TOOLS_BIN/golangci-lint-$GOLANGCI_LINT_VERSION"

GO_RC=0
UI_RC=0
FAILED=()

ensure_golangci() {
    if ! command -v go >/dev/null 2>&1; then
        echo "lint.sh: ERROR — go toolchain not found (required for Go checks)" >&2
        exit 1
    fi
    if [ ! -x "$GOLANGCI" ]; then
        echo "lint.sh: installing golangci-lint $GOLANGCI_LINT_VERSION into $TOOLS_BIN ..."
        mkdir -p "$TOOLS_BIN"
        local tmp
        tmp="$(mktemp -d)"
        GOBIN="$tmp" go install "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$GOLANGCI_LINT_VERSION"
        mv "$tmp/golangci-lint" "$GOLANGCI"
        rm -rf "$tmp"
    fi
}

resolve_base() {
    if [ -n "$BASE" ]; then
        echo "$BASE"
        return 0
    fi
    git -C "$ROOT" merge-base HEAD origin/main 2>/dev/null && return 0
    git -C "$ROOT" merge-base HEAD main 2>/dev/null && return 0
    return 1
}

discover_modules() {
    if [ "${#PATHS[@]}" -eq 0 ]; then
        find "$ROOT/services" "$ROOT/libs" -name go.mod -not -path '*/node_modules/*' -print0 \
            | xargs -0 -n1 dirname | sort -u
    else
        local p target
        for p in "${PATHS[@]}"; do
            case "$p" in
                /*) target="$p" ;;
                *)  target="$ROOT/${p#./}" ;;
            esac
            find "$target" -name go.mod -not -path '*/node_modules/*' -print0 2>/dev/null \
                | xargs -0 -r -n1 dirname
        done | sort -u
    fi
}

run_go() {
    ensure_golangci
    local base=""
    if [ "$FMT_ONLY" -eq 0 ]; then
        if ! base="$(resolve_base)"; then
            echo "lint.sh: WARNING — cannot resolve a merge base with origin/main or main;" >&2
            echo "lint.sh: WARNING — running the linter UN-GATED (whole-module findings, never fewer)." >&2
            base=""
        fi
    fi

    local moddir rel fmt_out
    while IFS= read -r moddir; do
        rel="${moddir#"$ROOT"/}"

        # ---- formatter layer: tree-wide, never rev-gated -------------------
        if [ "$CHECK" -eq 1 ]; then
            if fmt_out="$(cd "$moddir" && "$GOLANGCI" fmt --diff -c "$ROOT/.golangci.yml" ./... 2>&1)" \
                    && [ -z "$fmt_out" ]; then
                : # clean
            else
                echo "lint.sh: FMT FAIL — $rel"
                printf '%s\n' "$fmt_out" | head -40 || true
                GO_RC=1
                FAILED+=("fmt:$rel")
            fi
        else
            if ! (cd "$moddir" && "$GOLANGCI" fmt -c "$ROOT/.golangci.yml" ./...); then
                echo "lint.sh: FMT ERROR — $rel"
                GO_RC=1
                FAILED+=("fmt:$rel")
            fi
        fi

        # ---- linter layer: rev-gated to new code (design.md §5) ------------
        if [ "$FMT_ONLY" -eq 0 ]; then
            local -a lintargs=(run -c "$ROOT/.golangci.yml")
            if [ "$CHECK" -eq 0 ]; then
                lintargs+=(--fix)
            fi
            if [ -n "$base" ]; then
                lintargs+=(--new-from-rev "$base")
            fi
            if ! (cd "$moddir" && "$GOLANGCI" "${lintargs[@]}" ./...); then
                echo "lint.sh: LINT FAIL — $rel"
                GO_RC=1
                FAILED+=("lint:$rel")
            fi
        fi
    done < <(discover_modules)
}

run_ui() {
    local uidir="$ROOT/services/atlas-ui"
    if ! command -v node >/dev/null 2>&1; then
        echo "lint.sh: ERROR — node not found; atlas-ui checks need Node $NODE_MAJOR_REQUIRED (try: nvm use $NODE_MAJOR_REQUIRED)" >&2
        UI_RC=1
        FAILED+=("ui:node-missing")
        return
    fi
    local major
    major="$(node --version | sed 's/^v//' | cut -d. -f1)"
    if [ "$major" != "$NODE_MAJOR_REQUIRED" ]; then
        echo "lint.sh: ERROR — node v$major found, need v$NODE_MAJOR_REQUIRED (try: nvm use $NODE_MAJOR_REQUIRED)" >&2
        UI_RC=1
        FAILED+=("ui:node-version")
        return
    fi
    if [ ! -d "$uidir/node_modules" ]; then
        echo "lint.sh: bootstrapping atlas-ui dev tooling (npm ci) ..."
        (cd "$uidir" && npm ci)
    fi

    if [ "$CHECK" -eq 1 ]; then
        if ! (cd "$uidir" && npm run format:check); then
            echo "lint.sh: UI FMT FAIL — services/atlas-ui"
            UI_RC=1
            FAILED+=("ui:prettier")
        fi
        if [ "$FMT_ONLY" -eq 0 ]; then
            if ! (cd "$uidir" && npm run lint); then
                echo "lint.sh: UI LINT FAIL — services/atlas-ui"
                UI_RC=1
                FAILED+=("ui:eslint")
            fi
        fi
    else
        if ! (cd "$uidir" && npm run format); then
            UI_RC=1
            FAILED+=("ui:prettier")
        fi
        if [ "$FMT_ONLY" -eq 0 ]; then
            if ! (cd "$uidir" && npm run lint -- --fix); then
                echo "lint.sh: UI LINT FAIL — unfixable findings remain (services/atlas-ui)"
                UI_RC=1
                FAILED+=("ui:eslint")
            fi
        fi
    fi
}

if [ "$DO_GO" -eq 1 ]; then
    run_go
fi
if [ "$DO_UI" -eq 1 ]; then
    run_ui
fi

if [ "$GO_RC" -ne 0 ] || [ "$UI_RC" -ne 0 ]; then
    echo ""
    echo "lint.sh: FAIL — ${#FAILED[@]} failing target(s):"
    printf 'lint.sh:   %s\n' "${FAILED[@]}"
    if [ "$CHECK" -eq 1 ]; then
        echo "lint.sh: run 'tools/lint.sh' (fix mode) locally, then commit the result."
    fi
    exit 1
fi
echo "lint.sh: OK"
```

Then: `chmod +x tools/lint.sh`

- [x] **Step 5: Verify bootstrap + check mode on one small module**

Run: `tools/lint.sh --check --go libs/atlas-model`

Expected: first run prints `installing golangci-lint v2.12.2 ...` (takes a few minutes — it compiles from source), then either `lint.sh: OK` or `FMT FAIL — libs/atlas-model` with a diff (the tree is pre-baseline; a formatting diff here is a legitimate finding, NOT a script bug). The failure condition for this step is a crash: bootstrap error, `unknown flag`, config parse error, or golangci-lint panic. If `golangci-lint fmt ./...` rejects the `./...` argument, drop `./...` from both `fmt` invocations (it then defaults to the current directory tree — same scope) and re-run.

- [x] **Step 6: Verify check mode catches a deliberately broken file (Go)**

```bash
mkdir -p libs/atlas-model/scratchlint
cat > libs/atlas-model/scratchlint/scratch.go <<'EOF'
package scratchlint

import "os"

func Bad( ) {
os.Open("nope")
}
EOF
git add libs/atlas-model/scratchlint
git commit -m "scratch: deliberate lint/format violation (will be dropped)"
tools/lint.sh --check --go libs/atlas-model; echo "exit=$?"
```

Expected: `FMT FAIL — libs/atlas-model` (gofumpt spacing/indent diff) AND `LINT FAIL — libs/atlas-model` (errcheck: unchecked `os.Open` error — a NEW file relative to the merge base, so `--new-from-rev` must catch it), summary listing both, `exit=1`.

- [x] **Step 7: Drop the scratch commit and verify fix mode is a no-op on a clean module**

```bash
git reset --hard HEAD~1
tools/lint.sh --fmt --go libs/atlas-constants && git diff --exit-code -- libs/atlas-constants; echo "exit=$?"
```

Expected: if atlas-constants is already gofumpt-clean, `lint.sh: OK` and `exit=0` (no diff). If it produces a formatting diff, that is pre-baseline drift — `git checkout -- libs/atlas-constants` to revert it (the baseline reformat lands in Task 4, not here) and treat the step as passed (fix mode wrote only formatter output).

- [x] **Step 8: Commit**

```bash
git add tools/lint.versions tools/lint.sh .golangci.yml .gitignore
git commit -m "feat(task-171): shared lint & format guard — tools/lint.sh + golangci-lint v2 config"
```

---

### Task 2: atlas-ui Prettier wiring

**Files:**
- Create: `services/atlas-ui/.prettierrc`
- Create: `services/atlas-ui/.prettierignore`
- Modify: `services/atlas-ui/package.json` (scripts + devDependencies)
- Modify: `services/atlas-ui/eslint.config.js`

**Interfaces:**
- Consumes: nothing from Task 1 (independent).
- Produces: npm scripts `format` (`prettier --write .`) and `format:check` (`prettier --check .`) — called by `lint.sh`'s UI layer (Task 1's contract) and by the hook indirectly (Task 7 uses `npx prettier` directly).
- Produces: `eslint-config-prettier` appended last in the eslint `extends` chain (FR-3.4).

- [x] **Step 1: Install exact-pinned devDependencies**

```bash
source ~/.nvm/nvm.sh && nvm use 22
cd services/atlas-ui
npm install --save-dev --save-exact prettier@3.9.5 eslint-config-prettier@10.1.8
```

Expected: `package.json` devDependencies gain `"prettier": "3.9.5"` and `"eslint-config-prettier": "10.1.8"` (no `^`); `package-lock.json` updated. If either exact version has vanished from the registry, STOP and report — do not substitute a different version silently.

- [x] **Step 2: Add the npm scripts**

In `services/atlas-ui/package.json`, extend the `scripts` block (existing entries unchanged):

```json
  "scripts": {
    "dev": "vite",
    "build": "tsc -b && vite build",
    "preview": "vite preview",
    "lint": "eslint .",
    "format": "prettier --write .",
    "format:check": "prettier --check .",
    "test": "vitest run",
    "test:watch": "vitest",
    "test:coverage": "vitest run --coverage"
  },
```

- [x] **Step 3: Write `services/atlas-ui/.prettierrc`**

```json
{}
```

(Pure Prettier defaults. The file's existence — not its contents — is what pins determinism; house-style overrides can be added later as their own reviewed change.)

- [x] **Step 4: Write `services/atlas-ui/.prettierignore`**

```
# Mirrors eslint.config.js globalIgnores. Entries are root-anchored (leading
# slash) because .prettierignore uses gitignore semantics — an unanchored
# name like `components` would also ignore src/components.
/dist
/node_modules
/.next
/app
/components
/context
/hooks
/lib
/services
/types
/tests
# Not lint/format surface (PRD non-goal: docs & generated content).
/coverage
/docs
/public
/package-lock.json
*.md
```

- [x] **Step 5: Append eslint-config-prettier to `eslint.config.js`**

Full resulting file (two changes vs current: the new import and `eslintConfigPrettier` LAST in `extends`):

```js
import js from "@eslint/js";
import globals from "globals";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import tseslint from "typescript-eslint";
import eslintConfigPrettier from "eslint-config-prettier/flat";
import { defineConfig, globalIgnores } from "eslint/config";

export default defineConfig([
  globalIgnores(["dist", "node_modules", ".next", "app", "components", "context", "hooks", "lib", "services", "types", "tests"]),
  {
    files: ["src/**/*.{ts,tsx}"],
    extends: [
      js.configs.recommended,
      tseslint.configs.recommended,
      reactHooks.configs.flat.recommended,
      reactRefresh.configs.vite,
      eslintConfigPrettier,
    ],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
    },
  },
  {
    files: [
      "src/components/providers/**/*.{ts,tsx}",
      "src/components/ui/**/*.{ts,tsx}",
      "src/context/**/*.{ts,tsx}",
    ],
    rules: {
      "react-refresh/only-export-components": "off",
    },
  },
]);
```

If the `eslint-config-prettier/flat` import path errors under this eslint-config-prettier version, use the package default export instead: `import eslintConfigPrettier from "eslint-config-prettier";` — same object shape for flat-config use.

- [x] **Step 6: Verify the wiring executes**

```bash
cd services/atlas-ui
npx prettier --check src/main.tsx || true
npm run format:check || true
npm run lint || true
```

Expected: `prettier --check` runs and reports (pass or "Code style issues found" — the tree is pre-baseline, failures are legitimate); `format:check` exercises the ignore file without erroring on config; `npm run lint` still runs (existing errors expected — remediated in Task 3). Failure condition: config parse errors, unknown-option errors, or eslint crashing on the new import.

- [x] **Step 7: Commit**

```bash
git add services/atlas-ui/package.json services/atlas-ui/package-lock.json \
        services/atlas-ui/.prettierrc services/atlas-ui/.prettierignore \
        services/atlas-ui/eslint.config.js
git commit -m "feat(task-171): atlas-ui prettier + eslint-config-prettier wiring"
```

---

### Task 3: atlas-ui ESLint remediation to zero errors

The design makes `npm run lint` a hard CI gate (`lint-ui`), and it currently FAILS: **52 errors, 7 warnings** (measured 2026-07-15). ESLint's default exit code fails on errors only, so warnings may remain. Measured error distribution: `react-refresh/only-export-components` ≈30, `@typescript-eslint/no-unused-vars` 12, `no-useless-escape` 4, `no-useless-assignment` 2, `preserve-caught-error` 1, remainder singletons. Re-measure at execution time — the numbers may have drifted.

**Files:**
- Modify: `services/atlas-ui/eslint.config.js` (scoped overrides only, appended to the existing override block pattern)
- Modify: the TS/TSX files eslint names (list obtained from the lint run)

**Interfaces:**
- Consumes: Task 2's eslint.config.js (edits build on it).
- Produces: `npm run lint` exit 0 — required by `lint.sh --check --ui` (Task 1 contract) and the `lint-ui` CI job (Task 6).

- [x] **Step 1: Capture the authoritative error list**

```bash
source ~/.nvm/nvm.sh && nvm use 22
cd services/atlas-ui
npm run lint 2>&1 | tee /tmp/eslint-before.txt
```

Expected: non-zero exit; the file list drives Steps 2–4.

- [x] **Step 2: Fix `react-refresh/only-export-components` via scoped config overrides**

These fire in colocated non-page modules (columns/forms files that export helpers alongside components) — the codebase's deliberate colocation pattern, already exempted for `providers/ui/context` in the existing override block. Extend that block rather than refactoring ~30 files: collect the offending file paths from `/tmp/eslint-before.txt`, derive the narrowest glob(s) that cover them (e.g. `src/pages/**/*-columns.tsx`, `src/components/features/**/*-columns.tsx` — derive from the actual list, do not guess), and add them to the existing `files` array of the override that sets `"react-refresh/only-export-components": "off"`. If a handful of offenders don't fit a colocation glob, fix those individually by moving the non-component export to its own file. Do NOT turn the rule off globally.

- [x] **Step 3: Fix `@typescript-eslint/no-unused-vars` (≈12)**

For each: delete the unused binding if it is dead code; if it must stay for signature/documentation reasons, prefix it with `_` AND add the ignore-pattern config once, in the main config object's `rules`:

```js
    rules: {
      "@typescript-eslint/no-unused-vars": [
        "error",
        { argsIgnorePattern: "^_", varsIgnorePattern: "^_", caughtErrorsIgnorePattern: "^_" },
      ],
    },
```

- [x] **Step 4: Fix the mechanical remainder**

- `no-useless-escape` (4): remove the unnecessary `\` escapes (e.g. `\.` inside a character class in `src/services/errorLogger.ts:58`).
- `no-useless-assignment` (2): delete the dead assignment.
- `preserve-caught-error` (1, `src/services/api/npcs.service.ts:94`): attach the caught error: `throw new Error("...", { cause: err })`.
- Anything else on the list: fix mechanically per the rule's documented remedy. If a fix would change runtime behavior non-trivially, STOP and flag it rather than guessing.

- [x] **Step 5: Verify lint is clean and nothing broke**

```bash
cd services/atlas-ui
npm run lint          # expect: exit 0 (warnings allowed, zero errors)
npm test              # expect: PASS
npm run build         # expect: tsc -b + vite build succeed
```

- [x] **Step 6: Commit**

```bash
git add services/atlas-ui
git commit -m "fix(task-171): atlas-ui eslint remediation to zero errors"
```

---

### Task 4: Baseline reformat (formatter-only commit)

**Files:**
- Modify: machine-generated formatter output across all 80 Go modules + atlas-ui (no hand edits)

**Interfaces:**
- Consumes: `tools/lint.sh --fmt` (Task 1), npm `format` script (Task 2).
- Produces: a tree where `tools/lint.sh --check --fmt` passes with zero diffs — required by Tasks 5, 6, 9.

- [x] **Step 1: Confirm a clean working tree, then run the formatter layer tree-wide**

```bash
git status --porcelain   # expect: empty
tools/lint.sh --fmt      # Go: golangci-lint fmt per module; UI: prettier --write .
```

Expected: exits 0. (Whole-tree run over 80 modules — expect minutes, not seconds.)

- [x] **Step 2: Verify the diff is formatter-only**

```bash
git diff --stat | tail -5
git diff | grep -E '^\+' | grep -vE '^\+\+\+' | grep -iE 'TODO|FIXME|func [a-z]+[A-Z]' | head
```

Spot-read a few hunks (`git diff -- <a-large-service-file>`): every change must be whitespace, import ordering/grouping, or gofumpt/prettier canonicalization. No logic, no renames, no new symbols. If anything else appears, STOP — a formatter must never do that.

- [x] **Step 3: Verify the formatter layer is now clean and idempotent tree-wide**

```bash
tools/lint.sh --check --fmt     # expect: lint.sh: OK
tools/lint.sh --fmt && git diff --exit-code   # second fix run must be a no-op
```

- [x] **Step 4: Verify nothing broke — full Go + UI test sweep**

```bash
tools/test-all-go.sh 2>&1 | tail -20    # expect: no test failures (long run)
cd services/atlas-ui && npm test && npm run build && cd ../..
```

Note: the reformat touches `.go` source but NO `go.mod` files, so the CLAUDE.md bake mandate ("every service whose go.mod was touched") does not trigger; CI's docker matrix will build all images on the PR regardless (expected, per design §5).

- [x] **Step 5: Commit (single, isolated, machine-generated)**

```bash
git add -A
git commit -m "style(task-171): baseline reformat — gofumpt + goimports + prettier (machine-generated, no manual edits)"
```

---

### Task 5: Go linter residue remediation

`--new-from-rev <merge-base>` flags findings on changed lines; the baseline reformat changed lines tree-wide, so pre-existing `standard`-group findings sitting on reformatted lines surface on THIS branch only (design §5). Size is empirically unknown until run.

**Files:**
- Modify: whichever Go files the linter names (ordinary, reviewed fixes)
- Modify (escape hatch only): `.golangci.yml`

**Interfaces:**
- Consumes: `tools/lint.sh --check --go` (Task 1), post-reformat tree (Task 4).
- Produces: `tools/lint.sh --check` exit 0 on this branch — required by Task 6 (CI green) and Task 9.

- [x] **Step 1: Collect the residue**

```bash
tools/lint.sh --check --go 2>&1 | tee /tmp/lint-residue.txt
grep -c 'LINT FAIL' /tmp/lint-residue.txt || true
```

Expected: `FMT FAIL` count must be ZERO (Task 4 guarantees it). `LINT FAIL` modules are the work list. If the list is empty, skip to Step 4.

- [x] **Step 2: Fix findings module by module, in reviewed commits**

For each failing module, re-run to see the findings (`cd <module> && ../../../../.cache/tools/bin/golangci-lint-v2.12.2 run -c <repo-root>/.golangci.yml --new-from-rev "$(git merge-base HEAD origin/main)" ./...` — or just re-run `tools/lint.sh --check --go <module-path>`). Typical `standard`-group fixes:
- `errcheck`: handle or explicitly assign the error (`_ = f.Close()` only where ignoring is genuinely correct and obvious).
- `staticcheck`/`govet`: apply the documented remedy for the specific check.
- `unused`/`ineffassign`: delete dead code.

An errcheck fix that would change behavior (e.g. surfacing an error that was silently dropped in a hot path) is still in-scope — these are real findings on lines this PR touches — but if one looks risky, prefer the explicit `_ =` acknowledgment over a behavior change, and note it in the commit message. Group fixes into per-service or per-theme commits: `git commit -m "fix(task-171): lint residue — <module or theme>"`.

- [x] **Step 3 (escape hatch, only if a cluster is pathological): scoped exclusion**

If one module has an unreasonable finding count (e.g. hundreds of errcheck hits), add a scoped exclusion to `.golangci.yml` instead — visible, commented, tracked:

```yaml
linters:
  default: standard
  exclusions:
    rules:
      # task-171 burn-down (docs/TODO.md "Lint burn-down"): pre-existing
      # errcheck debt in <module>; remove this block when remediated.
      - path: services/<module>/
        linters:
          - errcheck
```

(Adjust `path`/`linters` to the actual cluster. Every escape-hatch block MUST carry the burn-down comment.)

- [x] **Step 4: Verify both ecosystems fully clean**

```bash
tools/lint.sh --check    # expect: lint.sh: OK, exit 0
```

Run the affected services' tests for every module Step 2 touched: `cd <module> && go test -race ./... && go vet ./...` — clean. If any touched module's fix altered non-test source, this is behavior-adjacent: also `go build ./...` in it.

- [x] **Step 5: Final commit for any stragglers**

```bash
git status --porcelain    # expect: empty (everything committed in Steps 2–3)
```

---

### Task 6: CI wiring — `lint-go` + `lint-ui` jobs

**Files:**
- Modify: `.github/workflows/pr-validation.yml`

**Interfaces:**
- Consumes: `tools/lint.sh --check` contract (Task 1); `detect-changes` outputs `go-services-matrix` / `go-libraries-matrix` (entries carry `module_path`, e.g. `services/atlas-account/atlas.com/account`), `has-ui-changes`, `has-workflow-changes`.
- Produces: required PR checks `Lint & Format Guard (Go)` and `Lint & Format Guard (UI)`; both wired into `pr-validation-complete`.

- [x] **Step 1: Add the two jobs**

Insert after the `gen-lb-ports` job (keeping the guard jobs grouped):

```yaml
  # ============================================
  # Lint & Format Guard (task-171)
  # tools/lint.sh --check is the single source of
  # truth; these jobs mirror it in CI. Formatting
  # is enforced tree-wide on the changed modules;
  # lint findings are gated to new code via
  # --new-from-rev (burn-down: docs/TODO.md).
  # Fail closed: a missing tool or script error
  # fails the job, never a silent pass.
  # ============================================
  lint-go:
    name: Lint & Format Guard (Go)
    needs: detect-changes
    if: needs.detect-changes.outputs.go-services-matrix != '[]' || needs.detect-changes.outputs.go-libraries-matrix != '[]'
    runs-on: ubuntu-latest

    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          # Full history: lint.sh needs the merge base for --new-from-rev.
          fetch-depth: 0

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}

      - name: Cache lint tooling
        uses: actions/cache@v4
        with:
          path: |
            ~/.cache/golangci-lint
            .cache/tools/bin
          key: lint-tools-${{ runner.os }}-${{ hashFiles('tools/lint.versions') }}

      - name: Lint & format guard (Go)
        env:
          SERVICES: ${{ needs.detect-changes.outputs.go-services-matrix }}
          LIBRARIES: ${{ needs.detect-changes.outputs.go-libraries-matrix }}
        run: |
          set -euo pipefail
          mapfile -t MODULES < <({ jq -r '.[].module_path' <<<"$SERVICES"; jq -r '.[].module_path' <<<"$LIBRARIES"; })
          echo "Linting ${#MODULES[@]} changed module(s)"
          if [ -n "${GITHUB_BASE_REF:-}" ]; then
            BASE="$(git merge-base "origin/${GITHUB_BASE_REF}" HEAD)"
            ./tools/lint.sh --check --go --base "$BASE" "${MODULES[@]}"
          else
            ./tools/lint.sh --check --go "${MODULES[@]}"
          fi

  lint-ui:
    name: Lint & Format Guard (UI)
    needs: detect-changes
    if: needs.detect-changes.outputs.has-ui-changes == 'true' || needs.detect-changes.outputs.has-workflow-changes == 'true' || github.event.inputs.force-all == 'true'
    runs-on: ubuntu-latest

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Set up Node.js
        uses: actions/setup-node@v4
        with:
          node-version: ${{ env.NODE_VERSION }}
          cache: 'npm'
          cache-dependency-path: services/atlas-ui/package-lock.json

      - name: Install dependencies
        working-directory: services/atlas-ui
        run: npm ci

      - name: Lint & format guard (UI)
        run: ./tools/lint.sh --check --ui
```

- [x] **Step 2: Wire both jobs into `pr-validation-complete`**

Three edits to the existing summary job:

1. `needs:` list gains `lint-go, lint-ui`:

```yaml
    needs: [detect-changes, test-go-libraries, test-go-services, test-ui, build-docker, update-pr-overlay, redis-key-guard, outbox-guard, goroutine-guard, gen-lb-ports, lint-go, lint-ui]
```

2. Result variables + table rows, alongside the existing ones:

```bash
          LINT_GO_RESULT="${{ needs.lint-go.result }}"
          LINT_UI_RESULT="${{ needs.lint-ui.result }}"
```

```bash
          echo "| Lint & Format Guard (Go) | $LINT_GO_RESULT |" >> $GITHUB_STEP_SUMMARY
          echo "| Lint & Format Guard (UI) | $LINT_UI_RESULT |" >> $GITHUB_STEP_SUMMARY
```

3. The failure `if` gains both (skipped is allowed, failure is not — same as the other gated jobs):

```bash
          if [ "$LIBS_RESULT" == "failure" ] || [ "$SERVICES_RESULT" == "failure" ] || [ "$UI_RESULT" == "failure" ] || [ "$DOCKER_RESULT" == "failure" ] || [ "$OVERLAY_RESULT" == "failure" ] || [ "$GUARD_RESULT" == "failure" ] || [ "$OUTBOX_GUARD_RESULT" == "failure" ] || [ "$GOROUTINE_GUARD_RESULT" == "failure" ] || [ "$LBPORTS_RESULT" == "failure" ] || [ "$LINT_GO_RESULT" == "failure" ] || [ "$LINT_UI_RESULT" == "failure" ]; then
```

- [x] **Step 3: Validate the workflow file parses**

```bash
python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/pr-validation.yml')); print('yaml OK')"
```

Expected: `yaml OK`. (If `actionlint` happens to be installed, run it too; do not install it just for this.)

- [x] **Step 4: Commit**

```bash
git add .github/workflows/pr-validation.yml
git commit -m "ci(task-171): Lint & Format Guard jobs (lint-go, lint-ui) in pr-validation"
```

---

### Task 7: Claude Code format-on-write hook

**Files:**
- Create: `.claude/hooks/format-on-write.sh`
- Modify: `.claude/settings.json`

**Interfaces:**
- Consumes: `tools/lint.versions` + the cached binary path convention `.cache/tools/bin/golangci-lint-$GOLANGCI_LINT_VERSION` (Task 1); atlas-ui's installed prettier (Task 2).
- Produces: automatic formatting of just-written `.go` and atlas-ui `.ts/.tsx` files during agent sessions. Deliberately fail-open (design §3.6): this hook must NEVER block an edit.

- [x] **Step 1: Write `.claude/hooks/format-on-write.sh`**

```bash
#!/usr/bin/env bash
# PostToolUse hook — format the file a Write/Edit just touched (task-171).
#
# DELIBERATELY FAIL-OPEN (design.md §3.6): a local convenience hook must never
# block an edit. Missing toolchain, missing cached binary, unparseable input,
# tool error — all exit 0 silently. CI (lint-go / lint-ui) is the enforcement
# point. To avoid a multi-minute stall on first Write, the hook never
# bootstraps golangci-lint itself; it uses the binary only if tools/lint.sh
# has already cached it.
set -u

[ -t 0 ] && exit 0

input="$(cat)"
fp="$(printf '%s' "$input" | jq -r '.tool_input.file_path // empty' 2>/dev/null)" || exit 0
[ -z "$fp" ] && exit 0
[ -f "$fp" ] || exit 0

ROOT="${CLAUDE_PROJECT_DIR:-$(pwd)}"

case "$fp" in
    *.go)
        # shellcheck source=../../tools/lint.versions
        source "$ROOT/tools/lint.versions" 2>/dev/null || exit 0
        GOLANGCI="$ROOT/.cache/tools/bin/golangci-lint-${GOLANGCI_LINT_VERSION:-}"
        [ -x "$GOLANGCI" ] || exit 0
        # Format from the file's own module dir so gofumpt sees its go.mod.
        moddir="$(dirname "$fp")"
        while [ "$moddir" != "/" ] && [ ! -f "$moddir/go.mod" ]; do
            moddir="$(dirname "$moddir")"
        done
        [ -f "$moddir/go.mod" ] || exit 0
        (cd "$moddir" && "$GOLANGCI" fmt -c "$ROOT/.golangci.yml" "$fp") >/dev/null 2>&1 || true
        ;;
    */services/atlas-ui/*.ts|*/services/atlas-ui/*.tsx)
        (cd "$ROOT/services/atlas-ui" && npx --no-install prettier --write "$fp") >/dev/null 2>&1 || true
        ;;
esac

exit 0
```

Then: `chmod +x .claude/hooks/format-on-write.sh`

- [x] **Step 2: Register it in `.claude/settings.json`**

Add a `PostToolUse` block alongside the existing `PreToolUse` one (rest of the file unchanged):

```json
    "PostToolUse": [
      {
        "matcher": "Write|Edit",
        "hooks": [
          {
            "type": "command",
            "command": "$CLAUDE_PROJECT_DIR/.claude/hooks/format-on-write.sh"
          }
        ]
      }
    ],
```

- [x] **Step 3: Verify by simulating the hook's stdin contract**

```bash
mkdir -p libs/atlas-model/scratchhook
printf 'package scratchhook\n\nfunc  Ugly( ) int {return 1}\n' > libs/atlas-model/scratchhook/x.go
CLAUDE_PROJECT_DIR="$(pwd)" bash -c 'echo "{\"tool_input\":{\"file_path\":\"$(pwd)/libs/atlas-model/scratchhook/x.go\"}}" | .claude/hooks/format-on-write.sh'
cat libs/atlas-model/scratchhook/x.go
echo '{"tool_input":{"file_path":"/nonexistent/y.go"}}' | CLAUDE_PROJECT_DIR="$(pwd)" .claude/hooks/format-on-write.sh; echo "exit=$?"
rm -rf libs/atlas-model/scratchhook
```

Expected: `x.go` comes back gofumpt-formatted (`func Ugly() int { return 1 }` with proper spacing); the nonexistent-file call prints nothing and `exit=0` (fail-open).

- [x] **Step 4: Commit**

```bash
git add .claude/hooks/format-on-write.sh .claude/settings.json
git commit -m "feat(task-171): PostToolUse format-on-write hook"
```

---

### Task 8: Documentation — CLAUDE.md checklist item + burn-down follow-up

**Files:**
- Modify: `CLAUDE.md` (Build & Verification section)
- Modify: `docs/TODO.md`

**Interfaces:**
- Consumes: the finished guard (Tasks 1–7) — the docs describe reality, so they land last-but-one.
- Produces: the FR-6.2 checklist obligation and the FR-4.4(b) tracked follow-up.

- [x] **Step 1: Add checklist item 7 to CLAUDE.md**

In the `## Build & Verification` numbered list, after item 6 (`tools/goroutine-guard.sh`), add:

```markdown
7. **`tools/lint.sh --check` clean from the repo root.** The shared lint &
   format guard (task-171): golangci-lint v2 formatters (gofumpt + goimports,
   tree-wide) and `standard` linters (rev-gated to new code) across every Go
   module, plus Prettier + ESLint for atlas-ui. Fix mode (`tools/lint.sh`,
   no flags) rewrites files in place — run it before committing. Item 2's
   standalone `go vet` is intentionally retained (it runs full-module;
   the guard's govet is diff-gated).
```

(Item 2 `go vet` stays unchanged — design decision 4.)

- [x] **Step 2: File the burn-down follow-up in docs/TODO.md**

Locate `docs/TODO.md` (Grep for "Priority Summary" to confirm the path) and add under `### High Priority (Feature Incomplete)`:

```markdown
- [ ] **Lint burn-down (task-171 follow-up)** - The Go linter layer of
  `tools/lint.sh` is rev-gated (`--new-from-rev` merge-base) so only new code
  fails CI. Burn down: fix pre-existing `standard`-group findings per module
  (run `tools/lint.sh --check --go --base <ancient-rev>` to enumerate), remove
  any escape-hatch exclusions in `.golangci.yml` marked "task-171 burn-down",
  then delete the `--new-from-rev` gating from `tools/lint.sh` so the linter
  layer enforces whole-tree like the formatters already do.
```

- [x] **Step 3: Commit**

```bash
git add CLAUDE.md docs/TODO.md
git commit -m "docs(task-171): lint guard verification checklist item + burn-down follow-up"
```

---

### Task 9: End-to-end verification (idempotence, clean gate, deliberate failure)

**Files:**
- None kept — scratch commits only, dropped before the branch is done.

**Interfaces:**
- Consumes: everything.
- Produces: the evidence for the PRD acceptance criteria. The CI-observation half of the deliberate-failure criterion runs at PR time (see Step 4) — it needs an open PR to show a red check.

- [x] **Step 1: Full clean gate + idempotence**

```bash
source ~/.nvm/nvm.sh && nvm use 22
tools/lint.sh --check && echo "CHECK CLEAN"
tools/lint.sh && git diff --exit-code && echo "FIX MODE IDEMPOTENT"
```

Expected: both echo lines print. Fix mode over the whole clean tree must produce zero diff.

- [x] **Step 2: Existing guards + verification checklist still pass**

```bash
tools/redis-key-guard.sh && tools/goroutine-guard.sh && tools/outbox-guard.sh
tools/gen-lb-ports.sh --check && tools/check-version-coverage.sh
```

Expected: all exit 0. (The Go/UI test sweeps already ran in Tasks 4/5; re-run any module you have touched since.)

- [x] **Step 3: Deliberate-failure exercise, local half**

```bash
mkdir -p libs/atlas-model/scratchlint
cat > libs/atlas-model/scratchlint/scratch.go <<'EOF'
package scratchlint

import "os"

func Bad( ) {
os.Open("nope")
}
EOF
cat > services/atlas-ui/src/scratch-lint-check.tsx <<'EOF'
export const  scratchLintCheck    =   1
EOF
git add -A && git commit -m "scratch(task-171): deliberate lint/format violations — WILL BE DROPPED"
tools/lint.sh --check; echo "exit=$?"
```

Expected: `FMT FAIL — libs/atlas-model`, `LINT FAIL — libs/atlas-model`, `UI FMT FAIL`, summary naming all three, `exit=1`.

- [ ] **Step 4: Deliberate-failure exercise, CI half (deferred to PR time)**

This step executes during `superpowers:finishing-a-development-branch`, after the PR is opened: push the branch WITH the scratch commit, confirm the PR shows `Lint & Format Guard (Go)` and `Lint & Format Guard (UI)` both red (screenshot/link the failed runs for the task record), then drop the scratch commit and force-push:

```bash
git reset --hard HEAD~1
git push --force-with-lease origin task-171-lint-format-enforcement
```

Confirm both guard jobs then go green on the re-run. If executing this plan before any PR exists, perform Step 3, then `git reset --hard HEAD~1` immediately, and leave this step's CI half as the documented final action of the PR checklist — it is the one acceptance criterion that structurally cannot run pre-PR.

- [x] **Step 5: Confirm the tree is back to clean**

```bash
git log --oneline -3        # no scratch commit
git status --porcelain      # empty
tools/lint.sh --check       # lint.sh: OK
```

---

## Acceptance-Criteria Traceability (PRD §10)

| Criterion | Where |
|---|---|
| `tools/lint.sh` exists, fix + `--check` + `--help` | Task 1 |
| Fix mode applies all tools; idempotent | Tasks 1, 4 (Step 3), 9 (Step 1) |
| `--check` non-zero + readable file list / 0 on clean | Task 1 (Step 6), 9 |
| Versions pinned in one place, CI + local identical | Task 1 (pin file), Task 6 (cache key + script reuse), Task 2 (exact npm pins) |
| Root `.golangci.yml`, per-module, no `go work sync` | Task 1 |
| Prettier + config + ignore + scripts + eslint-config-prettier | Task 2 |
| Isolated formatter-only baseline commit; formatter check passes tree-wide | Task 4 |
| §4.4(b) mechanism implemented, branch green | Tasks 1 (`--new-from-rev`), 5 |
| CI jobs gated on detect-changes; broken commit shown failing | Tasks 6, 9 (Steps 3–4) |
| Claude hook configured | Task 7 |
| CLAUDE.md checklist item | Task 8 |
| Existing verifications still pass | Tasks 4 (Step 4), 5 (Step 4), 9 (Step 2) |
| Follow-up filed for baseline burn-down | Task 8 (Step 2) |
