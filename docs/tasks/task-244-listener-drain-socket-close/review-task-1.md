# Review — Task 1: `libs/atlas-socket` Bind/Serve split

Range: `cb51b28ed..23cc82c5c` (1 commit, `23cc82c5c`)
Reviewer surface: `libs/atlas-socket/server.go`, `libs/atlas-socket/bind_serve_test.go` (new),
read-only reference `libs/atlas-socket/server_test.go`, `libs/atlas-socket/opts.go`,
`services/atlas-login/atlas.com/login/socket/init.go` (compile-compat check only).

## Scope confirmation

`git diff --stat cb51b28ed..23cc82c5c`:

```
libs/atlas-socket/bind_serve_test.go | 151 +++++++++++++++++++++++++++++++++++
libs/atlas-socket/server.go          |  76 +++++++++++++++---
2 files changed, 214 insertions(+), 13 deletions(-)
```

Matches exactly the two files the brief named. No stray edits anywhere else
(`git status --porcelain -- services/atlas-login libs/atlas-socket` is empty
after checkout). Single commit in range, matches `23cc82c5c` from the report.

## Requirement-by-requirement

1. **`Bind` is synchronous `net.Listen`, returns error directly.**
   `server.go:126-134`. `net.Listen("tcp", ...)`, returns `(nil, err)` on
   failure with a single-print `Errorf` (not the old double-printing
   `Errorln`), `(lis, nil)` on success. PASS.

2. **`Serve` blocks the accept loop over an already-bound listener; `Run`
   is a thin `Bind`-then-`Serve` wrapper.**
   `server.go:147-203` (`Serve`), `server.go:206-213` (`Run`). `Run` calls
   `buildConfig` once (to extract `ipAddress`/`port` for `Bind`), binds, then
   delegates the full accept loop to `Serve(l, ctx, wg, wg, lis,
   configurators...)`. PASS.

3. **Session waitgroup counts `run()` goroutines only, never the accept
   loop.** `Serve`'s own `wg.Add(1)`/`defer wg.Done()` (`server.go:148-149`)
   brackets `Serve`'s lifetime on the *first* `WaitGrouper` argument (`wg`),
   distinct from `sessionWg`, the second argument, which is only touched at
   the accept site (`server.go:200`) and in `run()`'s `defer wg.Done()`
   (`server.go:219`). For a caller that passes two different waitgroups
   (the whole point of the two-argument split), the accept-loop lifetime and
   the session lifetime are counted independently. `Run` collapses both into
   the same `wg` for `atlas-login` backward compatibility, which is correct
   per the brief — `atlas-login` doesn't need the split. PASS.

4. **`sessionWg.Add(1)` moved to the accept site, before the goroutine
   spawns — closing the accept→Add gap.**
   `server.go:200-201`:
   ```go
   sessionWg.Add(1)
   routine.Go(l, ctx, func(_ context.Context) { run(l, ctx, sessionWg)(c, conn, uuid.New(), 4) })
   ```
   `run()` no longer calls `Add` at all (`server.go:215-219`); it only holds
   `defer wg.Done()`, with the accompanying comment pointing back to the
   accept-site `Add`. I checked `routine.Go` (`libs/atlas-routine/routine.go`)
   before trusting the `Done()`-always-fires premise the brief calls out: it
   invokes `fn` unconditionally and only recovers a panic from it, so the
   spawned closure — and therefore its `defer wg.Done()` — always runs once
   the goroutine starts. Because `Add(1)` now happens synchronously in the
   accept loop before `routine.Go` is even called, there is no window where a
   concurrent `Wait()` could observe a zero counter between the accept and
   the `Add` (the old bug: `Add` execution was deferred until the spawned
   goroutine got scheduled, which could race a `Wait()` returning on a
   momentarily-zero counter). This is a real fix, not a relocation of the
   same race. PASS.

5. **Accept loop exits on `net.ErrClosed` regardless of ctx state.**
   `server.go:177-194`: the `errors.Is(err, net.ErrClosed)` check is
   evaluated and returns *before* the `select { case <-ctx.Done(): ... }`
   block, so a closed listener always terminates the loop whether or not
   `ctx` has also been canceled. The design's failure mode ("later task
   closes the listener while ctx is still live") is exactly what
   `TestServeReturnsWhenListenerClosedWhileContextLive` drives: `ctx` is
   never canceled during the test (`defer cancel()` only fires at test
   teardown), only `lis.Close()` is called, and the assertion is that
   `Serve` returns within 2s with `errors.Is(err, net.ErrClosed)`. Ran it
   locally under `-race`: PASS (0.05s, well under the 2s timeout — confirms
   the loop is not spinning and is not blocked on `ctx.Done()`).

6. **`atlas-login` compiles unchanged.**
   `services/atlas-login/atlas.com/login/socket/init.go:61` passes
   `wg *sync.WaitGroup` into `socket.Run(l, ctx, wg, ...)` — `*sync.WaitGroup`
   satisfies the new `WaitGrouper` interface (`Add(int)`, `Done()`) with no
   source change. Verified: `go build ./...` from
   `services/atlas-login/atlas.com/login` exits 0, and
   `git status --porcelain -- services/atlas-login` is empty. PASS.

7. **`Run`'s only signature change is `wg`'s type.**
   Old: `func Run(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup, configurators ...Configurator) error`.
   New: `func Run(l logrus.FieldLogger, ctx context.Context, wg WaitGrouper, configurators ...Configurator) error`
   (`server.go:206`). Everything else identical. PASS.

8. **No stubs, no placeholder comments.** Confirmed by reading the full diff
   — every added function has a complete body; no `TODO`/`FIXME`/panic
   placeholders. PASS.

9. **`sync` import removed, others retained.** `server.go:3-20` — `sync` is
   gone from the import block; `errors`, `fmt`, `net`, `uuid`, `routine`
   (`atlas-routine`) all present and used. PASS. `go build ./...` and
   `go vet ./...` in `libs/atlas-socket` both exit clean (no unused-import
   errors).

10. **Line endings / no reformatting of untouched lines.** The diff is a
    minimal unified diff (63 insertions / 13 deletions across one file,
    matching `git diff --stat`'s reported delta); no wholesale file rewrite,
    no untouched-line churn visible in the hunks. PASS.

## Concurrency scrutiny (the task's explicit ask)

Traced every `Add`/`Done` pairing:

- `wg.Add(1)` / `defer wg.Done()` in `Serve` (`server.go:148-149`) — brackets
  `Serve`'s own lifetime, unconditional entry/exit, no path skips the defer
  (it's the second statement in the function, before anything that can
  return early).
- `sessionWg.Add(1)` at the accept site (`server.go:200`), immediately before
  `routine.Go` spawns the per-connection goroutine. `run()`'s `defer
  wg.Done()` (`server.go:219`, called with `sessionWg`) is the sole matching
  `Done()`. Since `routine.Go` always invokes its closure and only recovers
  a panic (verified against `libs/atlas-routine/routine.go:15`), the
  `defer wg.Done()` inside `run()`'s returned closure always executes once
  the goroutine is scheduled — no leaked `Add`.
- No code path calls `Done()` without a preceding `Add()`, and no code path
  calls `Add()` without a corresponding `Done()` — every `Add` site has
  exactly one structurally-guaranteed `Done` (a `defer` at function entry,
  not conditionally reached).
- The old bug (`wg.Add(1)` executed *inside* the spawned goroutine, racing a
  concurrent `Wait()` observing a zero counter) is closed: the `Add` now
  executes synchronously on the accept-loop goroutine before the new
  goroutine is even created, so there is no window where the counter can be
  zero between "connection accepted" and "Add observed."
- Verified with `-race`: `go test ./... -race -run 'TestBind|TestServe' -v`
  passes clean, no data race reported, including
  `TestServeIncrementsSessionWaitGroupAtAcceptSite`, which polls `sessionWg`
  counts through the `countingWG` mutex-guarded accessor rather than reading
  raw fields, so the assertions themselves are race-safe against the
  goroutine performing the `Add`/`Done`.

## Test honesty

- `TestServeReturnsWhenListenerClosedWhileContextLive`: the brief's own
  claim that this test hangs against the pre-fix code (ctx-only exit check)
  is credible and the test's own comment documents the failure mode
  (`t.Fatal("Serve did not return on a closed listener — accept loop is
  spinning")` on timeout). I did not revert the fix and re-run to confirm
  the hang myself (task instructions say not to re-run tests the implementer
  already validated), but the test's structure — closing the listener while
  `ctx` is deliberately kept alive via `defer cancel()` only at teardown —
  is the correct discriminator between "ctx-conditional exit" and
  "unconditional net.ErrClosed exit," so it is not a test that would pass
  either way.
- `TestServeIncrementsSessionWaitGroupAtAcceptSite`: asserts `sessionWg`
  reaches `Adds()==1` after dial and `Dones()==1` after close, and separately
  that `wg.Adds()==1` (Serve's own bracket fires once, not once per
  connection) — this discriminates the accept-site-`Add` design from a
  design that still calls `Add` inside `run()`, and from a design that
  conflates `wg` and `sessionWg`. Genuine assertion of the new contract.
- `TestBindFailsWhenPortAlreadyBound` / `TestBindSucceedsOnEphemeralPort`:
  straightforward, assert exactly what the brief specifies.

## Non-blocking observations

- `Run` now calls `buildConfig(configurators...)` twice — once directly
  (to read `c.ipAddress`/`c.port` for `Bind`) and once again inside `Serve`
  (`server.go:207` then `server.go:151` via `Serve`'s own
  `buildConfig(configurators...)` call). Every `Configurator` in
  `opts.go` is a pure field-assignment closure (verified by reading all of
  `opts.go`), so double invocation is idempotent for every configurator
  currently in the codebase — including `SetHandlers`, which calls
  `producer()` a second time and discards the first map. This is inherent to
  the `Bind`/`Serve` split as specified verbatim in the brief's Step 3/4 code
  (not an implementer deviation), and causes no observable defect today. It
  is worth a design note if a future `HandlerProducer` implementation ever
  acquires a side effect (e.g., a metric registration) — calling it twice
  would double the side effect silently. Not blocking for this task.
- `TestServeReturnsWhenListenerClosedWhileContextLive` uses a fixed
  `time.Sleep(50 * time.Millisecond)` before closing the listener, rather
  than synchronizing on "Serve has entered its Accept() call." This is a
  common, low-risk pattern in this test file already (see
  `server_test.go`'s existing style) and the 2-second timeout gives ample
  margin; not a real flake risk, noted only for completeness.

## Not evaluable

- None. Everything the brief specifies for this task is inside the diff and
  was checked against either the diff itself, `libs/atlas-routine/routine.go`
  (the one external contract the design explicitly calls out to verify), or
  a local `go build`/`go vet`/`go test -race` run scoped to
  `libs/atlas-socket` and the `atlas-login` compile check.
- Tasks 3/4's actual consumption of `WaitGrouper`/`Bind`/`Serve` in
  `atlas-channel` is out of scope for this review (separate task unit) and
  was not evaluated.

## Verdict

APPROVED. The diff matches the brief line-for-line, the design properties
(accept-site `Add`, unconditional `net.ErrClosed` exit, session-only
waitgroup, thin `Run` wrapper) are all satisfied and verified against
`file:line`, the concurrency reasoning holds under a live `-race` run, and
`atlas-login` compiles with zero edits.
