# Review — Task 5: thread `h.Ctx` into `AddBody`, retry a failed listener add

Commit range: `345593a61..92ae4c480` (single commit `92ae4c480`)
Reviewer surface: `services/atlas-channel/atlas.com/channel/configuration/projection/loop.go`,
`.../configuration/projection/projection_test.go` — the only two files the diff touches.

## Scope confirmation

The diff touches exactly the two files named in `task-5-brief.md` (`loop.go`,
`projection_test.go`) and nothing else — `git diff --stat 345593a61..92ae4c480`
confirms `2 files changed, 258 insertions(+), 6 deletions(-)`. No files under
`listener/`, `socket/`, or `main.go` are touched by this commit. Scope matches
the brief.

## Requirement-by-requirement

### Step 3 — thread `h.Ctx` into `AddBody`

`loop.go` (in `execute`'s `OpAdd` branch): `return a.AddBody(h.Ctx, op.Key, op.Cfg, h)`,
with `Registry.Add`'s outer `parent` argument left as `ctx` (the apply-loop's
own context). Verified against `listener/registry.go:108-113`:
`ctx, cancel := context.WithCancel(parent); h := &Handle{..., Ctx: ctx, Cancel: cancel, ...}`
— `h.Ctx` is per-Handle and derives from the apply-loop `ctx`, and
`Drain`'s phase 4 (`listener/registry.go:230`, `h.Cancel()`) is the only thing
that cancels it. Passing the apply-loop's own long-lived `ctx` (the pre-fix
behavior) would never observe a per-channel `Drain`, which is exactly
task-244 defect 1. **PASS** — matches the brief's Step 3 code block verbatim,
comment included.

### Step 4 — pending-retry loop

`ApplyLoop` gained `pending map[server.Key]Op` and `retries map[server.Key]int`,
both initialized in `Run` before the ticker loop. Each tick: retry `pending`
first (dropping any key no longer in `ComputeOps(nil, nil, nextSvc, nextTenants)`'s
flattened desired set — verified `flatten`/`ComputeOps` semantics at
`apply.go:60-85`, confirms `ComputeOps(nil, nil, next, next)` yields exactly the
currently-desired `OpAdd`s with no drains, since `prevDesired` is empty), then
run the freshly diffed `ops`, queueing any failed `OpAdd` into `pending`.
`execute` now returns `error`; `OpDrain` and `OpAdd` failure paths both
`return err`; `ErrDraining` is distinguished from other errors and logged at
Debug (expected churn) vs. Warn-once/Debug-thereafter for persistent
conflicts, keyed off `a.retries[op.Key]`. **PASS** — matches the brief's Step 4
code block verbatim, including the Warn-then-Debug log-level logic and the
`"errors"` import addition.

### Step 1/2/5 — tests, and honesty of the RED claim

Three new tests (`TestApplyLoop_AddBodyReceivesAContextCanceledByDrain`,
`TestApplyLoop_RetriesAFailedAddOnTheNextTick`,
`TestApplyLoop_DropsAPendingAddWhenTheKeyLeavesConfig`) drive a real
`listener.NewRegistry`/`ApplyLoop.Run` tick loop, matching the brief's setup
(stub `AddBody`, stub `ServerModel`, `defer server.GetRegistry().Deregister(key)`,
`defer cancel()` on every test's context). Ran locally:

```
$ go build ./...
$ go test ./configuration/projection/... -v -run TestApplyLoop
=== RUN   TestApplyLoop_AddBodyReceivesAContextCanceledByDrain
--- PASS: TestApplyLoop_AddBodyReceivesAContextCanceledByDrain (0.01s)
=== RUN   TestApplyLoop_RetriesAFailedAddOnTheNextTick
--- PASS: TestApplyLoop_RetriesAFailedAddOnTheNextTick (0.03s)
=== RUN   TestApplyLoop_DropsAPendingAddWhenTheKeyLeavesConfig
--- PASS: TestApplyLoop_DropsAPendingAddWhenTheKeyLeavesConfig (0.11s)
PASS
```

Test honesty (not vacuous, traced by hand rather than by re-running a revert,
since edits to the working tree are out of bounds for a reviewer): the
defect-1 test captures the `parent` context argument `AddBody` receives, then
calls `registry.Drain(key)` and asserts `parent.Err() == context.Canceled`.
Only `h.Cancel()` at `listener/registry.go`'s Drain phase 4 can produce that
transition, and `h.Cancel()` only reaches a context derived from the
Handle's own `h.Ctx` — not the apply-loop's outer `ctx`. So this assertion is
genuinely load-bearing: the pre-fix code (passing `ctx` instead of `h.Ctx`)
would leave the captured context uncancelled forever, and the implementer's
reported RED run (`task-5-report.md` lines 16-27) is consistent with that
reasoning. The retry test counts real `AddBody` invocations via
`atomic.Int32` and asserts real registry state via `registry.Get(key)`, not a
mock-call assertion — genuine coverage of the retry path. The drop test
snapshots the call count right after `ApplyServiceTombstone()` and asserts no
further growth over 5+ tick intervals — genuine coverage of the "stop
retrying once the key leaves config" path.

### Step 6 — full suite

Not independently re-run under `-race` per the task instructions (the known
`listener/registry_test.go` `-race` flake is assigned to a separate fix round
and out of scope here); the module-local flagless build+test above is
consistent with the implementer's report. `tools/verify.sh` is a controller
gate, not part of this review.

## Design-level observation (non-blocking)

`loop.go`'s primary-`ops` loop unconditionally does
`a.retries[op.Key] = 1` on a fresh `OpAdd` failure, rather than incrementing —
if a key were simultaneously present in `a.pending` (from a prior tick) *and*
reappeared in the same tick's freshly-diffed `ops` (a same-tick
drain-then-re-add landing while a retry is also outstanding for the same
key), the retry counter would reset to 1 rather than continue climbing. This
is an edge case in the state machine as specified, not an implementation
defect: the diff matches the brief's Step 4 code block verbatim, so this is a
design-time decision already baked into `task-5-brief.md`/`design.md`, not
something Task 5's implementer introduced or could have deviated from.
Flagging for visibility only; does not block this task.

## Out of scope (per controller instruction)

The `listener/registry_test.go` `-race` flake reported in `task-5-report.md`'s
"Concerns" section is real but belongs to a separate, already-in-flight fix
round; not evaluated here and not counted against Task 5.

## Verdict rationale

Every Task 5 brief step (3, 4, 1/2/5) is implemented exactly as specified,
verified against the actual callee contracts (`Registry.Add`, `Handle.Cancel`,
`ComputeOps`/`flatten`) rather than taken on faith, and the new tests are
non-vacuous. No blocking defects found within this commit's diff.
