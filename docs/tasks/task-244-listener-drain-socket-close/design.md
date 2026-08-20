# Listener Drain Socket Lifecycle Fix — Design

Version: v1
Status: Draft
Created: 2026-08-20
Source PRD: `docs/tasks/task-244-listener-drain-socket-close/prd.md`
---

## 1. Root Cause Recap

All three defects share one root cause: the socket service's startup/shutdown/waitgroup
signals are never actually threaded through the `listener.Handle` the registry manages.

1. `configuration/projection/loop.go:88-89` calls `a.AddBody(ctx, ...)` with the apply-loop's
   parent `ctx`, not `h.Ctx`. `buildListener` (`main.go:394`) builds `socket.NewListenerContext`
   from that same wrong `ctx`, so `h.Cancel()` (`Registry.Drain` phase 4) never reaches the
   socket service.
2. `socket.CreateSocketService` (`socket/init.go:56-124`) buries the entire call — including
   `net.Listen`, which today already runs synchronously and already returns an error before any
   goroutine starts — inside a fire-and-forget `routine.Go`. `buildListener` unconditionally
   `return handles, nil`s, so `Registry.Add` never sees a bind failure.
3. `h.Wg` (`&sync.WaitGroup{}`, allocated fresh per `Handle` in `Registry.Add`) is never passed
   anywhere; `main.go:635` passes `tdm.WaitGroup()` (the process-wide waitgroup) into
   `CreateSocketService` instead. `Registry.Drain` phase 3's `h.Wg.Wait()` waits on an empty,
   never-incremented waitgroup and returns immediately regardless of in-flight sessions.

## 2. Two Corrections Discovered During Design

Two problems surfaced while working through the PRD's open questions that are not obvious from
reading the defects in isolation. Both change the shape of the fix from what a literal reading of
FR 4.2/4.3 might suggest, so they're called out explicitly.

### 2.1 A synchronous bind signal needs a real signal, not just "don't background it"

`socket.Run` already does `net.Listen` synchronously and already returns an error immediately on
failure — but **only on failure**. On success, `Run` blocks in its `Accept` loop for the entire
listener lifetime, returning only when `ctx` cancels. Simply moving the existing `socket.Run` call
out of `routine.Go` and blocking on its return value would work for the failure case but deadlock
`Registry.Add` (and the whole projection apply-loop goroutine) forever on every *successful* add,
because nothing would ever unblock the caller.

The fix is a **`Bind`/`Serve` split** in `libs/atlas-socket`: `Bind` does only `net.Listen` and
returns `(net.Listener, error)` synchronously; `Serve` takes an already-bound listener and runs
today's accept loop. `Run` becomes a thin `Bind`-then-`Serve` wrapper so its existing callers
(`atlas-login`) are unaffected. `CreateSocketService` calls `Bind` directly and can return its
error before doing anything else.

### 2.2 `h.Wg` must not count the listener's own accept-loop goroutine

FR 4.3 describes `h.Wg` as tracking "the socket-service's listener goroutine and each
per-connection `run()` goroutine." Taken literally, that's self-defeating: the accept-loop
goroutine (and the `CreateSocketService` goroutine that ends in `<-ctx.Done()`) legitimately runs
for the handle's entire `Active` lifetime and only exits when `h.Ctx` is canceled — which happens
in **phase 4**, after phase 3's bounded wait. If `h.Wg` counted that goroutine, phase 3 would hit
its full `DrainDeadline` timeout on *every* drain, defeating the PRD's own user story #3 ("a fast
completion means sessions actually finished").

`h.Wg` therefore tracks **session `run()` goroutines only**. The listener/accept-loop goroutine's
lifetime is bounded the way it already is today — by `h.Cancel()` in phase 4, not by being waited
on in phase 3. Phase 3 becomes a genuine "are there real in-flight client sessions" check: zero
sessions passes immediately, an in-flight session blocks until it finishes or the deadline hits.

## 3. `libs/atlas-socket` API Changes

All changes are additive/source-compatible with `atlas-login`'s existing `socket.Run` call
(`services/atlas-login/atlas.com/login/socket/init.go:62-76`).

```go
// WaitGrouper is the minimal surface Serve/Run needs from a waitgroup. A
// *sync.WaitGroup satisfies it today with no caller changes; atlas-channel
// uses it to fan Add/Done out to more than one underlying waitgroup.
type WaitGrouper interface {
    Add(delta int)
    Done()
}

// Bind performs net.Listen synchronously. Callers that need to observe a
// bind failure before doing anything else (Registry.Add's rollback path)
// call this directly instead of Run.
func Bind(l logrus.FieldLogger, ipAddress string, port int) (net.Listener, error)

// Serve runs the accept loop against an already-bound listener. wg brackets
// Serve's own lifetime (Add(1) on entry, Done() on return — mirrors what Run
// does today). sessionWg is passed to each per-connection run() goroutine
// separately, so a caller can track session lifetime independently of the
// listener's own lifetime.
func Serve(l logrus.FieldLogger, ctx context.Context, wg WaitGrouper, sessionWg WaitGrouper, lis net.Listener, configurators ...Configurator) error

// Run keeps its existing signature and behavior (Bind then Serve, wg used
// for both parameters) — atlas-login's call site is unaffected.
func Run(l logrus.FieldLogger, ctx context.Context, wg WaitGrouper, configurators ...Configurator) error {
    c := buildConfig(configurators...)
    lis, err := Bind(l, c.ipAddress, c.port)
    if err != nil {
        return err
    }
    return Serve(l, ctx, wg, wg, lis, configurators...)
}
```

`Serve`'s body is today's `Run` body starting immediately after the `net.Listen` call, with one
change: the per-connection goroutine now receives `sessionWg` instead of `wg`:

```go
routine.Go(l, ctx, func(_ context.Context) { run(l, ctx, sessionWg)(c, conn, uuid.New(), 4) })
```

`run()`'s own parameter type changes from `*sync.WaitGroup` to `WaitGrouper` to match.

The `defer lis.Close()` and the `routine.Go(ctx.Done() -> lis.Close())` goroutine move into `Serve`
unchanged (they operate on the listener, which `Serve` now receives already-bound). The existing
`net.ErrClosed` double-close guard is untouched, so `Drain`'s idempotency guarantee (NFR
"Idempotency") is unaffected.

## 4. `atlas-channel` Changes

### 4.1 Context threading (`configuration/projection/loop.go`)

`ApplyLoop.execute`'s `OpAdd` branch already has `h` in scope (it's the body closure's parameter).
Change the `AddBody` call to pass `h.Ctx` instead of the apply-loop's `ctx`:

```go
_, err := a.Registry.Add(ctx, op.Key, sc, func(h *listener.Handle) ([]listener.HandlerHandle, error) {
    return a.AddBody(h.Ctx, op.Key, op.Cfg, h)
})
```

`Registry.Add`'s own `parent` argument (used for `context.WithCancel` to produce `h.Ctx` itself,
`registry.go:108`) is unrelated and stays as the apply-loop's `ctx` — that's the parent the handle's
own context derives from, which is correct and unchanged.

No signature changes to `AddBody`, `Registry.Add`, or `buildListener`: `buildListener`'s returned
closure already receives `ctx` as a parameter (`main.go:394`) — that parameter now *is* `h.Ctx` by
construction, so every downstream use (`socket.NewListenerContext(ctx, t)`, the session context,
`CreateSocketService`'s `ctx`) is threaded through automatically.

### 4.2 `socket.CreateSocketService` (`socket/init.go`) returns an error

```go
func CreateSocketService(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup, sessionWg *sync.WaitGroup) func(hp socket.HandlerProducer, rw socket.OpReadWriter, wp writer.Producer, sc server.Model, ipAddress string, port int) error {
    return func(hp socket.HandlerProducer, rw socket.OpReadWriter, wp writer.Producer, sc server.Model, ipAddress string, port int) error {
        lis, err := socket.Bind(l, ipAddress, port)
        if err != nil {
            return fmt.Errorf("bind %s:%d: %w", ipAddress, port, err)
        }

        chakra.GetRegistry().StartSweeper(l, ctx)

        hasMapleEncryption := true
        t := sc.Tenant()
        if t.Region() == "JMS" {
            hasMapleEncryption = false
        }
        locale := byte(8)
        if t.Region() == "JMS" {
            locale = 3
        }

        sp := session.NewProcessor(l, ctx)
        fanOut := dualWaitGroup{a: wg, b: sessionWg}

        routine.Go(l, ctx, func(_ context.Context) {
            err := socket.Serve(l, ctx, wg, fanOut, lis,
                socket.SetHandlers(hp),
                socket.SetCreator(sp.Create(sc.Channel(), locale)),
                socket.SetMessageDecryptor(sp.Decrypt(true, hasMapleEncryption)),
                socket.SetDestroyer(/* unchanged */),
                socket.SetReadWriter(rw),
                socket.SetIdleNotifier(session.SendPing(l, ctx, wp), idleThreshold),
            )
            if err != nil {
                if errors.Is(err, net.ErrClosed) {
                    return
                }
                l.WithError(err).Errorf("Socket service encountered error")
            }
        })

        routine.Go(l, ctx, func(_ context.Context) {
            if err := channel.NewProcessor(l, ctx).Register(sc.Channel(), ipAddress, port); err != nil {
                l.WithError(err).Errorf("Socket service registration error.")
            }
            <-ctx.Done()
            l.Infof("Shutting down server on port %d", port)
        })

        return nil
    }
}
```

`dualWaitGroup` is a small unexported type local to this package:

```go
type dualWaitGroup struct{ a, b *sync.WaitGroup }

func (d dualWaitGroup) Add(n int) { d.a.Add(n); d.b.Add(n) }
func (d dualWaitGroup) Done()     { d.a.Done(); d.b.Done() }
```

`wg` (bracketing `Serve`'s own lifetime, i.e. the accept-loop goroutine — §2.2) stays
`tdm.WaitGroup()`-only, matching today's behavior and keeping phase 3 from counting the
accept-loop. `fanOut` (passed as `sessionWg`, bracketing each per-connection session) increments
both `h.Wg` and `tdm.WaitGroup()`, so phase 3's wait is accurate and `tdm.WaitGroup()` keeps
seeing everything it sees today, at the per-connection granularity it already effectively had.

Bind happens before any other side effect (sweeper start, session processor, channel
registration) so a failed bind leaves nothing else to unwind — `Registry.Add`'s existing rollback
(delete entry, decrement `refs`, cancel `h.Ctx`) is the only cleanup needed.

### 4.3 `buildListener` (`main.go`) propagates the bind error and passes `h.Wg`

```go
hp := handlerProducer(fl)(handler.AdaptHandler(fl)(t, wp))(tenantCfg.Socket.Handlers, validatorMap, handlerMap)
if err := socket.CreateSocketService(fl, tctx, tdm.WaitGroup(), h.Wg)(hp, rw, wp, sc, cfg.IPAddress, cfg.Port); err != nil {
    return nil, err
}

return handles, nil
```

`h` is already a parameter of the returned `AddBody` closure (`main.go:394`) — no signature
change needed here either. Returning a non-nil error here is what makes `Registry.Add`'s existing
`if err != nil` rollback branch (`registry.go:122-133`) actually fire for a bind failure; `Add`
itself needs no changes.

### 4.4 Observability

The existing `l.WithError(err).WithField("key", key).Warn("projection.applied add_failed")` in
`loop.go:92` already logs tenant/key/error for any `Add` failure, satisfying the PRD's
observability requirement — a bind failure now flows through that exact path instead of being
silently swallowed. No new log line needed.

## 5. `atlas-login` Impact

None. `atlas-login`'s `CreateSocketService` (`services/atlas-login/atlas.com/login/socket/init.go`)
keeps calling `socket.Run(l, ctx, wg, opts...)` exactly as today — `Run`'s signature changes only
the parameter type from `*sync.WaitGroup` to `WaitGrouper`, and a `*sync.WaitGroup` value already
satisfies that interface, so the call site compiles and behaves identically with zero edits.

## 6. Testing Strategy

- **`libs/atlas-socket`**: unit tests for `Bind` (success against an ephemeral port; failure
  against an already-bound port) and for `Serve` accepting a pre-bound listener (existing
  frame-read tests in `server_test.go` continue to exercise `run()`/`handle()` unchanged, since
  those functions' behavior isn't touched beyond the `wg` parameter's type).
- **`services/atlas-channel/.../listener`** (`registry_test.go`): new tests per PRD acceptance
  criteria —
  - per-channel `Drain` closes the socket (connect-after-drain is refused) — requires the test's
    `Dependencies`/harness to run a real `CreateSocketService`-equivalent against a Handle, or a
    minimal fake body that wires `socket.Bind`/`socket.Serve` directly against `h.Ctx`/`h.Wg` the
    way `buildListener` does, so the test exercises the actual context-threading path rather than
    re-asserting `h.Cancel()` was called.
  - `Add` returns an error and leaves no entry / no leaked `refs` when the port is already bound
    (bind a port out-of-band first, then call `Add` with a `body` that calls `socket.Bind` against
    the same port and returns its error).
  - phase 3 blocks on an in-flight fake session `run()`-equivalent goroutine registered on `h.Wg`
    (increment `h.Wg` directly, hold it past a short delay, assert `Drain` doesn't return before
    the goroutine completes and doesn't hit the full `DrainDeadline` when it completes quickly).
  - `DrainAll` still drains every handle correctly after the `h.Ctx` threading change (existing
    `TestRegistry_DrainRunsAllFourPhases`-style coverage, re-run against the new wiring).
- **`services/atlas-channel/.../socket`** (`init_test.go`): a test that `CreateSocketService`
  returns a non-nil error when the port is already bound, without starting any goroutine.
- Module-local `go build ./... && go test ./...` in `services/atlas-channel/atlas.com/channel`,
  `services/atlas-login/atlas.com/login`, and `libs/atlas-socket` per PRD AC.

## 7. Non-Goals (unchanged from PRD)

No change to frame-read/encryption/idle-detection behavior, no retry/backoff for a failed bind, no
bind-failure sentinel type (plain wrapped error — nothing in this task's scope needs to
`errors.Is`/`As` it), no change to `projection.AddBody`'s handler-registration behavior.
