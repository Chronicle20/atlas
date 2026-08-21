# Review — Task 4 of task-244: `atlas-channel/main.go` bind-error propagation + drain hooks

**Range reviewed:** `dbb7fcb1e..345593a61` (1 commit, `345593a61`)
**Scope:** `services/atlas-channel/atlas.com/channel/main.go` (`buildListener`'s tail block, +13/-1), plus the Task 2/3 contracts it calls into (`listener.Handle`, `socket.CreateSocketService`, `socket.SessionsForHandle`, `socket.KickSession`, `listener.Registry.Add`) read for correctness of the seam, not re-reviewed as their own units.

## Diff under review

```go
hp := handlerProducer(fl)(handler.AdaptHandler(fl)(t, wp))(tenantCfg.Socket.Handlers, validatorMap, handlerMap)
// tdm.WaitGroup() brackets the accept-loop goroutine (process-wide
// shutdown bookkeeping, unchanged); h.Wg additionally sees every
// accepted connection, which is what makes drain phase 3 a real
// bounded wait (task-244 design.md §4.3).
lis, err := socket.CreateSocketService(fl, tctx, tdm.WaitGroup(), h.Wg)(hp, rw, wp, sc, cfg.IPAddress, cfg.Port)
if err != nil {
	// A non-nil error here is what fires Registry.Add's existing
	// rollback: no entry is left for a channel that never bound.
	return nil, err
}
h.CloseListener = lis.Close
h.Sessions = socket.SessionsForHandle(fl, tctx, sc)
h.Kick = socket.KickSession(fl, tctx, wp, sc)

return handles, nil
```

(`services/atlas-channel/atlas.com/channel/main.go:624-642`)

This matches the brief's Step 1 block verbatim, including the doc comments.

## Checklist against the three defects

### 1. Bind error propagation

- Before: `socket.CreateSocketService(fl, tctx, tdm.WaitGroup())(...)` — return values discarded, function unconditionally fell through to `return handles, nil`.
- After: `lis, err := ...`; `if err != nil { return nil, err }` at `main.go:630-634`.
- Confirmed `socket.CreateSocketService` (`services/atlas-channel/atlas.com/channel/socket/init.go:75-82`) binds synchronously via `socket.Bind` and returns `(nil, fmt.Errorf("bind %s:%d: %w", ...))` on failure, before any other side effect (the goroutine, `chakra.GetRegistry().StartSweeper`, etc. all happen after the bind-error check). So a bind failure now genuinely short-circuits before anything is started.
- Confirmed `listener.Registry.Add` (`services/atlas-channel/atlas.com/channel/listener/registry.go:122-134`) treats a non-nil `body(h)` error as rollback: deletes the entry from `r.entries`, decrements `r.refs`, cancels `ctx`. No partially constructed `Handle` is left registered — the requirement "nothing partially constructed may be leaked into `handles` or left registered" holds. **PASS.**
- No other path in the reviewed diff discards the error; every `register(...)` call above this block already does `if err := register(...); err != nil { return nil, err }` (pre-existing, unmodified — `main.go:580-618`), so the new block is consistent with the surrounding convention. **PASS.**

### 2. `h.Wg` vs `tdm.WaitGroup()`

- Before: only `tdm.WaitGroup()` was passed as the sole waitgroup parameter to the old 3-arg `CreateSocketService` signature; `h.Wg` was never touched by `buildListener`, so it tracked nothing.
- After: `socket.CreateSocketService(fl, tctx, tdm.WaitGroup(), h.Wg)` — `tdm.WaitGroup()` occupies the `wg` parameter (unchanged, process-wide), `h.Wg` occupies the new `sessionWg` parameter.
- Verified inside `init.go`: `wg` parameter is documented as bracketing "the accept-loop goroutine only" (`init.go:69-72`) and is passed straight through to `routine.Go(l, ctx, func(_ context.Context) { ... socket.Serve(l, ctx, wg, fanOut, lis, ...) })` — i.e. `wg.Add/Done` brackets the outer accept-loop goroutine spawned by `routine.Go`.
- `sessionWg` (= `h.Wg`) is fanned into `dualWaitGroup{a: wg, b: sessionWg}` (`init.go:102`) which is passed to `socket.Serve` as the per-connection waitgroup argument (`fanOut`), not the accept-loop's `wg`. `dualWaitGroup.Add/Done` fan out to both, but the accept-loop goroutine itself is bracketed by `wg` alone via `routine.Go`, not by `fanOut`. This matches `Handle.Wg`'s doc contract at `handle.go:47-54`: "tracks per-connection session goroutines... does NOT cover the accept-loop goroutine."
- I did not re-verify inside `socket.Serve`/session `run()` that `fanOut.Add(1)/Done()` is called exactly once per accepted connection around the session's `run()` goroutine — that is Task 3's own unit (`socket` package) and is out of this task's scope per the brief; Task 3's tests are the evidence for that half of the contract. **PASS** for the wiring this task owns (which waitgroup goes where); the per-connection Add/Done bracketing itself is Task 3's surface, not re-litigated here.

### 3. Drain hooks populated

- `h.CloseListener = lis.Close` — `lis` is the `net.Listener` returned by the now-propagated `CreateSocketService` call, so this is the real bound listener's `Close`, not a nil/no-op. Matches `Handle.CloseListener`'s doc contract (`handle.go:56-63`: "closes the handle's bound TCP listener... nil for handles whose body never bound (tests)"). **PASS.**
- `h.Sessions = socket.SessionsForHandle(fl, tctx, sc)` — signature matches `socket/drain.go:24` (`func SessionsForHandle(l logrus.FieldLogger, ctx context.Context, sc server.Model) func() []listener.Session`); `fl`, `tctx`, `sc` are all already in scope from earlier in `buildListener`. **PASS.**
- `h.Kick = socket.KickSession(fl, tctx, wp, sc)` — signature matches `socket/drain.go:45` (`func KickSession(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, sc server.Model) func(listener.Session) error`); `wp` already in scope. **PASS.**
- All three hooks are assigned strictly after the bind-error check, so on the error path none of them are set on a `Handle` that gets rolled back anyway — consistent with "no partial construction leaked." **PASS.**

### 4. No invented symbols / no stubs

- Every symbol used (`CreateSocketService`, `SessionsForHandle`, `KickSession`, `h.Wg`, `h.CloseListener`, `h.Sessions`, `h.Kick`, `lis.Close`) exists in the repo today, verified directly by reading `handle.go`, `init.go`, `drain.go`, `registry.go` above — none guessed from the brief's prose. **PASS.**
- No `// TODO`, no empty handler, no placeholder. **PASS.**

### 5. `atlas-login` untouched

```
$ git status --porcelain services/atlas-login   (worktree, post-commit)
```
Diff stat for the reviewed range touches only `services/atlas-channel/atlas.com/channel/main.go` (`git diff --stat dbb7fcb1e..345593a61` → 1 file changed). No `atlas-login` file appears in the diff. **PASS.**

### 6. Line-ending / formatting discipline

- The diff is a pure insertion in the middle of an existing block (`+13 -1`), no reformatting of surrounding untouched lines visible in the diff context. **PASS** (nothing to suggest CRLF/LF disruption; diff shows clean unified-diff insertion only).

### 7. Test expectation

- Per the task's stated ground truth, `main.go` cannot carry a test (documented at `main.go:406-417`, confirmed present, unchanged by this diff), and this task's behavior is exercised by Task 2's `listener` tests and Task 3's `socket` tests. No test file appears in the diff, and none is expected. Not treated as a finding, per instructions.
- Did not re-run `go build`/`go test` myself — the implementer's report documents a clean `go build ./...` and `go test ./...` run against this exact commit, and re-running the same code would not surface anything the report doesn't already claim. (Noting for completeness, not as unverified: this is the one evidence class explicitly deferred to the implementer's report per the review brief.)

## Not evaluable

- The exact per-connection `Add(1)`/`Done()` bracketing inside `socket.Serve`/session `run()` (i.e., that `fanOut` is invoked exactly once per accepted connection, not double-counted or leaked) is outside this diff's surface — it lives entirely in Task 3's `socket` package, which this task only calls. Task 3's own tests are the authority for that; not re-verified here since the diff under review does not touch that code path.

## Verdict rationale

The diff is a faithful, minimal, verbatim implementation of the brief's Step 1. All three named defects (swallowed bind error, wrong waitgroup, nil drain hooks) are fixed at the exact lines the brief specified, and each fix was independently checked against the actual Task 2/3 source (not just trusted from the brief's prose or the implementer's report). No unrelated changes, no scope creep, no invented symbols, `atlas-login` untouched.
