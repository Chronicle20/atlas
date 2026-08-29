# Review — Task 2: Disk-backed scratch root and its sweeper (Layer 0)

Range: `583be745e..52f568d7b` (1 commit, `52f568d7b`).
Files: `tools/scratch-sweep.sh` (new), `tools/scratch-sweep_test.sh` (new),
`docs/verification.md` (+87 lines), `docs/tasks/task-286-build-verify-concurrency/measurements.md` (+46 lines).

## Scope confirmation

Diff matches the brief's file list exactly (`git diff --stat 583be745e..52f568d7b`):
4 files, 376 insertions, 0 deletions, no deletions in either doc (both are
pure appends — verified with `git diff ... | grep '^-'`, which returns only
the `--- a/...` header line for `docs/verification.md`). No `.envrc` appears
anywhere in the diff. Scope matches the brief.

Note: the worktree currently carries *uncommitted* changes to
`tools/verify.sh` / `tools/verify_test.sh` and an untracked
`tools/zz-verify-probe_test.sh`, which are not part of commit `52f568d7b`
(confirmed: `HEAD` = `52f568d7b`, and `git diff --stat HEAD -- tools/verify.sh
tools/verify_test.sh` shows a dirty working tree). These belong to other
in-flight work in this worktree, not to Task 2 — see "Not evaluable" below.

## Findings

### PASS — C2 (no tracked `.envrc`) honored

`git diff 583be745e..52f568d7b` touches no `.envrc`. `docs/verification.md`
explicitly documents why one is not shipped, naming the exact hazard
("an untracked personal `.envrc` already exists in the main checkout and is
not gitignored, so a tracked one would break `git checkout` of this branch
there"). Confirmed the main-checkout file exists exactly as described (the
main repo's `.envrc`, 30 bytes, mode 644, not in `.gitignore`).

### PASS — measurements.md: no fabricated after-figure

`## Layer 0 — scratch` (measurements.md) records a "before" `df -h /tmp` /
`ls /tmp | wc -l` / `free -h` / `/etc/fstab` grep, then an explicit "Not
applied at implementation time" for "After," naming the reason (`wsl
--shutdown` and an `/etc/fstab` edit are operator/host actions, out of an
implementer session's reach, and would destroy the "before" baseline).
Re-ran the same checks live on this host and they match the claims:

```
$ grep -n tmp /etc/fstab   → (no output — no /tmp line)
$ df -h /tmp               → tmpfs 16G  5.0G  11G  32%  /tmp
$ free -h                  → Mem: 31Gi total
```

`Mem: 31Gi total` matches the doc's "VM currently gets 31 GiB by the 50%
default" claim, and the missing `/tmp` fstab line matches "host tuning not
applied." No after-figure is invented anywhere.

### PASS — Task 1's `## Layer 1 — build context` section untouched

`git diff` for `measurements.md` is a pure append after the existing content
(`+` lines only, first `+` immediately follows the last pre-existing line);
Layer 1's heading and body are byte-for-byte unchanged, not renumbered.

### PASS — `## Host tuning (WSL2)` section: placement, content, structure

- Sits at `docs/verification.md:100`, immediately before `## The Go layer`
  (confirmed via `grep -n '^## '`).
- Carries the `.wslconfig` block verbatim (`memory=52GB`, `processors=24`,
  `swap=16GB`) at the placeholder path `C:\Users\<windows-user>\.wslconfig`,
  with `wsl --shutdown` as the apply step and the measured rationale (host
  ~64 GiB / 24 logical CPUs, VM at 31 GiB by the 50% default).
- Carries the `/etc/fstab` line verbatim
  (`tmpfs /tmp tmpfs rw,nosuid,nodev,size=4G,nr_inodes=1048576 0 0`) with the
  "26 GiB by the post-bump default, worse than today's 16 GiB" reasoning.
- Documents `export TMPDIR=/var/tmp/atlas/scratch`, the explicit no-`.envrc`
  rationale, and the `CLAUDE_JOB_DIR` best-effort / `TMPDIR` load-bearing
  note.
- Documents both systemd user unit files (`atlas-scratch-sweep.service`
  running `<repo-root>/tools/scratch-sweep.sh`, `atlas-scratch-sweep.timer`
  with `OnCalendar=daily`) and the enable command.
- Opens with a pointer to Task 7's preflight detecting the un-tuned
  condition rather than assuming the section was applied.
- No literal home path: `git diff 583be745e..52f568d7b | grep -E '/home/'`
  returns nothing across the whole diff, not just this section.
- Structured as one `##` heading with four `###` subsections ending on
  "### Sweeper — systemd user timer"; a Task-6 `### Build slots` subsection
  can append cleanly before `## The Go layer`.

### PASS — dangerous-root guard runs before any mutation, and the guarded
patterns are correct

`tools/scratch-sweep.sh:50-60` — the `case`/`if` guard block sits before the
first filesystem mutation (`mkdir` at line 63 / `chmod` at line 67). Verified
by reading the control flow directly, not by inference from test results.

- `/`, `/tmp`, `/tmp/`, `/var/tmp`, `/var/tmp/` — refused by literal `case`
  match (line 51).
- `$HOME` and `$HOME/` — refused by explicit comparison (line 53-55).
- Fewer than two `/`-delimited path components — refused (lines 57-60).
- Unset `ATLAS_SCRATCH_ROOT` — `${ATLAS_SCRATCH_ROOT:-/var/tmp/atlas/scratch}`
  (line 28) falls back to a safe absolute default; the colon form also
  covers an **empty** `ATLAS_SCRATCH_ROOT=""`, so `set -u` is not doing the
  guarding work here — the default itself is safe, as required.
- `--root ""` (explicit empty via the flag) is separately caught by
  `"${2:?--root needs a directory}"` (line 34), which dies before reaching
  the guard at all.
- Path with spaces: guard logic is fully quoted throughout (`"$root"` in
  every comparison, `tr`/`sed`/`wc` pipeline on the quoted variable); traced
  the component-count computation by hand for a root containing a space and
  confirmed it does not split on the space, only on `/`.
- Symlinked root: reasoned through GNU `find`'s default (`-P`) behavior for
  a symlink given as the search root, then verified empirically —
  `find <symlink-to-dir> -mindepth 1 -maxdepth 1` does **not** descend
  through the symlink (returns nothing), so even a root that resolves via
  symlink to a dangerous directory cannot be swept through this code path.
  Confirmed with a live reproduction (`ln -s realdir link; find link
  -mindepth 1` → empty).
- `--age-days` off-by-one: reasoned and empirically confirmed
  `-mtime +$((n-1))` matches the brief's semantics (a 2-day-old file is
  caught by `--age-days 1` → `-mtime +0`; a 10-day-old file is caught by the
  default `--age-days 7` → `-mtime +6`).

### FINDING (blocking) — dangerous-root test cases run the real destructive
invocation against real system paths

`tools/scratch-sweep_test.sh:92,96,100,104` invoke the script as
`ATLAS_SCRATCH_ROOT=/ "$SWEEP" --now`, `ATLAS_SCRATCH_ROOT=/tmp "$SWEEP"
--now`, `ATLAS_SCRATCH_ROOT=/var/tmp "$SWEEP" --now`, and
`ATLAS_SCRATCH_ROOT="$HOME" "$SWEEP" --now` — i.e., the four dangerous-root
refusal cases run *outside* the `mktemp -d` fixture, directly against `/`,
`/tmp`, `/var/tmp`, and the real `$HOME`, with `--now` (age 0 — sweep
everything). Given the guard's current correctness (see the PASS above),
these calls die with exit 2 before any mutation and today's run is safe. But
the test's safety is entirely contingent on the guard under test being
correct — exactly the scenario the task instructions flag as blocking "even
if it passes today." A future edit that reorders the guard relative to the
`mkdir`/`find`/`rm` block, or narrows a `case` pattern, would not be caught
by a failing assertion; it would be caught by this test suite actually
sweeping `/`, `/tmp`, `/var/tmp`, or the operator's real home directory.

This traces back to the brief's own test table (`.superpowers/sdd/plan/task-2-brief.md`),
which specifies exactly these four invocations with `--now`. The
implementer followed the brief literally. The fix does not require
deviating from the brief's intent: the guard fires unconditionally before
the `dry_run` branch (confirmed above — it is the first thing after
argument parsing, at line 50, well before the `dry_run` fork at line 77), so
substituting `--dry-run` (or simply dropping `--now`) for these four
invocations would preserve the identical exit-2/`refusing` assertions while
removing essentially all blast radius from a guard regression — a
regression would then print a scary candidate list instead of deleting
`/tmp` or `$HOME`.

Action needed: change `tools/scratch-sweep_test.sh:92,96,100,104` from
`--now` to `--dry-run` (or no age flag at all, since none of these cases
exercise age logic).

### Non-blocking — relative `--root`/`ATLAS_SCRATCH_ROOT` is not rejected

The guard checks path-component count but never requires an absolute path.
Reproduced live: from a temp directory, `ATLAS_SCRATCH_ROOT="foo/bar"
scratch-sweep.sh` exits 0 and creates/sweeps `foo/bar` relative to the
caller's cwd (`tools/scratch-sweep.sh:28,50-60`). This cannot reach a
catastrophic path via this code path (a relative value can't resolve to
`/`), but it is a real robustness gap for the exact class of "broken/empty
env var" scenario the guard exists to defend against — an unexpected cwd
plus a relative `ATLAS_SCRATCH_ROOT` silently sweeps the wrong directory
rather than refusing. Not in the brief's test table, not required by the
brief's contract text, so not blocking — flagging for awareness since the
review brief explicitly asked me to reason through the relative-path case.

### Non-blocking — `mkdir -p -m 700` only sets mode on the leaf directory

`tools/scratch-sweep.sh:63` — `shellcheck` flags this itself (SC2174,
warning level, not gating `shell-guard.sh`'s `-S error` threshold).
Intermediate directories created along the way (e.g. a not-yet-existing
`atlas` parent under `/var/tmp`) get default umask-based permissions, not
`700`; only the final component is 700. Confirmed live: `mkdir -p -m 700
foo/bar` leaves `foo` at mode `775`. Minor information-disclosure-adjacent
nitpick (scratch content itself is still 700), not required by the brief's
contract text.

### PASS — test suite otherwise operates only inside the `mktemp -d` fixture

All 11 non-dangerous-root cases (`tools/scratch-sweep_test.sh:22-88,
114-141`) create their working roots under `$tmp` (from `mktemp -d`, with
`trap 'rm -rf "$tmp"' EXIT` at line 20) and pass `ATLAS_SCRATCH_ROOT="$tmp/..."`
or `--root "$tmp/..."`. None of them touch a real scratch root or any path
outside the fixture.

### PASS — tests assert behavior, not just script execution

Every case checks either the exit code, a filesystem post-condition (file
gone/present, mode, directory existence), or stdout/stderr content — not
merely that the script ran. The dangerous-root cases assert both exit code
2 and the `refusing` substring (not just "didn't crash"). This satisfies the
brief's "the guard is checked, not just exercised" requirement structurally,
independent of the blocking finding above about *which* paths those checks
run against.

## Verification commands re-run

```
$ ./tools/scratch-sweep_test.sh
... (34 "ok" lines matching all 15 case groups in the brief's table)
scratch-sweep_test.sh: all assertions passed
$ echo $?
0
```

```
$ ./tools/shell-guard.sh --require-shellcheck
shell-guard: 1 of 77 script(s) failed.
```

This differs from the implementer's reported "76 script(s) OK, exit 0."
The single failure is `tools/verify_test.sh` (SC2218 — `cleanup` referenced
before definition), which is **not part of this diff** — it is an
uncommitted, dirty change to a file this commit does not touch (`HEAD` =
`52f568d7b`; `git status --short` shows `M tools/verify.sh`,
`M tools/verify_test.sh`, `?? tools/zz-verify-probe_test.sh`, none of which
appear in `git diff --stat 583be745e..52f568d7b`). Ran `shellcheck -S error`
(the exact severity `shell-guard.sh` gates on) against only the two new
files and got zero error-level findings — the only findings were SC2174
(warning) and SC2015 (info), both below the gating threshold. Task 2's own
files pass the gate; the reported regression belongs to other in-flight work
sharing this worktree.

## Not evaluable

- The state of `tools/verify.sh` / `tools/verify_test.sh` / the untracked
  `tools/zz-verify-probe_test.sh` — dirty working-tree state unrelated to
  commit `52f568d7b`, out of this unit's scope. Flagging its existence so
  the controller is aware `shell-guard.sh --require-shellcheck` will not
  currently exit 0 for the *worktree as a whole*, even though it exits 0 for
  this commit's files.
- `tools/verify.sh --quick --base 583be745e` — explicitly out of scope per
  the dispatch brief; running separately, verdict not mine to produce.
- The actual behavior of the systemd unit files and the `.wslconfig`/`fstab`
  changes on a real host — none of this is applicable inside a Linux
  worktree without a Windows host to apply `.wslconfig` against; documented
  correctly per the brief but its *effect* is untestable from here.

## Verdict rationale

One blocking finding: the dangerous-root test cases run the real
destructive invocation (`--now`) against real system paths instead of a
safer equivalent that would prove the same guard behavior without exposing
`/`, `/tmp`, `/var/tmp`, or `$HOME` to an undetected guard regression. Two
non-blocking findings (relative-path root not rejected; leaf-only `mkdir -m`
mode). Everything else — the guard's correctness as currently written, doc
placement/content, `.envrc` omission, and the measurements' honesty about
the unmeasured "after" state — passes with cited evidence.

---

## Re-review — fix round 1 (`90b0c2a05..42a59c45d`)

Range: `90b0c2a05..42a59c45d` (1 commit, `42a59c45d`,
`fix(scratch-sweep): dry-run dangerous-root tests, reject relative root`).
Files touched: `tools/scratch-sweep.sh`, `tools/scratch-sweep_test.sh`. This
section verdicts the three open findings above and reports anything new
found while probing the fix, per the re-review dispatch brief. It does not
re-litigate anything already marked PASS above; those hold unchanged (the fix
diff does not touch any of that surface).

### Finding 1 (blocking) — dangerous-root tests use `--now` against real paths

**ADDRESSED.** `tools/scratch-sweep_test.sh:98,102,106,110` now invoke
`--dry-run` instead of `--now` for all four dangerous-root cases (`/`,
`/tmp`, `/var/tmp`, `$HOME`), with the exit-2/`refusing` assertions
unchanged in intent (`assert_eq ... "2" "$rc"` / `assert_has ... "refusing"
"$out"` at each site). Confirmed the guard still fires before the `dry_run`
fork: the guard block is the first thing after argument parsing
(`tools/scratch-sweep.sh:50-67`), well before the `dry_run` branch at line
85 — so even if a future edit narrowed a `case` pattern, the worst outcome
of these four test invocations is now a printed candidate list under
`--dry-run`, never a `rm -rf`. Re-ran the suite live; all four dangerous-root
assertions still pass (`ok - refuses / : exit 2`, etc., see rerun below).

### Finding 2 — no absolute-path requirement; relative root swept the caller's cwd

**PARTIALLY ADDRESSED — a materially equivalent bypass remains open,
reported as new below.**

The literal case named in the finding is fixed: `tools/scratch-sweep.sh:64-67`
adds `case "$root" in /*) ;; *) die ... ;; esac`, and the new test case
(`tools/scratch-sweep_test.sh:116-118`) reproduces exactly the original
repro (`ATLAS_SCRATCH_ROOT="relative/scratch"` from a non-root cwd) and
asserts exit 2 / `refusing`. Verified live:

```
$ ATLAS_SCRATCH_ROOT="foo/bar" ./tools/scratch-sweep.sh
scratch-sweep: refusing to sweep dangerous root: foo/bar (not an absolute path)
$ echo $?
2
```

Also verified the new test case's subshell claim: `out="$(cd "$tmp" &&
ATLAS_SCRATCH_ROOT="relative/scratch" "$SWEEP" --dry-run 2>&1)"`
(`tools/scratch-sweep_test.sh:116`) — the entire right-hand side is a
`$(...)` command substitution, which bash always runs in a forked subshell
regardless of the `cd` inside it, so the main test script's cwd cannot leak.
Confirmed empirically: `pwd` before and after this line, run standalone, is
unchanged.

However, the guard's own comment claims the fix makes "the resolved root
... always exactly what was configured"
(`tools/scratch-sweep.sh:61-63`). That is not true: the guard checks only
that the literal string starts with `/`; it does not canonicalize `.` or
`..` segments. A crafted absolute path containing a `..` component can
resolve to one of the exact dangerous roots the guard exists to block, while
passing every check (`case` literal match, `$HOME` comparison, component
count, and the new absolute-path check) because none of those checks
operate on the resolved path. Reproduced live, non-destructively first,
then with an actual fixture sweep to prove it is not merely cosmetic:

```
$ ATLAS_SCRATCH_ROOT="/var/tmp/../tmp" ./tools/scratch-sweep.sh --dry-run
chmod: changing permissions of '/var/tmp/../tmp': Operation not permitted
$ echo $?
1
```

(This only fails here because the current unprivileged user cannot `chmod`
`/tmp`; the guard did not stop it — the failure is a permission error from
past the guard, not a `refusing` exit 2. As root, or against any directory
this user *can* chmod, this proceeds.) Confirmed the destructive path
directly with a fixture the user does own:

```
$ mkdir -p "$tmp/real" "$tmp/decoy"; touch -d '10 days ago' "$tmp/real/old.txt"
$ ATLAS_SCRATCH_ROOT="$tmp/decoy/../real" ./tools/scratch-sweep.sh --now
scratch-sweep: removed 1 entry from /tmp/tmp.XXXXXXXXXX/decoy/../real
$ ls "$tmp/real"          # old.txt is gone — a real, unguarded sweep
```

`$tmp/decoy/../real` is absolute (passes the new check), does not literally
match any of `/`, `/tmp`, `/var/tmp`, `$HOME` (passes the literal `case`),
and has more than two components (passes the component-count check) — yet
it resolves to exactly `$tmp/real` and is swept for real. The same
construction against a real, attacker-writable path pointing at `/tmp` or
`/var/tmp` (e.g. any absolute path with an existing parent that a `..`
walks back out of into `/tmp`) reaches the literal dangerous root the guard
exists to stop, entirely through a legitimately absolute string.

This is not the scenario Finding 2 named (a bare relative string), and
strictly it predates this fix — the pre-fix guard's literal `case` match
was always vulnerable to the same `..`-segment mismatch between the
configured string and the resolved path; the relative-path fix just adds
one more check to the same non-canonicalizing family without closing the
underlying class. Since the fix's own comment asserts the class is now
closed, and the re-review brief specifically asked for this vector to be
checked, it is reported here as new:

**FINDING (blocking, new) — `tools/scratch-sweep.sh:50-67`**: the guard
compares the literal, uncanonicalized `$root` string against dangerous
patterns and component counts; it does not resolve `.`/`..` segments before
comparing. An absolute path containing a `..` component that resolves to
`/`, `/tmp`, `/var/tmp`, or `$HOME` passes every existing check and reaches
the unconditional `chmod 700 "$root"` (line 75) and, if the guard is ever
regressed further, the `find`/`rm -rf` sweep. Proven destructively against a
non-system fixture above (a real file was removed via a `..`-relative
`ATLAS_SCRATCH_ROOT` that never literally names a dangerous path). Suggested
fix: canonicalize before comparing, e.g. `root="$(cd "$root" 2>/dev/null &&
pwd -P || printf '%s' "$root")"` immediately after argument parsing and
before any guard check (for the not-yet-existing case, canonicalize the
parent and append the leaf), so every subsequent check operates on the
resolved path rather than the literal string.

### Finding 3 — `mkdir -p -m 700` sets mode on the leaf only (SC2174)

**ADDRESSED,** for what it named. `tools/scratch-sweep.sh:70-71` splits
`mkdir -p -m 700 "$root"` into `mkdir -p "$root"` followed by `chmod 700
"$root"`, which silences SC2174. Re-ran `shellcheck` (no severity filter)
against `tools/scratch-sweep.sh` standalone: zero findings of any severity
(previously SC2174 warning + SC2015 info were present; SC2015 also appears
gone — not part of this fix's scope but noted). Effective leaf permissions
are unchanged: `mkdir -p "$root"; chmod 700 "$root"` leaves the leaf at
`700`, same as `mkdir -p -m 700 "$root"` did — confirmed via the existing
"created root is mode 700" test case, which still passes.

Two things worth recording, both non-blocking and not part of Finding 3's
original ask:

- **Intermediate directories are still not 700** (unchanged from before the
  fix — `mkdir -p -m 700` never set intermediate-directory modes either, so
  this is not a regression, just an unresolved partial fix of the
  underlying SC2174 concern). Confirmed live: `mkdir -p "$tmp/a/b"; chmod
  700 "$tmp/a/b"` leaves `$tmp/a` at the ambient umask-derived mode, not
  `700`.
- **A genuine, if narrow, permission window is introduced by the split**:
  `mkdir -p -m 700` sets the leaf's mode atomically at creation (the mode is
  passed to the creating syscall); `mkdir -p` followed by a separate `chmod
  700` leaves the leaf briefly world/group-readable at its default
  umask-derived mode between the two commands. At creation time the
  directory is empty, so this cannot leak scratch *content*, but it is a
  small regression in atomicity that did not exist in the pre-fix code.
  Deferred (non-blocking): the directory is empty at this point in every
  code path, so there is nothing to disclose through the window today, but
  a future caller that races this script could observe or write into an
  over-permissive directory for one syscall's worth of time.

### Verification commands re-run

```
$ ./tools/scratch-sweep_test.sh
ok   - creates a missing root: exit 0
...
ok   - refuses relative root: exit 2
ok   - refuses relative root: stderr says refusing
...
ok   - -h: stdout contains usage:

scratch-sweep_test.sh: all assertions passed
$ echo $?
0
```

33 assertion lines, 0 failures — matches the implementer's report.

```
$ ./tools/shell-guard.sh --require-shellcheck
shell-guard: 76 script(s) OK (syntax + shellcheck -S error).
$ echo $?
0
```

Matches the implementer's report exactly (76/76, exit 0) — the dirty
`tools/verify.sh`/`tools/verify_test.sh` state noted as "Not evaluable" in
the original review is no longer present in the working tree (`git status
--short` now shows only new, untracked review artifacts under
`docs/tasks/task-286-build-verify-concurrency/`), so that prior caveat no
longer applies.

### Not evaluable (re-review)

- Whether the new "absolute path with `..` segments" gap is reachable
  through any *other* entry point into this script (e.g. the systemd unit
  files documented in `docs/verification.md`, which are host state and out
  of this diff's scope) — not evaluated; the finding is scoped to the
  script's own guard logic as written.

### Re-review verdict rationale

Finding 1 is fully addressed — the destructive `--now` invocations against
real dangerous paths are gone, replaced with `--dry-run` equivalents that
preserve the same assertions with no blast radius. Finding 3 is addressed
for its stated scope (SC2174 silenced, leaf mode unchanged), with two
non-blocking observations recorded (intermediate dirs still not 700; a
narrow non-atomic window introduced by the split, empty-directory so no
content exposure today). Finding 2's named scenario (bare relative string)
is fixed and covered by a genuine new test, but the fix's stated invariant —
"the resolved root is always exactly what was configured" — is false as
written: a `..`-segment absolute path bypasses every check in the guard and
was proven live to reach and sweep a real, unguarded root. That gap is
reported above as a new blocking finding, since it defeats the purpose
Finding 2 was raised for even though it takes a different literal shape.
