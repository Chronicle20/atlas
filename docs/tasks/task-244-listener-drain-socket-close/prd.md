# Listener Drain Socket Lifecycle Fix — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-20
---

## 1. Overview

`atlas-channel`'s `listener.Registry` owns the per-(tenant, world, channel) lifecycle of a TCP socket listener: `Add` starts one, `Drain` is supposed to tear one down cleanly when the projection apply loop removes a channel from config (or on SIGTERM via `DrainAll`). Three defects in the wiring between `listener.Registry`, `main.go`'s `buildListener`, and `libs/atlas-socket`'s `socket.Run` mean this contract is not actually met:

1. **Drain never closes the TCP socket on a per-channel basis.** `Registry.Drain` cancels `h.Ctx` in phase 4, but `buildListener`'s `AddBody` (`main.go:417`, called from `configuration/projection/loop.go:88-89`) builds the socket-service context from the apply-loop's parent `ctx`, not `h.Ctx`. `socket.Run`'s listener-close goroutine (`libs/atlas-socket/server.go:135-145`) only reacts to that parent ctx, which is canceled only on full pod shutdown. A config-driven drain (channel removed while the pod stays up) leaves the port bound and accepting connections indefinitely, even though the registry has already marked the handle `Removed`.
2. **Bind failures are invisible to the registry.** `socket.CreateSocketService` (`socket/init.go:56-124`) starts `socket.Run` inside a fire-and-forget `routine.Go`; a `net.Listen` failure is logged and swallowed there. `buildListener` calls `CreateSocketService(...)(...)` (`main.go:635`) and unconditionally returns `handles, nil` on the next line, so `Registry.Add` reports success regardless of whether the bind actually succeeded. A port conflict currently produces a registry entry for a channel nothing is listening on.
3. **`h.Wg` never tracks anything.** `Registry.Drain` phase 3 waits on `h.Wg` (`listener/registry.go:210`) expecting it to reflect in-flight session goroutines, but nothing in the socket-service call path increments `h.Wg` — `socket.Run`/`run()` increment the `*service.Manager`-wide waitgroup (`tdm.WaitGroup()`) passed in separately. Phase 3 is currently a no-op regardless of the other two defects, and fixing per-handle context threading requires wiring a per-handle waitgroup through the same path anyway.

These three defects share a root cause: the socket service's startup/shutdown/waitgroup signals are not actually threaded through the `listener.Handle` the registry manages — `Add` and `Drain` believe they are tracking a listener's real state when they are not.

## 2. Goals

Primary goals:
- A per-channel `Drain` (config removal) synchronously closes the bound TCP socket via `h.Ctx`/`h.Cancel`, the same mechanism `Registry.Drain` already uses for kafka handler teardown.
- A `net.Listen` failure during `Registry.Add` surfaces as an error from `Add` itself; no `Handle` is inserted into the registry for a channel whose socket never bound.
- `h.Wg` accurately reflects in-flight session goroutines for the handle being drained, so `Registry.Drain` phase 3's bounded wait is a real wait, not a no-op.
- `libs/atlas-socket`'s public API changes stay compatible with `atlas-login`'s existing single-listener usage (no per-tenant registry there).

Non-goals:
- Redesigning `atlas-socket`'s frame-read loop, encryption, or idle-detection behavior.
- Changing `DrainAll`/SIGTERM behavior beyond what naturally falls out of fixing per-channel `Drain` (it already works today because the whole-process ctx cancellation masks these bugs).
- Adding retry/backoff for a failed bind — `Add` reporting the failure synchronously is in scope; automatic retry policy is not.
- Any change to `projection.AddBody`'s handler-registration behavior (account/asset/buddylist/... `InitHandlers` calls) unrelated to the socket lifecycle.

## 3. User Stories

- As an operator removing a channel from tenant config, I want the channel's TCP port to actually stop accepting connections when the drain completes, so the port is free and no stray client can still connect to a channel the registry considers gone.
- As an operator running multiple pods where two channels race to bind the same port, I want the losing `Add` to fail loudly and leave no registry entry, so the projection apply loop and its logs make the conflict visible instead of silently pretending the channel is live.
- As a developer debugging a drain that appears to hang, I want phase 3's wait to reflect real in-flight session activity, so a timeout warning means something and a fast completion means sessions actually finished.

## 4. Functional Requirements

### 4.1 Per-handle context threading (closes defect 1)

- `configuration/projection/loop.go`'s call into `AddBody` must pass `h.Ctx` (or a context derived from it) as the context the socket service ultimately observes for shutdown — not the apply-loop's parent ctx. Concretely: `a.AddBody(ctx, op.Key, op.Cfg, h)` at `loop.go:89` is invoked from inside `Registry.Add`'s `body` closure, which already has `h` in scope; the `ctx` argument passed to `AddBody` must originate from `h.Ctx`, and `buildListener` must use that (not `ctx` unrelated to `h`) when constructing `socket.NewListenerContext`.
- `socket.CreateSocketService`'s `ctx` parameter (`socket/init.go:56`) must, by the time it reaches `socket.Run`, be a context that `h.Cancel()` actually cancels.
- After the fix: draining a single handle (channel removed from config) must cause `socket.Run`'s listener-close goroutine to fire and `lis.Close()` to execute, without requiring full pod shutdown.

### 4.2 Bind result propagates synchronously (closes defect 2)

- `socket.Run` (or a new entry point in `libs/atlas-socket`) must give its caller a synchronous signal of whether `net.Listen` succeeded before `CreateSocketService`/`buildListener` return control to `Registry.Add`. The `Accept` loop and per-connection handling remain asynchronous; only the bind result needs to be observable before `Add` returns.
- `CreateSocketService` must propagate a bind failure out to its caller instead of only logging it inside the inner `routine.Go`.
- `buildListener`'s `AddBody` must return a non-nil `error` when the bind fails, instead of unconditionally `return handles, nil`.
- `Registry.Add` must treat a bind failure the same as any other `body(h)` error: run its existing rollback path (delete the entry, decrement `refs`, cancel `h.Ctx`) and return the error to the caller — no `Handle` is left in `r.entries` for a channel that never bound.
- `libs/atlas-socket`'s public API change must not break `atlas-login`'s existing call to `socket.Run` (`services/atlas-login/atlas.com/login/socket/init.go:62-76`), which has no per-tenant registry and currently only checks the returned `error` for `net.ErrClosed`. Prefer an additive change (e.g., an option or a second constructor) over a breaking signature change to `socket.Run`, unless a signature change can be made source-compatible across both call sites in the same commit.

### 4.3 `h.Wg` tracks real session activity (closes defect 3)

- The per-handle `*sync.WaitGroup` (`h.Wg`) must be the waitgroup actually incremented/decremented around the socket-service's listener goroutine and each per-connection `run()` goroutine for that handle — not the service-manager-wide `tdm.WaitGroup()`.
- `tdm.WaitGroup()` (`*service.Manager`, used elsewhere for process-wide shutdown bookkeeping) may continue to be tracked in parallel if other code depends on it for full-process shutdown; this task does not need to remove that usage, only make `h.Wg` additionally accurate for the handle-scoped wait in `Registry.Drain` phase 3.
- After the fix: `Registry.Drain` phase 3's `h.Wg.Wait()` blocks until the handle's socket-service goroutine and any active session `run()` goroutines for that handle actually complete (bounded by `cfg.DrainDeadline`, unchanged).

## 5. API Surface

No JSON:API or REST endpoints are affected. This task changes internal Go function signatures/contracts only:

- `libs/atlas-socket`: `socket.Run`'s bind-failure signal must become observable to the caller before `Run` returns control (exact mechanism — return channel, callback option, or split bind/serve functions — is an implementation decision at design time, constrained by 4.2's compatibility requirement).
- `services/atlas-channel/.../configuration/projection.AddBody` (type at `loop.go:19`) — no signature change required, but its invocation and the `buildListener` implementation change to thread `h.Ctx` and propagate bind errors.
- `services/atlas-channel/.../listener.Registry.Add` — behavior change only (rolls back on a body error it already receives); no signature change.
- `services/atlas-channel/.../listener.Handle.Wg` — same field, now actually populated.

## 6. Data Model

No persistent data model changes. `listener.Handle` (in-memory struct, `listener/handle.go`) keeps its existing fields; no new fields are anticipated, though the design phase may determine a wrapper (e.g., a bind-ready channel or listener reference) needs to be threaded through `CreateSocketService`'s closure rather than stored on `Handle` itself.

## 7. Service Impact

- **atlas-channel**: primary target. Changes span `listener/registry.go`, `listener/handle.go` (if `Wg` wiring needs a new field), `socket/init.go` (`CreateSocketService`), and `main.go` (`buildListener`, the `configuration/projection` loop wiring).
- **atlas-login**: consumes `libs/atlas-socket` but has no per-tenant `listener.Registry`/`Drain` concept — a single static listener started at boot. Must remain source-compatible with whatever `atlas-socket` API changes land; verify its existing `socket.Run` call site (`services/atlas-login/atlas.com/login/socket/init.go:62-76`) still compiles and behaves identically (single listener, bind failure already logged and treated as fatal-ish today via the outer error check).
- **libs/atlas-socket**: shared library, locally `replace`-d by both services (no separate versioning/publishing step needed) — a change here is picked up by both `go.mod` replace directives in the same commit.

## 8. Non-Functional Requirements

- **Observability**: a bind failure during `Add` must produce a clear log line (existing `listener.added`/`listener.drain_phase` style fields — `key`, tenant, error) so operators can see why a channel failed to come up, not just a generic apply-loop error.
- **No regression to SIGTERM/`DrainAll` behavior**: `DrainAll` already works today (it happens to work despite the bugs, because the whole-process ctx cancellation masks per-handle context threading defects) — verify it still drains all handles correctly after `h.Ctx` threading changes, since `DrainAll` now depends correctly on the same per-handle mechanism instead of accidentally working via process shutdown.
- **Concurrency safety**: `Registry.Add`'s existing lock discipline (`r.mu`) must still hold under the new synchronous bind-check — the bind check should not hold `r.mu` while blocking on `net.Listen`, matching the existing pattern where `body(h)` already runs outside the lock.
- **Idempotency**: `Registry.Drain`'s existing idempotency (`State == Draining` / `Removed` guards) must be preserved; the socket-close fix must not introduce a path where `Drain` is called twice and `lis.Close()` panics or double-fires (existing `net.ErrClosed` handling in `atlas-socket` already guards this — verify it still applies).

## 9. Open Questions

- Exact shape of the `atlas-socket` API addition (bind-ready channel vs. split `Bind`/`Serve` functions vs. a synchronous pre-check using a throwaway `net.Listen` before handing off to `Run`) — deferred to `/design-task`.
- Whether `h.Wg` should be created fresh per `Handle` (already true — `Add` allocates `&sync.WaitGroup{}` per handle) and simply passed instead of `tdm.WaitGroup()` at the `CreateSocketService` call site, or whether both waitgroups need to be tracked in parallel for other callers of `tdm.WaitGroup()` — deferred to `/design-task` after auditing other `tdm.WaitGroup()` consumers.
- Whether the bind-failure rollback path in `Registry.Add` needs a distinct error type/sentinel so the projection apply loop can distinguish "bind conflict, maybe transient" from "handler init failed, config problem" for future retry-policy work (out of scope for this task's implementation, but worth flagging so the error isn't opaque).

## 10. Acceptance Criteria

- [ ] A per-channel `Registry.Drain` call (not full pod shutdown) results in the TCP listener's `Accept` loop returning and the socket being closed, verified by a test that drains a handle and then confirms a connection attempt to that port is refused.
- [ ] `Registry.Add` returns a non-nil error and leaves no entry in `r.entries` (and no leaked `refs` increment) when the underlying `net.Listen` fails (e.g., port already bound), verified by a test that binds a port out-of-band first, then calls `Add` for a config pointing at that port.
- [ ] `h.Wg` is incremented for the socket-service listener goroutine and each active session `run()` goroutine belonging to that handle, and reaches zero when they exit, verified by a test asserting `Registry.Drain` phase 3 actually blocks until an in-flight fake session goroutine completes (within `DrainDeadline`) rather than returning immediately.
- [ ] `DrainAll` (SIGTERM path) still drains every registered handle correctly after the `h.Ctx` threading change — existing behavior unchanged, covered by existing or updated tests in `registry_test.go`.
- [ ] `atlas-login`'s `socket.Run` call site compiles unchanged (or with a mechanical, source-compatible update) and its existing behavior (single listener, boot-time bind, log-and-continue on non-fatal errors) is unaffected.
- [ ] `go build ./... && go test ./...` passes in both `services/atlas-channel/atlas.com/channel` and `services/atlas-login/atlas.com/login`, plus `libs/atlas-socket`.
- [ ] `tools/verify.sh` (flagless) exits 0 before the branch is considered done.
