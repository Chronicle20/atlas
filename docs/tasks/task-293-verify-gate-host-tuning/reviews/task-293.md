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

## Re-review (9d3ee1cd3)

Range `d2d89a3af..HEAD`, one fix commit: `9d3ee1cd3` ("build(verify): derive K
and the Go pool width from one slot thread budget"). Files touched:
`tools/lib/build-slot.sh`, `tools/verify.sh`, `tools/lib/build-slot_test.sh`,
`docs/verification.md`, `docs/tasks/task-293-verify-gate-host-tuning/notes.md`,
plus this review artifact. Scope matches the prior finding exactly — no drift.

### 1. Blocking finding resolved for defaults, and for the doc's own "raise it" example

`_build_slot_threads()` (`tools/lib/build-slot.sh:79-85`) introduces
`ATLAS_SLOT_THREADS` (default 6, sanitized to a positive integer). `K` is now
`cores / slot_threads` (`tools/lib/build-slot.sh:96`), and `verify.sh` derives
`GO_JOBS_DEFAULT = slot_threads / GO_P`, floored at 1 (`tools/verify.sh:44-46`).

Measured on this host (`bash tools/lib/build-slot.sh` sourced, `cores=12`):

- Defaults: `threads=6, K=2, GO_JOBS=1, GO_P=6` → demand `= 2*1*6 = 12 = cores`. No oversubscription — the exact defect the prior review blocked on is gone.
- Doc's own "raise it" example, `ATLAS_SLOT_THREADS=12`: `K=1, GO_JOBS=2` → demand `= 1*2*6 = 12 = cores`. Holds.

I also swept the full range `threads=1..24` to check the general invariant
(`K = max(1, floor(cores/threads))`, `GO_JOBS = max(1, floor(threads/GO_P))`,
`GO_P=6` fixed, `cores=12`):

```
threads=1  K=12 GOJOBS=1 demand=72  OVERSUBSCRIBED
threads=2  K=6  GOJOBS=1 demand=36  OVERSUBSCRIBED
threads=3  K=4  GOJOBS=1 demand=24  OVERSUBSCRIBED
threads=4  K=3  GOJOBS=1 demand=18  OVERSUBSCRIBED
threads=5..15  ...                  OK (demand<=12)
threads=18 K=1  GOJOBS=3 demand=18  OVERSUBSCRIBED
threads=24 K=1  GOJOBS=4 demand=24  OVERSUBSCRIBED
```

The invariant is NOT universal: it holds for `threads` in roughly `[5, 17]`
on this 12-core host, but breaks both below `GO_P` (small `ATLAS_SLOT_THREADS`
combined with the fixed `-p 6` floor forces `GO_JOBS` to stay at 1 while `K`
grows unbounded) and above the core count (`K` floors at 1 but `GO_JOBS`
keeps growing with `threads`). This is a real residual gap in the general
formula, but it is narrower than what the task asked me to re-confirm:
the task's own phrasing was "holds with defaults, and holds when
`ATLAS_SLOT_THREADS` is raised" — and `docs/verification.md:216-219`'s only
worked "raise it" example is `ATLAS_SLOT_THREADS=12`, which I confirmed above
stays exactly at the core budget. Nothing in the diff or the docs endorses
raising past the physical core count or dropping below `ATLAS_GO_P`, so I am
not blocking on the `threads=18`/`threads=24` or `threads<5` cases, but they
are worth naming as a **non-blocking finding**: `_build_slot_threads()` does
not validate `ATLAS_SLOT_THREADS` against `nproc`/`ATLAS_GO_P`, so an operator
who follows the doc's own hint ("raise `ATLAS_SLOT_THREADS`... on a host that
can afford it") past the actual core count, or who lowers it below `ATLAS_GO_P`
on a small VM, gets silent oversubscription with no error — the same class of
bug this task exists to eliminate, just requiring a deliberate value instead
of the shipped default.

### 2. Sourcing order and arithmetic safety under `set -euo pipefail`

`tools/verify.sh:26-27` sources `tools/lib/build-slot.sh` immediately after
`set -euo pipefail`, well before `_build_slot_threads` is first called at
`tools/verify.sh:44`. Confirmed by `grep -n` — only one `. "$ROOT/tools/lib/build-slot.sh"` in the file, at line 27.

`GO_P` is read and validated at `tools/verify.sh:41-42`
(`case "$GO_P" in ''|*[!0-9]*|0) ... exit 2 ;; esac`) *before* the division at
`tools/verify.sh:44` (`GO_JOBS_DEFAULT=$(( $(_build_slot_threads) / GO_P ))`),
so a non-numeric or zero `ATLAS_GO_P` is rejected with a clear message and
exit 2 rather than reaching arithmetic (which would otherwise be a shell
syntax/divide-by-zero error under `set -euo pipefail`, since `$(())` is not
subject to `-e` in the same way but a divide-by-zero is a hard shell error
regardless). `_build_slot_threads` itself sanitizes `ATLAS_SLOT_THREADS`
against non-digit input and clamps to a minimum of 1
(`tools/lib/build-slot.sh:80-83`), so it can never return `0` or an empty
string to the divisor on the `build-slot.sh` side either. Both guards are in
place and ordered correctly. PASS.

### 3. Doc table and prose agree with the new defaults

`docs/verification.md:195-207` table: `ATLAS_SLOT_THREADS | 6`,
`GOMAXPROCS`/`go build -p` (`ATLAS_GO_P`) `| 6`, `Go pool workers per gate
(ATLAS_VERIFY_GO_JOBS) | slot threads / -p = 1`, `K (ATLAS_BUILD_SLOTS) |
physical cores / slot threads = 2 on a 12-core host`. All four match the code
(`_build_slot_threads` default 6, `tools/verify.sh:41` `GO_P` default 6,
computed `GO_JOBS_DEFAULT=1`, `_build_slot_default` `K=2` on this host, per
`build-slot_test.sh`'s own `cores=12` assertion). The prose at
`docs/verification.md:216-219` ("Now 2 slots × 1 worker × 6 threads = 12 = the
cores... raise `ATLAS_SLOT_THREADS` to 12... and both K and the pool follow")
matches the arithmetic verified in §1. No mismatch remaining. PASS.

`notes.md:30-38` was also updated to describe the new single-budget model and
explicitly records "Review round 1 caught the first cut of this" — an honest
paper trail of the prior blocking finding. Consistent with the diff.

### 4. No regression in the earlier-passing checks

- `bash tools/lib/build-slot_test.sh`: 11/11 assertions pass, including the two new cases (`slot budget above the core count floors K at 1`, `a 1-thread slot budget yields K = physical cores`) and the original 9. `cores=12` reconfirmed on this host.
- `shellcheck -S error` on all four changed shell files (`tools/verify.sh`, `tools/lib/build-slot.sh`, `tools/lib/build-slot_test.sh`, `tools/verify_test.sh`): clean, exit 0.
- `tools/verify_test.sh` was not touched by this fix commit (absent from `git diff --stat d2d89a3af..HEAD`); the structural assertions the prior review already checked (`acquire_build_slot` once, `with-build-slot.sh` once, label normaliser) are untouched by this commit and were not re-broken — `go_layer()`'s only change is reading `$GO_P` instead of `${ATLAS_GO_P:-6}` inline (`tools/verify.sh:511`), same value, no behavior change to the build command itself.
- The `--facts` isolation, `renice`/`ionice` block, `step()` timing, and `.claude/commands/execute-task.md` findings from the prior review are all outside this fix commit's diff (`git diff --stat` above) and unaffected.

### Not evaluable (unchanged from round 1, plus one addition)

- Real-world contention/scheduling behavior on the dev host — still a runtime claim a static review cannot confirm.
- Whether an operator would ever actually set `ATLAS_SLOT_THREADS` outside the `[GO_P, physical_cores]` range in practice — the arithmetic gap in §1 is verifiable from the code, its practical likelihood is not.

## Re-review verdict

APPROVED_WITH_FINDINGS. The blocking finding from round 1 is resolved: with
shipped defaults, `K × GO_JOBS × GO_P = 12 = physical cores` on the 12-core
reference host (previously 24), and the doc's own worked "raise it" example
(`ATLAS_SLOT_THREADS=12`) also lands exactly at the core budget. Sourcing
order and arithmetic guards are correct under `set -euo pipefail`; the doc
table and prose agree with the code; no regression in the previously-passing
checks. One non-blocking finding remains: `_build_slot_threads()` does not
validate `ATLAS_SLOT_THREADS` against `ATLAS_GO_P`/`nproc`, so a value chosen
outside the documented `[GO_P, physical_cores]` window (e.g. below 6 or above
12 on this host) silently reproduces oversubscription — narrower than, but
the same class of bug as, the one this task exists to fix.
