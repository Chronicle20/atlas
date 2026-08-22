# Review: fix round 1 (task-244)

Commit range `92ae4c480..901b64606` (single commit `901b64606`).
Brief: `.superpowers/sdd/plan/fix-round-1-brief.md`
Report: `.superpowers/sdd/plan/fix-round-1-report.md`

Scope: the three blocking findings (goroutineguard, errcheck, data race) plus
the one verify-only item from the brief. Judged against the brief, not the
original plan.

## Diff surveyed

```
libs/atlas-socket/bind_serve_test.go                                     |  4 +--
services/atlas-channel/atlas.com/channel/listener/registry.go            | 29 +++++++--
services/atlas-channel/atlas.com/channel/listener/registry_test.go       | 38 +++++++++++--
```

## Finding 1 — bare `go` in `DrainAll` (goroutineguard)

PASS. `registry.go:283-292`: the bare `go func(h *Handle){...}(h)` is replaced
with `routine.Go(r.l, h.Ctx, func(_ context.Context) { defer wg.Done(); ... })`.

Verified the join/panic-safety claim against `routine.Go`'s actual body
(`libs/atlas-routine/routine.go:16-25`):

```go
func Go(l logrus.FieldLogger, ctx context.Context, fn func(context.Context)) {
	go func() {
		defer func() { if r := recover(); r != nil { ... } }()
		fn(ctx)
	}()
}
```

Defer ordering on a panic inside `fn`: `fn`'s own `defer wg.Done()` (registered
first, inside `fn`'s frame) unwinds before the panic propagates to the outer
goroutine literal's frame, where `routine.Go`'s `recover()` catches it. So
`wg.Done()` always fires whether `Drain` returns normally or panics, and
`wg.Wait()` in the caller still joins — the report's claim is correct, not
just asserted.

Loop-variable capture: `for _, h := range r.Snapshot() { ...; routine.Go(...,
func(_ context.Context) { ...r.Drain(h.Key)... }) }` — safe under per-iteration
loop-variable scoping (Go 1.22+; toolchain is go1.26.5 per the brief header),
so no shared-`h` capture bug was introduced.

Doc comment on `DrainAll` (`registry.go:279-282`) updated to match — no longer
claims a plain `go` was required for joining.

`go build ./...` clean at `services/atlas-channel/atlas.com/channel` (verified
directly, not just per the report).

## Finding 2 — unchecked `Close()` in `bind_serve_test.go` (errcheck)

PASS. `bind_serve_test.go:42,58` changed from `defer lis.Close()` /
`defer held.Close()` to `defer func() { _ = lis.Close() }()` / `defer func() {
_ = held.Close() }()`, matching the existing `_ = client.Close()` style already
in `libs/atlas-socket/server_test.go:71`. No test assertions touched; test
still exercises the same behavior.

## Finding 3 — data race in `listener` tests under `-race`

This is the highest-value check. Verdict: **genuine race fixes, not
sleeps/hidden**.

### Race A — `TestRegistry_DrainClosesTheBoundSocket`

Before: the test dialed a connection, closed it, then immediately called
`r.Drain(key)`, with nothing tying the accept-site `h.Wg.Add(1)` (running in
the `socket.Serve` goroutine) to `Drain` phase 3's `h.Wg.Wait()`
(`registry.go:247-252`, run inside a `routine.Go`-spawned goroutine). This is
a real `sync.WaitGroup` "Add concurrently with Wait" race per the memory
model — two goroutines touch the same `WaitGroup` with no synchronizing
event between them.

Fix (`registry_test.go:227-267`): a `acceptNotifyWG` wrapper around `h.Wg`
that `close()`s a `ready` channel (via `sync.Once`) the first time `Add(1)` is
observed. The test now blocks on `<-accepted` before calling `r.Drain(key)`.

Traced the happens-before chain by hand rather than trusting the run:

1. `w.inner.Add(delta)` (real `h.Wg.Add`) completes, then `close(w.ready)`
   runs — both in the accept goroutine, program-order-sequenced.
2. `close(ready)` **happens-before** the test goroutine's `<-accepted`
   receive completes (Go memory model: a close happens-before a receive that
   observes the close).
3. The test goroutine's subsequent `r.Drain(key)` call is program-order-after
   the receive.
4. Inside `Drain`, phase 3's `routine.Go(..., func(_ context.Context) {
   h.Wg.Wait(); ... })` is a `go` statement reached only after phases 1-2 run
   in the same (test-calling) goroutine — program-order-after step 3.
5. The `go` statement itself happens-before the start of the spawned
   goroutine's body, i.e. before `h.Wg.Wait()` executes.

Chchaining 1→5 gives a real happens-before edge from `Add(1)` to `Wait()`,
which is exactly what was missing before. This is not a sleep and not a
widened window — it is a structural fix that makes the race detector's
finding permanently unreachable, independent of scheduling.

Confirmed the wrapper only instruments `Add`/`Done` and forwards to the same
underlying `*sync.WaitGroup` (`w.inner`) that `Drain` phase 3 waits on
(`sessionWg := &acceptNotifyWG{inner: h.Wg, ...}` at `registry_test.go:256`,
while `h.Wg` itself — unwrapped — is what phase 3 calls `.Wait()` on). So the
production code under test is unmodified; only the test gained an
observability hook. No production-code path was available for this one within
the brief's file list (`registry.go`/`registry_test.go` only) — the report's
argument for keeping this test-side, citing `Handle.Wg`'s doc comment
rejecting counting the accept loop, is consistent with what's in
`registry.go`'s `Handle` doc comments (not independently re-verified word for
word here, but the tradeoff described — burning the full drain deadline on
every drain if the accept loop is counted — is a real cost, and design.md
§4.4 is cited as the source of that tradeoff).

### Race B — `TestRegistry_AddRefusesADrainingHandle`

Before: the test read `h.State` directly off a `*Handle` obtained from
`r.Get(key)` inside `require.Eventually`, unsynchronized, while `Drain`
writes `h.State = Draining` under `r.mu` (`registry.go:188`, inside the
`Drain` phase-1 critical section at `registry.go:186-190`). Two goroutines,
one write one read, no synchronization — a real data race, not an artifact.

Fix: new `Registry.State(key) (State, bool)` (`registry.go:157-166`) takes
`r.mu` and returns `h.State` under the same lock `Drain` uses to write it —
this is the identical lock, so it's a genuine synchronizing read, not a
different lock that merely narrows the window. `Get`'s doc comment
(`registry.go:145-148`) was updated to warn against reading `.State` off its
returned pointer. The test (`registry_test.go:388-390`) now polls
`r.State(key)` instead of `r.Get(key)` + `h.State`.

Checked for other production call sites that still read `.State` off a
`Get()`-return unsynchronized: `grep` across `atlas-channel` for
`listener.Registry` usage outside `registry.go`/`registry_test.go` turned up
only comment references and the `configuration/projection` package, which
does not touch `Handle.State` at all. So the fix is scope-complete, not just
patching the one test that happened to be flagged.

### Verification (reproduced independently, not just trusting the report)

```
$ cd services/atlas-channel/atlas.com/channel && go test ./listener/... -race -count=10
ok  	atlas-channel/listener	5.919s
```

10/10 clean, run fresh by this review (brief's bar was 5/5; report ran 20/20).

## Finding 4 — `socket/init.go:14 could not import fmt` (verify only)

PASS (no-op, as expected). Reproduced independently:

```
$ cd services/atlas-channel/atlas.com/channel && go build ./...   # clean
$ cd libs/atlas-socket && go build ./... && go vet ./...          # clean
```

No file touched for this finding, matching the brief's "if it does not
reproduce, change nothing" instruction. Diff confirms no edits to
`socket/init.go`.

## Other checks

- Module-local build/test: `go build ./...` and `go test ./listener/...
  -race -count=10` both green, run directly by this review (not solely from
  the report).
- No `//goroutine-guard:allow` annotation was added — `routine.Go` was used,
  per the brief's stated preference.
- No `-count=1` masking, no `sleep`-based synchronization anywhere in the
  diff — confirmed by reading every hunk.
- Commit is a single commit on `task-244-listener-drain-socket-close`,
  matches the brief's file list; no files touched outside
  `registry.go`/`registry_test.go`/`bind_serve_test.go`.
- Did not verify `tools/lint.sh`/`tools/verify.sh` myself per the task
  instructions (a `--quick` gate was running concurrently over the same
  tree); relied on direct code/memory-model inspection plus module-local
  `go build`/`go vet`/`go test -race` runs instead.

## Not evaluable

- Whether `tools/verify.sh --quick`'s `goroutineguard` analyzer specifically
  stops flagging `registry.go:269` (the exact reported line) was not run
  directly by this review (workspace-module wiring issue when invoking the
  vet tool standalone from this worktree, and running `tools/lint.sh`/
  `tools/verify.sh` was explicitly out of scope for this review while the
  gate runs concurrently). The diff itself replaces the exact bare `go`
  statement the analyzer flagged with `routine.Go`, which is the fix the
  brief describes as sufficient — the controller's concurrent gate run is
  the authoritative check for this specific line.

## Verdict rationale

All three blocking findings are fixed at the root cause, not papered over:
finding 1 preserves join/panic-recovery semantics correctly reasoned through
`routine.Go`'s defer ordering; finding 2 is a straightforward style match;
finding 3's two races are both closed by real happens-before edges (a
close/receive edge for Race A, the shared registry mutex for Race B), traced
by hand against the Go memory model rather than accepted on a green
`-race` run alone. Finding 4 was correctly left untouched per the brief.
