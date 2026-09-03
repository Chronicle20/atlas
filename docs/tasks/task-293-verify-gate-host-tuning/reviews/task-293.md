# Review: task-293 — verify-gate host tuning

Commit under review: `d2d89a3af` (`build(verify): size the gate for physical
cores and slot/nice the inner loop (task-293)`), range `main..HEAD`.

Brief: `docs/tasks/task-293-verify-gate-host-tuning/notes.md` (trivial-fix
tier, brainstorming record — 7 numbered changes).

## Scope confirmed

Shell tooling only: `tools/verify.sh`, `tools/lib/build-slot.sh`,
`tools/lib/build-slot_test.sh`, `tools/verify_test.sh`, `docs/verification.md`,
`.claude/commands/execute-task.md`. No Go changed. Matches
`tools/change-surfaces.sh` output and the notes' seven items. No scope
mismatch.

## Findings

### BLOCKING — the K derivation ignores `GO_JOBS`, so shipped defaults still oversubscribe a 12-core host by 2x

`tools/lib/build-slot.sh:75-77` derives K from a stated assumption that **one
slot costs 6 threads** ("One slot is budgeted at 6 threads (GOMAXPROCS / go
build -p ...), so K = physical_cores / 6"). That formula has no term for
`ATLAS_VERIFY_GO_JOBS` — the number of `go build -p 6` workers `verify.sh`'s
Go pool runs *concurrently inside one acquired slot* (`tools/verify.sh:519-533`,
loop bound `while [ "$(jobs -rp | wc -l)" -ge "$GO_JOBS" ]`).

With the shipped defaults (`tools/verify.sh:43` `GO_JOBS=2`, `go_layer()`
`tools/verify.sh:504` `go build -p "${ATLAS_GO_P:-6}"`), a single gate's Go
pool can run **2 concurrent `go build -p 6` processes** the moment two or
more modules are in the change set — i.e. exactly the `libs/` fan-out
scenario the notes name as a trigger. That is 12 threads for *one* gate, not
the 6 threads `_build_slot_default` budgeted per slot.

`tools/verify.sh:38-40` and `docs/verification.md:213-214` both say this out
loud and treat it as correct: "Two workers at `go build -p 6` is 12
threads — one slot's worth" / "a gate is 2 workers × 6 threads = one slot."
That statement is internally inconsistent with the K formula two paragraphs
above it (`docs/verification.md:195`, `K = physical_cores / 6`): if one slot's
true footprint is 12 threads, K should be `physical_cores / 12` (= 1 on this
12-core host), not `physical_cores / 6` (= 2).

As shipped, `_build_slot_count` returns K=2 on the reference 12-core host
(confirmed by running `build-slot_test.sh`: `cores=12`, and the new case
asserts `default K is physical_cores/6 floored at 1`). The broker therefore
lets **two gates run concurrently**, each of which can peak at
`GO_JOBS × ATLAS_GO_P` = 12 threads → **24 threads of demand on a 12-thread**
(12-physical-core) host. That is the same class of 2x oversubscription this
task's whole stated purpose was to eliminate (notes.md: "Four slots × 6
threads was ~2x oversubscribed... Inside each slot the Go pool ran 4 workers
at `go build -p 6` — 24 threads in a slot budgeted for 6") — just diluted from
~4x to 2x, not fixed. The fix is either `K = physical_cores / (GO_JOBS *
ATLAS_GO_P)` in `_build_slot_default`, or `GO_JOBS` default of 1 (matching the
"6 threads per slot" convention the divisor already encodes) — but not both
"K=2" and "GO_JOBS=2" as currently shipped, since they double-count the same
core budget.

This is blocking because it is not a docs nit: it is the numeric core of the
change the task exists to make, and the contradiction is verifiable by
comparing `tools/lib/build-slot.sh:75-77` against `tools/verify.sh:38-40` /
`docs/verification.md:213-214` — all three describe the *same* number
differently (6 threads/slot in the divisor, 12 threads/slot in the prose two
lines away).

## Everything else checked — PASS

**`step()` timing under `set -euo pipefail`** (`tools/verify.sh:154-174`).
`"$@"` runs inside an `if`, so the -e trap never fires on a failing step; `rc`
is captured explicitly. `STEP_SECS_OVERRIDE` is read once and cleared
(`tools/verify.sh:163-166`), so it cannot leak into the next step() call.

**`STEP_SECS` / `GO_POOL_SECS` / summary line.** `declare -A STEP_SECS=()` and
`VERIFY_T0=$SECONDS` are set once at top level (`tools/verify.sh:129-132`);
`GO_POOL_SECS=0` before `launch_go_layers` is defined and updated by the
function running in the main shell, not a subshell, so the assignment
survives (`tools/verify.sh:515-533`). The summary line at
`tools/verify.sh:1119-1121` reads `STEP_SECS[$s]:-0` defensively for any label
without a recorded time.

**`renice`/`ionice` block** (`tools/verify.sh:94-102`). Gated on
`[ "$FACTS" -eq 0 ]`, placed after option parsing so `FACTS` is already set;
both calls are `|| true` and `ionice` is only invoked if present via
`command -v`. `ATLAS_VERIFY_NICE=0` disables it as documented.

**`--facts` isolation.** `STEP_SECS_OVERRIDE` is only read inside
`if [ "$FACTS" -eq 0 ]` (`tools/verify.sh:582-584`); `launch_go_layers` (which
sets `GO_LOG_DIR`) is only ever called from the `[ "$FACTS" -eq 0 ]` branch
(`tools/verify.sh:567-579`), so `GO_LOG_DIR` is never referenced under
`--facts`. The `--facts` path exits at `tools/verify.sh:1109` before the
summary block that uses `STEP_SECS`/`GO_POOL_SECS`, so none of the new timing
state can affect `--facts` output. `build-slot_test.sh` and `verify_test.sh`
were run; both pass (`shellcheck -S error` clean on all four changed shell
files; `bash tools/lib/build-slot_test.sh` — 9/9 assertions ok, confirms
`cores=12` on this host).

**`verify_test.sh` structural assertions.** `acquire_build_slot` appears
exactly once, at `tools/verify.sh:573`, inside the `# --- go modules` banner
section (`tools/verify.sh:278-594`); `with-build-slot.sh` appears exactly
once, at `tools/verify.sh:680`, inside `# --- docker` (`594-685`); nothing
from `# --- guards` (685) through `# --- summary` (1112) references either.
That satisfies "exactly one acquire_build_slot in the go-modules section, two
total, none from guards through summary" (`tools/verify_test.sh:325-340`).
The label normaliser (`tools/verify_test.sh:76`) strips ANSI first
(`sed 's/\x1b\[[0-9;]*m//g'`), then `s/  ([0-9]*s)$//` — correct, since the
printed line is `✓ label  \033[2m(Ns)\033[0m`, and stripping color codes
first leaves a literal `label  (Ns)` for the second sed to match.

**`--quick` and `lint.sh --fmt`.** `tools/verify.sh:964-978` builds
`LINT_ARGS=(--check --go [--base <rev>] [--fmt])` under `--quick`, matching
`lint.sh`'s documented check-mode formatter path
(`tools/lint.sh:26-31,204-225`): `CHECK=1` + `FMT_ONLY=1` runs
`golangci-lint fmt --diff` (a parse) and skips the `run` linter block
(`tools/lint.sh:212-213,227`) — genuinely exercises the check-mode formatter
path, not a no-op.

**Doc/code agreement (besides the blocking K/GO_JOBS mismatch).**
`docs/verification.md`'s "Go pool workers per gate
(`ATLAS_VERIFY_GO_JOBS`) | 2" table row matches `tools/verify.sh:43`;
"`--quick` runs `go build/vet` and now takes a slot too" matches
`tools/verify.sh:568-573`; `ATLAS_VERIFY_NICE` default 10 matches
`tools/verify.sh:96`; the "resolved footguns" list (golangci-lint lock
contention fixed by #1413, nvm-on-PATH fixed by `node-env.sh`) is consistent
with the notes' claim that both were already fixed elsewhere — this review
did not re-verify #1413/`node-env.sh` themselves (out of this unit's diff;
see Not evaluable).

**`.claude/commands/execute-task.md` — item 6 (gate per range).** The four
trigger conditions in the diff (two-or-more tasks landed, `libs/`/`go.work`
touched, next-is-handoff-or-plan-end, `DONE_WITH_CONCERNS`) match notes.md's
list verbatim. `--quick --base` per-range wording and the "only the flagless
run counts as verified" sentence are unchanged in substance, just extended to
name what `--quick` now skips (linters, `-race`).

## Not evaluable

- Whether the renice/ionice block actually improves interactive-session
  responsiveness on the real dev host — this is a runtime/host-tuning claim
  that a static review of the diff cannot confirm or refute; the notes flag
  it as inference-turned-into-a-lever, and the summary's new timing output is
  the intended way to check it empirically over time. Not something this
  review can evaluate from the diff alone.
- Whether the 2x-oversubscription math above is actually observable as
  contention in practice (vs. absorbed by kernel scheduling headroom,
  I/O-bound phases, or the two gates rarely both hitting peak
  `GO_JOBS × ATLAS_GO_P` at the same instant) — the arithmetic mismatch in
  the shipped constants is verifiable from the diff; its real-world severity
  is not, and would need to be measured with the new per-step/pool timing
  this same commit adds.

## Verdict

CHANGES_REQUIRED. The blocking finding is squarely inside the diff's own
numbers (`tools/lib/build-slot.sh:75-77` vs `tools/verify.sh:38-40` vs
`docs/verification.md:213-214`) and undermines the task's stated purpose:
fixing gate-induced host contention on a 12-core box. Everything else in the
seven-item change list — timing, `--facts` isolation, `verify_test.sh`
structural invariants, `--quick`'s `--fmt` path, and the `execute-task.md`
cadence rule — checks out against the brief and against the code.
