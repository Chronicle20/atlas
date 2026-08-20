# Review — Task 10: `bootstrap.sh` activation step (FR-4.1/4.2)

Commit reviewed: `fff75ebba` only. Confirmed `100764219` (Task 9) is excluded —
`git show fff75ebba` touches only `services/atlas-pr-bootstrap/scripts/bootstrap.sh`
(+88) and `services/atlas-pr-bootstrap/test/bootstrap_test.bats` (+50); no
overlap with Task 9's diff hunks was re-reviewed here.

## Scope

Read: full diff of `fff75ebba` (both files), `env-record.sh` (env_record_get /
env_record_patch contract), `services/atlas-configurations/.../environments/processor.go`
(server-side `validatePhaseTransition`, `UpdateByName`) as the seam the PATCH
depends on, and the surrounding un-diffed regions of `bootstrap.sh` (Task 9's
override-set loop, `record_environment_tenant`) needed to judge correctness of
this task's insertion point and variable reuse. Ran the bats suite and
`shell-guard.sh --require-shellcheck` myself rather than trusting the report's
numbers.

## 1. Phase-transition safety

`env_record_patch` in the `activate` branch (bootstrap.sh:810-815) sends the
literal phase `ACTIVE`, not the record's current phase — this is the
constraint-lift the brief calls for, and it is real: `record_environment_tenant`
(bootstrap.sh:110-126) preserves whatever phase it read, this block overwrites
it.

The safety net is server-side, not client-side, and it holds:
`environments/processor.go:82-88` (`validatePhaseTransition`) rejects any
transition except no-op or exactly one step along
`PROVISIONING -> ACTIVE -> DEACTIVATING -> DELETED`
(`environments/processor.go:47-89`). So even though `bootstrap.sh`'s
`activation_decision` check-then-act has a TOCTOU window (phase is read, then
PATCHed, with no lock in between), the two races that matter are both closed:
- A concurrent teardown that already advanced the record past PROVISIONING
  before the PATCH lands gets `ErrIllegalPhaseTransition` from the server (the
  bootstrap script's `env_record_patch` call fails, `|| exit 1` fires,
  bootstrap.sh:816).
- A second/duplicate activate racing the first: if the first PATCH lands
  first, the second's PATCH (also literal `ACTIVE`) is a same-phase no-op,
  which `validatePhaseTransition` explicitly allows (`ti == fi`,
  processor.go:85).
No path clobbers a later phase or resurrects an earlier one. This composes
correctly with the existing library contract; verified, not assumed.

Rollout is verified, not assumed, before any PATCH: every deployment in the
override set gets `kubectl rollout status ... || { log error; exit 1; }`
(bootstrap.sh:781-788), fatal, before the phase branch runs at all. Confirmed
this is a hard exit, not `log warn` — matches the brief's explicit callout.

## 2. Sparse-only gating

The whole block is behind `if [ "${ATLAS_MODE:-isolated}" = "sparse" ]; then`
(bootstrap.sh:773), identical guard expression to Task 9's service-config loop
(bootstrap.sh:633) and `record_environment_tenant`'s call site
(bootstrap.sh:381). Verified this is the literal guard, not a look-alike —
`grep -n` confirms one occurrence at :773. The bats test for this
(`activation is sparse-only`, bootstrap_test.bats +196-207) greps the script
for the exact string rather than exercising the branch; the implementer's
report and the test's own comment both flag this as weak coverage (would not
catch an inverted-then-renegated condition). Accepted per the brief's
documented fallback — a genuine black-box seam does not exist in this test
file (confirmed: no fixture drives the script past `require_env`/preflight
without a live cluster).

## 3. Idempotency / re-run

- Already-ACTIVE re-run: `activation_decision ACTIVE` → `skip`
  (bootstrap.sh:139), no PATCH sent, only a log line. Covered by
  `activation_decision: ACTIVE skips` (bootstrap_test.bats +183-188).
- A re-run that reaches `activate` again after a successful first activation
  is unreachable in practice (phase would already be ACTIVE by then) but is
  also safe by the server-side same-phase rule above.
- The mandatory-socket check (bootstrap.sh:790-794) runs unconditionally
  before the phase switch, including on the `skip` path — matches the
  brief's stated ordering (rollout loop → mandatory-socket assert → phase
  decision) and the implementer's self-review note.

## 4. Interaction with Task 9

- Insertion point: verified by `grep -n 'ATLAS_STEP='` that this task's block
  sits at bootstrap.sh:773-819, immediately before `ATLAS_STEP=done` at
  bootstrap.sh:820 — correctly located post-Task-9 by content, not by the
  brief's stale pre-Task-9 line number (:676). Matches the implementer's
  self-review claim.
- `overrides` variable reuse: Task 9's loop (bootstrap.sh:651) and this
  block (bootstrap.sh:778, again at :812) both assign a top-level (non-local)
  `overrides` variable. No collision — each assignment is a fresh
  `env_record_get | jq ...` read immediately before use, and the two blocks
  never run interleaved (script is single-threaded, Task 9's loop is long
  finished by the time this block runs). Confirmed no other code path reads
  `overrides` after Task 9's loop and before this block's first
  reassignment.
- `tools/derive-service-id.sh`'s no-trailing-newline contract: not touched.
  Task 10 does not call `derive-service-id.sh` or read `SERVICE_ID_*`
  anywhere in this diff (confirmed via `grep -n derive-service-id
  bootstrap.sh` — only Task 9's pre-existing comment references at
  lines 544/552, both outside this diff). No regression surface here.

## Tests / verification run myself

```
bats services/atlas-pr-bootstrap/test/bootstrap_test.bats
```
`1..14`, all `ok`, including the 6 new activation tests (`ok 9`-`ok 14`).
Matches the report's numbers.

```
tools/shell-guard.sh --require-shellcheck
```
`shell-guard: 71 script(s) OK (syntax + shellcheck -S error).` Matches.

## Non-blocking findings

1. **`bootstrap.sh:778`** — `overrides=$(env_record_get | jq -r '.data.attributes.overrides // {} | keys[]')`
   has no `|| ...` fallback, unlike the second `env_record_get` call in the
   same block (bootstrap.sh:806, `body=$(env_record_get) || body=""`) and
   unlike `record_environment_tenant`'s pattern. Since `pipefail` is active
   (inherited from the top of the script) and `set -e` is restored at
   bootstrap.sh:20, a failed GET here (e.g. a 404 if the control-plane
   record vanished between rollout and activation) aborts the script via
   bare `set -e` rather than through the deliberate `log error; exit 1`
   path the rest of the block uses. The net effect is still a non-zero exit
   and no PATCH sent — not a safety defect — but the error message the
   operator sees would be curl's raw stderr instead of a clear
   "activation: ..." log line. This mirrors a pre-existing gap in Task 9's
   own identical line (bootstrap.sh:651), so it is not a regression
   Task 10 introduced, but Task 10 had the opportunity to close it in the
   one line it added a twin of and didn't.
2. Commit message ("via the same current-phase-preserving PATCH pattern
   record_environment_tenant uses") is a little imprecise: the *plumbing*
   (four-`jq`-read-then-PATCH shape) is shared, but the actual phase value
   sent is not preserved — it is the literal `ACTIVE`, which is the entire
   point of the block. The code and header comment are accurate; only the
   commit subject/body summary slightly overstates the similarity. Not
   worth a fixup commit on its own.

## Not evaluable

- Live-cluster behavior of `kubectl rollout status` under real timeout/
  failure conditions — no cluster available in this review; judged the
  fatal-exit wiring only (bootstrap.sh:781-788), not runtime behavior against
  a real Deployment.
- Server-side `UpdateByName`'s own race window between `GetByName` and the
  transactional `update()` (processor.go:246-266) — pre-existing library
  code untouched by this task's diff; noted only to establish that the
  transition-legality guarantee this task depends on is real, not to audit
  the library itself.

## Verdict rationale

All four brief requirements are implemented and verified against the actual
code (not just the report's claims): the phase transition is safe via a real
server-side invariant, sparse-only gating is the correct literal guard,
idempotency holds via `skip` + same-phase-transition allowance, and Task 9's
insertion point / variable reuse / trailing-newline contract are all
undisturbed. The one issue found (item 1 above) is a robustness/error-message
gap, not a correctness or safety defect, and mirrors an existing pattern in
the file rather than introducing a new one.
