# Review — Task 3: atlas-channel/socket bind-first CreateSocketService, dualWaitGroup, drain-phase-2 helpers

Range: `a2fe09919..dbb7fcb1e` (1 commit, `dbb7fcb1e`)
Scope: `services/atlas-channel/atlas.com/channel/socket/{init.go,init_test.go,drain.go,drain_test.go}` — matches the file list in `.superpowers/sdd/plan/task-3-brief.md` exactly; no `main.go` edit present, which is correct (Task 4's job).

## Discovery

```
git diff --stat a2fe09919..dbb7fcb1e
 socket/drain.go       | 58 +++
 socket/drain_test.go  | 32 +++
 socket/init.go        | 138 +++--- (82 ins, 56 del)
 socket/init_test.go   | 110 +++ (pure addition)
 4 files changed, 282 insertions(+), 56 deletions(-)
```

## Findings

### 1. Bind-before-side-effects — PASS

`socket/init.go:76-82`: `CreateSocketService`'s returned closure calls `socket.Bind(l, ipAddress, port)` as its first statement and returns `(nil, err)` immediately on failure, before `l.Infof`, `chakra.GetRegistry().StartSweeper`, `session.NewProcessor`, `dualWaitGroup` construction, or either `routine.Go`. Confirmed `socket.Bind` (`libs/atlas-socket/server.go:126-134`) does nothing but `net.Listen` — no registry mutation, no goroutine. On a bind failure nothing downstream runs, so `wg.Add`/`sessionWg.Add` are never called — matches the test assertions in `init_test.go:73-78` (`wg.Adds()==0`, `sessionWg.Adds()==0`). `go test -race ./socket/...` confirms this passes (see Verification below).

### 2. `dualWaitGroup` fan-out — PASS

`socket/init.go:60-63`: `Add(n)` calls `d.a.Add(n); d.b.Add(n)`; `Done()` calls `d.a.Done(); d.b.Done()`. Both methods are unconditional two-line calls with no branch, so there is no path that updates one underlying group and not the other. `TestDualWaitGroupFansOutAddAndDone` (`init_test.go:117-130`) pins `Add(1)+Add(2)` → both sides see `Adds()==3`, one `Done()` → both sides see `Dones()==1`.

`fanOut := dualWaitGroup{a: wg, b: sessionWg}` (`init.go:102`) is passed as `Serve`'s `sessionWg` argument (`init.go:105`), and `wg` (the plain process-wide grouper) is passed separately as `Serve`'s `wg` argument — matches the design comment at `init.go:71-74` (`wg` brackets the accept loop only; `sessionWg` is fanned per-connection via `fanOut`). Confirmed against `libs/atlas-socket/server.go:147-149`: `Serve`'s own `wg.Add(1)/defer wg.Done()` brackets its lifetime, and `sessionWg` (here `fanOut`) is incremented per accepted connection at the accept site — so a session's lifetime is visible to both the handle-scoped and process-wide accounting, exactly as the doc comment claims.

### 3. Return signature / listener ownership — PASS

`CreateSocketService(...) func(...) (net.Listener, error)` returns `lis, nil` at `init.go:148` after starting both `routine.Go` goroutines — the bound listener is handed back to the caller (Task 4's `buildListener`) for `Handle.CloseListener` wiring, matching the brief's produced interface (`task-3-brief.md:56`). `TestCreateSocketServiceReturnsTheBoundListener` (`init_test.go:84-115`) exercises the ephemeral-port happy path and confirms a real assigned port and a clean first `Close()`.

### 4. `socket.SetPort`/import changes — PASS

`socket.SetPort(port)` is absent from the `Serve` configurator list (`init.go:105-131`), consistent with `Serve` binding nothing itself and ignoring ipAddress/port configurators (`libs/atlas-socket/server.go:145-146`). Import block: `sync` removed, `fmt` added (used only for `fmt.Errorf` at `init.go:81`) — matches the brief's exact instruction.

### 5. `drain.go` placement and bodies — PASS, deviation is the documented one

`SessionsForHandle`/`KickSession` live in `socket/drain.go`, exported, per the brief's deliberate deviation from design §4.5 (file cannot be `main.go` because `main_test.go` note at `main.go:406-417` blocks a test there). Bodies match the brief's Step 4 block verbatim byte-for-byte (diffed manually against `task-3-brief.md:191-249` — identical, including comments). No behavioral drift.

- `KickSession` (`drain.go:45-58`) type-asserts `s.(session.Model)`; on failure returns `fmt.Errorf("unexpected session type %T", s)` — satisfies `TestKickSessionRejectsANonSessionModel`'s substring check (`"unexpected session type"`).
- The announce call is best-effort: a write failure is logged at Debug and swallowed, then `sp.Destroy(m)` still runs (`drain.go:52-56`) — matches the design's "best effort" comment and doesn't let a doomed write skip session teardown.
- `SessionsForHandle` (`drain.go:24-37`) returns `nil` (not an error) on a lookup failure, logging at Warn — matches the brief's body verbatim; this is a snapshot-provider shape (`func() []listener.Session`) so swallowing to `nil` here is the specified contract, not an invented one.

### 6. Symbol grounding — PASS, no invented symbols

Verified directly against source, not assumed:
- `socket.WaitGrouper` (`libs/atlas-socket/server.go:103`), `socket.Bind` (`server.go:126`), `socket.Serve` (`server.go:147`) — all exist with the exact signatures the diff calls.
- `session.Processor.AllInChannelProvider(worldId, channelId) ([]Model, error)` (`session/processor.go:43,92`), `Destroy(s Model) error` (`processor.go:66,407`), `Announce(l)(ctx)(wp)(name)(encoder)(Model)` (`processor.go:247`), `NewProcessor(l, ctx) Processor` (`processor.go:71`).
- `writer.WorldMessagePopUpBody(string) packet.Encode` (`socket/writer/world_message.go:47`).
- `chatpkt.WorldMessageWriter` constant (`libs/atlas-packet/chat/clientbound/world_message.go:14`).
- `server.Model.WorldId()`/`ChannelId()` (`server/model.go:30,34`).
- `listener.Session` is `type Session any` (`listener/registry.go:36`) — so both the `session.Model` type assertion in `KickSession` and the direct `Model`→`Session` append in `SessionsForHandle` compile with no adapter, as the report claims.

### 7. Pre-existing tests untouched — PASS

`git diff a2fe09919..dbb7fcb1e -- socket/init_test.go` shows a pure addition (110 insertions, 0 deletions) — `TestNewListenerContextCarriesThisPodsEnvironment`, `TestNewListenerContextOnMainIsTheLegacyValue`, `TestWithSelfEnvironmentCarriesThisPodsEnvironment` are byte-identical to the pre-image.

### 8. Registration goroutine wg tracking — not a regression, note only

`init.go:140-146`, the channel-registration goroutine (`channel.NewProcessor(...).Register` then `<-ctx.Done()`) is started via a bare `routine.Go` with no `wg`/`sessionWg` accounting, same as the pre-image (it was nested inside an untracked outer `routine.Go` before — see `git show a2fe09919:.../init.go:64-125`). Not a regression introduced by this diff; out of this task's scope since neither the brief nor the design assign this goroutine to either waitgroup. Noted for completeness, not a finding.

## Verification (package-scoped gate, per the task's own instructions)

```
$ cd services/atlas-channel/atlas.com/channel
$ go build ./socket/...        # clean, no output
$ go vet ./socket/...          # clean, no output
$ go test ./socket/...         # ok x4 packages
$ go test -race ./socket/...   # ok x4 packages (socket 1.082s, handler 2.102s, model 1.051s, writer 1.064s)
```
All 8 tests in package `socket` pass, including the 4 new ones (`TestCreateSocketServiceReturnsErrorWhenPortIsAlreadyBound`, `TestCreateSocketServiceReturnsTheBoundListener`, `TestDualWaitGroupFansOutAddAndDone`, `TestKickSessionRejectsANonSessionModel`) and the 3 pre-existing ones. This is a fresh run of the exact commands the implementer already ran with the same code — re-confirmed, not re-tested from scratch, per the review-scope note.

Did not run module-wide `go build ./...`: it is expected to fail at `main.go:627` (old two-arg, no-return-value call site), documented in the brief's Step 5, `context.md`'s ordering constraints, and the implementer's report. Not reported as a finding, per the task instructions.

## Not evaluable

None. The full review surface (the 4 changed files plus the read-only symbol contracts they call into) was directly inspected and verified against source.

## Verdict

APPROVED. No blocking findings. The diff matches the brief's Step 3/Step 4 code blocks verbatim, the bind-before-side-effects property is real (not just test-shaped), `dualWaitGroup` has no partial-fan path, all called symbols exist with the signatures used, the deliberate `drain.go` placement deviation is exactly the one the brief pre-authorized, and the three pre-existing tests are untouched.
