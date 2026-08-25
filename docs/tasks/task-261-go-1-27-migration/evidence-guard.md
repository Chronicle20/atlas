# Task 7 evidence: `tools/toolchain-pin-guard.sh` (AC-11)

Verbatim guard-run output recorded per Task 7 Step 8: the clean run, the
selftest, and a real live mutation of a tracked file with its full failing
output and restore confirmation.

## 1. Clean run against the branch (Step 4)

Command:

```
tools/toolchain-pin-guard.sh; echo "GUARD_EXIT=$?"
```

Output:

```
toolchain-pin-guard: clean (103 go.mod + go.work + 4 Dockerfile ARGs + 2 bake vars + 7 CI pins + README checked)
GUARD_EXIT=0
```

103 go.mod files checked, matching `git ls-files '*go.mod' 'go.mod' | grep -v
'^tools/cideps/testdata/' | wc -l` = 103 (111 tracked go.mod files minus the 8
exempt fixture files under `tools/cideps/testdata/`).

## 2. Selftest (Step 5)

Command:

```
tools/toolchain-pin-guard.sh --selftest; echo "SELFTEST_EXIT=$?"
```

Output:

```
toolchain-pin-guard: selftest PASS
SELFTEST_EXIT=0
```

Confirmed nothing under the real tree was mutated by the selftest — `git
status --porcelain` after the run showed only the guard script itself and this
evidence file as new/untracked, no tracked file modified (Step 6).

## 3. Real live mutation (Step 7, AC-11)

The guard has only ever been exercised against `--selftest`'s throwaway copy
so far; this section proves it also catches a real drift in a real tracked
`go.mod`.

Mutation:

```
go -C libs/atlas-retry mod edit -go=1.26.0
```

Run:

```
tools/toolchain-pin-guard.sh > /tmp/t261-guard-fail.log 2>&1; echo "GUARD_EXIT=$?"
cat /tmp/t261-guard-fail.log
```

Full output (verbatim, not summarized):

```
GUARD_EXIT=1
libs/atlas-retry/go.mod:3: expected go 1.27.0, got go 1.26.0
toolchain-pin-guard: violations found (see above)
```

Restore:

```
go -C libs/atlas-retry mod edit -go=1.27.0
git diff --exit-code -- libs/atlas-retry/go.mod; echo "RESTORED=$?"
tools/toolchain-pin-guard.sh; echo "GUARD_EXIT=$?"
```

Output:

```
RESTORED=0
toolchain-pin-guard: clean (103 go.mod + go.work + 4 Dockerfile ARGs + 2 bake vars + 7 CI pins + README checked)
GUARD_EXIT=0
```

`git status --porcelain` after restore confirmed `libs/atlas-retry/go.mod` was
back to its committed state and no other tracked file was touched.

## 4. Task 7 fix round: absent-key reporting + structural `go-test` scan

Findings from `.superpowers/sdd/plan/task-7-review.md`, fixed per
`.superpowers/sdd/plan/task-7-fix-brief.md`. All probes below ran against a
throwaway local clone of the worktree at `/tmp/tmp.HY2arWz6P3` (removed after
the run) with the fixed `tools/toolchain-pin-guard.sh` copied in; nothing
under the real worktree was mutated. `git status --short` in the real
worktree after every probe showed only ` M tools/toolchain-pin-guard.sh` (the
intended fix) plus the pre-existing untracked
`docs/tasks/task-261-go-1-27-migration/agent-ledger.tsv`.

### Finding 1 — absent pin key now reports a violation instead of dying silently

Three sites were probed by deleting the searched-for key outright from a copy
and rerunning the guard against that copy.

**`gomod_check` — `libs/atlas-retry/go.mod` with its `go ` directive deleted:**

```
libs/atlas-retry/go.mod: expected a 'go' directive line, found none
toolchain-pin-guard: violations found (see above)
EXIT=1
```

**`ci_check` JSON branch — `.github/config/services.json` with `"go_version":` deleted:**

```
.github/config/services.json: expected a '"go_version":' key, found none
toolchain-pin-guard: violations found (see above)
EXIT=1
```

**`ci_check` go-test branch — `.github/actions/go-test/action.yml` with `  go-version:` deleted:**

```
.github/actions/go-test/action.yml: expected a 'default:' line inside the go-version input block, found none
toolchain-pin-guard: violations found (see above)
EXIT=1
```

All three now print a `path: ...` violation line and exit 1 — no more silent,
zero-output death under `set -euo pipefail`.

### Finding 2 — `go-test/action.yml`'s `default:` is now found structurally, not by fixed offset

The `+3`-line offset was replaced with an `awk` block scan (mirroring
`bake_check`'s style) that starts at `  go-version:`, scans forward for the
first `default:` line, and stops at the next top-level input key
(`^  [a-zA-Z]`, e.g. `  race-detection:`).

**Blank line inserted immediately after `  go-version:` (shifts `default:` from
line 11 to line 12), value unchanged — still clean:**

```
toolchain-pin-guard: clean (103 go.mod + go.work + 4 Dockerfile ARGs + 2 bake vars + 7 CI pins + README checked)
EXIT=0
```

Then the shifted `default:` (now line 12) mutated to `1.26.0` — correctly
detected at the new line number, not the old fixed offset:

```
.github/actions/go-test/action.yml:12: expected default: '1.27.0', got default: '1.26.0'
toolchain-pin-guard: violations found (see above)
EXIT=1
```

**Blank line inserted before `  go-version:` (shifts `go-version:` itself down
one line), value unchanged — still clean:**

```
toolchain-pin-guard: clean (103 go.mod + go.work + 4 Dockerfile ARGs + 2 bake vars + 7 CI pins + README checked)
EXIT=0
```

Then the corresponding `default:` (now line 12) mutated to `1.26.0` — still
found and flagged correctly:

```
.github/actions/go-test/action.yml:12: expected default: '1.27.0', got default: '1.26.0'
toolchain-pin-guard: violations found (see above)
EXIT=1
```

**`race-detection`'s own `default:` mutated instead (from `'false'` to
`'true'`) — correctly ignored, guard stays clean:**

```
toolchain-pin-guard: clean (103 go.mod + go.work + 4 Dockerfile ARGs + 2 bake vars + 7 CI pins + README checked)
EXIT=0
```

The scan is now insensitive to line insertion/removal on both sides of the
`go-version:` block and still never matches `race-detection`'s block.

### Finding 3 — no more doubled space in `got ...` messages

`docker-bake.hcl`'s `GO_VERSION` `default =` line (2-space indent) mutated to
`1.26.0` in a copy:

```
docker-bake.hcl:26: expected default = "1.27.0", got default = "1.26.0"
toolchain-pin-guard: violations found (see above)
EXIT=1
```

`got default = ...` — a single space, no longer `got  default = ...`. The
same full-leading-whitespace strip (`${var#"${var%%[![:space:]]*}"}"`) was
applied at `bake_check`'s ALPINE_VERSION site and `ci_check`'s go-test branch.

### Regression bar (Step 8)

```
tools/toolchain-pin-guard.sh; echo "EXIT=$?"
```
```
toolchain-pin-guard: clean (103 go.mod + go.work + 4 Dockerfile ARGs + 2 bake vars + 7 CI pins + README checked)
EXIT=0
```

```
tools/toolchain-pin-guard.sh --selftest; echo "EXIT=$?"
```
```
toolchain-pin-guard: selftest PASS
EXIT=0
```

`git status --short` right after: `M tools/toolchain-pin-guard.sh` (the
intended fix) plus the pre-existing untracked
`docs/tasks/task-261-go-1-27-migration/agent-ledger.tsv` — nothing else.

```
shellcheck --severity=warning tools/toolchain-pin-guard.sh; echo "SHELLCHECK_EXIT=$?"
```
```
SHELLCHECK_EXIT=0
```

Exact-not-prefix matching still holds (probed on a copy):

```
libs/atlas-retry/go.mod:3: expected go 1.27.0, got go 1.27
toolchain-pin-guard: violations found (see above)
EXIT=1
```
```
libs/atlas-retry/go.mod:3: expected go 1.27.0, got go 1.27.1
toolchain-pin-guard: violations found (see above)
EXIT=1
```

Both `go 1.27` (prefix) and `go 1.27.1` (over-precise) are still rejected.
