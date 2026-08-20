# Listener Drain Socket Lifecycle Fix — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `listener.Registry`'s per-channel `Drain` actually close the TCP socket, make a bind failure surface synchronously from `Registry.Add`, and make `h.Wg` reflect real in-flight session goroutines.

**Architecture:** `libs/atlas-socket` splits `Run` into `Bind` (synchronous `net.Listen`) and `Serve` (accept loop over an already-bound listener), with a `WaitGrouper` interface so a caller can track session lifetime separately from listener lifetime. `atlas-channel`'s `CreateSocketService` calls `Bind` first and returns `(net.Listener, error)`, letting `buildListener` install a `Handle.CloseListener` hook and propagate bind failures into `Registry.Add`'s existing rollback path. `Registry.Drain` reorders so the listener closes at the end of phase 1 (before the phase-3 `h.Wg.Wait()`), and phase 2 is wired to real session enumeration + destroy so phase 3 is a genuine bounded wait rather than a guaranteed deadline burn.

**Tech Stack:** Go, `sync.WaitGroup`, `context`, `net`, `logrus`, `testify/require`, local `replace`-d modules (`libs/atlas-socket` ← both `atlas-channel` and `atlas-login`).

**Spec:** `docs/tasks/task-244-listener-drain-socket-close/design.md` (PRD: `docs/tasks/task-244-listener-drain-socket-close/prd.md`, design review: `design-review.md`)

## Global Constraints

- Worktree: `.worktrees/task-244-listener-drain-socket-close/`, branch `task-244-listener-drain-socket-close`. All paths below are repo-relative from the worktree root.
- **No stubs.** Do not land a `// TODO`, an empty handler, or a "wire this later" comment. The three stubbed `Dependencies` fields this task removes are exactly what that rule is about.
- **`atlas-login` must keep compiling unchanged.** `socket.Run`'s only permitted signature change is `wg *sync.WaitGroup` → `wg WaitGrouper`. `services/atlas-login/atlas.com/login/socket/init.go:62-76` must not need an edit.
- **Never invent a symbol.** Every function this plan calls exists in the repo today unless the task explicitly says `new`.
- Preserve existing line endings; do not reformat untouched lines.
- Module-local verification only per task: `go build ./... && go test ./...` from the named module root. Repo-wide `tools/verify.sh` is the controller's job, not the implementer's.
- `routine.Go(l, ctx, func(context.Context))` (`libs/atlas-routine/routine.go:15`) always invokes `fn`; it only recovers panics. That is what makes the accept-site `Add(1)` / deferred `Done()` pairing safe.

---

## Task 1: `libs/atlas-socket` — `Bind`/`Serve` split, `WaitGrouper`, accept-loop `net.ErrClosed` exit

**Module root for `go build`/`go test`:** `libs/atlas-socket`

### Files

- `libs/atlas-socket/server.go` — all changes: add `WaitGrouper`, `buildConfig`, `Bind`, `Serve`; rewrite `Run` as a thin wrapper; change `run()`'s `wg` param type and drop its internal `wg.Add(1)`
- `libs/atlas-socket/bind_serve_test.go` — **new file**; the tests for `Bind`, `Serve`, the `net.ErrClosed` exit, and session-waitgroup accounting
- `libs/atlas-socket/server_test.go` — read-only reference for the existing `serve`/`nopWriter`/`writeAll`/`header`/`frame` helpers (same package `socket`, so the new file reuses them directly — do NOT redefine them). No edit expected: it calls `run(l, ctx, wg)` with a `*sync.WaitGroup`, which satisfies `WaitGrouper`.
- `libs/atlas-socket/opts.go` — read-only; `SetIpAddress`/`SetPort` stay as-is (`Run` still consumes them)

Patterns to copy: `libs/atlas-socket/server_test.go:39-73` (`serve` helper — how a test drives `run()` over a `net.Pipe`).

### Interfaces

- Produces (consumed by Tasks 3 and 4):
  - `type WaitGrouper interface { Add(delta int); Done() }`
  - `func Bind(l logrus.FieldLogger, ipAddress string, port int) (net.Listener, error)`
  - `func Serve(l logrus.FieldLogger, ctx context.Context, wg WaitGrouper, sessionWg WaitGrouper, lis net.Listener, configurators ...Configurator) error`
  - `func Run(l logrus.FieldLogger, ctx context.Context, wg WaitGrouper, configurators ...Configurator) error` (signature change: `wg` type only)

- [ ] **Step 1: Write the failing tests**

Create `libs/atlas-socket/bind_serve_test.go`, package `socket` (internal — it needs the unexported `config`/`run`, and it reuses `nopWriter` from `server_test.go`).

Four test functions. Helpers to reuse from `server_test.go`, not redefine: `nopWriter`.

| test func | what it drives | exact assertions |
|---|---|---|
| `TestBindSucceedsOnEphemeralPort` | `Bind(l, "127.0.0.1", 0)` | `err == nil`; returned `net.Listener` is non-nil; `lis.Addr().(*net.TCPAddr).Port != 0`; `defer lis.Close()` |
| `TestBindFailsWhenPortAlreadyBound` | bind a port with `net.Listen("tcp", "127.0.0.1:0")` first, extract its port, then `Bind(l, "127.0.0.1", thatPort)` | second call returns `err != nil` and a `nil` listener |
| `TestServeReturnsWhenListenerClosedWhileContextLive` | `Bind` an ephemeral port; `ctx, cancel := context.WithCancel(context.Background())` with `defer cancel()` (never canceled during the assertion); run `Serve` in a goroutine writing its error to a `chan error`; after `50*time.Millisecond` call `lis.Close()` | `Serve` returns within `2*time.Second` (select on the chan vs `time.After`); the returned error satisfies `errors.Is(err, net.ErrClosed)`; the test FAILS the old code by hanging, so the timeout arm must call `t.Fatal("Serve did not return on a closed listener — accept loop is spinning")` |
| `TestServeIncrementsSessionWaitGroupAtAcceptSite` | `Bind` ephemeral; `Serve` in a goroutine with a `countingWG` as `sessionWg` and a separate `countingWG` as `wg`; `net.Dial("tcp", lis.Addr().String())`; poll `sessionWg.Adds()` until `>= 1` (bounded by `2*time.Second`), then close the client conn and poll `sessionWg.Dones()` until `>= 1` | `adds == 1` after the dial; `dones == 1` after the conn closes; and `wg.Adds() == 1` (Serve brackets its own lifetime once) |

`countingWG` is a new test-only type in this file:

```go
type countingWG struct {
    mu    sync.Mutex
    adds  int
    dones int
}

func (c *countingWG) Add(delta int) { c.mu.Lock(); c.adds += delta; c.mu.Unlock() }
func (c *countingWG) Done()         { c.mu.Lock(); c.dones++; c.mu.Unlock() }
func (c *countingWG) Adds() int     { c.mu.Lock(); defer c.mu.Unlock(); return c.adds }
func (c *countingWG) Dones() int    { c.mu.Lock(); defer c.mu.Unlock(); return c.dones }
```

Config for `Serve` in the last two tests: pass no configurators at all — `buildConfig` supplies `defaultCreator`/`defaultMessageDecryptor`/`defaultDestroyer` and an empty handler map, and `rw` being nil is fine because no frame is ever completed in these tests. Use a `logrus.New()` with `SetOutput(nopWriter{})`.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd libs/atlas-socket && go test ./... -run 'TestBind|TestServe' -v`
Expected: compile failure — `undefined: Bind`, `undefined: Serve`, `undefined: WaitGrouper`.

- [ ] **Step 3: Add `WaitGrouper`, `buildConfig`, and `Bind`**

In `libs/atlas-socket/server.go`, after the `config` struct (`server.go:88-98`):

```go
// WaitGrouper is the minimal surface Serve/Run needs from a waitgroup. A
// *sync.WaitGroup satisfies it with no caller changes; atlas-channel uses
// it to fan Add/Done out to more than one underlying waitgroup
// (task-244 design.md §3.1).
type WaitGrouper interface {
	Add(delta int)
	Done()
}

func buildConfig(configurators ...Configurator) *config {
	c := &config{
		creator:   defaultCreator,
		decryptor: defaultMessageDecryptor,
		destroyer: defaultDestroyer,
		ipAddress: "0.0.0.0",
		port:      5000,
		handlers:  make(map[uint16]request.Handler),
	}
	for _, configurator := range configurators {
		configurator(c)
	}
	return c
}

// Bind performs net.Listen synchronously. Callers that must observe a
// bind failure before starting any side effect -- listener.Registry.Add's
// rollback path in atlas-channel -- call this instead of Run.
func Bind(l logrus.FieldLogger, ipAddress string, port int) (net.Listener, error) {
	l.Infof("Starting tcp server on [%s:%d]", ipAddress, port)
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", ipAddress, port))
	if err != nil {
		l.WithError(err).Errorf("Error listening on [%s:%d].", ipAddress, port)
		return nil, err
	}
	return lis, nil
}
```

Note the deliberate change from the old `l.WithError(err).Errorln("Error listening:", err.Error())` — the old form double-printed the error. Keep the new form.

- [ ] **Step 4: Add `Serve` and rewrite `Run`**

Replace the whole existing `Run` body (`server.go:100-164`) with:

```go
// Serve runs the accept loop against an already-bound listener. wg
// brackets Serve's own lifetime (Add(1) on entry, Done() on return),
// matching what Run does today. sessionWg is incremented once per
// accepted connection AT THE ACCEPT SITE and released by the
// per-connection goroutine, so a caller can track session lifetime
// independently of the listener's. The accept-site increment is what
// closes the accept->Add gap and makes a concurrent Wait safe
// (task-244 design.md §2.3).
//
// Serve ignores the ipAddress/port configurators -- the listener is
// already bound. Every other configurator still applies.
func Serve(l logrus.FieldLogger, ctx context.Context, wg WaitGrouper, sessionWg WaitGrouper, lis net.Listener, configurators ...Configurator) error {
	wg.Add(1)
	defer wg.Done()

	c := buildConfig(configurators...)

	defer func(lis net.Listener) {
		err := lis.Close()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			l.WithError(err).Error("Error closing listener")
		}
	}(lis)

	routine.Go(l, ctx, func(_ context.Context) {
		<-ctx.Done()
		l.Infof("Closing listener.")
		err := lis.Close()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			l.WithError(err).Errorf("Error closing listener.")
		}
	})

	for {
		conn, err := lis.Accept()
		if err != nil {
			// A closed listener is terminal whether or not ctx has been
			// canceled -- drain phase 1 closes the listener while the
			// handle's ctx is still live, and a ctx-only check would spin
			// a hot loop here (task-244 design.md §3.3).
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

		l.Infof("Client [%s] connected.", conn.RemoteAddr())

		// Add before the goroutine starts: run()'s own Add(1) would race a
		// concurrent Wait and panic with "WaitGroup misuse".
		sessionWg.Add(1)
		routine.Go(l, ctx, func(_ context.Context) { run(l, ctx, sessionWg)(c, conn, uuid.New(), 4) })
	}
}

//goland:noinspection GoUnusedExportedFunction
func Run(l logrus.FieldLogger, ctx context.Context, wg WaitGrouper, configurators ...Configurator) error {
	c := buildConfig(configurators...)
	lis, err := Bind(l, c.ipAddress, c.port)
	if err != nil {
		return err
	}
	return Serve(l, ctx, wg, wg, lis, configurators...)
}
```

Then change `run`'s signature and drop its internal `Add`. At `server.go:166-169` the current code is:

```go
func run(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup) func(config *config, conn net.Conn, sessionId uuid.UUID, headerSize int) {
	return func(config *config, conn net.Conn, sessionId uuid.UUID, headerSize int) {
		wg.Add(1)
		defer wg.Done()
```

becomes:

```go
func run(l logrus.FieldLogger, ctx context.Context, wg WaitGrouper) func(config *config, conn net.Conn, sessionId uuid.UUID, headerSize int) {
	return func(config *config, conn net.Conn, sessionId uuid.UUID, headerSize int) {
		// The matching Add(1) happens at Serve's accept site, before this
		// goroutine is spawned (task-244 design.md §2.3).
		defer wg.Done()
```

Nothing else in `run`, `handle`, or the frame-read loop changes.

The `"sync"` import in `server.go` is now unused — remove it from the import block (`server.go:10`). `errors`, `fmt`, `net`, `uuid`, `routine` all stay.

- [ ] **Step 5: Run the tests to verify they pass**

Run: `cd libs/atlas-socket && go build ./... && go test ./... -race`
Expected: PASS, including the pre-existing `TestReadAssemblesFrameSplitAcrossReads` and `TestReadHandlesByteAtATimeDelivery`.

`server_test.go:67` calls `run(l, ctx, wg)` with `wg := &sync.WaitGroup{}` — that still compiles against `WaitGrouper`. If it does not, the fix is in `server.go`, not in `server_test.go`.

- [ ] **Step 6: Confirm `atlas-login` still compiles unchanged**

Run: `cd services/atlas-login/atlas.com/login && go build ./...`
Expected: exit 0 with **zero edits** to `services/atlas-login/atlas.com/login/socket/init.go`. If it fails, `Run`'s signature diverged from the plan — fix `Run`, do not edit atlas-login.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-socket/server.go libs/atlas-socket/bind_serve_test.go
git commit -m "feat(atlas-socket): split Run into Bind/Serve and exit the accept loop on a closed listener"
```

---

## Task 2: `atlas-channel/listener` — handle hooks, drain reordering, `ErrDraining`, concurrent `DrainAll`

**Module root for `go build`/`go test`:** `services/atlas-channel/atlas.com/channel`

### Files

- `services/atlas-channel/atlas.com/channel/listener/handle.go` — add `CloseListener`, `Sessions`, `Kick` fields; update `Wg`'s doc comment
- `services/atlas-channel/atlas.com/channel/listener/registry.go` — remove three `Dependencies` fields; add `ErrDraining`; `Add` refuses a draining key; `Drain` phase reorder; concurrent `DrainAll`
- `services/atlas-channel/atlas.com/channel/listener/registry_test.go` — update `nopDeps`, add the new cases
- `services/atlas-channel/atlas.com/channel/main.go:283-295` — delete the three stub `Dependencies` fields from the `listener.NewRegistry` call so the service still compiles. **Nothing else in main.go changes in this task.**

Patterns to copy: `services/atlas-channel/atlas.com/channel/listener/registry_test.go:49-70` (registry test setup — `makeTenant`/`makeServerModel`/`nullLogger` and the `defer server.GetRegistry().Deregister(key)` discipline).

### Interfaces

- Consumes from Task 1: `socket.Bind`, `socket.Serve`, `socket.WaitGrouper` (used by the drain-closes-the-socket test only; import path `socket "github.com/Chronicle20/atlas/libs/atlas-socket"`).
- Produces (consumed by Tasks 3 and 4):
  - `listener.Handle.CloseListener func() error`
  - `listener.Handle.Sessions func() []listener.Session`
  - `listener.Handle.Kick func(s listener.Session) error`
  - `var listener.ErrDraining error`
  - `listener.Dependencies` reduced to `UnregisterChannel` + `RemoveHandler`

- [ ] **Step 1: Write the failing tests**

Edit `services/atlas-channel/atlas.com/channel/listener/registry_test.go`.

First, `nopDeps` loses the three removed fields:

```go
func nopDeps() listener.Dependencies {
	return listener.Dependencies{
		UnregisterChannel: func(channel.Model) error { return nil },
		RemoveHandler:     func(string, string) error { return nil },
	}
}
```

`TestRegistry_DrainRunsAllFourPhases` (`registry_test.go:72-111`) currently sets `deps.SessionsForKey` / `deps.DestroySession`. Rewrite those three lines to drive the handle hooks instead: drop `deps.SessionsForKey` and `deps.DestroySession`, and inside the `Add` body set

```go
h.Sessions = func() []listener.Session { return []listener.Session{"s1", "s2", "s3"} }
h.Kick = func(listener.Session) error { destroyCalls.Add(1); return nil }
```

Every other assertion in that test stays byte-for-byte identical (1 unreg call, 3 destroy calls, 2 removeHandler calls, `context.Canceled`, both registries empty).

Then add these new test functions:

| test func | setup | exact assertions |
|---|---|---|
| `TestRegistry_DrainClosesTheBoundSocket` | `Add` with a body that does `lis, err := socket.Bind(nullLogger(), "127.0.0.1", 0)`, requires no error, records `port := lis.Addr().(*net.TCPAddr).Port`, sets `h.CloseListener = lis.Close`, and starts `go socket.Serve(nullLogger(), h.Ctx, &sync.WaitGroup{}, h.Wg, lis)`. Confirm the port is live with a successful `net.DialTimeout("tcp", addr, time.Second)` (close that conn). Then `r.Drain(key)`. | after drain, `net.DialTimeout("tcp", addr, 200*time.Millisecond)` returns a non-nil error. Retry the dial up to 20 times at 25ms if it unexpectedly succeeds, to absorb TIME_WAIT-style flake; fail with `t.Fatalf("port %d still accepting after drain", port)` |
| `TestRegistry_DrainClosesListenerBeforePhase3Wait` | same body as above, plus `h.Wg.Add(1)` before calling `Drain` and a goroutine that sleeps `150*time.Millisecond` then `h.Wg.Done()`. `DrainDeadline: 2 * time.Second`. Run `Drain` in a goroutine; from the test goroutine, sleep `50*time.Millisecond` (so we are inside phase 3) and dial. | the mid-phase-3 dial returns a non-nil error — the listener is already closed. Then wait for `Drain` to return (bounded 3s) and require no error. This is the §2.3 ordering assertion; with the close left at phase 4 the dial succeeds and the test fails |
| `TestRegistry_AddReturnsErrorAndLeavesNoEntryWhenBodyFails` | bind a port out of band (`net.Listen("tcp","127.0.0.1:0")`, keep it open, `defer close`); `Add` with a body that calls `socket.Bind` on that same port and returns `(nil, err)` | `Add` returns `err != nil`; `r.Get(key)` returns `ok == false`; `r.Snapshot()` is empty; a subsequent successful `Add` for a *different* key on the same tenant followed by `Drain` still fires the evictor exactly once (proves `refs` was not leaked — use `listener.SetEvictorsForTest`) |
| `TestRegistry_AddRefusesADrainingHandle` | `Add` key with a body that sets `h.Sessions = func() []listener.Session { return []listener.Session{"s"} }` and `h.Kick = func(listener.Session) error { <-block; return nil }` where `block` is a channel the test closes later, so `Drain` is parked in phase 2. `DrainDeadline: 2*time.Second`. Run `Drain` in a goroutine, wait until the handle reports `Draining` (poll `r.Get(key)` for `h.State == listener.Draining`, bounded 1s), then call `Add` again for the same key. | second `Add` returns `nil` handle and `errors.Is(err, listener.ErrDraining)`; `r.Snapshot()` still has exactly 1 entry. Close `block` and let the drain finish |
| `TestRegistry_Phase3CompletesAsSoonAsTheWaitGroupReleases` | `Add`, then `h.Wg.Add(1)` and a goroutine that `Done()`s after `100*time.Millisecond`. `DrainDeadline: 3 * time.Second` | `Drain` elapsed `>= 100*time.Millisecond` and `< 1*time.Second` — it waits for real, and does not burn the deadline |
| `TestRegistry_Phase2ContinuesPastAKickError` | `Add` with `h.Sessions` returning `[]listener.Session{"a","b","c"}` and `h.Kick` incrementing a counter and returning `errors.New("boom")` for `"b"` only | kick counter `== 3`; `Drain` returns `nil` |
| `TestRegistry_DrainAllIsBoundedByOneDeadline` | 4 handles across 4 keys, each with `h.Wg.Add(1)` and no `Done` (so each burns the deadline). `DrainDeadline: 200 * time.Millisecond` | total `DrainAll` elapsed `< 700*time.Millisecond` (one deadline plus slack), not `>= 800ms` (4 × 200ms). Also require `r.Snapshot()` is empty afterwards |

`TestRegistry_DrainWarnsOnDeadlineButCompletes` (`registry_test.go:146-170`) already asserts the deadline fall-through and needs no change beyond the `nopDeps` update.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./listener/... -v`
Expected: compile failure — `unknown field CloseListener`, `unknown field Sessions`, `unknown field Kick`, `undefined: listener.ErrDraining`.

- [ ] **Step 3: Add the `Handle` fields**

In `services/atlas-channel/atlas.com/channel/listener/handle.go`, replace the `Handle` struct (`handle.go:36-45`) with:

```go
// Handle is the per-(t,w,c) listener state.
//
// CloseListener/Sessions/Kick are populated by the Add body (main.go's
// buildListener) outside r.mu. That is safe because Registry.Add
// re-acquires r.mu after body returns and every Drain reads the handle
// only after acquiring r.mu -- the lock supplies the happens-before
// edge. Do not remove that post-body lock acquisition.
type Handle struct {
	Key    server.Key
	State  State
	Ctx    context.Context
	Cancel context.CancelFunc

	// Wg tracks per-connection session goroutines (atlas-socket's run())
	// for this handle, and nothing else. It deliberately does NOT cover
	// the accept-loop goroutine -- that runs for the handle's whole
	// Active lifetime, so counting it would make drain phase 3 burn its
	// full deadline every time. It also does not cover the per-packet
	// handle() goroutines or the per-session ctx-watcher, neither of
	// which is tracked by any waitgroup today. Phase 3 therefore waits
	// on sessions, not on all in-flight work (task-244 design.md §2.2).
	Wg *sync.WaitGroup

	ServerModel   server.Model
	KafkaHandlers []HandlerHandle

	// CloseListener closes the handle's bound TCP listener. Invoked at
	// the end of drain phase 1 so the port stops accepting before the
	// phase-3 wait -- otherwise a newly accepted connection's Add(1)
	// races h.Wg.Wait() and panics with "WaitGroup misuse"
	// (task-244 design.md §2.3). nil for handles whose body never bound
	// (tests). Safe to call twice: atlas-socket's Serve guards
	// net.ErrClosed and phase 4's ctx cancellation closes it again.
	CloseListener func() error

	// Sessions snapshots the sessions bound to this handle. nil means
	// phase 2 has nothing to enumerate.
	Sessions func() []Session

	// Kick sends the shutdown notice to s and destroys it, closing the
	// underlying conn so the session's run() goroutine exits and
	// releases Wg. nil means phase 2 kicks nobody.
	Kick func(s Session) error
}
```

- [ ] **Step 4: Trim `Dependencies` and add `ErrDraining`**

In `services/atlas-channel/atlas.com/channel/listener/registry.go`:

Delete the `SessionsForKey`, `SendShutdownNotice`, and `DestroySession` fields and their doc comments from `Dependencies` (`registry.go:29-38`), leaving:

```go
type Dependencies struct {
	// UnregisterChannel calls atlas-world's DELETE channel endpoint. A
	// 404 from upstream is success.
	UnregisterChannel func(ch channel.Model) error

	// RemoveHandler maps to consumer.Manager.RemoveHandler -- invoked
	// once per HandlerHandle during phase 4.
	RemoveHandler func(topic, id string) error
}
```

Keep the `Session` type alias (`registry.go:48`) — `Handle.Sessions`/`Handle.Kick` still use it.

Add, above `Registry`:

```go
// ErrDraining is returned by Add when a Handle for the key exists but is
// mid-Drain. Add does not race a Drain to revive a terminal Handle; the
// projection apply loop retries the op on its next tick
// (task-244 design.md §4.6).
var ErrDraining = errors.New("listener: handle is draining")
```

Add `"errors"` and `"net"` to the import block (`net` is needed in Step 5's phase 1).

- [ ] **Step 5: Make `Add` refuse a draining handle**

Replace `registry.go:103-107`:

```go
	r.mu.Lock()
	if existing, ok := r.entries[key]; ok && existing.State == Active {
		r.mu.Unlock()
		return existing, nil
	}
```

with:

```go
	r.mu.Lock()
	if existing, ok := r.entries[key]; ok {
		if existing.State == Active {
			r.mu.Unlock()
			return existing, nil
		}
		// Draining: inserting a second Handle here would let the old
		// drain's phase-4 delete(r.entries, key) remove the NEW handle and
		// decrement refs for it. Refuse; the apply loop retries.
		r.mu.Unlock()
		return nil, ErrDraining
	}
```

Also update `Add`'s doc comment (`registry.go:98-101`) so "the caller must wait" becomes "Add returns ErrDraining and the caller retries."

- [ ] **Step 6: Reorder `Drain`'s phases**

In `registry.go`, update the `Drain` doc comment's phase list to:

```
//	Phase 1 (quiesce): mark Draining, deregister from server.Registry,
//	         call atlas-world DELETE, then CLOSE THE LISTENER so no new
//	         client can connect and no Accept can race phase 3's Wait.
//	Phase 2 (save-and-kick): enumerate h.Sessions(), h.Kick each one.
//	Phase 3 (deadline): wait up to cfg.DrainDeadline for h.Wg; warn on
//	         timeout.
//	Phase 4 (teardown): cancel ctx, RemoveHandler per kafka handle, mark
//	         Removed, decrement tenant ref, fire evictors if zero.
```

Append to phase 1, immediately before the `phase=1` log line (`registry.go:195`):

```go
	if h.CloseListener != nil {
		if err := h.CloseListener(); err != nil && !errors.Is(err, net.ErrClosed) {
			r.l.WithError(err).WithField("key", key).Warn("listener.drain.close_listener_failed")
		}
	}
```

Replace phase 2 (`registry.go:197-205`) with:

```go
	// Phase 2: save-and-kick existing sessions. Kicking is what makes
	// phase 3 a real bounded wait rather than a guaranteed deadline burn:
	// Kick ends in session.Model.Disconnect(), which closes the conn so
	// the session's run() goroutine returns and releases h.Wg.
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

Phases 3 and 4 are unchanged.

- [ ] **Step 7: Make `DrainAll` concurrent**

Replace `DrainAll` (`registry.go:248-257`) with:

```go
// DrainAll drains every Handle in the current snapshot concurrently, so
// total SIGTERM drain time is bounded by one DrainDeadline rather than
// N x DrainDeadline -- sequential drains blow past a typical
// terminationGracePeriod once phase 3 is a real wait
// (task-244 design.md §4.4). Concurrent calls are safe; each Drain
// serializes itself and touches only its own handle. A plain `go` rather
// than routine.Go because DrainAll must join.
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

- [ ] **Step 8: Delete the three stub deps in `main.go`**

In `services/atlas-channel/atlas.com/channel/main.go`, the `listener.NewRegistry` call at `main.go:283-295` currently supplies five `Dependencies` fields. Delete three of them outright — `SessionsForKey` (three stub comment lines plus `return nil`), `SendShutdownNotice`, and `DestroySession` — leaving exactly:

```go
	listenerRegistry := listener.NewRegistry(l, listener.Dependencies{
		UnregisterChannel: func(ch channel2.Model) error {
			return channel3.NewProcessor(l, rt.Context()).Unregister(ch)
		},
		RemoveHandler: consumer.GetManager().RemoveHandler,
	}, listener.Config{
		DrainDeadline: parseDrainDeadline(),
	})
```

The deleted stubs' comment block goes with them — Task 4 wires the real path onto `Handle.Sessions`/`Handle.Kick`. Leave `UnregisterChannel` and `RemoveHandler` bodies exactly as they are. If `server` becomes an unused import in `main.go` as a result, leave it — `buildListener` still takes a `server.Key`.

- [ ] **Step 9: Run the tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./listener/... -race -v`
Expected: PASS, all cases.

- [ ] **Step 10: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/listener/handle.go services/atlas-channel/atlas.com/channel/listener/registry.go services/atlas-channel/atlas.com/channel/listener/registry_test.go services/atlas-channel/atlas.com/channel/main.go
git commit -m "fix(atlas-channel): close the listener in drain phase 1 and refuse an Add against a draining handle"
```

---

## Task 3: `atlas-channel/socket` — bind-first `CreateSocketService`, `dualWaitGroup`, drain-phase-2 helpers

**Module root for `go build`/`go test`:** `services/atlas-channel/atlas.com/channel`

### Files

- `services/atlas-channel/atlas.com/channel/socket/init.go` — `CreateSocketService` binds first, takes `socket.WaitGrouper` params, returns `(net.Listener, error)`; add `dualWaitGroup`
- `services/atlas-channel/atlas.com/channel/socket/drain.go` — **new file**; `SessionsForHandle` and `KickSession`
- `services/atlas-channel/atlas.com/channel/socket/init_test.go` — add the bind-failure, listener-returned, and `dualWaitGroup` cases (keep the three existing `NewListenerContext`/`WithSelfEnvironment` tests untouched)
- `services/atlas-channel/atlas.com/channel/socket/drain_test.go` — **new file**; `KickSession`'s non-`session.Model` rejection
- `services/atlas-channel/atlas.com/channel/session/processor.go` — read-only; `AllInChannelProvider(worldId world.Id, channelId channel.Id) ([]Model, error)` at line 92, `Destroy(s Model) error` at line 407, `Announce` at line 247, `NewProcessor(l, ctx) Processor` at line 71
- `services/atlas-channel/atlas.com/channel/socket/writer/world_message.go` — read-only; `WorldMessagePopUpBody(message string) packet.Encode` at line 47

Patterns to copy: `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation.go:389` — the exact `session.Announce(l)(ctx)(wp)(chatpkt.WorldMessageWriter)(writer.WorldMessagePopUpBody(msg))(s)` call shape, including the `chatpkt "github.com/Chronicle20/atlas/libs/atlas-packet/chat/clientbound"` import alias.

**Placement note (deviation from design §4.5, deliberate):** the design put `sessionsForHandle`/`kickSession` in `main.go`. They go in `socket/drain.go` instead, exported as `SessionsForHandle`/`KickSession`, for the reason `main.go:406-417` already states in a comment about `NewListenerContext`: a file literally named `main.go` cannot carry a test. The bodies are the design's bodies verbatim; only the file and the capitalization change. `socket` already imports `session`, `writer`, and `server`; `listener` imports only `atlas-channel/server`, so `socket` → `listener` introduces no cycle.

### Interfaces

- Consumes from Task 1: `socket.Bind`, `socket.Serve`, `socket.WaitGrouper`.
- Consumes from Task 2: `listener.Session`.
- Produces (consumed by Task 4):
  - `func CreateSocketService(l logrus.FieldLogger, ctx context.Context, wg socket.WaitGrouper, sessionWg socket.WaitGrouper) func(hp socket.HandlerProducer, rw socket.OpReadWriter, wp writer.Producer, sc server.Model, ipAddress string, port int) (net.Listener, error)`
  - `func SessionsForHandle(l logrus.FieldLogger, ctx context.Context, sc server.Model) func() []listener.Session`
  - `func KickSession(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, sc server.Model) func(listener.Session) error`

- [ ] **Step 1: Write the failing tests**

Add to `services/atlas-channel/atlas.com/channel/socket/init_test.go` (package `socket`, internal — `dualWaitGroup` is unexported):

| test func | setup | exact assertions |
|---|---|---|
| `TestCreateSocketServiceReturnsErrorWhenPortIsAlreadyBound` | `pre, err := net.Listen("tcp", "127.0.0.1:0")`, require no error, `defer pre.Close()`, `port := pre.Addr().(*net.TCPAddr).Port`. Build `sc` via `server.NewProcessor(logrus.New(), ctx).Register(testTenant(t), channel.NewModel(1, 0), "127.0.0.1", port)` with `ctx := NewListenerContext(context.Background(), tm)`; `defer server.GetRegistry().Deregister(server.KeyOf(sc))`. Pass `wg`/`sessionWg` as two `*countingWG` (new test type, same shape as Task 1's). `hp` is `func() map[uint16]request.Handler { return nil }`, `rw` is `socket.ShortReadWriter{}`, `wp` is `nil`. | returned `net.Listener` is nil; `err != nil`; **`wg.Adds() == 0` and `sessionWg.Adds() == 0`** — the bind-before-side-effects property that makes `Registry.Add`'s rollback sufficient (design §4.2/§6). Note: the design also wanted a "sweeper not started" assertion; `chakra.GetRegistry().StartSweeper` is guarded by a process-global `sync.Once` with no test seam, so it is not observable — the two `Adds() == 0` assertions cover the ordering property |
| `TestCreateSocketServiceReturnsTheBoundListener` | same, but bind port `0` (ephemeral, no pre-bound conflict); tenant + `sc` as above; cancel the ctx via `defer cancel()` at the end | `err == nil`; listener non-nil; `lis.Addr().(*net.TCPAddr).Port != 0`; calling `lis.Close()` returns nil the first time. Cancel the ctx and `t.Cleanup` the deregistration so no goroutine leaks into the next test |
| `TestDualWaitGroupFansOutAddAndDone` | `a, b := &countingWG{}, &countingWG{}`; `d := dualWaitGroup{a: a, b: b}`; `d.Add(1)`, `d.Add(2)`, `d.Done()` | `a.Adds() == 3 && b.Adds() == 3`; `a.Dones() == 1 && b.Dones() == 1` |

`server.NewProcessor(...).Register` has a real side effect on the process-global `server.GetRegistry()`, which is why each case deregisters — copy the `defer server.GetRegistry().Deregister(key)` discipline from `listener/registry_test.go:53`.

Create `services/atlas-channel/atlas.com/channel/socket/drain_test.go` (package `socket`) with one case:

| test func | setup | exact assertions |
|---|---|---|
| `TestKickSessionRejectsANonSessionModel` | `KickSession(nullLogger, NewListenerContext(context.Background(), testTenant(t)), nil, sc)` invoked with `listener.Session("not-a-model")` | returns `err != nil` whose `Error()` contains `"unexpected session type"`. This is the only branch reachable without a live socket + kafka producer; the happy path is exercised end-to-end by Task 2's `TestRegistry_DrainClosesTheBoundSocket` shape and by the service at runtime |

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/ -run 'TestCreateSocketService|TestDualWaitGroup|TestKickSession' -v`
Expected: compile failure — `CreateSocketService(...) used as value` (it returns nothing today), `undefined: dualWaitGroup`, `undefined: KickSession`.

- [ ] **Step 3: Rewrite `CreateSocketService`**

Replace `services/atlas-channel/atlas.com/channel/socket/init.go:56-124` with:

```go
// dualWaitGroup fans Add/Done out to two waitgroups so a caller can track
// the same event in a handle-scoped waitgroup and the process-wide one at
// once. Held as the interface, not *sync.WaitGroup, so it is exercisable
// against a counting fake.
type dualWaitGroup struct{ a, b socket.WaitGrouper }

func (d dualWaitGroup) Add(n int) { d.a.Add(n); d.b.Add(n) }
func (d dualWaitGroup) Done()     { d.a.Done(); d.b.Done() }

// CreateSocketService binds the listener SYNCHRONOUSLY and returns it, so
// listener.Registry.Add can surface a bind failure through its existing
// rollback path and buildListener can install Handle.CloseListener. The
// accept loop and per-connection handling stay asynchronous -- only the
// bind result is observable before this returns (task-244 design.md §4.2).
//
// wg brackets the accept-loop goroutine only (process-wide bookkeeping).
// sessionWg is fanned out per accepted connection, so a handle-scoped
// waitgroup sees real session lifetime without also counting the
// accept loop, which lives as long as the handle does.
func CreateSocketService(l logrus.FieldLogger, ctx context.Context, wg socket.WaitGrouper, sessionWg socket.WaitGrouper) func(hp socket.HandlerProducer, rw socket.OpReadWriter, wp writer.Producer, sc server.Model, ipAddress string, port int) (net.Listener, error) {
	return func(hp socket.HandlerProducer, rw socket.OpReadWriter, wp writer.Producer, sc server.Model, ipAddress string, port int) (net.Listener, error) {
		// Bind before any other side effect: a failed bind must leave
		// nothing for Registry.Add's rollback to unwind.
		lis, err := socket.Bind(l, ipAddress, port)
		if err != nil {
			return nil, fmt.Errorf("bind %s:%d: %w", ipAddress, port, err)
		}

		l.Infof("Creating channel socket service for [%s] on port [%d].", sc.String(), port)

		chakra.GetRegistry().StartSweeper(l, ctx)

		hasMapleEncryption := true
		t := sc.Tenant()
		if t.Region() == "JMS" {
			hasMapleEncryption = false
			l.Debugf("Service does not expect Maple encryption.")
		}

		locale := byte(8)
		if t.Region() == "JMS" {
			locale = 3
		}
		l.Debugf("Service locale [%d].", locale)

		sp := session.NewProcessor(l, ctx)
		fanOut := dualWaitGroup{a: wg, b: sessionWg}

		routine.Go(l, ctx, func(_ context.Context) {
			err := socket.Serve(l, ctx, wg, fanOut, lis,
				socket.SetHandlers(hp),
				socket.SetCreator(sp.Create(sc.Channel(), locale)),
				socket.SetMessageDecryptor(sp.Decrypt(true, hasMapleEncryption)),
				socket.SetDestroyer(func(sessionId uuid.UUID) {
					sp.IfPresentById(sessionId, func(s session.Model) error {
						shopscanner.GetRegistry().ClearCharacter(t, s.CharacterId())
						// Without this the throttle map leaks one entry per
						// character ever seen by this pod (task-190).
						statreset.GetRegistry().ClearCharacter(t, s.CharacterId())
						// Channel change and disconnect both destroy the
						// session; without this the window map leaks one
						// entry per character ever seen by this pod
						// (PRD FR-5.5, FR-2.2).
						chakra.GetRegistry().Clear(t, s.CharacterId())
						// Channel change and disconnect both destroy the
						// session; without this the pending-unlock map
						// leaks one entry per character ever seen by this
						// pod (task-221).
						remotemerchant.GetRegistry().ClearCharacter(t, s.CharacterId())
						return nil
					})
					sp.DestroyByIdWithSpan(sessionId)
				}),
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

Two things that changed and must stay changed: `socket.SetPort(port)` is **dropped** from the configurator list (`Bind` takes the port directly and `Serve` ignores those fields — keeping it would leave two ways to say the same thing), and the `sync` import is replaced by `fmt`. Final import block for `init.go`: same as today minus `"sync"`, plus `"fmt"`. `errors`, `net`, `time`, `uuid`, `logrus`, `env`, `routine`, `socket`, `tenant` and all the `atlas-channel/...` imports stay.

- [ ] **Step 4: Add `socket/drain.go`**

Create `services/atlas-channel/atlas.com/channel/socket/drain.go`:

```go
package socket

import (
	"atlas-channel/listener"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	chatpkt "github.com/Chronicle20/atlas/libs/atlas-packet/chat/clientbound"
)

// shutdownNotice is what a player sees when their channel is drained out
// from under them.
const shutdownNotice = "This channel is shutting down. You will be disconnected shortly; please log back in."

// SessionsForHandle snapshots this channel's sessions for drain phase 2.
// AllInChannelProvider filters the tenant registry by world and channel,
// which is exactly the server.Key triple -- the tenant comes from ctx,
// which is already tenant-scoped by NewListenerContext.
func SessionsForHandle(l logrus.FieldLogger, ctx context.Context, sc server.Model) func() []listener.Session {
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

// KickSession sends the shutdown notice and destroys the session. Destroy
// emits the logout/destroyed Kafka events and then calls
// Model.Disconnect(), which closes the conn -- that is what makes the
// session's run() goroutine return and release the handle's Wg, so drain
// phase 3 completes before its deadline instead of always timing out
// (task-244 design.md §4.5).
func KickSession(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, sc server.Model) func(listener.Session) error {
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

Confirmed symbols: `server.Model.WorldId()` (`server/model.go:30`), `server.Model.ChannelId()` (`server/model.go:34`), `chatpkt.WorldMessageWriter` (`libs/atlas-packet/chat/clientbound/world_message.go:14`), `writer.WorldMessagePopUpBody` (`socket/writer/world_message.go:47`).

- [ ] **Step 5: Run the tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./socket/ -race -v`
Expected: the three new cases plus `TestKickSessionRejectsANonSessionModel` PASS; the three pre-existing `NewListenerContext`/`WithSelfEnvironment` tests still PASS.

`go build ./...` will FAIL at `main.go` because `buildListener` still calls the old two-argument `CreateSocketService` with no return value — that is expected and is Task 4's job. Run `go vet ./socket/...` and `go test ./socket/...` for this task's gate, and note the known `main.go` break in the report.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/init.go services/atlas-channel/atlas.com/channel/socket/drain.go services/atlas-channel/atlas.com/channel/socket/init_test.go services/atlas-channel/atlas.com/channel/socket/drain_test.go
git commit -m "feat(atlas-channel): bind the socket synchronously and return the listener from CreateSocketService"
```

---

## Task 4: `atlas-channel/main.go` — propagate the bind error and populate the handle

**Module root for `go build`/`go test`:** `services/atlas-channel/atlas.com/channel`

### Files

- `services/atlas-channel/atlas.com/channel/main.go:633-636` — the `hp := ...` / `socket.CreateSocketService(...)` / `return handles, nil` block inside `buildListener`

This is the one task with no test of its own: `main.go` cannot carry a test (see the comment at `main.go:406-417`). Every behavior it wires is covered by Task 2's registry tests and Task 3's socket tests; this task's gate is that the service compiles and `go test ./...` stays green.

### Interfaces

- Consumes from Task 2: `listener.Handle.CloseListener`, `.Sessions`, `.Kick`.
- Consumes from Task 3: the new `CreateSocketService` signature, `socket.SessionsForHandle`, `socket.KickSession`.

- [ ] **Step 1: Rewrite the tail of `buildListener`**

In `services/atlas-channel/atlas.com/channel/main.go`, the block currently reading:

```go
		hp := handlerProducer(fl)(handler.AdaptHandler(fl)(t, wp))(tenantCfg.Socket.Handlers, validatorMap, handlerMap)
		socket.CreateSocketService(fl, tctx, tdm.WaitGroup())(hp, rw, wp, sc, cfg.IPAddress, cfg.Port)

		return handles, nil
```

becomes:

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

`h` is already the `AddBody` closure's parameter (`main.go:394`), and `err` is already a declared name in that scope — if the compiler reports `no new variables on left side of :=`, split it into `var lis net.Listener` + `lis, err = ...`, and add `"net"` to main.go's imports only if it is not already there.

- [ ] **Step 2: Build and test the whole service**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./... -race`
Expected: PASS. This is the first point in the plan where `go build ./...` for atlas-channel is expected to be clean.

- [ ] **Step 3: Confirm atlas-login is still untouched**

Run: `git status --porcelain services/atlas-login` and `cd services/atlas-login/atlas.com/login && go build ./... && go test ./...`
Expected: empty `git status` output for atlas-login, and both commands exit 0.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/main.go
git commit -m "fix(atlas-channel): propagate the bind error from buildListener and populate the drain hooks"
```

---

## Task 5: `atlas-channel/configuration/projection` — thread `h.Ctx` and retry failed adds

**Module root for `go build`/`go test`:** `services/atlas-channel/atlas.com/channel`

### Files

- `services/atlas-channel/atlas.com/channel/configuration/projection/loop.go` — pass `h.Ctx` into `AddBody`; add the `pending` retry map; `execute` returns an error
- `services/atlas-channel/atlas.com/channel/configuration/projection/projection_test.go` — add the three new cases (all existing cases stay untouched)
- `services/atlas-channel/atlas.com/channel/configuration/projection/apply.go` — read-only; `Op`, `OpKind`, `OpAdd`, `OpDrain`, `ListenerConfig`, `ComputeOps`
- `services/atlas-channel/atlas.com/channel/configuration/projection/state.go` — read-only; `NewState`, `ApplyService`, `ApplyTenant`, `Snapshot`
- `services/atlas-channel/atlas.com/channel/configuration/projection/caughtup.go` — read-only; `NewCaughtUp`, `SetEndOffsets`

Patterns to copy: `services/atlas-channel/atlas.com/channel/configuration/projection/projection_test.go:87-144` (`TestComputeOps_AddRemovePortChangeUnchanged` — how a test builds a `configuration.RestModel` + `tenant.RestModel` pair and the matching `server.Key`); `services/atlas-channel/atlas.com/channel/listener/registry_test.go:39-47` (`makeTenant`/`makeServerModel`, needed here to build the `server.Model` the `ServerModelFn` stub returns).

### Interfaces

- Consumes from Task 2: `listener.ErrDraining`, `listener.Registry.Add`, `listener.Registry.Drain`.
- Produces: nothing consumed by a later task.

- [ ] **Step 1: Write the failing tests**

Add to `services/atlas-channel/atlas.com/channel/configuration/projection/projection_test.go` (package `projection_test`).

All three cases drive a real `listener.NewRegistry` with `listener.Dependencies{UnregisterChannel: func(channel.Model) error { return nil }, RemoveHandler: func(string, string) error { return nil }}` and an `ApplyLoop` whose `ServerModel` is a stub returning `server.NewProcessor(logrus.New(), context.Background()).Register(tm, channel.NewModel(w, c), "127.0.0.1", port)`, with `defer server.GetRegistry().Deregister(key)`.

| test func | setup | exact assertions |
|---|---|---|
| `TestApplyLoop_AddBodyReceivesAContextCanceledByDrain` | `AddBody` captures its `parent` argument into a package-local channel (`got := make(chan context.Context, 1)`) and returns `(nil, nil)`. Call `loop.Registry.Add(context.Background(), key, sc, func(h *listener.Handle) (...) { return loop.AddBody(h.Ctx, key, cfg, h) })` — i.e. drive `execute` via a one-op `ApplyLoop.Run` tick with `Interval: 10*time.Millisecond` and a `CaughtUp` that is already caught up (`c.SetEndOffsets("T", map[int]int64{})`), then `loop.Registry.Drain(key)` | the captured ctx's `Err()` is `context.Canceled` after `Drain` returns. **This is the assertion that pins defect 1**: with `loop.go:89` still passing the apply loop's `ctx`, the captured ctx is still live and the test fails |
| `TestApplyLoop_RetriesAFailedAddOnTheNextTick` | `AddBody` returns `(nil, errors.New("bind failed"))` on its first call and `(nil, nil)` on every later call, counting calls with an `atomic.Int32`. Run the loop with `Interval: 10*time.Millisecond` against a snapshot holding one channel, held constant across ticks. Bound the wait: poll the counter for up to 2s | the counter reaches `>= 2` — the failed op is retried even though `ComputeOps` emits nothing new on tick 2 (`prevSvc` already advanced). After the retry succeeds, `loop.Registry.Get(key)` returns `ok == true` |
| `TestApplyLoop_DropsAPendingAddWhenTheKeyLeavesConfig` | `AddBody` always returns an error. Run one tick with the channel present, then `ApplyServiceTombstone()` so the key leaves the snapshot; let 5 more ticks pass | the `AddBody` call counter stops increasing once the key is gone: capture the count right after the tombstone, sleep `100*time.Millisecond` (≥5 ticks at 10ms), and require the count is unchanged |

Cancel the loop's ctx at the end of each case (`defer cancel()`) so no goroutine outlives the test.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./configuration/projection/ -run TestApplyLoop -v`
Expected: `TestApplyLoop_AddBodyReceivesAContextCanceledByDrain` fails with the captured ctx's `Err()` being `nil`; the two retry cases fail because the loop never re-attempts.

- [ ] **Step 3: Thread `h.Ctx` into `AddBody`**

In `services/atlas-channel/atlas.com/channel/configuration/projection/loop.go`, the `OpAdd` branch (`loop.go:86-95`) currently passes the apply loop's `ctx` into `AddBody`:

```go
		_, err := a.Registry.Add(ctx, op.Key, sc, func(h *listener.Handle) ([]listener.HandlerHandle, error) {
			return a.AddBody(ctx, op.Key, op.Cfg, h)
		})
```

Change the inner argument to `h.Ctx`:

```go
		_, err := a.Registry.Add(ctx, op.Key, sc, func(h *listener.Handle) ([]listener.HandlerHandle, error) {
			// h.Ctx, NOT the apply loop's ctx: buildListener builds the
			// socket service's context from this argument, so a per-channel
			// Drain's h.Cancel() must be able to reach it. Passing the
			// apply-loop ctx here is task-244 defect 1 -- the port stayed
			// bound until full pod shutdown.
			return a.AddBody(h.Ctx, op.Key, op.Cfg, h)
		})
```

`Registry.Add`'s own first argument stays `ctx` — that is the parent `h.Ctx` derives from (`registry.go:108`), which is correct as-is.

- [ ] **Step 4: Add the pending-retry loop**

Change `execute` to return an error, and add `pending` to `ApplyLoop`.

In the `ApplyLoop` struct (`loop.go:33-43`), add after `Interval`:

```go
	// pending holds ops whose execution failed, keyed by the op's key, so
	// the next tick retries them. Without this a transient bind conflict
	// leaves the channel dead until config changes again: prevSvc/
	// prevTenants advance unconditionally, so ComputeOps never re-emits
	// the OpAdd (task-244 design.md §4.6).
	pending map[server.Key]Op
	// retries counts consecutive failures per pending key, so a persistent
	// conflict logs once at Warn and thereafter at Debug rather than
	// flooding at the tick cadence.
	retries map[server.Key]int
```

In `Run`, before the ticker loop, initialize both maps:

```go
	a.pending = make(map[server.Key]Op)
	a.retries = make(map[server.Key]int)
```

Replace the tick body (`loop.go:66-73`) with:

```go
		case <-t.C:
			nextSvc, nextTenants := a.State.Snapshot()
			ops := ComputeOps(prevSvc, prevTenants, nextSvc, nextTenants)

			// Retry pending ops first, and only those whose key is still
			// desired -- a key that left config is dropped, not retried
			// forever.
			stillDesired := ComputeOps(nil, nil, nextSvc, nextTenants)
			desiredKeys := make(map[server.Key]bool, len(stillDesired))
			for _, op := range stillDesired {
				desiredKeys[op.Key] = true
			}
			for key, op := range a.pending {
				if !desiredKeys[key] {
					delete(a.pending, key)
					delete(a.retries, key)
					continue
				}
				if err := a.execute(ctx, l, op); err != nil {
					a.retries[key]++
					continue
				}
				delete(a.pending, key)
				delete(a.retries, key)
			}

			for _, op := range ops {
				if err := a.execute(ctx, l, op); err != nil && op.Kind == OpAdd {
					a.pending[op.Key] = op
					a.retries[op.Key] = 1
				}
			}
			prevSvc = nextSvc
			prevTenants = nextTenants
```

`ComputeOps(nil, nil, nextSvc, nextTenants)` is the flatten of the *desired* state — every key config currently wants, as `OpAdd`s. That is the cheapest way to ask "is this key still in config" without exporting `flatten`.

Then change `execute`'s signature and its two failure branches:

```go
func (a *ApplyLoop) execute(ctx context.Context, l logrus.FieldLogger, op Op) error {
	switch op.Kind {
	case OpDrain:
		if err := a.Registry.Drain(op.Key); err != nil {
			l.WithError(err).WithField("key", op.Key).Warn("projection.applied drain_failed")
			return err
		}
		l.WithField("key", op.Key).WithField("op", "drain").Debug("projection.applied")
	case OpAdd:
		sc := a.ServerModel(op.Key, op.Cfg)
		_, err := a.Registry.Add(ctx, op.Key, sc, func(h *listener.Handle) ([]listener.HandlerHandle, error) {
			// h.Ctx, NOT the apply loop's ctx -- see Step 3.
			return a.AddBody(h.Ctx, op.Key, op.Cfg, h)
		})
		if err != nil {
			// ErrDraining is expected churn (a re-add landing while the
			// old handle drains), not an operator-visible conflict -- the
			// next tick retries it either way.
			if errors.Is(err, listener.ErrDraining) {
				l.WithField("key", op.Key).WithField("retries", a.retries[op.Key]).Debug("projection.applied add_draining")
				return err
			}
			if a.retries[op.Key] > 0 {
				l.WithError(err).WithField("key", op.Key).WithField("retries", a.retries[op.Key]).Debug("projection.applied add_failed")
			} else {
				l.WithError(err).WithField("key", op.Key).Warn("projection.applied add_failed")
			}
			return err
		}
		l.WithField("key", op.Key).WithField("op", "add").Debug("projection.applied")
	}
	return nil
}
```

Add `"errors"` to `loop.go`'s import block.

- [ ] **Step 5: Run the tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./configuration/... ./listener/... ./socket/ -race -v`
Expected: PASS, including every pre-existing projection case.

- [ ] **Step 6: Run the full service test suite**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./... -race`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/configuration/projection/loop.go services/atlas-channel/atlas.com/channel/configuration/projection/projection_test.go
git commit -m "fix(atlas-channel): thread h.Ctx into AddBody and retry a failed listener add"
```

---

## Acceptance mapping

| PRD acceptance criterion | Where it is met |
|---|---|
| Per-channel `Drain` closes the socket; post-drain dial refused | Task 2 `TestRegistry_DrainClosesTheBoundSocket` + Task 5's `h.Ctx` threading |
| `Add` errors and leaves no entry / no leaked `refs` on bind failure | Task 2 `TestRegistry_AddReturnsErrorAndLeavesNoEntryWhenBodyFails`, Task 3 `TestCreateSocketServiceReturnsErrorWhenPortIsAlreadyBound`, Task 4's `return nil, err` |
| `h.Wg` reflects real sessions; phase 3 is a real wait | Task 1 `TestServeIncrementsSessionWaitGroupAtAcceptSite`, Task 2 `TestRegistry_Phase3CompletesAsSoonAsTheWaitGroupReleases`, Task 4's `h.Wg` argument |
| `DrainAll` still drains every handle | Task 2 `TestRegistry_DrainAllIsBoundedByOneDeadline` |
| `atlas-login` compiles unchanged | Task 1 Step 6, Task 4 Step 3 |
| `go build ./... && go test ./...` in atlas-channel, atlas-login, atlas-socket | Task 1 Steps 5-6, Task 4 Steps 2-3, Task 5 Step 6 |
| Flagless `tools/verify.sh` exits 0 | Controller gate after Task 5, not an implementer step |
