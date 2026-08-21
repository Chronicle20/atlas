# Review — task-244 Task 2: `atlas-channel/listener` handle hooks, drain reordering, `ErrDraining`, concurrent `DrainAll`

Range reviewed: `23cc82c5c..a2fe09919` (1 commit, `a2fe09919`).
Files touched: `listener/handle.go`, `listener/registry.go`, `listener/registry_test.go`, `main.go` — exactly the four files the brief scopes this task to (`git diff --stat` confirms no other file changed).

## Method

Read `task-2-brief.md` and `task-2-report.md` in full, then diffed the range directly (`git diff 23cc82c5c..a2fe09919`) rather than trusting the report's self-review. Compared every hunk against the brief's literal code blocks (Steps 3–8) and against the four design properties in the task prompt. Read the full post-diff `registry.go` and `handle.go` to check phase ordering and lock scope in context rather than from hunks alone. Did not re-run `go test` — the implementer's report already carries a `-race -v` run with all-PASS output for `./listener/...` plus a clean full-module `go build ./... && go test ./...`.

## Findings by design property

### 1. Listener closes at end of phase 1, not phase 4

`registry.go:191-199` (Phase 1): `server.GetRegistry().Deregister(key)` → `r.deps.UnregisterChannel(...)` → `if h.CloseListener != nil { h.CloseListener() ... }` → the `phase=1` log line. This is strictly before phase 2 (session kick, `registry.go:202-215`) and phase 3's `h.Wg.Wait()` (`registry.go:217-228`). Confirmed against the design's stated failure mode: with the close left at phase 4, a still-accepting listener could hand `atlas-socket`'s `run()` a fresh connection whose `Add(1)` races phase 3's `Wait()` — the WaitGroup-misuse panic the design calls out. The reorder eliminates that ordering, and the doc comment on `Drain` (`registry.go:168-175`) states the new phase list correctly.

`net.ErrClosed` is explicitly excused from the warn-log (`registry.go:196`), matching the doc comment on `Handle.CloseListener` (`handle.go:62-69`) that says it is safe to call twice — once here in phase 1, once implicitly via `atlas-socket`'s own ctx-cancel-driven close in phase 4/session teardown.

Test coverage: `TestRegistry_DrainClosesTheBoundSocket` (post-drain dial fails) and `TestRegistry_DrainClosesListenerBeforePhase3Wait` (mid-phase-3 dial fails while `Drain` is still blocked on `h.Wg.Wait()`, proven by parking a goroutine on `h.Wg` for 150ms and dialing at the 50ms mark) both directly exercise ordering, not just end state. `TestRegistry_DrainClosesListenerBeforePhase3Wait` is the test that would fail if the close were left at phase 4 — confirmed by inspection of what it asserts (dial must fail while `Drain` has not yet returned).

**PASS.**

### 2. `h.Wg` counts session `run()` goroutines only, doc comment says so

`handle.go:49-57`: the doc comment on `Wg` states it "tracks per-connection session goroutines (atlas-socket's `run()`) for this handle, and nothing else," explicitly calls out that the accept loop, per-packet `handle()` goroutines, and the per-session ctx-watcher are excluded, and ties the reasoning to `task-244 design.md §2.2`. No code in this diff adds an `Add`/`Done` pairing that would contradict the comment — `Wg` is only read (`h.Wg.Wait()`, `registry.go:220`) in this diff, never incremented; increments happen in `atlas-socket`'s `Serve`/`run()` (Task 1, out of this diff's scope) and in tests directly (`h.Wg.Add(1)`).

**PASS.**

### 3. `Add` refuses a `Draining` key with `ErrDraining`; rollback path leaks nothing

`registry.go:96-108`: the existing-entry branch now distinguishes `Active` (idempotent return, unchanged) from any other state (`Draining` or theoretically `Removed`, though `Removed` entries are deleted from the map in phase 4 so in practice this is `Draining`) — returns `nil, ErrDraining` without touching `r.entries` or `r.refs`. Comment at `registry.go:103-105` correctly explains why a second insert under the same key would be unsafe (old drain's phase-4 `delete` would remove the new handle).

Rollback path on `body` failure (`registry.go:122-134`) — `delete(r.entries, key)`, `r.refs[key.TenantId]--`, delete the tenant's zero-count entry — is unchanged from before this diff and still correctly reached; verified live by `TestRegistry_AddReturnsErrorAndLeavesNoEntryWhenBodyFails`, which asserts `r.Get(key)` returns `ok == false`, `r.Snapshot()` is empty, and — the load-bearing assertion — a second independent key's `Add`+`Drain` still fires the evictor exactly once (`evictCalls == 1`), which would be `2` (or the evictor would never fire at all, depending on how the leak manifested) if the failed `Add` had left a stray `refs` increment for the tenant.

`TestRegistry_AddRefusesADrainingHandle` parks phase 2's `Kick` on a channel to hold the handle in `Draining` for the duration of a concurrent second `Add`, and asserts `h2 == nil`, `errors.Is(addErr, ErrDraining)`, and `r.Snapshot()` still length 1. This is the concurrent case, not just the sequential one.

**PASS.**

### 4. `DrainAll` is concurrent, bounded by one deadline

`registry.go:265-277`: `DrainAll` snapshots handles, spawns one `go func(h *Handle) { defer wg.Done(); r.Drain(h.Key) ... }(h)` per handle (closure argument, not loop-variable capture — correct regardless of Go version), and `wg.Wait()`s for all to finish. Each `Drain` call independently acquires `r.mu` only for its own short critical sections (state transition, map delete, refs decrement) — no lock is held across the deadline wait, so concurrent drains of different handles don't serialize on each other.

`TestRegistry_DrainAllIsBoundedByOneDeadline` sets up 4 handles each holding `h.Wg` open (so each would burn the full 200ms deadline if drained sequentially) and asserts total elapsed `< 700ms`, which would fail at `>= 800ms` under the old sequential `DrainAll`. This is a real regression test for the concurrency change, not just a smoke test.

Doc comment at `registry.go:258-264` explains the deliberate choice of a plain `go` over `routine.Go` ("DrainAll must join") — consistent with the stated fact that `routine.Go` always invokes `fn` and only recovers panics; using `routine.Go` here would have added panic-recovery at the cost of needing a separate signal for "goroutine finished," which the code already gets for free from `wg.Wait()`. Not a defect — matches the brief exactly.

**PASS.**

## Other checks

- **`Dependencies` trim** (`registry.go:23-31`): reduced to exactly `UnregisterChannel` + `RemoveHandler`, matching the brief's Step 4 code block verbatim (aside from a `—`→`--` comment-dash normalization from an auto-formatter, called out in the implementer's self-review and confirmed harmless).
- **`main.go` scope**: the diff's only hunk is `@@ -283,15 +283,7 @@ func main()`, deleting the three stub `Dependencies` fields (`SessionsForKey`, `SendShutdownNotice`, `DestroySession`) and their comments, leaving `UnregisterChannel`/`RemoveHandler` untouched. Confirmed via `git diff --stat` that this is the *only* hunk in `main.go` — `main.go:633-636` / `buildListener` is untouched, matching the explicit out-of-scope guard in the task prompt. `buildListener` (`main.go:372+`) does not set `h.CloseListener`/`h.Sessions`/`h.Kick` — those remain nil in production until Task 4, which is the stated-correct state for this task (nil hooks are guarded at every call site: `registry.go:195` checks `h.CloseListener != nil`, `registry.go:207` checks both `h.Sessions != nil && h.Kick != nil`).
- **No stubs introduced**: the three deleted `main.go` fields (including their `TODO` comment) are gone outright, not replaced with placeholders. `Handle.CloseListener`/`Sessions`/`Kick` are real function-typed fields consumed by real (if currently nil-in-prod) code paths, not stub handlers.
- **Happens-before comment on `Handle`** (`handle.go:38-42`): correctly describes why setting the three new hook fields inside the `Add` body (outside `r.mu`) is safe — `Add` re-acquires `r.mu` after `body` returns (`registry.go:136-138`, and also on the rollback path `registry.go:125-131`) and `Drain` only reads the handle after acquiring `r.mu` (`registry.go:178`), giving a lock-mediated happens-before edge. Verified this pairing is real in both the success and failure paths of `Add`.
- **No line-ending or formatting drift**: diff hunks are clean adds/removes: no evidence of unrelated reformatting of untouched lines.
- **No invented symbols**: `socket.Bind`, `socket.Serve`, `socket.WaitGrouper` (Task 1), `listener.SetEvictorsForTest` (pre-existing in `evict.go`, not part of this diff) — all resolve to real, already-defined functions. `routine.Go` referenced only in a doc comment, not called with new semantics.

## Not evaluable

- Whether Task 1's `socket.Bind`/`socket.Serve` genuinely honor `WaitGrouper`/close-on-`Ctx`-cancel semantics the way the new tests assume — that contract lives in `libs/atlas-socket`, outside this diff's surface, and Task 1 was reviewed separately. This task's tests depend on that contract holding, but re-verifying it here would be auditing a sibling package's implementation, which is out of this unit's scope.
- Production wiring of `Handle.Sessions`/`Kick`/`CloseListener` in `buildListener` (Task 4) — correctly out of scope per the task prompt; noted only to confirm it was not silently attempted here.

## Verdict rationale

Every one of the four concurrency-bearing design properties in the review brief is implemented as specified and covered by a test that would fail under the pre-diff (or a plausibly-wrong) implementation, not just one that passes either way. The `main.go` edit is exactly and only the scoped stub-field deletion. No stubs, no invented symbols, no scope creep. Diff matches the brief's literal code blocks closely enough that deviations (a comment-dash normalization) are cosmetic and were called out by the implementer's own self-review, which I independently confirmed.

No blocking findings.
