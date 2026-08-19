# Review: 70afb62d7 — fix(lint): allow parallel golangci-lint runners across worktrees

Worktree: `.worktrees/fix-golangci-parallel-runners`, branch `fix-golangci-parallel-runners`.
Scope: commit `70afb62d7`, single file `tools/lint.sh` (+12/-6).

## 1. Correctness of the shell change

`tools/lint.sh:199` (current file):

```
local -a lintargs=(run --allow-parallel-runners -c "$ROOT/.golangci.yml")
```

- The flag is a bare boolean (`--allow-parallel-runners`, no `=value`), placed
  immediately after the `run` subcommand and before `-c`. Confirmed against
  `golangci-lint run --help`:
  ```
  --allow-parallel-runners   Allow multiple parallel golangci-lint instances running.
                              If false (default) - golangci-lint acquires file lock on start.
  ```
  Standard Cobra/pflag parsing accepts flags in any order relative to other
  flags, so placement before `-c` is not load-bearing, but it is also not
  wrong.
- `lintargs` is a bash array (`local -a`), and `--fix` / `--new-from-rev
  "$base"` are appended with `+=` afterwards (`tools/lint.sh:202-206`). The
  new flag does not sit between `run` and a value that could be mis-parsed as
  its argument (it takes none), and does not disturb the `--fix` /
  `--new-from-rev` append logic — verified by reading the surrounding lines,
  which are otherwise unchanged from before the commit.
- Quoting: `"$ROOT/.golangci.yml"` was already quoted before this commit and
  remains quoted; the new bareword flag needs no quoting.
- The `fmt` layer (`tools/lint.sh:182,192`, `golangci-lint fmt ...`) is
  untouched by the diff. `golangci-lint fmt --help` has no
  `--allow-parallel-runners` flag at all, consistent with the PR description's
  claim that only `run` takes the flock — verified directly:
  ```
  $ .cache/tools/bin/golangci-lint-v2.12.2 run --help | grep -i allow-parallel
        --allow-parallel-runners            Allow multiple parallel golangci-lint instances running.
  $ .cache/tools/bin/golangci-lint-v2.12.2 fmt --help | grep -i allow-parallel
  (no output)
  ```
- PASS — the flag is correctly placed, correctly quoted (trivially, since it
  takes no argument), and does not touch the `--fix` / `--new-from-rev` /
  `fmt` code paths.

## 2. Does per-tree cache keying actually remove the hazard the lock protects against?

Reproduced the failure mode directly against the pinned binary
(`.cache/tools/bin/golangci-lint-v2.12.2`) to confirm the exit code and
message the new comment cites:

```
$ flock "$TMPDIR/golangci-lint.lock" -c "sleep 20" &
$ golangci-lint run --allow-parallel-runners=false -c .golangci.yml ./...
exit: 3
Error: parallel golangci-lint is running
The command is terminated due to an error: parallel golangci-lint is running
```

This matches the comment's claim of "exited 3" (`tools/lint.sh:76`) exactly —
not invented, independently reproduced against the actual pinned binary.

On the substantive question: the lock genuinely guards concurrent writers to
one `GOLANGCI_LINT_CACHE` directory (that's what golangci-lint's own docs and
`--allow-serial-runners`'s wording — "serialize them around a lock" — imply).
Since `GOLANGCI_LINT_CACHE` is keyed to `$ROOT` (`tools/lint.sh:79`, unchanged
by this commit) and `$ROOT` is the absolute path of the worktree running
`lint.sh`, two *different* worktrees get two different cache directories by
construction, so the hazard the lock exists to prevent does not occur between
worktrees under default operation. This reasoning is sound and matches what
the rewritten comment says.

The one real gap — which the commit's own diff already calls out — is:
`export GOLANGCI_LINT_CACHE="${GOLANGCI_LINT_CACHE:-$ROOT/.cache/golangci-lint}"`
(`tools/lint.sh:79`, unchanged) means an *inherited* `GOLANGCI_LINT_CACHE` env
var (e.g. exported in a parent shell, `.envrc`, or a CI env block shared
across concurrent jobs) would silently point two trees at the same cache
directory while `--allow-parallel-runners` is now also skipping the only
mechanism (the flock) that would have caught concurrent writers to that
shared cache. This is a genuine, if narrow, regression in the failure mode:
before this commit, an accidental shared-cache override would at worst
produce a spurious "parallel golangci-lint is running" failure (annoying but
safe); after this commit, it can produce silent cache corruption / incorrect
results with no error at all.

The comment does state this caveat plainly (`tools/lint.sh:82-83`: "if you
override GOLANGCI_LINT_CACHE to a path shared between trees, you give that
protection up"), so this is not an *undocumented* hazard — it's a correctly
disclosed one. Nothing in the repo (checked: no `.github/workflows/*.yml`,
no `.envrc` in this worktree, no CI env block) currently sets
`GOLANGCI_LINT_CACHE`, so the hazard is theoretical today, not live. I am
flagging this as a **non-blocking note**, not a blocker: the comment is
accurate and adequate as documentation, but the code does not defend against
the case it describes (e.g. it could `readlink -f "$GOLANGCI_LINT_CACHE"` and
compare against a per-tree-derived path, or simply ignore an inherited
override). That defense was not asked for in the stated fix scope and adding
it unasked would be scope creep on this commit; noting it for the record.

## 3. `--allow-parallel-runners` vs `--allow-serial-runners`

Per the task framing, the repo owner's choice of `--allow-parallel-runners`
over `--allow-serial-runners` is not being re-litigated. For the record: the
help text shows `--allow-serial-runners` "serialize them around a lock" —
i.e. it would have kept mutual exclusion as a fallback net (queue instead of
fail) if the per-tree cache-keying assumption in §2 above is ever violated
(inherited/shared `GOLANGCI_LINT_CACHE`), at the cost of a run occasionally
blocking on another worktree's lint instead of running concurrently. That is
a real tradeoff the owner is accepting knowingly (the commit message
explicitly reasons through why the lock's protection is redundant here), so
this is informational only, not a finding.

## 4. Accuracy of the rewritten comment block

Rewritten comment, `tools/lint.sh:73-83`:

```
# The per-tree cache is NOT enough on its own: `golangci-lint run` also takes an
# exclusive flock on $TMPDIR/golangci-lint.lock, a machine-global path that no
# cache setting isolates. Two worktrees running verify.sh at once meant the
# loser exited 3 with "parallel golangci-lint is running" and no findings — a
# spurious guard failure, not a lint result. `run` is passed
# --allow-parallel-runners below to skip that lock. What the lock protects
# against is concurrent writers to ONE cache, which the per-tree keying above
# already rules out; if you override GOLANGCI_LINT_CACHE to a path shared
# between trees, you give that protection up.
```

Checked claim by claim:

- "takes an exclusive flock on $TMPDIR/golangci-lint.lock" — matches the
  task's stated, empirically-confirmed diagnosis (file observed appearing
  during a run at that path). Not independently re-observed by me in this
  review (would require an actual concurrent run against a real project, out
  of scope for a comment-accuracy check), but consistent with golangci-lint's
  documented lock behavior and the exit-3/message reproduction in §2.
- "a machine-global path that no cache setting isolates" — accurate:
  `GOLANGCI_LINT_CACHE` only affects the cache directory, not the lock file
  path, which golangci-lint derives from `os.TempDir()` independent of that
  env var. Consistent with the task's confirmed diagnosis.
- "loser exited 3 with 'parallel golangci-lint is running' and no findings" —
  reproduced directly in §2 (`exit: 3`, exact message match).
- "`run` is passed --allow-parallel-runners below" — true, verified at
  `tools/lint.sh:199`.
- "What the lock protects against is concurrent writers to ONE cache, which
  the per-tree keying above already rules out" — reasoned through in §2;
  consistent with `--allow-serial-runners`'s help text ("serialize them
  around a lock") implying the lock exists to protect one shared resource.
- "if you override GOLANGCI_LINT_CACHE to a path shared between trees, you
  give that protection up" — accurate and is the one caveat identified in §2.

No claim in the comment is unsupported or invented; all are either directly
reproduced here or are accurate descriptions of the code immediately below
them. PASS.

One small precision note (non-blocking): the comment says "Two worktrees
running verify.sh at once" — the actual failure mode is two worktrees running
`tools/lint.sh` (directly or via `verify.sh`) concurrently; `verify.sh` is the
common caller per the task's framing but is not the only one (a bare
`tools/lint.sh --check` invocation from two shells hits the same lock). This
doesn't misstate the mechanism, just narrows the framing to the observed
symptom. Not worth blocking on.

## 5. CI impact

```
$ grep -n "GOLANGCI_LINT_CACHE\|TMPDIR\|lint.sh" .github/workflows/*.yml
pr-validation.yml:436:  # tools/lint.sh --check is the single source of...
pr-validation.yml:490:  ./tools/lint.sh --check --go --base "$BASE" "${MODULES[@]}"
pr-validation.yml:492:  ./tools/lint.sh --check --go "${MODULES[@]}"
pr-validation.yml:517:  ./tools/lint.sh --check --ui
```

`.github/workflows/pr-validation.yml` invokes `tools/lint.sh --check` in two
separate jobs/steps (Go, UI), each on its own GitHub Actions runner VM with
its own filesystem and `$TMPDIR` — there is no shared `$TMPDIR` or
`GOLANGCI_LINT_CACHE` across CI jobs to race on, and no workflow in
`.github/workflows/` invokes `lint.sh` more than once concurrently within a
single runner. So this change has no observable effect on CI behavior either
way — it only matters for the local multi-worktree-on-one-machine scenario
the commit targets. No CI regression risk found.

## Summary of findings

- No blocking findings. Flag placement, quoting, and non-interference with
  `--fix` / `--new-from-rev` / `fmt` are all correct and verified against the
  actual pinned `golangci-lint` binary.
- Comment accuracy: every factual claim checked out against direct
  reproduction or code reading; nothing invented.
- Non-blocking: the cache-keying safety property this fix relies on
  (`GOLANGCI_LINT_CACHE:-$ROOT/...`) can be defeated by an inherited env var
  override, turning a previously-safe (if noisy) failure mode into a silent
  one. The comment discloses this correctly; the code does not defend
  against it. Worth a follow-up if `GOLANGCI_LINT_CACHE` is ever set in CI or
  a shared shell profile, but out of scope for this fix.
- Non-blocking: comment's "Two worktrees running verify.sh" framing slightly
  narrows the actual trigger (any two concurrent `lint.sh`/`golangci-lint run`
  invocations sharing a `$TMPDIR`), not materially inaccurate.
