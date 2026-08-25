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
