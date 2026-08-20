# task-244 — Implementation Context

Companion to `plan.md`. Everything an implementer or reviewer needs that is not
a step in a task section.

## The three defects, and where each is actually fixed

| Defect (PRD §1) | Real site | Fixed in |
|---|---|---|
| 1. Drain never closes the socket per channel | `configuration/projection/loop.go:89` passes the apply loop's `ctx` into `AddBody`, not `h.Ctx` | Task 5 Step 3 (+ Task 2's phase-1 close, which is the *ordering* half) |
| 2. Bind failures invisible to the registry | `socket/init.go:56-124` buries `net.Listen` in `routine.Go`; `main.go:635` returns `handles, nil` unconditionally | Tasks 1, 3, 4 |
| 3. `h.Wg` never tracks anything | `main.go:635` passes `tdm.WaitGroup()`, never `h.Wg` | Tasks 1, 3, 4 |

## Key decisions carried from `design.md` v2

- **`Bind`/`Serve` split rather than "don't background `Run`".** `Run` returns an
  error immediately on bind failure but blocks in the accept loop forever on
  success, so simply un-backgrounding it would deadlock the projection apply
  loop on every successful add (design §2.1).
- **`h.Wg` counts session `run()` goroutines only** — never the accept loop,
  which lives for the handle's whole `Active` lifetime and would make phase 3
  burn its full deadline every time (design §2.2). It also does not cover the
  per-packet `handle()` goroutines or the per-session ctx-watcher; those are
  untracked today and stay untracked (PRD non-goals).
- **The listener closes at the end of phase 1, not phase 4** (design §2.3). Two
  independent reasons: `run()`'s old `wg.Add(1)` inside the goroutine could fire
  on a zero counter while phase 3's `Wait` is in flight (`sync: WaitGroup
  misuse`, a process panic), and leaving the port open through the drain window
  lets a client connect to a channel the registry already deregistered.
- **`sessionWg.Add(1)` moves to the accept site**, closing the accept→`Add` gap.
- **The accept loop must exit on `net.ErrClosed` regardless of ctx state**
  (design §3.3) — required by the phase-1 close, and a latent-bug fix for
  atlas-login too.
- **Phase 2 is wired for real** (design §2.4/§4.5). The three `Dependencies`
  stubs in `main.go:286-293` are deleted, not left as fallbacks. Without a real
  kick, one connected player makes phase 3 block for the entire
  `DrainDeadline`, and sequential `DrainAll` turns SIGTERM into
  `N × DrainDeadline`.
- **`DrainAll` becomes concurrent** so total drain is bounded by one deadline.
- **`Add` refuses a `Draining` key with `ErrDraining`, and the apply loop
  retries pending adds** (design §4.6) — otherwise fixing defect 2 converts a
  silent partial failure into a permanent channel outage.

## Deliberate deviation from the design

**`sessionsForHandle` / `kickSession` live in `socket/drain.go`, not `main.go`.**
Design §4.5 put them in `main.go`. The plan exports them as
`socket.SessionsForHandle` / `socket.KickSession` instead, for the reason
`main.go:406-417` already documents in a comment about `NewListenerContext`: a
file named `main.go` cannot carry a test. The bodies are the design's bodies
verbatim; only the file and capitalization change. No import cycle:
`socket` already imports `session`, `writer`, and `server`, and `listener`
imports only `atlas-channel/server`.

## Known test-scope limitation

Design §6 asked `init_test.go` to assert that a failed bind starts no chakra
sweeper. `chakra.GetRegistry().StartSweeper` is guarded by a process-global
`sync.Once` (`character/chakra/registry.go:164-167`) with no test seam, so
"sweeper not started" is not observable without adding one — out of scope here.
The bind-before-side-effects property is instead asserted through the two
counting `WaitGrouper`s: on a bind failure, neither sees a single `Add`.

## Task sizing

Five tasks. Sizes, against the >6-files / >1-service split rule:

| Task | Files touched | Services | Note |
|---|---|---|---|
| 1 | 2 (+2 read-only) | `libs/atlas-socket` | Verified against atlas-login too (build only, no edit) |
| 2 | 4 | atlas-channel | `main.go` edit is a 3-field deletion needed to keep the module compiling |
| 3 | 4 (+2 read-only) | atlas-channel | `go build ./...` is expected to fail at `main.go` until Task 4 — noted in the task |
| 4 | 1 | atlas-channel | No test of its own; `main.go` is untestable by construction |
| 5 | 2 (+3 read-only) | atlas-channel | |

None is deliberately oversized. Task 4 is deliberately *under*sized — splitting
`main.go`'s call-site change out of Task 3 is what lets Task 3's socket-package
tests gate on their own, since the two cannot both compile at once.

## Ordering constraints

Strictly sequential: 1 → 2 → 3 → 4 → 5.

- Task 3 consumes Task 1's `Bind`/`Serve`/`WaitGrouper` and Task 2's `Handle`
  fields.
- Task 4 consumes Task 3's `CreateSocketService` signature.
- **Between Task 3's commit and Task 4's commit, `atlas-channel` does not
  build.** That is expected and stated in Task 3 Step 5. A verifier run in that
  window will fail on `main.go`; do not dispatch one there.
- Task 5's defect-1 test (`TestApplyLoop_AddBodyReceivesAContextCanceledByDrain`)
  only needs Task 2's registry, so it could in principle run earlier — it is
  sequenced last so the whole service builds when it runs.

## Cross-service seam to check by hand at review

`libs/atlas-socket` has exactly two `Run` callers repo-wide:
`services/atlas-login/atlas.com/login/socket/init.go:62` and
`services/atlas-channel/atlas.com/channel/socket/init.go:79`. atlas-login must
compile with **zero edits** — the only permitted `Run` change is `wg
*sync.WaitGroup` → `wg WaitGrouper`. atlas-login also inherits the §3.3
accept-loop behavior change; on its normal path (ctx cancel closes the
listener) the call site's existing `errors.Is(err, net.ErrClosed)` check already
swallows the returned error, so observable behavior is unchanged.

## Gates

- Per task: module-local `go build ./... && go test ./...` from the module root
  named in the task's Files block. `-race` on `libs/atlas-socket`,
  `./listener/...`, `./socket/`, and `./configuration/...` specifically, given
  the phase-ordering work.
- Branch: flagless `tools/verify.sh` must exit 0 before the branch is claimed
  done — a controller gate, not an implementer step.
- Code review before the PR, per CLAUDE.md.
