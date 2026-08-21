# Review — fix round 3 (task-244)

Range: `1e1042a10..9233f9339` (2 commits)
- `9bce698ea` fix(atlas-channel): deregister kafka handlers when Add's body fails
- `9233f9339` test(atlas-channel): fix synchronization race in tombstone-drop test

Brief: `.superpowers/sdd/plan/fix-round-3-brief.md`
Implementer report: `.superpowers/sdd/plan/fix-round-3-report.md`

## Scope confirmed

`git diff --stat 1e1042a10..9233f9339`:

```
configuration/projection/loop.go            |   5 +
configuration/projection/projection_test.go |  23 ++
listener/handle.go                          |   8 ++
listener/registry.go                        |  23 +++-
listener/registry_test.go                   |  43 +++++
main.go                                     | 132 +++++++++------
6 files changed, 169 insertions(+), 65 deletions(-)
```

Matches the brief's two findings exactly — no unrelated file touched. The
commit split (one commit per finding, conventional-commit style) matches the
brief's instruction.

## Finding A — Kafka handler leak on Add's rollback path

**Shape taken:** exactly the brief's preferred shape — `AddBody`/`body`
returns handles-on-error, `Registry.Add`'s rollback deregisters via
`r.deps.RemoveHandler`, mirroring `Drain` phase 4.

- `listener/registry.go:130-155` — `handlers, err := body(h)`; on error, loops
  `for _, hh := range handlers { r.deps.RemoveHandler(hh.Topic, hh.Id) }`
  before the existing `delete(r.entries, key)` / `refs--` / `cancel()`
  sequence, warn-logging failures with
  `listener.add.rollback_remove_handler_failed` — same shape as `Drain`
  phase 4 (`registry.go:251-258`, unchanged). Verified by reading the
  function in full; the pre-existing rollback ordering (delete entry, decr
  refs, cancel ctx) is untouched, only the new deregister loop is prepended.
- `listener/handle.go:28-41` and `registry.go:89-98` — the contract
  ("`body` MUST return every `HandlerHandle` it has already registered even
  when it also returns a non-nil error") is documented on both
  `HandlerHandle` and `Add`. `configuration/projection/loop.go:17-22`
  documents the same contract on `AddBody`'s type declaration, which is the
  interface `main.go`'s closure satisfies.
- `listener/registry_test.go:368-410` — new
  `TestRegistry_AddRollsBackHandlersRegisteredBeforeBodyFails`: `body`
  returns 3 `HandlerHandle`s plus a bind-style error;
  `require.ElementsMatch(registered, removed)` proves every handle body
  registered is passed to `RemoveHandler`; `Get`/`Snapshot` prove the entry
  is gone. Ran it standalone (`go test ./listener/... -run
  TestRegistry_AddRollsBackHandlersRegisteredBeforeBodyFails -race
  -count=5`) — 5/5 pass. This test is genuinely load-bearing: against
  pre-fix `registry.go` (no deregister loop in the rollback branch)
  `removed` would stay empty and `require.ElementsMatch` would fail — not
  coverage that passes either way.

**Scope widening — verified site by site.** The brief cited one leak site
(`main.go:630-634`, the socket bind-error return). The implementer changed
every `return nil, err` in `buildListener`'s body from the point `handles`
is declared (`main.go:433`) through the end of the function (`main.go:443`
through `main.go:639`) to `return handles, err` — approximately 55
`register()` failure sites plus the bind-error site.

- Read the full diff for `main.go` (`git diff 1e1042a10..9233f9339 --
  main.go`): every changed line matches `-return nil, err` /
  `+return handles, err`, one-for-one, with a single non-mechanical hunk (a
  comment expansion at the bind-error site, `main.go:632-639`). No stray
  edits — confirmed by grepping the diff for any `+`/`-` line that isn't one
  of those two forms; only the comment block matched as an exception.
- The claim that each site is "the same leak at an earlier failure point"
  holds uniformly: `handles` is declared once (`main.go:433`) and
  accumulated only through the local `register` closure
  (`main.go:434-440`), which appends `hh` only when the wrapped
  `InitHandlers` call succeeds. Every subsequent `if err := register(...); err
  != nil { return handles, err }` therefore returns exactly the handles
  registered by every *prior* successful `register()` call in the same
  invocation — never handles from a call that hasn't run yet, never a stale
  or double-counted set. This is the identical bug at every one of the ~55
  sites: an `N`th `InitHandlers` failure previously discarded the `N-1`
  handler registrations already live.
- The two return sites *before* `handles` is declared (`main.go:390`,
  tenant missing from state; `main.go:395`, `tenant.Register` failure) are
  correctly left as `nil, err` — nothing has been registered yet at those
  points, so `handles, err` would be a no-op accompanied by a bogus claim
  the site returns handles.
- No error-path site after the bind (`main.go:640-646`) exists that was
  missed — the function ends at `return handles, nil` immediately after the
  bind succeeds.

**One residual gap, not introduced by this round, correctly out of scope.**
`register`'s wrapping only accumulates handles from `InitHandlers` calls
that returned successfully in full; it does not protect against a single
`InitHandlers` call that registers a subset of several topics and then
fails partway through (e.g. `kafka/consumer/fame/consumer.go:33-49`
discards its own partial `handles` on a mid-call `rf` failure via `return
nil, err` at line 42, one level below `main.go`). That intra-package
leak is a real but narrower and pre-existing gap the brief never scoped to
(the brief's files list is `registry.go`, `handle.go`, `registry_test.go`,
`main.go`, and the `configuration/projection` package only — not
`kafka/consumer/*`), and this round did not touch any `InitHandlers`
implementation. Noting it as non-blocking for a future round, not a defect
in this one.

**Test-double coverage.** Brief required every `AddBody` implementation and
test double be updated to match the contract, `projection_test.go` named
explicitly. Verified: `grep -rl AddBody` across the module surfaces only
`main.go` (real implementation, fixed above) and
`configuration/projection/projection_test.go` (3 inline `AddBody` closures
at lines 327, 380, 425). All three already `return nil, err` on the error
path — a valid zero-handles instance of the "return every handle already
registered" contract (they register nothing before failing in their test
scenarios), so no change to them is required. This is a correct,
non-lazy read of the contract, not a shortcut.

Finding A verdict: **holds**. Shape matches the brief, the widening is
uniform and individually justified at every site, the contract is
documented in all three places the brief named, and the new test is
load-bearing.

## Finding B — `TestApplyLoop_DropsAPendingAddWhenTheKeyLeavesConfig`

**Diagnosis (test-sync, not a real `loop.go` defect) is defensible from the
code.** Read `configuration/projection/loop.go:81-121`: `ApplyLoop.Run` is a
single goroutine; each tick takes exactly one `a.State.Snapshot()`
(`loop.go:86`) and derives both `stillDesired`/`desiredKeys`
(`loop.go:92-96`) and the pending-retry loop (`loop.go:97-109`) from that
same snapshot before the pending loop runs. There is no intra-tick
staleness between the drop check and the snapshot it's based on — a tick
that reads a snapshot *after* the tombstone is applied will always drop the
key from `a.pending` before attempting any retry in that same tick. The
only possible staleness is a tick whose `Snapshot()` call landed a moment
before `s.ApplyServiceTombstone()`'s effect became visible — that tick
still sees the tenant present and legitimately retries once more. This
diagnosis is grounded correctly, not assumed.

**Fix mechanism — polling-based quiescence, not a strict happens-before,
but defensible given the loop's known bound.**
`projection_test.go:445-466`: after `ApplyServiceTombstone()`, the test
polls `calls.Load()` every 10ms and requires it to be unchanged for 50ms
(5x the 10ms tick interval) before taking the baseline, with a 2s
fail-fast deadline. This is not a channel/waitgroup-style happens-before —
it is a bounded polling loop, and in the general case polling-for-stability
can theoretically be fooled by a scheduler pathology. But it is defensible
here specifically because the diagnosis proves *at most one* extra call can
land after `ApplyServiceTombstone()` returns (never an unbounded stream):
once that single possible extra tick lands, `calls` never changes again for
the life of the test, so the 50ms/5-tick stabilization window is
guaranteed to eventually observe quiescence (bounded by the 2s deadline,
which fails loud rather than silently passing). This is the right test for
the bug being tested: it is a strictly stronger check than the original
"snapshot immediately after the call returns," and it fails loud
(`t.Fatal`) rather than silently widening tolerance if the loop ever
regresses to genuinely unbounded retries. Ran it standalone (`go test
./configuration/projection/... -run
TestApplyLoop_DropsAPendingAddWhenTheKeyLeavesConfig -race -count=10`) —
10/10 pass, consistent with the implementer's reported 20/20.

**Compliance with the brief's explicit prohibitions:** did not widen the
assertion to `LessOrEqual`, did not touch the trailing `100 * time.Millisecond`
sleep or the final `require.Equal(countAtTombstone, ...)` — confirmed by the
diff, which only adds the stabilization block before the pre-existing
`countAtTombstone := calls.Load()` line.

Finding B verdict: **holds**. The diagnosis is grounded in `loop.go`'s
actual single-goroutine, single-snapshot-per-tick structure, and the fix
establishes a bound the underlying design actually guarantees, not a wider
sleep that happens to work today.

## Other checks

- `gofmt -l` on every touched file (`listener/registry.go`,
  `listener/handle.go`, `listener/registry_test.go`, `main.go`,
  `configuration/projection/loop.go`,
  `configuration/projection/projection_test.go`): empty.
- No `errcheck`-relevant unchecked error returns introduced — the new
  `RemoveHandler` call is checked (`if rmErr := ...; rmErr != nil`).
- Commit hygiene: two commits, one per finding, conventional-commit
  prefixes (`fix`, `test`), named files only (verified via `git show
  --stat` on each commit — no stray files).
- Ran the two new/changed tests standalone under `-race` as a
  test-honesty spot check (not a substitute for the flagless gate, which
  is running separately): both pass repeatably and the Finding A test is
  confirmed load-bearing against the pre-fix rollback branch by reading
  the diff (pre-fix rollback had no `RemoveHandler` loop, so `removed`
  would be empty and `require.ElementsMatch` would fail).

## Not evaluable

- Did not re-run `tools/verify.sh` or `tools/lint.sh` per the controller's
  instruction (a flagless run is already in flight against this tree).
  Full-suite pass/fail is out of this review's evidence; the implementer's
  reported `-race -count=20` clean run for `listener/...`, `socket/...`,
  `configuration/...` (excluding the pre-existing, out-of-scope
  `configuration` package flake) was not independently reproduced at
  `-count=20`, only spot-checked at lower counts above.

## Non-blocking notes

1. `kafka/consumer/fame/consumer.go:42` (and, by the same pattern, other
   `InitHandlers` implementations under `kafka/consumer/*`) still discards
   partial handle registrations on a mid-call failure — a narrower,
   pre-existing variant of finding A's leak that this round correctly did
   not scope to (brief's file list excludes `kafka/consumer/*`). Worth a
   future task if the leak surface matters at that granularity.
