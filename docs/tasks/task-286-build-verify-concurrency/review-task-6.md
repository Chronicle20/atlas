# Review: Task 6 — build slot broker

Commit range: `19187a6b7..7b26b2d79` (single commit `7b26b2d79`, "feat(tools): add
machine-wide build slot broker").

Brief: `.superpowers/sdd/plan/task-6-brief.md` (including the "Additional owed
fix" section for `tools/buildx-bootstrap_test.sh`).

## Scope

`git diff --stat 19187a6b7..7b26b2d79`:

```
docs/verification.md           |  51 +++++++++++++++
tools/buildx-bootstrap_test.sh |  19 +++++-
tools/lib/build-slot.sh        | 144 +++++++++++++++++++++++++++++++++++++++++
tools/lib/build-slot_test.sh   | 114 ++++++++++++++++++++++++++++++++
tools/with-build-slot.sh       | 101 +++++++++++++++++++++++++++++
tools/with-build-slot_test.sh  | 116 +++++++++++++++++++++++++++++++++
6 files changed, 544 insertions(+), 1 deletion(-)
```

This matches the brief's file list exactly (`tools/lib/build-slot.sh`,
`tools/lib/build-slot_test.sh`, `tools/with-build-slot.sh`,
`tools/with-build-slot_test.sh`, `docs/verification.md`) plus the owed fix to
`tools/buildx-bootstrap_test.sh`. No scope mismatch. (`tools/verify.sh` is
modified in the working tree but that change is not part of this commit — it
belongs to the in-flight Task 7 and is out of scope here.)

## Findings

### 1. Sourceability (the requirement Task 7 depends on) — PASS

`tools/lib/build-slot.sh:37-38` states "Deliberately no `set -e`/`set -u` at
file scope." Verified no `set -e`/`set -u`/`set -o pipefail` anywhere in the
file, and no top-level side effects outside function bodies (only comments and
function definitions execute at source time).

Verified directly by sourcing it under a caller's `set -euo pipefail` (Task
7's actual context, since `launch_go_layers` runs inside `verify.sh`, which
sets its own strict mode):

```
$ bash -c 'set -euo pipefail; source tools/lib/build-slot.sh; export ATLAS_SLOT_DIR=/tmp/slot-review-sete; acquire_build_slot "sete-test"; echo "acquired: $BUILD_SLOT"; release_build_slot; echo "released ok"'
build-slot: 'sete-test' acquired slot 1 after 0s
acquired: 1
released ok
```

No spurious exit under `-e` from the internal `case`/`if` constructs. This is
the load-bearing requirement for Task 7 and it holds.

### 2. fd discipline — PASS

Fixed fd 200 used throughout (`tools/lib/build-slot.sh:100,116,124`), never fd
0-2, never fd 9 (reserved for Task 8's module-cache lock per the brief).
`grep -n "9>"` across `tools/lib/build-slot.sh` and `tools/with-build-slot.sh`
finds nothing; the only fd-9 usage anywhere in the diff is inside
`tools/lib/build-slot_test.sh`'s own local verification subshell
(`build-slot_test.sh:59`, `flock -n 9 9>"$f"`), which is an isolated test-only
fd unrelated to (and never concurrent with) the library's own fd 9 reservation
constraint — non-blocking finding, noted below.

`shellcheck -S error` via `tools/shell-guard.sh` is clean on all 5
touched/added scripts:

```
$ tools/shell-guard.sh --require-shellcheck tools/lib/build-slot.sh tools/lib/build-slot_test.sh tools/with-build-slot.sh tools/with-build-slot_test.sh tools/buildx-bootstrap_test.sh
shell-guard: 5 script(s) OK (syntax + shellcheck -S error).
```

### 3. Exit 75 contract — PASS

`acquire_build_slot` returns 75 only on the timeout path
(`tools/lib/build-slot.sh:126`), distinct from return 2 (invalid
`ATLAS_BUILD_SLOTS`, missing `flock`) and distinct from a wrapped command's own
status, which `tools/with-build-slot.sh` propagates via a separate `"$@"`
invocation after the slot is already held (`tools/with-build-slot.sh:99-101`).
Verified end-to-end: `with-build-slot.sh t -- sh -c 'exit 7'` returns 7 (not
conflated with 75), and the contention case returns 75 with
`no build capacity` on stderr.

### 4. No stale-lock cleanup path — PASS

`grep -n "stale\|rm -f\|find.*-mtime\|cleanup" tools/lib/build-slot.sh` finds
only the header comment explaining *why* there is no such path
(`tools/lib/build-slot.sh:18-21`). No cleanup/removal logic was added.
`release_build_slot` (`tools/lib/build-slot.sh:141-144`) only flocks/closes fd
200 — no file removal. Correct: the brief explicitly forbids this, and the
implementer's report correctly self-reports "no stale-lock cleanup was added."

### 5. Slot dir default — PASS

`_build_slot_dir` defaults to `/var/tmp/atlas/slots`
(`tools/lib/build-slot.sh:44-46`), overridable via `ATLAS_SLOT_DIR`, matching
the machine-global intent.

### 6. Test honesty — PASS, verified by deliberate breakage

Both suites pass as shipped:

```
$ bash tools/lib/build-slot_test.sh       # 7/7 ok
$ bash tools/with-build-slot_test.sh      # 17/17 ok
```

To confirm the contention/timeout and no-wait assertions are not vacuous, I
copied the scripts to a scratch dir and deliberately broke the broker in two
ways, re-running the (copied) CLI suite against each:

- **Broke enforcement** (`if flock -n 200; then` → `if true; then`, so the
  broker never actually contends): the suite genuinely failed —
  `FAIL - all slots busy + --timeout exits 75 (want '75', got '0')` and
  `FAIL - ... stderr mentions no build capacity`. All other cases still
  passed.
- **Broke the no-wait guarantee** (inserted `sleep 2` before the first
  non-blocking attempt): the suite genuinely failed —
  `FAIL - a free slot is taken without waiting (missing 'after 0s' ...)` and
  `FAIL - --slots overrides the env var: no wait (...)`.

Both breakages produced the expected, and only the expected, failures — the
tests are load-bearing, not decorative. (Scratch copies deleted after the
check; no tracked file was modified.)

### 7. `_build_slot_count` validation — PASS

Verified directly: `ATLAS_BUILD_SLOTS=-1`, `=abc`, `=3.5` all rejected with rc
2 and a message naming the variable
(`tools/lib/build-slot.sh:58-68`).

### 8. `release_build_slot` without a prior `acquire` — PASS

Verified it is a safe no-op and does not leak a permanent stderr redirect onto
the calling shell (the concern the report calls out):

```
$ bash -c 'source tools/lib/build-slot.sh; release_build_slot; echo "no-op release rc=$?"; echo "stderr still works" >&2'
no-op release rc=0
stderr still works
```

### 9. Owed fix — `tools/buildx-bootstrap_test.sh` builder save/restore — PASS

`docker buildx ls` before running the suite: `atlas*` selected. Ran the suite
directly (docker-gated block, in scope for this owed fix):

```
$ bash tools/buildx-bootstrap_test.sh
... (15 ok lines) ...
buildx-bootstrap_test: all assertions passed
```

`docker buildx ls` after: `atlas*` still selected — confirmed unchanged. The
fix (`tools/buildx-bootstrap_test.sh:62-76` in the diff) captures the
`*`-marked builder from `docker buildx ls` before creating the throwaway
`zz-atlas-test-$$` builder, and the `cleanup` trap restores it, falling back
to no-op restore if `PREV_BUILDER` is empty (matching the brief's
"fall back to the current behaviour rather than erroring" instruction). Header
note added at `tools/buildx-bootstrap_test.sh:4-11` documenting the
save/restore.

### 10. Documentation — PASS

`### Build slots` (`docs/verification.md:187`) is correctly placed as the last
subsection of `## Host tuning (WSL2)` (section headings verified via
`grep -n "^## \|^### "`). Covers the broker's purpose, the K=4 arithmetic
against 24 threads / 52 GiB, the per-slot resource table (`GOMAXPROCS` 6,
`go build -p` 6, `go test -p` 2, BuildKit `max-parallelism` 8), the
CLI-vs-sourced-library usage split (including the exact reason `verify.sh`
can't use the CLI wrapper), and the exit-75 contract.

### 11. Deterministic contended-slot selection — PASS

`target=$(( $$ % n + 1 ))` (`tools/lib/build-slot.sh:118`) correctly ranges
over `1..n` for any `n >= 1` (checked `n=1`: `$$ % 1 + 1 = 1`, always valid).

## Non-blocking notes

- `tools/lib/build-slot_test.sh:59` uses fd 9 for its own local
  post-release-lockability check (`flock -n 9 9>"$f"`). This is a different
  process/subshell than anything in `tools/lib/build-slot.sh` or
  `tools/with-build-slot.sh` and never runs concurrently with a real Task 8
  module-cache-lock holder in this suite, so there is no actual fd collision
  — but it is a stylistic near-miss of the "fd 9 is reserved" convention
  worth avoiding in a future edit (e.g. use fd 8 or another value) so a
  future reader doesn't have to reason through the "it's fine because it's a
  different process" argument.
- The report flags, and I confirm, a genuine but harmless ambiguity: the CLI
  usage synopsis in both the header comment and `usage()`
  (`tools/with-build-slot.sh:12,25`) shows `[--slots N] [--timeout SEC] <label>`
  (flags before label), while the brief's own acceptance-table invocations
  put `<label>` first. The parser (`tools/with-build-slot.sh:39-73`) accepts
  either order by design (loop over all leading tokens). Not a defect — both
  orders are exercised by the test suite and both work — but the usage text
  could note this explicitly for a human reader.

## Not evaluable

None. The full diff surface (six files) was read, all listed test suites were
executed directly against this worktree, `shellcheck -S error` was run via
`tools/shell-guard.sh`, deliberate-breakage checks were performed to confirm
test honesty, and the docker-gated owed fix was exercised end-to-end with
before/after `docker buildx ls` confirmation.

## Verdict

APPROVED. All brief requirements are met, the sourceability contract Task 7
depends on is verified to actually work under a caller's `set -euo pipefail`,
the exit-75/no-cleanup/fd-discipline constraints all hold, the tests are
demonstrably non-vacuous, and the owed buildx-bootstrap fix is verified to
restore the host's `atlas` builder.
