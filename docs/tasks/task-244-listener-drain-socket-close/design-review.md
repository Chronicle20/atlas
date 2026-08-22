# task-244 Design — Adversarial Review

Reviewer: controller (inline), 2026-08-20
Scope: `design.md` v1 against `prd.md` v1 and current branch source (`ba816d6f5`).
Note: no `plan.md` exists in this task folder yet — this review is of the design as the
input to `/plan-task`.

**Resolution (2026-08-20): all findings addressed in `design.md` v2.**
B1 → §2.3 (phase-1 listener close + `Add` at the accept site). B2 → §2.3/§4.4 (phase reorder).
B3 → §2.4/§4.5 (phase 2 wired) and §4.4 (concurrent `DrainAll`). B4 → §4.6 (`pending` retry +
`ErrDraining`). N1 → §3.1 (`buildConfig`, `Serve` ignores ip/port). N2/N3 → §4.2 (`WaitGrouper`
throughout). N4 → §2.2 (scope of the phase-3 wait stated). N5 → §6 (`projection_test.go` pins
defect 1). N6 → §6 (`init_test.go` asserts bind-before-side-effects). One finding surfaced while
writing v2 and is also fixed: the accept loop would hot-spin on a phase-1 close → §3.3.

Verdict: **CHANGES REQUESTED**. The three-defect diagnosis is correct and the
`Bind`/`Serve` split is the right shape. But making `h.Wg` live (§4.2) without changing
drain *ordering* introduces one crash risk and one guaranteed latency regression, and
FR 4.2's fail-loudly `Add` creates a new permanently-down failure mode in the apply loop.

---

## Blocking

### B1 — `h.Wg` can panic: `Add` on a zero counter while phase 3's `Wait` is in flight

`run()` calls `wg.Add(1)` *inside* the per-connection goroutine
(`libs/atlas-socket/server.go:161`), after `routine.Go` has already returned. The listener
is not closed until phase 4 (`listener/registry.go:222`), so the accept loop is still
running during phase 3's `h.Wg.Wait()` (`registry.go:209-212`).

Sequence: phase 3 starts with zero live sessions → `Wait()` blocks with the counter at 0 →
a new connection arrives → `run()` does `h.Wg.Add(1)`. That is exactly
`sync: WaitGroup misuse: Add called concurrently with Wait` — a process-wide panic, not a
logged error. Today this is impossible only because `h.Wg` is never incremented; the design
makes it reachable.

The same window also produces a *silent* miss: between `Accept` returning and `run()`
executing `Add(1)`, the counter reads 0, so phase 3 can complete and phase 4 cancel while
a just-accepted connection is unaccounted for.

Fix (either, prefer both):
- Close the listener **before** phase 3 rather than in phase 4 (see B2) so no `Add` can
  race the `Wait`; and
- Hoist the increment out of the goroutine: `sessionWg.Add(1)` in `Serve` immediately after
  `Accept` returns, with `run()` taking ownership of the matching `Done()`. This is the
  standard Add-before-`go` discipline and removes the accept→Add gap.

If neither is acceptable, replace `h.Wg` with a counter + broadcast channel, which has no
reuse restriction.

### B2 — The listener stays open through phases 1–3

Design §2.2 deliberately leaves listener teardown at phase 4 (`h.Cancel()`). Consequence:
for the whole drain window (up to `DrainDeadline`, 5s default / 10s ceiling) the port keeps
accepting clients for a channel that phase 1 already deregistered from `server.GetRegistry()`
and from atlas-world (`registry.go:191-193`). PRD user story #1 asks for the opposite: "no
stray client can still connect to a channel the registry considers gone."

Recommend: a `Handle`-scoped listener-close hook (or a second `quiesce` context) invoked at
the end of phase 1. `Serve`'s existing `net.ErrClosed` guard already makes a later phase-4
`lis.Close()` idempotent, so this costs nothing on the double-close path (PRD NFR
"Idempotency" preserved).

### B3 — Phase 3 will hit the full deadline on every drain, because phase 2 is a stub

`main.go:286-293` wires the registry with:

```go
SessionsForKey: func(key server.Key) []listener.Session { return nil }, // TODO
SendShutdownNotice: func(listener.Session) {},
DestroySession:     func(listener.Session) error { return nil },
```

Phase 2 therefore never kicks anybody. The only thing that closes a client conn is
`session.Model.Disconnect()` (`session/model.go:209`), reached via
`session/processor.go:431` — which phase 2 never calls. So a session `run()` goroutine
exits only when the client leaves or when phase 4 cancels `h.Ctx`.

That means: with `h.Wg` live and a single connected player, phase 3 blocks for the entire
`DrainDeadline`, every time. The design's own §2.2 argument against counting the accept-loop
goroutine ("phase 3 would hit its full timeout on *every* drain, defeating user story #3")
applies verbatim to sessions under the current deps wiring. The design does not mention this.

`DrainAll` iterates handles **sequentially** (`registry.go:251-257`), so SIGTERM shutdown
becomes `N_channels × DrainDeadline` — with 10 channels at the 10s ceiling that is 100s,
well past a typical `terminationGracePeriod`, and the pod gets SIGKILLed mid-drain. That is
a direct regression against PRD NFR "No regression to SIGTERM/`DrainAll` behavior."

Fix: pick one and state it in the design —
1. Wire `SessionsForKey`/`DestroySession` for real in this task (CLAUDE.md's
   "finish producible work" bar — the session processor exists and `Disconnect()` exists,
   this is producible now, and without it AC #3 is only testable against a fake); or
2. Make `DrainAll` drain handles concurrently and bound the *total* by `DrainDeadline`; or
3. Explicitly accept the stall, document it, and state why it is not a SIGTERM regression.

Option 1 is the honest one — with phase 2 a no-op, the entire point of a bounded phase-3
wait is unreachable.

### B4 — A bind failure now takes the channel down permanently

FR 4.2 makes `Add` fail loudly. But the apply loop treats an `Add` failure as terminal:

```go
// configuration/projection/loop.go:88-95
_, err := a.Registry.Add(...)
if err != nil { l....Warn("projection.applied add_failed"); return }
...
prevSvc = nextSvc   // <- runs regardless
```

`prevSvc`/`prevTenants` advance to the snapshot that *included* the failed key, so
`ComputeOps` never re-emits the `OpAdd`. A transient bind conflict (the exact scenario in
PRD user story #2 — two pods racing, or a re-add landing while the previous listener is
still closing) leaves that channel dead until the config changes again.

The most likely trigger is a config flap: `Registry.Add`'s early-return only covers
`State == Active` (`registry.go:104`), so an `OpAdd` arriving while the old handle is
`Draining` creates a *second* handle for the same key — and the old drain's phase 4
`delete(r.entries, key)` (`registry.go:233`) then deletes the new one and decrements `refs`
for it. With the port still bound by the draining listener (per B2), the new `Bind` also
fails. Before this task that was invisible; after it, it is a permanent outage.

At minimum the design must say what happens on `add_failed`: either don't advance `prev*`
for failed keys (so the next tick retries), or make `Add` refuse a non-`Active` existing
handle explicitly rather than silently double-inserting.

---

## Non-blocking

### N1 — `buildConfig` does not exist

§3's `Run` body calls `buildConfig(configurators...)`; today `Run` inlines the `config`
literal and the configurator loop (`server.go:97-110`). Fine, but it is a new function the
plan must create, and `Bind`'s `(ipAddress, port)` params mean `Serve` still consumes
`SetIpAddress`/`SetPort` configurators that it now ignores — leaving two ways to specify the
port. Either drop those fields from `config` post-`Bind` or note that `Serve` ignores them.

### N2 — `dualWaitGroup` should hold `WaitGrouper`, not `*sync.WaitGroup`

`type dualWaitGroup struct{ a, b *sync.WaitGroup }` (§4.2) pins the concrete type for no
reason, and makes the struct untestable against a counting fake. Use the interface the
design just introduced.

### N3 — `CreateSocketService`'s signature should take `WaitGrouper` too

§4.2 keeps `wg, sessionWg *sync.WaitGroup`. Same reasoning as N2 — and it is what makes the
`init_test.go` test in §6 ("returns non-nil error when the port is already bound, without
starting any goroutine") able to assert *"without starting any goroutine"* at all.

### N4 — Nothing tracks packet-handler goroutines

`handle()` runs in its own `routine.Go` (`server.go:262`), and each session spawns a
ctx-watcher goroutine (`server.go:176`). Neither is in any waitgroup. Phase 3 can therefore
complete with handler work still in flight. Pre-existing, out of this task's scope — but
since the task's claim is "phase 3 is now a real wait," the design should say what phase 3
does and does not cover.

### N5 — The AC #1 test as described does not test the fix

§6 proposes a "minimal fake body that wires `socket.Bind`/`socket.Serve` directly against
`h.Ctx`/`h.Wg` the way `buildListener` does." That asserts the *registry* works, not that
`buildListener` threads `h.Ctx` — and `main.go` cannot carry a test (the existing comment at
`main.go:406-417` says as much). The threading defect actually lives at
`loop.go:88-89`, which **is** testable: add a `projection_test.go` case asserting that the
`ctx` handed to `AddBody` is canceled by `h.Cancel()` / by `Drain`. That is the one
assertion that pins defect 1 at its real site. Keep the fake-body registry test as well, but
don't let it stand in for this.

### N6 — Bind-before-side-effects claim is worth an explicit test ordering note

§4.2 correctly moves `Bind` ahead of `StartSweeper`/session-processor/channel registration.
Worth adding to §6 as an assertion (bind failure ⇒ `chakra.GetRegistry()` sweeper not
started), since it is the property that makes `Add`'s existing rollback sufficient.

### N7 — Good calls worth keeping

- §2.1's deadlock catch (moving `Run` out of `routine.Go` would wedge the apply loop) is
  correct and non-obvious.
- §2.2's exclusion of the accept-loop goroutine from `h.Wg` is correct reasoning, even
  though B3 shows the same argument was not carried through to sessions.
- §4.4 (reuse the existing `add_failed` log line) correctly resolves the PRD's observability
  NFR without inventing a new log.
- §5's source-compatibility analysis for `atlas-login` is accurate: `socket.Run` has exactly
  two callers (`services/atlas-login/.../socket/init.go:62`,
  `services/atlas-channel/.../socket/init.go:79`), and a `*sync.WaitGroup` satisfies
  `WaitGrouper` with no edit.

---

## Suggested design deltas

1. Add a §2.3 on **drain ordering**: listener closes at end of phase 1; phase 2 kicks
   sessions; phase 3 waits on `h.Wg` with no possible concurrent `Add`. This resolves B1
   and B2 together and makes §2.2's argument sound.
2. Add a §4.5 on **phase 2's stubbed dependencies** — either wire them or state the
   accepted stall and the `DrainAll` bound (B3).
3. Add a §4.6 on **`Add` failure semantics in the apply loop** — retry vs. `prev*`
   advancement vs. `Draining`-key rejection (B4).
4. Move `sessionWg.Add(1)` to the accept site in `Serve` (B1) and change `dualWaitGroup` /
   `CreateSocketService` to `WaitGrouper` (N2, N3).
5. Replace §6's AC-#1 test with the `loop.go` context-cancellation assertion (N5), keeping
   the registry fake-body test as a complement.
