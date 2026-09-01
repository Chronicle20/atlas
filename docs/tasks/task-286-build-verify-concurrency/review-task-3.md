# Review: Task 3 — one bake solve instead of one per target (Layer 2, docker half)

Range reviewed: `52f568d7b..bb4a034b0` (1 commit, `bb4a034b0`).
Files: `docs/verification.md`, `tools/verify.sh`, `tools/verify_test.sh`.

## Scope confirmed

The diff matches the brief exactly: it replaces the per-target `docker buildx
bake` loop in `tools/verify.sh` with a single invocation, extends the block
comment above `BAKE_OUTPUT`, adds the five bake-selection assertions to
`tools/verify_test.sh` (using only the file's existing helpers), and updates
`## The docker layer` in `docs/verification.md`. No unrelated files touched.

## Findings

### PASS — target set preserved exactly, no target dropped or added

`bake_targets()` and the `TARGETS` array construction (`tools/verify.sh:369-386`)
are byte-for-byte unchanged from the pre-image (`git show 52f568d7b:tools/verify.sh`,
lines 293-315 of the old file vs. the same block in the new file). Only the
consumption changed:

- old: `for t in "${TARGETS[@]}"; do step "docker buildx bake $t" docker buildx bake --set "$BAKE_OUTPUT" "$t"; done`
- new: `step "docker buildx bake (${#TARGETS[@]} target(s))" docker buildx bake --set "$BAKE_OUTPUT" "${TARGETS[@]}"` (`tools/verify.sh:391-392`)

`step()` (`tools/verify.sh:101-114`) runs `"$@"` verbatim, so the new call
expands to `docker buildx bake --set VALUE t1 t2 ... tN` — the same target
list the loop iterated over, passed positionally to a single `bake` instead
of N separate ones. `TARGETS` is also read later, unchanged, at
`tools/verify.sh:761` for the `bake_targets=` fact line, confirming nothing
about the array itself was touched.

### PASS — no per-target loop remains

`grep -n 'for t in' tools/verify.sh` returns no matches in the working tree.
The new structural assertion (`tools/verify_test.sh:173-174`,
`grep -c 'for t in .\{0,4\}TARGETS' "$VERIFY"` == `0`) is sound against the
actual old code shape and passes.

### PASS — `tools/verify_test.sh`'s pre-existing agreement/structural
assertions are untouched and still pass

Ran the suite twice (the first run showed 4 stale failures traceable to a
leftover `tools/zz-verify-probe_test.sh` and other residue from a prior
session in this shared worktree, not from this diff — a clean rerun after
that residue cleared itself via the script's own `cleanup()` trap produced):

```
$ bash tools/verify_test.sh; echo EXIT=$?
ok   - SELECTED is appended in exactly one place
ok   - that one place is inside step()
ok   - no 'grep -q' in the change-detection path
ok   - selected gates agree (--quick --base HEAD)
ok   - skipped gate count agrees (--quick --base HEAD)
ok   - selected gates agree (--quick --base HEAD --no-ui)
ok   - skipped gate count agrees (--quick --base HEAD --no-ui)
ok   - --facts selects an untracked tools/*_test.sh
ok   - the real run selects it too
ok   - --facts lists it under guard_suites
ok   - a deploy/ change selects the LB port gate
ok   - removing it deselects the LB port gate
ok   - module count agrees
ok   - two changed go.mods select two bake targets
ok   - two bake targets produce exactly one bake gate
ok   - the gate names the target count
ok   - no probes, no bake gate
ok   - no per-target bake loop remains
ok   - --facts --all exits 0
ok   - --facts --all runs no check (0s)
ok   - --facts --all reports the fan-out reason
ok   - fact block carries 'base'
... (fact block carries × 9)
ok   - fact block is key=value lines only
ok   - no informational chatter on stdout

verify_test.sh: all assertions passed
EXIT=0
```

All 5 new bake-selection assertions from the brief's table are present and
passing, none of the pre-existing assertions (agreement loop, structural
checks, output contract) were weakened or deleted — confirmed by diffing the
file: the only removal is the single-line `cleanup()` body, replaced by a
multi-line body that does strictly more (adds cleanup of the two new probe
paths, keeps removing `probe_suite`/`probe_deploy`).

### PASS — `--facts --all` and `shellcheck` match the report's claims

```
$ bash tools/verify.sh --facts --all | grep -E 'bake_targets|docker buildx bake'
bake_targets=all-go-services
gate=docker buildx bake (1 target(s))

$ bash tools/shell-guard.sh --require-shellcheck
shell-guard: 76 script(s) OK (syntax + shellcheck -S error).
```

### PASS — probe fixture and cleanup correctness

`probe_bake_ban`/`probe_bake_account` are fixed names (not `$$`), declared
alongside the pre-existing `probe_suite`/`probe_deploy`, and folded into the
single `cleanup()` registered under the existing `trap ... EXIT`
(`tools/verify_test.sh:118-128`) exactly as the brief required ("extending the
existing `trap`", not adding a second `cleanup` definition — which the report
notes was caught by `shell-guard.sh` as SC2218 and fixed). The test body also
does an explicit interim cleanup of the two bake-probe files
(`tools/verify_test.sh:167-168`) before the "no probes, no bake gate"
assertion, which is correct and redundant with (not a bypass of) the trap.

### PASS — doc update reflects the code change and gives a reason

`docs/verification.md:207-212` replaces "`docker buildx bake atlas-<svc>` for
every service whose `go.mod` changed" with a statement that all selected
targets go into one solve, states the two reasons (one context transfer;
shared `libs/` mod-only/source layers within the solve), and adds the
failure-attribution note (BuildKit's own solve output names the failing
target/step). The "mandatory, not optional" sentence is preserved verbatim.
The layer-sharing claim is standard BuildKit multi-target-solve behavior
(shared DAG nodes across targets built from the same Dockerfile/base stages)
and is consistent with the existing cacheonly-layer-reuse note already in the
file a few lines below (`atlas-ban ... 27 CACHED steps ... 64 CACHED steps`);
I did not re-verify it by running a bake (out of scope per the dispatch), but
it is not a new or implausible technical claim.

### Non-blocking — doc sentence is slightly awkward

`docs/verification.md:210-211`: "...and BuildKit shares the `libs/` mod-only
and source layers across targets within the solve instead of resolving them
once per invocation." The clause "instead of resolving them once per
invocation" is describing the *old* per-target-loop behavior (resolved once
per each of N invocations = N times total), contrasted with "shared ...
within the solve" (once total, now). It reads ambiguously on a first pass
(could be misread as "once per invocation" being the *new* behavior). Not a
factual error, not blocking, but a candidate for a one-word tightening
("instead of once per separate invocation, as before") if anyone touches this
paragraph again.

### Cross-task shape — bake step is a single, extensible construct

The bake step is exactly one `step "..." docker buildx bake --set "$BAKE_OUTPUT" "${TARGETS[@]}"`
statement (`tools/verify.sh:391-392`), not inlined into the surrounding
`if/elif/else`. This is a clean attachment point for Task 5 (`--builder`/`--load`)
and Task 7 (build-slot acquire/release) to extend without restructuring.

## Not evaluable

- A real `docker buildx bake` was not run (per explicit dispatch instruction
  and controller ownership of the repo-wide `--quick` verification pass) —
  target-set equivalence was instead established by static comparison of the
  unchanged `bake_targets()`/`TARGETS` construction plus the passing
  `--facts` assertions, which is the evidence the dispatch asked for.
- The BuildKit cross-target layer-sharing claim in the doc update is accepted
  as plausible BuildKit behavior but was not independently confirmed by a
  real solve.

## Verdict rationale

No requirement from the brief was dropped. The defect class the dispatch
called out as most likely — a bake that silently builds fewer services than
before while still exiting 0 — is not present: the target-selection logic is
untouched, and the new invocation consumes the full `TARGETS` array. Tests are
additive, pass, and their pre-existing agreement/structural claims still hold.
The one doc wording note is cosmetic only.
