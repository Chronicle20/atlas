# Listener Drain Socket Lifecycle Fix — Design

Version: v2
Status: Draft
Created: 2026-08-20
Updated: 2026-08-20 (v2 — corrections from `design-review.md`)
Source PRD: `docs/tasks/task-244-listener-drain-socket-close/prd.md`
Review: `docs/tasks/task-244-listener-drain-socket-close/design-review.md`
---

## 0. What Changed in v2

v1 correctly diagnosed the three defects and chose the right `Bind`/`Serve` shape, but it
made `h.Wg` live without changing drain *ordering*, which is not safe, and it left two
adjacent failure modes unaddressed. v2 adds:

- **§2.3 drain ordering** — the listener now closes at the end of phase 1, not phase 4, and
  `sessionWg.Add(1)` moves to the accept site. Together these remove a `sync: WaitGroup
  misuse` panic window and stop a draining channel from accepting new clients.
- **§3.3 accept-loop exit on `net.ErrClosed`** — required by the reordering; without it a
  phase-1 close spins the accept loop.
- **§4.5 phase 2 is wired for real** — `SessionsForKey`/`SendShutdownNotice`/`DestroySession`
  are TODO stubs in `main.go` today, which would make phase 3 burn the full `DrainDeadline`
  on every drain and turn `DrainAll` into `N × DrainDeadline` on SIGTERM.
- **§4.6 `Add` failure semantics** — a failed `Add` is now retried by the apply loop instead
  of being lost, and `Add` refuses a key whose handle is still `Draining` instead of
  double-inserting.
- Interface hygiene (`WaitGrouper` throughout) and a test plan that pins defect 1 at its
  real site (`loop.go`), not at a registry fake.

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

## 2. Corrections Discovered During Design

Four problems surfaced that are not obvious from reading the defects in isolation. Each
changes the shape of the fix from what a literal reading of FR 4.1–4.3 might suggest.

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
lifetime is bounded by the phase-1 listener close (§2.3) and `h.Cancel()` in phase 4, not by being
waited on in phase 3. Phase 3 becomes a genuine "are there real in-flight client sessions" check.

**Scope of the phase-3 wait, stated precisely.** `h.Wg` covers per-connection session goroutines
(`run()`). It does *not* cover the per-packet `handle()` goroutines
(`libs/atlas-socket/server.go:262`) or the per-session ctx-watcher goroutine (`server.go:176`),
neither of which is tracked by any waitgroup today. Phase 3 can therefore complete with
packet-handler work still in flight. Bringing those under a waitgroup is a wider change to the
frame-read loop than this task's non-goals allow (PRD §2), so it stays out of scope — but the
claim "phase 3 is now a real wait" means *sessions*, not *all work*, and the code comment on
`h.Wg` must say so.

### 2.3 Drain ordering: the listener must close in phase 1, before the phase-3 wait

v1 left listener teardown at phase 4 (`h.Cancel()`). That is wrong in two independent ways once
`h.Wg` is live.

**Panic.** `run()` calls `wg.Add(1)` *inside* the per-connection goroutine
(`libs/atlas-socket/server.go:161`), after `routine.Go` has already returned. With the listener
still open through phase 3, the accept loop can take a new connection while phase 3 sits in
`h.Wg.Wait()` (`listener/registry.go:209-212`) with the counter at zero. `Add` on a zero counter
with a waiter present is `sync: WaitGroup misuse: Add called concurrently with Wait` — a
process-wide panic. Today this is unreachable only because `h.Wg` is never incremented.

**Silent miss.** Even without a panic, the window between `Accept` returning and `run()`
executing `Add(1)` reads as zero, so phase 3 can complete and phase 4 cancel while a
just-accepted connection is unaccounted for.

**Stray clients.** Phase 1 already deregisters the channel from `server.GetRegistry()` and from
atlas-world (`registry.go:191-193`). Leaving the port open for the rest of the drain window (up
to 10s at the ceiling) lets a client connect to a channel the registry considers gone — the exact
thing PRD user story #1 asks to prevent.

Both fixes apply:

1. **Close the listener at the end of phase 1**, via a new `Handle.CloseListener` closure
   (§4.4). No `Accept` can then return during phase 3, so no `Add` can race the `Wait`.
2. **Hoist the increment to the accept site**: `Serve` calls `sessionWg.Add(1)` immediately
   after `Accept` returns and before `routine.Go`; `run()` keeps only `defer sessionWg.Done()`.
   This is the standard Add-before-`go` discipline and closes the accept→Add gap. `routine.Go`
   always invokes `fn` (it only recovers panics — `libs/atlas-routine/routine.go`), and `run()`'s
   `Done` is deferred, so the pairing holds even if `run` panics.

Phase 4's `h.Cancel()` still closes the listener a second time; `Serve`'s existing `net.ErrClosed`
guard makes that a no-op, so the PRD's "Idempotency" NFR is preserved.

### 2.4 A live `h.Wg` is only useful if phase 2 actually kicks sessions

`main.go:286-293` wires the registry with three TODO stubs:

```go
SessionsForKey:     func(key server.Key) []listener.Session { return nil }, // TODO
SendShutdownNotice: func(listener.Session) {},
DestroySession:     func(listener.Session) error { return nil },
```

So phase 2 never disconnects anybody. The only thing that closes a client conn is
`session.Model.Disconnect()` (`session/model.go:209`), reached from `session.Processor.Destroy`
(`session/processor.go:431`) — which phase 2 never calls. A session `run()` goroutine therefore
exits only when the client leaves on its own or when phase 4 cancels `h.Ctx`.

Consequence, once `h.Wg` is live: **one connected player makes phase 3 block for the entire
`DrainDeadline`, on every drain.** That is §2.2's own argument, applied to sessions instead of
the accept loop, and it defeats user story #3 exactly as thoroughly. Worse, `DrainAll` iterates
handles **sequentially** (`registry.go:251-257`), so SIGTERM shutdown becomes
`N_channels × DrainDeadline` — 10 channels at the 10s ceiling is 100s, past a typical
`terminationGracePeriod`, and the pod is SIGKILLed mid-drain. That is a direct regression against
the PRD's "No regression to SIGTERM/`DrainAll` behavior" NFR.

Wiring phase 2 is producible right now — `session.ProcessorImpl.AllInChannelProvider(worldId,
channelId)` (`session/processor.go:92`) and `Destroy(s Model) error` (`processor.go:407`) already
exist, and `buildListener` already holds the tenant-scoped processor and writer producer needed to
call them. §4.5 does it. This is a deliberate scope addition beyond a literal reading of the PRD,
taken because the task's headline deliverable — "phase 3's bounded wait is a real wait" — is
unreachable without it.

## 3. `libs/atlas-socket` API Changes

All changes are source-compatible with `atlas-login`'s existing `socket.Run` call
(`services/atlas-login/atlas.com/login/socket/init.go:62-76`). `socket.Run` has exactly two
callers repo-wide (that one and `services/atlas-channel/.../socket/init.go:79`).

### 3.1 New surface

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
// does today). sessionWg is incremented once per accepted connection, at the
// accept site, and released by the per-connection goroutine, so a caller can
// track session lifetime independently of the listener's own lifetime.
//
// Serve ignores the ipAddress/port configurators; the listener is already
// bound. Everything else in the configurator set still applies.
func Serve(l logrus.FieldLogger, ctx context.Context, wg WaitGrouper, sessionWg WaitGrouper, lis net.Listener, configurators ...Configurator) error

// Run keeps its existing behavior (Bind then Serve, wg used for both
// parameters). Its only signature change is the wg parameter's type, from
// *sync.WaitGroup to WaitGrouper, which a *sync.WaitGroup satisfies — so
// atlas-login's call site is unaffected.
func Run(l logrus.FieldLogger, ctx context.Context, wg WaitGrouper, configurators ...Configurator) error {
    c := buildConfig(configurators...)
    lis, err := Bind(l, c.ipAddress, c.port)
    if err != nil {
        return err
    }
    return Serve(l, ctx, wg, wg, lis, configurators...)
}
```

`buildConfig(configurators ...Configurator) *config` is a new unexported helper extracting
today's inline `config` literal and configurator loop (`server.go:97-110`), so `Run` and `Serve`
share one construction path.

### 3.2 `Serve`'s body

`Serve`'s body is today's `Run` body starting immediately after the `net.Listen` call, with two
changes.

The per-connection increment moves to the accept site (§2.3):

```go
sessionWg.Add(1)
routine.Go(l, ctx, func(_ context.Context) { run(l, ctx, sessionWg)(c, conn, uuid.New(), 4) })
```

and `run()` drops its own `wg.Add(1)`, keeping `defer wg.Done()`. `run()`'s parameter type
changes from `*sync.WaitGroup` to `WaitGrouper` to match.

The `defer lis.Close()` and the `routine.Go(ctx.Done() -> lis.Close())` goroutine move into
`Serve` unchanged (they operate on the listener, which `Serve` now receives already-bound). The
existing `net.ErrClosed` double-close guard is untouched.

### 3.3 The accept loop must exit on `net.ErrClosed`, not only on ctx cancellation

Today the accept-error branch is:

```go
conn, err := lis.Accept()
if err != nil {
    select {
    case <-ctx.Done():
        l.Infof("Listener stopped accepting new connections.")
        return err
    default:
        l.WithError(err).Infof("Error accepting connection.")
        continue
    }
}
```

That is safe only while the listener can never close before `ctx`. §2.3 breaks that assumption:
phase 1 closes the listener while `h.Ctx` is still live, so `Accept` would return `net.ErrClosed`
forever and the `default` arm would spin a hot loop, logging on every iteration, until phase 4
cancels. `Serve` must therefore return on a closed listener regardless of ctx state:

```go
conn, err := lis.Accept()
if err != nil {
    // A closed listener is terminal whether or not ctx has been canceled --
    // Drain phase 1 closes the listener while h.Ctx is still live, and the
    // pre-existing ctx-only check would spin here (task-244 design §3.3).
    if errors.Is(err, net.ErrClosed) {
        l.Infof("Listener stopped accepting new connections.")
        return err
    }
    select {
    case <-ctx.Done():
        l.Infof("Listener stopped accepting new connections.")
        return err
    default:
        l.WithError(err).Infof("Error accepting connection.")
        continue
    }
}
```

This is also a latent-bug fix for `atlas-login`, where an externally-closed listener has the same
spin behavior today.

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

### 4.2 `socket.CreateSocketService` (`socket/init.go`) binds first and returns an error

```go
func CreateSocketService(l logrus.FieldLogger, ctx context.Context, wg socket.WaitGrouper, sessionWg socket.WaitGrouper) func(hp socket.HandlerProducer, rw socket.OpReadWriter, wp writer.Producer, sc server.Model, ipAddress string, port int) (net.Listener, error) {
    return func(hp socket.HandlerProducer, rw socket.OpReadWriter, wp writer.Producer, sc server.Model, ipAddress string, port int) (net.Listener, error) {
        // Bind before any other side effect: a failed bind must leave nothing
        // for Registry.Add's rollback to unwind (task-244 design §4.2).
        lis, err := socket.Bind(l, ipAddress, port)
        if err != nil {
            return nil, fmt.Errorf("bind %s:%d: %w", ipAddress, port, err)
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

        return lis, nil
    }
}
```

Returning `lis` is what lets `buildListener` install `Handle.CloseListener` (§4.4). The
`socket.SetPort(port)` configurator is dropped from the option list — `Bind` takes the port
directly, and `Serve` ignores those fields (§3.1), so keeping it would leave two ways to say the
same thing.

`dualWaitGroup` is a small unexported type local to this package, over the interface rather than
the concrete type so it can be exercised against a counting fake:

```go
type dualWaitGroup struct{ a, b socket.WaitGrouper }

func (d dualWaitGroup) Add(n int) { d.a.Add(n); d.b.Add(n) }
func (d dualWaitGroup) Done()     { d.a.Done(); d.b.Done() }
```

`wg` (bracketing `Serve`'s own lifetime, i.e. the accept-loop goroutine — §2.2) stays
`tdm.WaitGroup()`-only, matching today's behavior and keeping phase 3 from counting the
accept-loop. `fanOut` (passed as `sessionWg`, bracketing each accepted connection) increments
both `h.Wg` and `tdm.WaitGroup()`, so phase 3's wait is accurate and `tdm.WaitGroup()` keeps
seeing everything it sees today, at the per-connection granularity it already effectively had.

### 4.3 `buildListener` (`main.go`) propagates the bind error and populates the handle

```go
hp := handlerProducer(fl)(handler.AdaptHandler(fl)(t, wp))(tenantCfg.Socket.Handlers, validatorMap, handlerMap)
lis, err := socket.CreateSocketService(fl, tctx, tdm.WaitGroup(), h.Wg)(hp, rw, wp, sc, cfg.IPAddress, cfg.Port)
if err != nil {
    return nil, err
}
h.CloseListener = lis.Close
h.Sessions = sessionsForHandle(fl, tctx, sc)   // §4.5
h.Kick = kickSession(fl, tctx, wp, sc)         // §4.5

return handles, nil
```

`h` is already a parameter of the returned `AddBody` closure (`main.go:394`) — no signature
change needed. Returning a non-nil error here is what makes `Registry.Add`'s existing
`if err != nil` rollback branch (`registry.go:122-133`) actually fire for a bind failure.

**Memory-visibility note.** `body` writes `h.CloseListener`/`h.Sessions`/`h.Kick` outside
`r.mu`. That is safe because `Registry.Add` takes `r.mu` again after `body` returns
(`registry.go:135-137`, to assign `h.KafkaHandlers`), and every `Drain` reads the handle only
after acquiring `r.mu` — so the lock supplies the happens-before edge for all of `body`'s writes.
The plan must keep the post-`body` lock acquisition even if `KafkaHandlers` assignment moves;
`handle.go` gets a comment saying so.

### 4.4 `listener.Handle` and `Registry.Drain` phase reordering

`Handle` gains three optional closures, all populated by `buildListener`:

```go
// CloseListener closes the handle's bound TCP listener. Invoked at the end
// of drain phase 1 so the port stops accepting before the phase-3 wait --
// see task-244 design.md §2.3. nil for handles created by a body that
// never bound (tests). Safe to call twice: atlas-socket's Serve guards
// net.ErrClosed, and phase 4's ctx cancellation closes it again.
CloseListener func() error

// Sessions snapshots the sessions bound to this handle. nil means phase 2
// has nothing to enumerate.
Sessions func() []Session

// Kick sends the shutdown notice to s and destroys it, closing the
// underlying conn so the session's run() goroutine exits and releases Wg.
Kick func(s Session) error
```

`Wg`'s doc comment is updated to record what it does and does not cover (§2.2).

`Drain`'s phases become:

```
Phase 1 (quiesce): mark Draining, deregister from server.Registry, call
         atlas-world DELETE, then CLOSE THE LISTENER so no new client can
         connect and no Accept can race phase 3's Wg.Wait.
Phase 2 (save-and-kick): enumerate h.Sessions(), h.Kick each one.
Phase 3 (deadline): wait up to cfg.DrainDeadline for h.Wg.
Phase 4 (teardown): cancel ctx, RemoveHandler per kafka handle, mark
         Removed, decrement tenant ref, fire evictors if zero.
```

Phase 1's new tail:

```go
if h.CloseListener != nil {
    if err := h.CloseListener(); err != nil && !errors.Is(err, net.ErrClosed) {
        r.l.WithError(err).WithField("key", key).Warn("listener.drain.close_listener_failed")
    }
}
```

Phase 2 becomes:

```go
var kicked int
if h.Sessions != nil && h.Kick != nil {
    for _, s := range h.Sessions() {
        if err := h.Kick(s); err != nil {
            r.l.WithError(err).WithField("key", key).Warn("listener.drain.kick_session_failed")
        }
        kicked++
    }
}
r.l.WithField("key", key).WithField("sessions", kicked).Info("listener.drain_phase phase=2")
```

The three session-related `Dependencies` fields (`SessionsForKey`, `SendShutdownNotice`,
`DestroySession`) are **removed**, not left as fallbacks — they exist today only as the stubs
§2.4 quotes, and leaving both paths would leave dead code that silently wins in `main.go`.
`UnregisterChannel` and `RemoveHandler` stay; they are genuinely process-scoped.

`DrainAll` drains handles **concurrently** rather than sequentially, so total SIGTERM drain time
is bounded by one `DrainDeadline` rather than `N × DrainDeadline` (§2.4):

```go
func (r *Registry) DrainAll() {
    var wg sync.WaitGroup
    for _, h := range r.Snapshot() {
        wg.Add(1)
        go func(h *Handle) {
            defer wg.Done()
            if err := r.Drain(h.Key); err != nil {
                r.l.WithError(err).WithField("key", h.Key).Warn("listener.drain_all.failed")
            }
        }(h)
    }
    wg.Wait()
}
```

`Drain` is already documented and implemented as safe for concurrent use (the `Draining`/`Removed`
guards under `r.mu`), and each `Drain` touches only its own handle, so this needs no further
locking. A plain `go` is correct here rather than `routine.Go` because `DrainAll` must join.

### 4.5 Wiring phase 2 for real (`main.go`)

Two helpers in `main.go`, built from the same tenant-scoped values `buildListener` already has —
so there is no tenant re-resolution and no synthetic `tenant.Model`:

```go
// sessionsForHandle snapshots this channel's sessions for drain phase 2.
func sessionsForHandle(l logrus.FieldLogger, ctx context.Context, sc server.Model) func() []listener.Session {
    return func() []listener.Session {
        ms, err := session.NewProcessor(l, ctx).AllInChannelProvider(sc.WorldId(), sc.ChannelId())
        if err != nil {
            l.WithError(err).Warn("listener.drain.sessions_lookup_failed")
            return nil
        }
        out := make([]listener.Session, 0, len(ms))
        for _, m := range ms {
            out = append(out, m)
        }
        return out
    }
}

// kickSession sends the shutdown notice and destroys the session. Destroy
// ends in Model.Disconnect(), which closes the conn -- that is what makes
// the session's run() goroutine return and release h.Wg, so drain phase 3
// can complete before its deadline instead of always timing out.
func kickSession(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, sc server.Model) func(listener.Session) error {
    sp := session.NewProcessor(l, ctx)
    return func(s listener.Session) error {
        m, ok := s.(session.Model)
        if !ok {
            return fmt.Errorf("unexpected session type %T", s)
        }
        // Best effort -- a write failure must not stop the destroy.
        if err := session.Announce(l)(ctx)(wp)(chatpkt.WorldMessageWriter)(writer.WorldMessagePopUpBody(shutdownNotice))(m); err != nil {
            l.WithError(err).Debug("listener.drain.shutdown_notice_failed")
        }
        return sp.Destroy(m)
    }
}
```

`AllInChannelProvider` filters `getRegistry().GetInTenant(p.t.Id())` by world and channel
(`session/processor.go:92-102`), which is exactly the `server.Key` triple — the tenant comes from
`ctx` (`tctx`, already tenant-scoped by `socket.NewListenerContext`). `Destroy` emits the
logout/destroyed Kafka events before `Disconnect()` (`processor.go:407-432`), so the existing
crash-safe ordering is preserved and a drained player is indistinguishable downstream from a
normal disconnect.

`shutdownNotice` is a package constant with the operator-facing message.

### 4.6 `Add` failure semantics in the apply loop

FR 4.2 makes `Add` fail loudly, but the apply loop currently treats an `Add` failure as terminal
and then advances its snapshot anyway:

```go
// configuration/projection/loop.go:88-95
_, err := a.Registry.Add(...)
if err != nil { l....Warn("projection.applied add_failed"); return }
...
prevSvc, prevTenants = nextSvc, nextTenants   // runs regardless
```

`ComputeOps` therefore never re-emits the `OpAdd`, so a transient bind conflict — PRD user story
#2's exact scenario — leaves that channel dead until the config changes again. Without a fix,
FR 4.2 converts a silent partial failure into a permanent outage.

Two changes:

**Retry failed adds.** `ApplyLoop` gains `pending map[server.Key]Op`. `execute` returns the op's
error; a failed `OpAdd` is recorded in `pending`. At the top of each tick, before `ComputeOps`,
the loop retries every `pending` op whose key still exists in the new snapshot and drops the rest.
The first failure logs at `Warn` (`projection.applied add_failed`, unchanged — it already carries
key/tenant/error and satisfies the PRD's observability NFR); subsequent retries of the same key
log at `Debug` with a `retries` field so a persistent conflict is visible without flooding.
`prevSvc`/`prevTenants` continue to advance unconditionally — `pending` is what carries the
outstanding work, which keeps the snapshot-diff semantics intact.

**Refuse an add against a draining handle.** `Registry.Add`'s early return only covers
`State == Active` (`registry.go:104`), so an `OpAdd` arriving while the old handle is `Draining`
inserts a *second* handle under the same key, and the old drain's phase-4
`delete(r.entries, key)` (`registry.go:233`) then deletes the new one and decrements `refs` for
it. With the old listener still bound (until that drain's phase 1 completes), the new `Bind`
would fail anyway. `Add` gains:

```go
if existing, ok := r.entries[key]; ok {
    if existing.State == Active {
        r.mu.Unlock()
        return existing, nil
    }
    // Draining: do not race the drain to revive a terminal handle. The
    // apply loop retries this op on the next tick (design §4.6).
    r.mu.Unlock()
    return nil, ErrDraining
}
```

`ErrDraining` is a package sentinel — the one error in this task that a caller genuinely needs to
distinguish (the retry path treats it as expected, not as an operator-visible conflict, so it
logs at `Debug`). PRD open question #3 asked whether a bind-failure sentinel was needed: it is
not — bind failures stay plain wrapped errors, since `pending` retries them without inspecting
them.

### 4.7 Observability

The existing `l.WithError(err).WithField("key", key).Warn("projection.applied add_failed")` in
`loop.go:92` already logs tenant/key/error for any `Add` failure, satisfying the PRD's
observability requirement — a bind failure now flows through that exact path instead of being
silently swallowed. New log lines are limited to the two drain warnings in §4.4
(`close_listener_failed`, `kick_session_failed`) and the `retries` field in §4.6.

## 5. `atlas-login` Impact

Source-compatible. `atlas-login`'s `CreateSocketService`
(`services/atlas-login/atlas.com/login/socket/init.go:62`) keeps calling
`socket.Run(l, ctx, wg, opts...)` with no edit: `Run`'s only signature change is
`wg *sync.WaitGroup` → `wg WaitGrouper`, which `*sync.WaitGroup` satisfies.

One behavior change reaches it: §3.3's `net.ErrClosed` accept-loop exit. That replaces a hot
spin with a clean return if the login listener is ever closed out from under the loop, and is a
strict improvement; on the normal path (ctx cancel closes the listener) the outer
`errors.Is(err, net.ErrClosed)` check at the call site already swallows the returned error, so the
observable behavior is unchanged.

## 6. Testing Strategy

### `libs/atlas-socket`
- `Bind`: success against an ephemeral port; failure against an already-bound port.
- `Serve`: accepts a pre-bound listener; existing frame-read tests in `server_test.go` continue
  to exercise `run()`/`handle()` unchanged.
- **Accept-loop exit (§3.3)**: close the listener while ctx is still live; assert `Serve` returns
  rather than spinning (bound the test on the number of log lines or on a short deadline).
- **Session waitgroup accounting (§2.3)**: a fake `WaitGrouper` recording `Add`/`Done`; assert
  `Add` is observed before the connection is dispatched (dial, then assert the counter is
  non-zero without waiting on the session goroutine to be scheduled), and returns to zero when
  the conn closes.

### `services/atlas-channel/.../configuration/projection` (`projection_test.go`)
- **This is where defect 1 is pinned.** Assert the `ctx` handed to `AddBody` is canceled by
  `Drain` on that key — the fix lives at `loop.go:88-89`, and `main.go` cannot carry a test
  (`main.go:406-417` says as much). A registry-level fake-body test asserts the registry works;
  only this one asserts the *threading*.
- A failed `OpAdd` is retried on the next tick (§4.6), and is dropped if the key leaves config.
- `ErrDraining` from `Add` is retried, not reported as a hard failure.

### `services/atlas-channel/.../listener` (`registry_test.go`)
- Per-channel `Drain` closes the socket: a `body` that calls `socket.Bind`/`socket.Serve` against
  `h.Ctx`/`h.Wg` the way `buildListener` does, then assert a post-drain dial is refused.
- **Phase ordering (§2.3)**: assert the listener is closed *before* the phase-3 wait — dial
  during phase 3 (held open by an in-flight `h.Wg` entry) and assert refusal.
- `Add` returns an error and leaves no entry / no leaked `refs` when the port is already bound.
- `Add` returns `ErrDraining` and does not insert a second entry when the existing handle is
  `Draining` (§4.6).
- Phase 3 blocks on an in-flight `h.Wg` entry and completes as soon as it is released — asserting
  both that it waits *and* that it does not hit the full `DrainDeadline` on a fast release.
- Phase 2 calls `h.Kick` once per `h.Sessions()` entry and continues past a `Kick` error.
- `DrainAll` drains every handle, and its total elapsed time is bounded by roughly one
  `DrainDeadline` with several handles held past their deadline (§4.4 concurrency).

### `services/atlas-channel/.../socket` (`init_test.go`)
- `CreateSocketService` returns a non-nil error when the port is already bound, **and starts no
  goroutine and no sweeper** — asserting the bind-before-side-effects ordering that makes
  `Registry.Add`'s existing rollback sufficient (a counting `WaitGrouper` observes zero `Add`s;
  `chakra.GetRegistry()` shows no sweeper).
- On success it returns a non-nil `net.Listener` whose `Close` is what `buildListener` installs.
- `dualWaitGroup` fans `Add`/`Done` to both targets.

### Gates
- Module-local `go build ./... && go test ./...` in `services/atlas-channel/atlas.com/channel`,
  `services/atlas-login/atlas.com/login`, and `libs/atlas-socket`.
- The channel and socket packages are worth a `-race` run specifically, given §2.3.
- Flagless `tools/verify.sh` exits 0 before the branch is done.

## 7. Scope Notes

Unchanged non-goals from the PRD: no change to frame-read/encryption/idle-detection behavior, no
retry/backoff *policy* for bind failures beyond the next-tick retry in §4.6, no bind-failure
sentinel type, no change to `projection.AddBody`'s handler-registration behavior.

Deliberate additions beyond a literal PRD reading, with rationale:

| Addition | Why it is not optional |
|---|---|
| Listener closes in phase 1 (§2.3) | A live `h.Wg` plus an open listener is a `sync.WaitGroup` misuse panic. |
| `sessionWg.Add` at the accept site (§2.3) | Closes the accept→`Add` gap that would silently under-count. |
| Accept-loop `net.ErrClosed` exit (§3.3) | Required by the phase-1 close; otherwise a hot spin. |
| Phase 2 wired for real (§4.5) | Without it phase 3 always burns the full deadline — the task's headline deliverable is unreachable. |
| Concurrent `DrainAll` (§4.4) | Otherwise SIGTERM becomes `N × DrainDeadline` — a regression against the PRD's own NFR. |
| Apply-loop retry + `ErrDraining` (§4.6) | Otherwise FR 4.2 turns a silent partial failure into a permanent channel outage. |

Explicitly still out of scope, and now stated rather than implied: packet-handler (`handle()`)
and per-session ctx-watcher goroutines remain untracked by any waitgroup, so phase 3 waits on
sessions, not on all in-flight work (§2.2).
