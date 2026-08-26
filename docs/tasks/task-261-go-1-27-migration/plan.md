# Go 1.27 Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move every Go/Alpine/golangci-lint version pin in the repository to a single declared source of truth at Go 1.27.0 / Alpine 3.24 / golangci-lint v2.13.1, and add a guard that fails the build when any pin site disagrees with it.

**Architecture:** `tools/lint.versions` is renamed to `tools/toolchain.versions` and gains `GO_VERSION` and `ALPINE_VERSION`. The ~110 pin sites that no file format can cross-reference (103 `go.mod` directives, `go.work`, four Dockerfile `ARG` defaults, two bake variables, six CI pins, one README row) are edited mechanically and then machine-checked against that file by a new pure-bash `tools/toolchain-pin-guard.sh`, wired into `tools/verify.sh` (path-gated) and into `.github/workflows/pr-validation.yml` as its own job. One pin site — the `go.work` directive synthesized inside `Dockerfile`'s builder stage — is made *derived* from `ARG GO_VERSION` rather than policed. `renovate.json` gains two last-position override rules so a Go bump can never auto-merge past the guard.

**Tech Stack:** Go 1.27.0 (`go mod edit` / `go work edit` / `go work sync`), Bash (guard scripts, `tools/verify.sh`), Docker/BuildKit + `docker-bake.hcl`, GitHub Actions YAML, golangci-lint v2.13.1, Renovate JSON config.

**Spec:** `docs/tasks/task-261-go-1-27-migration/design.md` (PRD: `docs/tasks/task-261-go-1-27-migration/prd.md`; site inventory: `docs/tasks/task-261-go-1-27-migration/pin-inventory.md`)

## Global Constraints

Copied verbatim from the spec. Every task's requirements implicitly include these.

- **Target values.** `GO_VERSION=1.27.0`, `ALPINE_VERSION=3.24`, `GOLANGCI_LINT_VERSION=v2.13.1`.
- **Patch-precise form.** Every `go` directive is `go 1.27.0`, never `go 1.27` (FR-1.2). `tools/packet-audit`'s minor-only `go 1.24` normalizes to the patch-precise form.
- **No `toolchain` directives.** No `toolchain` line is added to any `go.mod` (FR-1.3, AC-2). The repo has none today.
- **No `go mod tidy` sweep.** FR-1.4 is amended by design §5: `go mod tidy` cannot run standalone on a service module in this repo (pre-existing, proven on the unmodified tree at design §1.3). Use `go mod edit -go=1.27.0` per module, `go work edit -go=1.27.0` + `go work sync` once at the root.
- **Expected `go.sum` churn: none.** The `go` directive selects the language version and the pruning mode; pruning has been constant since 1.17. If any `go.sum` changes, STOP and explain — do not commit it blind (design §5).
- **Fixtures are never bumped (FR-7).** The 8 `go.mod` files under `tools/cideps/testdata/`, the embedded string at `tools/cideps/graph_test.go:14`, and the temp-repo fixture at `tools/plan-context_test.sh:50` all keep their current version strings.
- **No `.go` source change (AC-16)** other than (a) lint-driven fixes required by FR-5.3 and (b) new guard source. Any other `.go` change is a scope violation.
- **No behavioral change** to any service. No Go 1.27 language feature, stdlib API, or `GOEXPERIMENT` is adopted.
- **Guard CLI contract.** `tools/<name>-guard.sh`, exit 0 clean, non-zero on violation, one `file:line: expected X, got Y` line per violation (PRD §5).
- **Do not rewrite history.** Historical `docs/tasks/task-171-*` artifacts referencing `tools/lint.versions` are correct as history and must not be edited.

---

## Task 1: Rename the pin file and make it the source of truth

### Files

- `tools/toolchain.versions` — **new file** (produced by `git mv tools/lint.versions tools/toolchain.versions`, then edited)
- `tools/lint.sh` — lines 19, 20, 38: shellcheck directive, `source` path, usage text
- `.claude/hooks/format-on-write.sh` — lines 28-29: shellcheck directive + `source` path
- `.golangci.yml` — line 12: comment reference only
- `.github/workflows/pr-validation.yml` — line 478: `hashFiles('tools/lint.versions')` → `hashFiles('tools/toolchain.versions')`

Patterns to copy: the file's existing shape at `tools/lint.versions:1-5` (comment block + shell-sourceable `KEY=value`).

Module root: none — no Go module is touched by this task.

### Interfaces

- **Produces:** `tools/toolchain.versions`, a shell-sourceable file declaring exactly `GO_VERSION=1.27.0`, `ALPINE_VERSION=3.24`, `GOLANGCI_LINT_VERSION=v2.13.1`. Task 7's guard sources this file; Tasks 4/5/6 must match these literals.
- **Consumes:** nothing.

- [ ] **Step 1: Rename the file**

```bash
git mv tools/lint.versions tools/toolchain.versions
```

- [ ] **Step 2: Write the new contents**

Replace the whole file with exactly this (design §2):

```sh
# tools/toolchain.versions — the single source of truth for the repo's
# toolchain pins (task-261). Shell-sourceable KEY=value; read by tools/lint.sh,
# .claude/hooks/format-on-write.sh, tools/toolchain-pin-guard.sh, and hashed by
# the CI lint-tools cache key.
#
# Nothing here is read at build time by go.mod, go.work, Dockerfile ARG
# defaults, or docker-bake.hcl — none of those formats can read an external
# file. Those pins are duplicated on purpose and machine-checked against this
# file by tools/toolchain-pin-guard.sh. Bump here, run the guard, fix what it
# names.
#
# gofumpt/goimports versions are embedded in the golangci-lint release;
# Prettier is pinned exactly in services/atlas-ui/package.json (package.json +
# lockfile own Node tooling). Originally tools/lint.versions (task-171).
GO_VERSION=1.27.0
ALPINE_VERSION=3.24
GOLANGCI_LINT_VERSION=v2.13.1
```

- [ ] **Step 3: Update the five consumers**

`tools/lint.sh:19-20`:

```sh
# shellcheck source=toolchain.versions
source "$ROOT/tools/toolchain.versions"
```

`tools/lint.sh:38` (inside the `usage()` heredoc):

```
Versions are pinned in tools/toolchain.versions. Exit: 0 clean, 1 violations, 2 usage.
```

`.claude/hooks/format-on-write.sh:28-29`:

```sh
        # shellcheck source=../../tools/toolchain.versions
        source "$ROOT/tools/toolchain.versions" 2>/dev/null || exit 0
```

`.golangci.yml:12` — the comment currently reads `  # tools/lint.versions.`; change to `  # tools/toolchain.versions.`

`.github/workflows/pr-validation.yml:478`:

```yaml
          key: lint-tools-${{ runner.os }}-${{ hashFiles('tools/toolchain.versions') }}
```

- [ ] **Step 4: Verify no live reference to the old name survives**

Run: `grep -rn "lint\.versions" --exclude-dir=.git .`

Expected: matches ONLY under `docs/tasks/task-171-lint-format-enforcement/` and `docs/tasks/task-261-go-1-27-migration/` (prd.md/design.md/pin-inventory.md prose). Those are historical/spec artifacts and MUST NOT be edited. Zero matches under `tools/`, `.claude/`, `.github/`, or `.golangci.yml`.

- [ ] **Step 5: Verify the file sources cleanly and carries all three values**

Run:

```bash
bash -c 'set -euo pipefail; source tools/toolchain.versions; echo "$GO_VERSION $ALPINE_VERSION $GOLANGCI_LINT_VERSION"'
```

Expected stdout, exactly: `1.27.0 3.24 v2.13.1`

- [ ] **Step 6: Verify the format-on-write hook still works after the rename**

Design §9 flags this: the hook swallows a bad `source` path with `2>/dev/null || exit 0`, so a broken rename is silent and `verify.sh` would not catch it. Prove the path resolves:

```bash
bash -c 'set -euo pipefail; ROOT="$(pwd)"; source "$ROOT/tools/toolchain.versions" 2>/dev/null || { echo "HOOK PATH BROKEN"; exit 1; }; echo "HOOK PATH OK: $GOLANGCI_LINT_VERSION"'
```

Expected stdout: `HOOK PATH OK: v2.13.1`

- [ ] **Step 7: Shellcheck the two edited scripts**

Run: `shellcheck tools/lint.sh .claude/hooks/format-on-write.sh`

Expected: exit 0, no output. (If shellcheck reports a pre-existing finding unrelated to lines 19-20/28-29, record it and do not fix it — AC-16 scope.)

- [ ] **Step 8: Commit**

```bash
git add tools/toolchain.versions tools/lint.sh .claude/hooks/format-on-write.sh .golangci.yml .github/workflows/pr-validation.yml
git commit -m "chore(task-261): rename lint.versions to toolchain.versions and pin Go 1.27.0"
```

---

## Task 2: Bump all 103 module directives and the workspace

### Files

- `libs/*/go.mod`, `libs/atlas-constants/gen/go.mod` (20 modules) — directive only
- `services/*/atlas.com/*/go.mod` and `services/atlas-kafka-precreate/go.mod` (67 modules) — directive only
- `tools/*/go.mod` (16 modules, including the 8 non-workspace guard analyzer modules) — directive only
- `go.work` — line 1 directive, via `go work edit`
- `go.work.sum` — regenerated by `go work sync`; commit whatever delta results
- `tools/cideps/testdata/**/go.mod` (8 files) — **read-only; MUST NOT be edited** (FR-7.1)
- `tools/cideps/graph_test.go` — **read-only; MUST NOT be edited** (FR-7.2, the `go 1.25.5` string at line 14)
- `docs/tasks/task-261-go-1-27-migration/pin-inventory.md` — read-only; the authoritative worklist

Module roots: every directory containing an edited `go.mod`; the sweep is driven from the repo root.

This task deliberately exceeds the 6-file guidance: it is one mechanical edit applied uniformly to 103 files by `go mod edit`, with no per-file judgement. See `context.md`.

### Interfaces

- **Consumes:** `GO_VERSION=1.27.0` from `tools/toolchain.versions` (Task 1).
- **Produces:** a tree where every non-fixture `go.mod` and `go.work` declares `go 1.27.0`. Task 3 measures the toolchain blast radius against this state; Task 7's guard asserts it.

- [ ] **Step 1: Snapshot the starting state (this is the "failing test")**

Run:

```bash
for f in $(git ls-files '*go.mod' 'go.mod'); do printf '%s\t%s\n' "$(grep -m1 '^go ' "$f")" "$f"; done | sort | awk -F'\t' '{print $1}' | uniq -c
```

Expected (design §1 / prd.md FR-1.1 table, plus the 8 fixtures):

| directive | count |
|---|---|
| `go 1.24` | 1 |
| `go 1.24.4` | 4 |
| `go 1.25.0` | 1 |
| `go 1.25.5` | 104 (96 in-scope + 8 fixtures) |
| `go 1.26.0` | 1 |
| **total** | **111** |

If the total is not 111, STOP — the module set moved since the inventory was taken and the plan's counts need re-deriving before any edit.

- [ ] **Step 2: Apply `go mod edit` to every non-fixture module**

`go mod edit -go=` writes the canonical patch-precise form, normalizes `tools/packet-audit`'s minor-only `go 1.24` without a regex, and touches nothing else in the file (design §5). Run from the repo root:

```bash
set -euo pipefail
for f in $(git ls-files '*go.mod' 'go.mod'); do
    case "$f" in
        tools/cideps/testdata/*) continue ;;   # FR-7.1 fixtures — never bumped
    esac
    go mod edit -go=1.27.0 -C "$(dirname "$f")"
done
```

If `go mod edit -C` is unavailable in this toolchain, use a subshell per module instead — `( cd "$(dirname "$f")" && go mod edit -go=1.27.0 )` — and do not change what is edited.

- [ ] **Step 3: Verify all 103 in-scope modules moved and all 8 fixtures did not**

Run:

```bash
for f in $(git ls-files '*go.mod' 'go.mod'); do printf '%s\t%s\n' "$(grep -m1 '^go ' "$f")" "$f"; done | sort | awk -F'\t' '{print $1}' | uniq -c
```

Expected, exactly two lines:

| directive | count |
|---|---|
| `go 1.25.5` | 8 |
| `go 1.27.0` | 103 |

Then confirm the 8 remaining are exactly the fixtures:

Run: `git ls-files 'tools/cideps/testdata/*go.mod' | wc -l`

Expected: `8`

- [ ] **Step 4: Verify no `toolchain` directive was introduced (AC-2)**

Run: `grep -rn '^toolchain ' $(git ls-files '*go.mod' 'go.mod')`

Expected: no matches, exit 1 from grep. Any match is an AC-2 violation — remove it.

- [ ] **Step 5: Edit the workspace directive**

`go.mod` edits land BEFORE the workspace is validated: Go rejects a workspace whose directive is below a member's, so doing `go.work` first leaves the tree transiently unbuildable and makes any mid-sweep `go build` misleading (design §5).

```bash
go work edit -go=1.27.0
```

Verify: `sed -n '1p' go.work` → expected `go 1.27.0`

- [ ] **Step 6: Sync the workspace**

```bash
go work sync
```

Then inspect the delta:

```bash
git diff --stat -- go.work.sum go.work
git diff -- $(git ls-files '*go.sum')
```

Expected: `go.work.sum` may drop the stale `github.com/golangci/golangci-lint/v2 v2.12.2` entries currently at `go.work.sum:273-274` (residue from a past `go install`; no tracked `go.mod` requires golangci-lint). That churn is benign and in scope — call it out in the PR description rather than reverting it.

Expected `go.sum` delta: **empty**. If any `go.sum` changed, STOP and explain what moved — that means something other than the directive changed, which AC-16 treats as a scope violation (design §5).

- [ ] **Step 7: Verify the workspace builds**

```bash
go build ./...
```

Expected: exit 0, no output. (Run from the repo root, workspace mode active.)

- [ ] **Step 8: Verify the excluded fixtures still pass their own tests (AC-13)**

```bash
cd tools/cideps && go test ./...
```

Expected: `ok  github.com/Chronicle20/atlas/tools/cideps` (plus `no test files` lines for any package without tests). `tools/cideps/graph_test.go:14` still reads `go 1.25.5` and `TestParseAtlasRequires_DirectAndIndirect`, `TestParseAtlasRequires_MalformedFails`, `TestBuildGraph_Simple`, `TestClosure_Transitive` all pass unchanged.

- [ ] **Step 9: Verify the 8 non-workspace guard modules also moved**

These sit outside `go.work`'s 95 `use` entries and are the easiest to miss with a workspace-only sweep (pin-inventory §A.3):

```bash
for m in atlasguards buffdurationguard envguard goroutineguard outboxguard producerseamguard rediskeyguard scopeguard; do
    printf '%s\t%s\n' "$(grep -m1 '^go ' "tools/$m/go.mod")" "tools/$m/go.mod"
done
```

Expected: eight lines, each starting `go 1.27.0`.

- [ ] **Step 10: Commit**

```bash
git add -A -- '*go.mod' go.work go.work.sum
git commit -m "chore(task-261): bump 103 module directives and go.work to go 1.27.0"
```

---

## Task 3: Measure and resolve the Go 1.27 / golangci-lint v2.13.1 blast radius

### Files

- `.golangci.yml` — only if FR-5.3 requires a commented exclusion
- `docs/TODO.md` — only if an exclusion is added; it must name a burn-down entry
- `docs/tasks/task-261-go-1-27-migration/evidence-lint.md` — **new file**; captured run output (AC-9)
- Any `.go` file a trivial, mechanical lint fix requires — permitted by AC-16(a), nothing else

Patterns to copy: the existing burn-down section in `docs/TODO.md` ("Lint burn-down", referenced by `tools/lint.sh:9-10`).

Module roots: driven from the repo root by `tools/lint.sh`, which walks modules in workspace mode.

### Interfaces

- **Consumes:** `GOLANGCI_LINT_VERSION=v2.13.1` (Task 1); the 1.27.0 module set (Task 2).
- **Produces:** `docs/tasks/task-261-go-1-27-migration/evidence-lint.md` holding the recorded `tools/lint.sh --check` output that AC-9 requires. Task 10 cites it.

This is PRD open question 3 — the one item that can materially change the size of the task. Design §9 requires measuring it **first, immediately after the directive sweep, while the branch is still small**. Do this before Tasks 4-9.

- [ ] **Step 1: Measure the linter blast radius**

```bash
tools/lint.sh --check 2>&1 | tee /tmp/t261-lint-check.log; echo "LINT_EXIT=${PIPESTATUS[0]}"
```

Record the exit code and the full finding list. `tools/lint.sh` will download the v2.13.1 release asset on first run (the pin changed in Task 1); release-asset availability was confirmed during design (§1.2).

- [ ] **Step 2: Measure the `go vet` blast radius**

`go vet` ships inside the toolchain, so 1.27 may report findings 1.26 did not:

```bash
go vet ./... 2>&1 | tee /tmp/t261-govet.log; echo "VET_EXIT=${PIPESTATUS[0]}"
```

- [ ] **Step 3: Triage every finding against FR-5.3**

Apply the policy verbatim — no finding may be left unaddressed:

| Finding kind | Action |
|---|---|
| Trivial and mechanical (formatting, unused import, `ineffassign`, an obvious `errcheck` on a `defer Close()`) | Fix it here. Permitted `.go` change under AC-16(a). |
| Needs behavioral judgement (a real error path, a concurrency question, a semantic change) | Add a `.golangci.yml` exclusion carrying a comment that names a `docs/TODO.md` burn-down follow-up entry, and add that entry. Never a bare exclusion. |
| Volume large enough to dominate the task | STOP and escalate to the user. Design §9: "that is a scope conversation, not a silent expansion." |

- [ ] **Step 4: Re-run both to green**

```bash
tools/lint.sh --check; echo "LINT_EXIT=$?"
go vet ./...; echo "VET_EXIT=$?"
```

Expected: `LINT_EXIT=0` and `VET_EXIT=0`.

- [ ] **Step 5: Record the evidence (AC-9)**

Write `docs/tasks/task-261-go-1-27-migration/evidence-lint.md` containing, verbatim and unsummarized:

1. `golangci-lint version` output, proving the v2.13.1 binary is the one that ran.
2. The full Step 1 command line and its final exit code.
3. The full Step 2 command line and its final exit code.
4. A table of every finding triaged in Step 3: `file:line`, linter name, disposition (`fixed` / `excluded → docs/TODO.md#<anchor>`), and one line of rationale.
5. If the finding list was empty, say so explicitly and paste the zero-finding output — an empty section is not evidence.

- [ ] **Step 6: Commit**

```bash
git add docs/tasks/task-261-go-1-27-migration/evidence-lint.md
git add -u
git commit -m "chore(task-261): resolve go 1.27 / golangci-lint v2.13.1 findings"
```

If Step 3 produced no `.go`, `.golangci.yml`, or `docs/TODO.md` change, commit only the evidence file with the message `docs(task-261): record clean lint/vet run under go 1.27`.

---

## Task 4: Container build pins

### Files

- `Dockerfile` — lines 17-18 (`ARG` defaults), a new `ARG GO_VERSION` re-declaration after the `build-env` `FROM` at line 20, and line 94's synthesized-workspace `printf`
- `services/atlas-kafka-precreate/Dockerfile` — lines 9-10 (`ARG` defaults)
- `docker-bake.hcl` — lines 9 and 13 (the `default` inside the `GO_VERSION` and `ALPINE_VERSION` variable blocks)
- `services/atlas-renders/Dockerfile` — **deleted** (FR-3.5)

Module roots: none — no Go module is edited.

### Interfaces

- **Consumes:** `GO_VERSION=1.27.0`, `ALPINE_VERSION=3.24` (Task 1).
- **Produces:** four `ARG` defaults and two bake variables at the target values, and one pin site (the synthesized workspace directive) removed entirely by derivation. Task 7's guard checks the six literals and explicitly does NOT check the derived one.

- [ ] **Step 1: Bump the root Dockerfile's global ARGs**

`Dockerfile:17-18`:

```dockerfile
ARG GO_VERSION=1.27.0
ARG ALPINE_VERSION=3.24
```

`Dockerfile:148`'s runtime `FROM alpine:3.24` is already 3.24 and is unchanged. `Dockerfile:20`'s `FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION}` is an indirection — do not hand-edit it.

- [ ] **Step 2: Re-declare `ARG GO_VERSION` inside the `build-env` stage**

An `ARG` declared before the first `FROM` is global and is NOT visible inside a build stage; it must be re-declared. Insert immediately after `FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS build-env` (`Dockerfile:20`), before the existing `ARG SERVICE` at line 22:

```dockerfile
# Re-declare after FROM: a global ARG is not visible inside a build stage.
# Consumed by the synthesized go.work below so the workspace directive can
# never drift from the builder image (task-261).
ARG GO_VERSION
```

- [ ] **Step 3: Derive the synthesized workspace directive**

`Dockerfile:94` currently reads `         printf 'go 1.26.0\n\nuse (\n'; \`. Replace it with:

```dockerfile
         # task-261: NOT a pinned site — derived from ARG GO_VERSION above, so
         # tools/toolchain-pin-guard.sh deliberately does not check this line.
         # A literal here must be >= every member module's directive or the
         # workspace fails to load and every go-service image build breaks.
         printf 'go %s\n\nuse (\n' "${GO_VERSION}"; \
```

Keep the surrounding `RUN` line continuations intact — this line sits inside the `{ ... } > go.work` block at `Dockerfile:92-102`.

- [ ] **Step 4: Bump the kafka-precreate Dockerfile**

`services/atlas-kafka-precreate/Dockerfile:9-10`:

```dockerfile
ARG GO_VERSION=1.27.0
ARG ALPINE_VERSION=3.24
```

This file is live: the `atlas-kafka-precreate` bake target sets `context = "services/atlas-kafka-precreate"`, so `dockerfile = "Dockerfile"` resolves here, not to the repo-root file. It synthesizes no workspace (flat module, `COPY go.mod go.sum ./` + `go mod download`), so it needs no Step 2/3 treatment.

- [ ] **Step 5: Bump the bake variables**

`docker-bake.hcl`:

```hcl
variable "GO_VERSION" {
  default = "1.27.0"
}

variable "ALPINE_VERSION" {
  default = "3.24"
}
```

- [ ] **Step 6: Delete the dead atlas-renders Dockerfile (FR-3.5)**

```bash
git rm services/atlas-renders/Dockerfile
```

It hardcodes `golang:1.25.5-alpine3.21` at line 1 and a literal `go 1.25.5` synthesized workspace at line 17, and it is not built: `atlas-renders` is a `go-service` in `docker-bake.hcl` and is produced by the matrix `go-service` target from the repo-root `Dockerfile`. `.claude/skills/backend-dev-guidelines/resources/scaffolding-checklist.md:169` already records it as "dead, not an exception to the rule."

- [ ] **Step 7: Verify the bake config resolves**

```bash
docker buildx bake --print atlas-renders 2>&1 | head -40
```

Expected: valid JSON naming `"dockerfile": "Dockerfile"` (the repo-root one) and `"GO_VERSION": "1.27.0"` / `"ALPINE_VERSION": "3.24"` in its `args`. This is the cheap proof that Step 6's deletion did not orphan a target; AC-7's real build happens in Task 10's flagless `verify.sh`.

- [ ] **Step 8: Build one go-service image end to end**

The synthesized-workspace change is the one edit in this task that can break every image build, and only an actual build exercises it:

```bash
docker buildx bake atlas-account
```

Expected: exit 0. A failure at workspace load (`go.work` directive below a member module's) means Step 2's `ARG GO_VERSION` re-declaration is missing or misplaced — a global `ARG` that was not re-declared expands to the empty string, producing `go \n` in the synthesized file.

- [ ] **Step 9: Build the kafka-precreate image**

```bash
docker buildx bake atlas-kafka-precreate
```

Expected: exit 0. This is the only target that exercises `services/atlas-kafka-precreate/Dockerfile`.

- [ ] **Step 10: Commit**

```bash
git add Dockerfile services/atlas-kafka-precreate/Dockerfile docker-bake.hcl
git add -u services/atlas-renders/
git commit -m "chore(task-261): bump container pins to go 1.27.0 / alpine 3.24, delete dead renders Dockerfile"
```

---

## Task 5: CI workflow and composite action pins

### Files

- `.github/workflows/pr-validation.yml` — line 41: `GO_VERSION: '1.26.0'` → `'1.27.0'`
- `.github/workflows/main-publish.yml` — line 33: `GO_VERSION: '1.26.0'` → `'1.27.0'`
- `.github/workflows/catalog-lint.yml` — line 17: `GO_VERSION: '1.25.5'` → `'1.27.0'`
- `.github/workflows/packet-matrix.yml` — line 13: `GO_VERSION: '1.25.5'` → `'1.27.0'`
- `.github/actions/go-test/action.yml` — line 11: `default: '1.25.5'` → `'1.27.0'`
- `.github/actions/detect-changes/action.yml` — line 280: `go-version: '1.26.0'` → `'1.27.0'`

Module roots: none.

### Interfaces

- **Consumes:** `GO_VERSION=1.27.0` (Task 1).
- **Produces:** the six literal CI pin sites at 1.27.0. Task 7's guard enumerates these six as explicit `file:pattern` pairs.

- [ ] **Step 1: Confirm the six sites are exactly where the inventory says**

Run: `grep -rn "GO_VERSION\|go-version" .github/workflows/*.yml .github/actions/*/action.yml`

Expected: the six literal pins above, plus eight `${{ env.GO_VERSION }}` / `${{ inputs.go-version }}` indirections at `catalog-lint.yml:29`, `packet-matrix.yml:25`, `pr-validation.yml:149,424,460,668,694`, `go-test/action.yml:40`, plus the `go-version:` input *declaration* at `go-test/action.yml:8`. The indirections and the declaration MUST NOT be hand-edited.

If the line numbers have moved, edit by matched content, not by line number.

- [ ] **Step 2: Apply the six edits**

Each is a single quoted-scalar replacement. Keep the existing single quotes and indentation:

| File | Before | After |
|---|---|---|
| `.github/workflows/pr-validation.yml` | `  GO_VERSION: '1.26.0'` | `  GO_VERSION: '1.27.0'` |
| `.github/workflows/main-publish.yml` | `  GO_VERSION: '1.26.0'` | `  GO_VERSION: '1.27.0'` |
| `.github/workflows/catalog-lint.yml` | `  GO_VERSION: '1.25.5'` | `  GO_VERSION: '1.27.0'` |
| `.github/workflows/packet-matrix.yml` | `  GO_VERSION: '1.25.5'` | `  GO_VERSION: '1.27.0'` |
| `.github/actions/go-test/action.yml` | `    default: '1.25.5'` | `    default: '1.27.0'` |
| `.github/actions/detect-changes/action.yml` | `        go-version: '1.26.0'` | `        go-version: '1.27.0'` |

`.github/actions/go-test/action.yml:11` is the `default:` under the `go-version:` input at line 8 — change the default, not the input name or its description.

- [ ] **Step 3: Verify no stale Go pin remains under `.github/` (AC-8)**

Run: `grep -rn "1\.2[456]" .github/`

Expected: no line that is a Go version pin. Matches that are NOT Go pins are acceptable and must be listed explicitly rather than "fixed" — e.g. an action `@v4`-style ref, a Node version, an image digest, or a `# syntax=docker/dockerfile:1.26` directive. Record which category each surviving match falls into.

- [ ] **Step 4: Verify all six workflows still parse**

```bash
for f in .github/workflows/*.yml .github/actions/*/action.yml; do yq -e '.' "$f" >/dev/null || echo "PARSE FAIL: $f"; done; echo "YAML_OK"
```

Expected: `YAML_OK` with no `PARSE FAIL` lines. (`yq` is in the repo toolchain.)

- [ ] **Step 5: Commit**

```bash
git add .github/workflows/pr-validation.yml .github/workflows/main-publish.yml .github/workflows/catalog-lint.yml .github/workflows/packet-matrix.yml .github/actions/go-test/action.yml .github/actions/detect-changes/action.yml
git commit -m "ci(task-261): pin all six CI Go version sites to 1.27.0"
```

---

## Task 6: README prerequisites and the `GOTOOLCHAIN` callout

### Files

- `README.md` — line 79, the Prerequisites table's Go row; plus a new note beneath the table

Module roots: none.

### Interfaces

- **Consumes:** `GO_VERSION=1.27.0` (Task 1).
- **Produces:** a `| Go | 1.27.0+ | ... |` row that Task 7's guard checks with the pattern `\| Go \| ${GO_VERSION}+ \|`. The guard's regex depends on this exact cell spelling — `1.27.0+` with the trailing `+`, single spaces around the pipes.

`README.md` is the contributor-facing statement of the required toolchain (currently two minors stale at `1.25.5+`) and is the home NFR-Compatibility requires for the patch-precise / `GOTOOLCHAIN` callout.

- [ ] **Step 1: Bump the Prerequisites row**

`README.md:79` currently reads:

```
| Go | 1.25.5+ | All backend services |
```

Replace with:

```
| Go | 1.27.0+ | All backend services |
```

The surrounding rows (`Node.js | 22+`, `Docker | Latest`, `Kafka | 3.x+`, `PostgreSQL | 14+`, `Redis | 7+`) are unchanged.

- [ ] **Step 2: Add the `GOTOOLCHAIN` callout beneath the table**

Insert immediately after the Prerequisites table's last row, before whatever section follows:

```markdown
> **Go toolchain.** Every module declares a patch-precise `go 1.27.0` directive.
> With the default `GOTOOLCHAIN=auto`, a contributor on an older toolchain gets
> an automatic download of go1.27.0 and nothing else to do. With
> `GOTOOLCHAIN=local` set, the build fails with a hard error instead — that is a
> deliberate consequence of the patch-precise choice, not a bug. Install
> go1.27.0 or unset `GOTOOLCHAIN`.
>
> All toolchain versions (Go, Alpine, golangci-lint) are declared once in
> [`tools/toolchain.versions`](tools/toolchain.versions) and machine-checked
> against every pin site by `tools/toolchain-pin-guard.sh`. Bump that file, run
> the guard, and fix what it names.
```

- [ ] **Step 3: Verify the row spelling the guard will depend on**

Run: `grep -n '^| Go |' README.md`

Expected, exactly one line: `79:| Go | 1.27.0+ | All backend services |` (line number may shift; the content must match character for character).

- [ ] **Step 4: Commit**

```bash
git add README.md
git commit -m "docs(task-261): bump README Go prerequisite to 1.27.0 and document GOTOOLCHAIN"
```

---

## Task 7: Write `tools/toolchain-pin-guard.sh`

### Files

- `tools/toolchain-pin-guard.sh` — **new file**, executable, pure bash
- `tools/service-name-guard.sh` — read-only; the shell-guard shape to copy
- `tools/npc-shop-contract-mirror-guard.sh` — read-only; `ROOT="$(cd "$(dirname "$0")/.." && pwd)"` + violation accumulator idiom
- `tools/toolchain.versions` — read-only; the values the guard sources (new file, created by Task 1)

Patterns to copy: `tools/npc-shop-contract-mirror-guard.sh:1-25` (header comment stating why the guard exists, `set -euo pipefail`, `ROOT=` resolution, `rc=0` accumulator); `tools/service-name-guard.sh:44-60` (accumulate → print → summary line → exit).

Module roots: none — pure bash, no Go, no `python3`. Design §3: the CI job must be `checkout` + `run` with no `setup-go`, because this guard's whole purpose is to be the thing that still works when the Go pin is wrong.

### Interfaces

- **Consumes:** `tools/toolchain.versions` (Task 1); the pin sites moved by Tasks 2, 4, 5, 6.
- **Produces:** `tools/toolchain-pin-guard.sh`, exit 0 clean / 1 on violation, one `file:line: expected X, got Y` line per violation, plus a `--selftest` flag. Task 8 wires it into `tools/verify.sh` and CI by that exact path and contract.

- [ ] **Step 1: Write the guard**

Full required behavior. Follow the repo's shell-guard shape exactly.

**Header comment** must state (design §4.1): the repo pins its Go/Alpine/golangci-lint versions in ~110 places that no format can cross-reference; before this guard, a partial Renovate bump left the tree building against three different Go versions at once with nothing failing; `tools/toolchain.versions` is the source of truth.

**Preamble:**

```sh
#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
# shellcheck source=toolchain.versions
source "$ROOT/tools/toolchain.versions"
```

**Exemptions**, as a literal array with the rationale inline (FR-6.4, FR-7.3 — the comment text is required, not optional):

```sh
# FR-7 (task-261): fixture data, NOT build inputs. tools/cideps' tests assert
# against these trees and the version string is incidental to what they
# exercise; bumping them would break `go test ./...` in tools/cideps for no
# gain. Do NOT "fix" these to match the pin file.
#
# Two more fixture strings live outside this list because they are not tracked
# go.mod files and `git ls-files '*go.mod'` never yields them — recorded here
# so a future sweep does not mistake them for misses:
#   tools/cideps/graph_test.go:14   `go 1.25.5` inside a go.mod string literal
#   tools/plan-context_test.sh:50   `go 1.24` written into a temp fixture repo
EXEMPT_GOMOD=(
    tools/cideps/testdata/  # both simple/ and transitive/ trees (8 files)
)
```

Match `EXEMPT_GOMOD` entries as **path prefixes**, not literal paths, so a ninth file added to the same fixture corpus inherits the exemption.

**Checked set** — exactly these seven classes (design §3.1):

| Class | How the guard finds the sites | Assertion |
|---|---|---|
| Module directives | `git ls-files '*go.mod' 'go.mod'`, minus `EXEMPT_GOMOD` prefixes | the `go ` line matches `^go ${GO_VERSION}$` exactly |
| Workspace | `go.work` line 1 | matches `^go ${GO_VERSION}$` |
| Dockerfile ARGs | `Dockerfile` and `services/atlas-kafka-precreate/Dockerfile` | `ARG GO_VERSION=${GO_VERSION}` and `ARG ALPINE_VERSION=${ALPINE_VERSION}` |
| Bake variables | `docker-bake.hcl` | `default = "${GO_VERSION}"` inside the `GO_VERSION` block; `default = "${ALPINE_VERSION}"` inside the `ALPINE_VERSION` block |
| CI pins | the six explicit `file:pattern` pairs below | the quoted scalar equals `${GO_VERSION}` |
| Lint pin | `tools/toolchain.versions` itself | `GOLANGCI_LINT_VERSION` is set and non-empty (presence check only — the pin file *is* the source) |
| README | `README.md` | a line matching `^\| Go \| ${GO_VERSION}\+ \|` exists |

The six CI sites are enumerated explicitly, never by pattern — `.github/**` contains many unrelated version strings and an over-broad pattern there produces false positives that get "fixed" by loosening the guard:

```sh
CI_SITES=(
    ".github/workflows/pr-validation.yml|GO_VERSION:"
    ".github/workflows/main-publish.yml|GO_VERSION:"
    ".github/workflows/catalog-lint.yml|GO_VERSION:"
    ".github/workflows/packet-matrix.yml|GO_VERSION:"
    ".github/actions/go-test/action.yml|default:"
    ".github/actions/detect-changes/action.yml|go-version:"
)
```

For `go-test/action.yml` the `default:` key appears more than once in the file (`race-detection` also has one); scope the match to the `go-version:` input's block, or match the exact line `    default: '<version>'` that follows `  go-version:` — do not match the first `default:` in the file.

**Exact match, not prefix match.** `^go 1\.27\.0$` must reject both `go 1.27` (the minor-only form FR-1.2 forbids) and `go 1.27.1`. Build the regex by escaping `.` in `$GO_VERSION`; do not interpolate it raw.

**`Dockerfile:94` is NOT checked.** Design §6: it is derived from `ARG GO_VERSION`, and parsing a `printf` format string inside a line-continued `RUN` yields a guard that passes while the bake fails. A comment in the guard must say so.

**Vacuous-pass protection** (design §9): after enumeration, assert the module count is non-zero and fail loudly if it is not — a guard that checked zero files must fail, not pass.

```sh
if [ "$checked_gomod" -lt 1 ]; then
    echo "toolchain-pin-guard: FAIL — enumerated 0 go.mod files; the git ls-files glob is broken" >&2
    exit 1
fi
```

**Output contract:** one line per violation, `path:line: expected X, got Y`, on stdout. On any violation print a summary line and `exit 1`. Clean run prints `toolchain-pin-guard: clean (N go.mod + go.work + 4 Dockerfile ARGs + 2 bake vars + 6 CI pins + README checked)` and exits 0.

**`--selftest` flag** (design §3.3): copy the tree's checked files into a `mktemp -d`, mutate one copied `go.mod` to `go 1.26.0`, run the checks against the copy, and assert BOTH a non-zero exit AND a violation line matching `expected 1\.27\.0, got 1\.26\.0`. It MUST mutate nothing under `$ROOT`. Print `toolchain-pin-guard: selftest PASS` and exit 0 when the mutation was correctly detected; print `selftest FAIL — mutation not detected` and exit 1 when it was not.

- [ ] **Step 2: Make it executable**

```bash
chmod +x tools/toolchain-pin-guard.sh
```

- [ ] **Step 3: Shellcheck it**

```bash
shellcheck tools/toolchain-pin-guard.sh
```

Expected: exit 0, no output.

- [ ] **Step 4: Run it against the branch — expect clean**

```bash
tools/toolchain-pin-guard.sh; echo "GUARD_EXIT=$?"
```

Expected: `GUARD_EXIT=0` and the clean summary line naming **103** go.mod files checked.

If it reports violations, they are real — Tasks 2/4/5/6 missed a site. Fix the site, not the guard.

- [ ] **Step 5: Run the selftest — expect PASS**

```bash
tools/toolchain-pin-guard.sh --selftest; echo "SELFTEST_EXIT=$?"
```

Expected: `toolchain-pin-guard: selftest PASS` and `SELFTEST_EXIT=0`.

- [ ] **Step 6: Prove the selftest mutates nothing under the repo**

```bash
git status --porcelain
```

Expected after Step 5: no modification to any `go.mod`, `go.work`, `Dockerfile`, `docker-bake.hcl`, workflow, or `README.md`. Only `tools/toolchain-pin-guard.sh` itself should be untracked/new.

- [ ] **Step 7: Prove the guard fails on a real live mutation (AC-11)**

A guard that has only ever passed is not verified. Mutate a real tracked file, capture the failure verbatim, restore:

```bash
go mod edit -go=1.26.0 -C libs/atlas-retry
tools/toolchain-pin-guard.sh 2>&1 | tee /tmp/t261-guard-fail.log; echo "GUARD_EXIT=${PIPESTATUS[0]}"
```

Expected: `GUARD_EXIT=1` and a line of the form `libs/atlas-retry/go.mod:3: expected go 1.27.0, got go 1.26.0` (the line number is whatever the `go ` directive sits on in that file).

Restore and confirm:

```bash
go mod edit -go=1.27.0 -C libs/atlas-retry
git diff --exit-code -- libs/atlas-retry/go.mod; echo "RESTORED=$?"
tools/toolchain-pin-guard.sh; echo "GUARD_EXIT=$?"
```

Expected: `RESTORED=0` and `GUARD_EXIT=0`.

- [ ] **Step 8: Record the AC-11 evidence**

Write `docs/tasks/task-261-go-1-27-migration/evidence-guard.md` — **new file** — containing verbatim:

1. The Step 4 clean run and its exit code.
2. The Step 5 selftest output and its exit code.
3. The Step 7 mutation, the FULL failing output pasted (not summarized), the exit code, and the restore confirmation.

AC-11 requires the failing output be recorded; a paraphrase does not satisfy it.

- [ ] **Step 9: Commit**

```bash
git add tools/toolchain-pin-guard.sh docs/tasks/task-261-go-1-27-migration/evidence-guard.md
git commit -m "feat(task-261): add tools/toolchain-pin-guard.sh drift guard"
```

---

## Task 8: Wire the guard into verify.sh, CI, and the docs

### Files

- `tools/verify.sh` — insert a new path-gated `step` block between the service name guard block (`:431-435`) and the template guards block (`:437`)
- `.github/workflows/pr-validation.yml` — add a `toolchain-pin-guard` job
- `docs/verification.md` — add the guard to the "Path-gated" inventory (the section begins at `:199`)
- `tools/toolchain-pin-guard.sh` — read-only; the script being wired (new file, created by Task 7)

Patterns to copy: `tools/verify.sh:431-435` (the service name guard's `if touched ... step ... else skip ... fi` shape); `.github/workflows/pr-validation.yml:207-218` (the `npc-shop-contract-mirror-guard` job); `docs/verification.md:201-202` (the "Registration lists" path-gated entry's one-line form).

Module roots: none.

### Interfaces

- **Consumes:** `tools/toolchain-pin-guard.sh` (Task 7), by path and by its exit-0/non-zero contract.
- **Produces:** the guard runs under `tools/verify.sh` and as a standalone CI job. Task 10's flagless `verify.sh` run depends on this wiring.

- [ ] **Step 1: Add the `verify.sh` step**

Insert after `fi` at `tools/verify.sh:435` and before the template-guards `if` at `:437`. The `touched` and `step`/`skip` helpers are defined at `tools/verify.sh:147`, `:94`, `:109`.

```sh
# task-261: the repo pins its Go/Alpine/golangci-lint versions in ~110 places
# that no format can cross-reference (go.mod and go.work directives, Dockerfile
# ARG defaults, bake variables, workflow env). Before this guard, a partial
# Renovate bump left the tree building against three different Go versions at
# once with nothing failing. tools/toolchain.versions is the source of truth;
# this asserts every pin site agrees with it.
if touched '(^|/)go\.mod$|^go\.work$|^Dockerfile$|/Dockerfile$|^docker-bake\.hcl$|^\.github/|^tools/toolchain\.versions$|^tools/toolchain-pin-guard\.sh$|^README\.md$'; then
    step "toolchain pin guard" ./tools/toolchain-pin-guard.sh
else
    skip "toolchain pin guard (no pin site changed)"
fi
```

The predicate includes the guard's own source and the pin file, mirroring the self-inclusion at `verify.sh:394` (scope guard) and `:411` (producer seam guard). `README.md` is included because Task 6 made it a checked site.

- [ ] **Step 2: Add the CI job**

CI does not run `tools/verify.sh` — it enumerates each shell guard as its own job — so FR-6.5's conditional resolves to its second branch. Add to `.github/workflows/pr-validation.yml`, alongside `npc-shop-contract-mirror-guard` (`:207`) and `npc-conversation-contract-mirror-guard` (`:228`):

```yaml
  # ============================================
  # Toolchain Pin Guard
  # Asserts every Go/Alpine/golangci-lint pin site agrees with
  # tools/toolchain.versions. The versions are duplicated across ~110 sites
  # (103 go.mod directives, go.work, 4 Dockerfile ARGs, 2 bake vars, 6 CI
  # pins, README) that no file format can cross-reference, so a partially
  # landed bump leaves the tree building against two Go versions with nothing
  # failing. task-261.
  # ============================================
  toolchain-pin-guard:
    name: Toolchain Pin Guard
    needs: detect-changes
    if: needs.detect-changes.outputs.deploy-only != 'true'
    runs-on: ubuntu-latest

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: toolchain pin guard
        run: ./tools/toolchain-pin-guard.sh
```

Deliberately **no `setup-go` step** — the guard is pure shell by design, because it must still work when the Go pin is the thing that is wrong. Gated only on `deploy-only`, like its two siblings: a `deploy/`-only change set cannot move a Go pin, and everything else can.

- [ ] **Step 3: Document it**

Add to `docs/verification.md`'s "Path-gated" section (begins at `:199`), matching the one-line form of the "Registration lists" entry at `:201-202`:

```markdown
**Toolchain pins** — `toolchain-pin-guard.sh`, when a `go.mod`, `go.work`,
`Dockerfile`, `docker-bake.hcl`, anything under `.github/`, `README.md`,
`tools/toolchain.versions`, or the guard's own source changed. Asserts every Go
/ Alpine / golangci-lint pin site agrees with `tools/toolchain.versions`.
Silent failure it prevents: a partially landed toolchain bump leaves the tree
building against two Go versions at once with nothing failing — which is how the
1.24/1.25/1.26 spread task-261 collapsed came to exist. The `tools/cideps`
fixtures are exempt by path prefix (FR-7); the synthesized workspace directive
in `Dockerfile` is derived from `ARG GO_VERSION`, not checked.
```

- [ ] **Step 4: Verify the wiring fires**

The step is path-gated, so force it with `--all` semantics or by having a pin site in the diff. Confirm the step is registered and runs:

```bash
tools/verify.sh --quick 2>&1 | grep -i "toolchain pin guard"
```

Expected: a line showing the step either ran (with an OK/PASS marker) or was explicitly skipped with the reason `no pin site changed`. If neither string appears, the block was inserted outside the executed region — fix the placement.

- [ ] **Step 5: Shellcheck verify.sh**

```bash
shellcheck tools/verify.sh
```

Expected: exit 0, or only findings that predate this branch. Do not fix pre-existing findings (AC-16 scope).

- [ ] **Step 6: Verify the workflow still parses**

```bash
yq -e '.jobs["toolchain-pin-guard"].steps[1].run' .github/workflows/pr-validation.yml
```

Expected stdout: `./tools/toolchain-pin-guard.sh`

- [ ] **Step 7: Commit**

```bash
git add tools/verify.sh .github/workflows/pr-validation.yml docs/verification.md
git commit -m "feat(task-261): wire toolchain pin guard into verify.sh and CI"
```

---

## Task 9: Stop Renovate auto-merging Go toolchain bumps

### Files

- `renovate.json` — append two override rules at the END of `packageRules` (currently ends at line 98, array closes at `:99`)
- `docs/tasks/task-261-go-1-27-migration/renovate-trace.md` — **new file**; the AC-14 reasoning trace

Module roots: none.

### Interfaces

- **Consumes:** nothing from earlier tasks.
- **Produces:** a `packageRules` array whose LAST two entries set `automerge: false` for the `go` gomod package and the `golang` dockerfile package. Position is the enforcement mechanism — later rules win in Renovate.

Editing the existing index-0 `go-version` rule in place does **not** achieve AC-14: the unfiltered `matchManagers: ["gomod"]` + `matchUpdateTypes: ["patch", "minor"]` rule at index 6 (`renovate.json:63-68`) carries no package filter, matches the `go` package too, and wins over index 0. That ordering is mechanically why today's drift exists (design §1.7).

- [ ] **Step 1: Append the two override rules**

Insert after the final existing rule (the npm major rule closing at `renovate.json:98`), as the last two elements of `packageRules`:

```json
    {
      "description": "task-261: Go toolchain bumps never auto-merge. The `go` directive is duplicated across ~110 pin sites (go.mod x103, go.work, Dockerfile ARGs, bake vars, CI env) that Renovate's gomod manager cannot all see, so a merged bump lands PARTIALLY and the tree builds against two Go versions at once — that is how the 1.24/1.25/1.26 spread this task collapses came to exist. MUST stay last in packageRules: the unfiltered gomod patch/minor rule above sets automerge:true and later rules win.",
      "matchManagers": ["gomod"],
      "matchPackageNames": ["go"],
      "automerge": false
    },
    {
      "description": "task-261: golang base-image bumps never auto-merge — same partial-landing reason as the `go` rule above. tools/toolchain-pin-guard.sh must pass before a Go/Alpine bump merges, and it cannot pass until every pin site moves together.",
      "matchManagers": ["dockerfile"],
      "matchPackageNames": ["golang"],
      "automerge": false
    }
```

The existing `groupName` assignments (`go-version` at index 0, `dockerfile-golang` at index 2) are **kept unchanged** — grouping is desirable, auto-merging is not. Neither override carries `matchUpdateTypes`: every update type, including patch, is manual.

- [ ] **Step 2: Verify the JSON parses and the overrides are last**

```bash
jq -e '.packageRules | length' renovate.json
jq -r '.packageRules[-2:] | .[] | "\(.matchManagers[0])\t\(.matchPackageNames[0])\tautomerge=\(.automerge)"' renovate.json
```

Expected: `13`, then exactly two lines:

```
gomod	go	automerge=false
dockerfile	golang	automerge=false
```

- [ ] **Step 3: Verify no later rule re-enables auto-merge for the `go` package**

```bash
jq -r 'to_entries | .[] | select(.key=="packageRules") | .value | to_entries | .[] | select(.value.matchManagers // [] | index("gomod")) | select((.value.matchPackageNames // ["<unfiltered>"]) | index("go") or index("<unfiltered>")) | "\(.key)\t\(.value.matchPackageNames // "unfiltered")\tautomerge=\(.value.automerge)"' renovate.json
```

Expected: the rules that can match the `go` package, in array order, with the LAST one reading `automerge=false`. Concretely: index 0 (`["go"]`, true), index 1 (`["go"]`, false, major-only), index 6 (unfiltered, true), index 7 (unfiltered, false, major-only), index 11 (`["go"]`, false). Index 11 is last and wins.

- [ ] **Step 4: Write the AC-14 reasoning trace**

AC-14 is satisfied by reasoning over the resulting rule array, since a live PR cannot be produced on a branch. Write `docs/tasks/task-261-go-1-27-migration/renovate-trace.md` containing:

1. The Renovate semantics being relied on, stated once: `packageRules` are applied in array order and a later matching rule overrides an earlier one.
2. A table of EVERY rule index that matches a `gomod` + `go` package minor/patch update, in order, with its `automerge` value and whether it carries a package filter. Take the values from the Step 3 output, pasted verbatim — do not retype them from memory.
3. The same table for a `dockerfile` + `golang` update.
4. The conclusion: the last matching rule in each table sets `automerge: false`, therefore neither a Go toolchain bump nor a golang base-image bump auto-merges.
5. The pre-change counterfactual: before this task, the last matching rule for `gomod`+`go` minor/patch was index 6 (unfiltered, `automerge: true`), which is why setting `automerge: false` on index 0 alone would not have worked.

- [ ] **Step 5: Commit**

```bash
git add renovate.json docs/tasks/task-261-go-1-27-migration/renovate-trace.md
git commit -m "chore(task-261): stop Renovate auto-merging Go toolchain and golang image bumps"
```

---

## Task 10: Full acceptance sweep and evidence

### Files

- `docs/tasks/task-261-go-1-27-migration/acceptance.md` — **new file**; the AC-1..AC-17 evidence record
- `docs/tasks/task-261-go-1-27-migration/evidence-lint.md` — read-only (new file, created by Task 3)
- `docs/tasks/task-261-go-1-27-migration/evidence-guard.md` — read-only (new file, created by Task 7)
- `docs/tasks/task-261-go-1-27-migration/renovate-trace.md` — read-only (new file, created by Task 9)

Module roots: repo root.

### Interfaces

- **Consumes:** everything from Tasks 1-9.
- **Produces:** the acceptance record the reviewer reads. Nothing consumes it.

- [ ] **Step 1: Confirm no unintended `.go` change (AC-16)**

```bash
git diff --stat 855fef4d1..HEAD -- '*.go'
```

Expected: only `tools/toolchain-pin-guard.sh`-adjacent additions (there are none — the guard is bash) and any FR-5.3 lint fix from Task 3. If a `.go` file appears that Task 3 did not record in `evidence-lint.md`, that is a scope violation — justify it explicitly or revert it.

- [ ] **Step 2: Run the guard clean**

```bash
tools/toolchain-pin-guard.sh; echo "GUARD_EXIT=$?"
```

Expected: `GUARD_EXIT=0`, summary naming 103 go.mod files.

- [ ] **Step 3: Run the flagless verification gate (AC-15)**

`--quick` / `--no-docker` do NOT satisfy AC-15; the bake and `-race` are the point.

```bash
tools/verify.sh 2>&1 | tee /tmp/t261-verify.log; echo "VERIFY_EXIT=${PIPESTATUS[0]}"
```

Expected: `VERIFY_EXIT=0`. Anything else is not done — fix and re-run in full, not with a flag.

- [ ] **Step 4: Write the acceptance record**

Write `docs/tasks/task-261-go-1-27-migration/acceptance.md` with one section per AC. Each cites a command and its **pasted, unsummarized** output.

| AC | Command that proves it |
|---|---|
| AC-1 | the Task 2 Step 3 directive tally (103 × `go 1.27.0`) |
| AC-2 | `grep -rn '^toolchain ' $(git ls-files '*go.mod' 'go.mod')` → no matches |
| AC-3 | `sed -n '1p' go.work` → `go 1.27.0` |
| AC-4 | `sed -n '17,18p' Dockerfile` |
| AC-5 | `sed -n '9,10p' services/atlas-kafka-precreate/Dockerfile` |
| AC-6 | `grep -A1 'variable "GO_VERSION"' docker-bake.hcl` and the same for `ALPINE_VERSION` |
| AC-7 | `test ! -e services/atlas-renders/Dockerfile` plus the Task 4 Step 7 bake print and Step 3's `verify.sh` bake |
| AC-8 | the Task 5 Step 1 grep and the Step 3 residual-match categorization |
| AC-9 | `evidence-lint.md` |
| AC-10 | `grep -rn 'GO_VERSION=\|ALPINE_VERSION=\|GOLANGCI_LINT_VERSION=' tools/toolchain.versions` — one declaration each; plus the note that every other site is a duplicate machine-checked by the guard |
| AC-11 | `evidence-guard.md` |
| AC-12 | `grep -n 'EXEMPT_GOMOD' -B14 tools/toolchain-pin-guard.sh` — the comment block naming both fixture classes |
| AC-13 | `grep -m1 '^go ' tools/cideps/testdata/simple/libs/lib-a/go.mod` and `cd tools/cideps && go test ./...` |
| AC-14 | `renovate-trace.md` |
| AC-15 | Step 3's `VERIFY_EXIT=0` and the tail of `/tmp/t261-verify.log` |
| AC-16 | Step 1's diffstat |
| AC-17 | filled in after code review runs (see Step 6) |

- [ ] **Step 5: Commit the acceptance record**

```bash
git add docs/tasks/task-261-go-1-27-migration/acceptance.md
git commit -m "docs(task-261): record acceptance evidence for the go 1.27 migration"
```

- [ ] **Step 6: Code review (AC-17)**

Run code review per `docs/review-protocol.md` before any PR is opened. A green `verify.sh` is not a substitute — the two things it cannot see here are (a) whether the guard's checked set actually covers every site the inventory lists, and (b) whether the `Dockerfile` `ARG` re-declaration is correctly scoped to the `build-env` stage. Record the verdict in `acceptance.md` under AC-17.

---

## Self-Review

**Spec coverage.** design §2 → Task 1. §5 / FR-1, FR-2 → Task 2. §9 risk 1 / FR-5, PRD OQ3 → Task 3. §6 / FR-3 → Task 4. FR-4 → Task 5. §1.5 / NFR-Compatibility → Task 6. §3 / FR-6.1-6.4, FR-7.3 → Task 7. §4 / FR-6.2, FR-6.3, FR-6.5 → Task 8. §7 / Renovate, AC-14 → Task 9. AC-1..AC-17 → Task 10. design §1.8 (`reconcile-bump-prs.yml`) and §7 (no `customManagers`) are resolutions of PRD open questions requiring **no edit**, and are correctly absent from every task. §10 out-of-scope items appear in Global Constraints as prohibitions.

**Placeholder scan.** No "TBD", "similar to Task N", or "add appropriate error handling". Task 3 is the one task whose edit set is not fully enumerable at plan time — by construction, since PRD OQ3 is only answerable by running the sweep; its steps therefore specify the measurement commands and the FR-5.3 triage table verbatim, plus an explicit escalation branch, rather than guessing a file list.

**Type consistency.** The pin file is `tools/toolchain.versions` in every task. The guard is `tools/toolchain-pin-guard.sh` in every task. `GO_VERSION` / `ALPINE_VERSION` / `GOLANGCI_LINT_VERSION` are spelled identically in Tasks 1, 4, 5, 7, 8. The three literal values (`1.27.0`, `3.24`, `v2.13.1`) match Global Constraints everywhere they appear. The README cell spelling asserted in Task 6 Step 3 is the same string Task 7's README check matches.
