# task-299 — Plan Context

Companion to `plan.md`. Inputs: `prd.md` (v1), `design.md` (v1), `inventory.md`
(captured at `31a791e3a`).

---

## Key files

| File | Role |
|---|---|
| `libs/atlas-routine/routine.go:15` | `Go(l, ctx, fn)` — the panic-containment primitive both scheduler goroutines run on. Logs a recovered panic at Error with value + stack, then ends the goroutine. |
| `libs/atlas-routine/routine_test.go:1-27` | The test package shape to copy: external `package routine_test`, `test.NewNullLogger()`, the `waitFor` polling helper. |
| `libs/atlas-routine/go.mod` | `logrus` only. The scheduler adds `context`/`sync`/`time` (stdlib), so **no `go.mod`/`go.sum` change is expected anywhere in the repo**. A dirty `go.mod` after any task is a signal something went wrong. |
| `libs/atlas-service/teardown.go:12` | `atlas-service` imports `atlas-routine`. This is why the `*sync.WaitGroup` is a `Register` **parameter** and not read from a `Manager` — the reverse import is a cycle. |
| `libs/atlas-service/teardown.go:55-65` | `Manager.Wait()`: `<-termChan` → `close(doneChan)` → TeardownFuncs fire concurrently → `cancel()` → `waitGroup.Wait()`. Unchanged by this task; task loops simply join that WaitGroup. |
| `libs/atlas-service/bootstrap.go:92-93` | `rt.Context()` and `rt.WaitGroup()` — both already in scope in every one of the 22 `main.go` files. |
| `services/atlas-account/atlas.com/account/tasks/task.go` | The variant-2 (majority, 18-file) `task.go` being replaced. Its loop body is what `routine.Register`'s loop reproduces verbatim, including the `"Stopping task execution."` log line. |
| `services/atlas-maps/atlas.com/maps/tasks/respawn.go:32-38` | The canonical impl transformation (`context.Background()` → `ctx` as the otel span root). |
| `services/atlas-maps/atlas.com/maps/main.go:132-143` | The canonical F-1 site: four `tasks.Register` calls each wrapped in a pointless `routine.Go`. |

## Decisions carried in from `design.md`

- **Panic semantics preserved (OQ-1).** A panic in `Run` ends that task's loop;
  `routine.Go`'s recover logs it. No per-tick recovery — that is a behavior change
  across 22 services and can be added later entirely inside `scheduler.go`.
- **5s drain (OQ-2)**, as an unexported package-level `var drainTimeout`. `var` not
  `const` solely as an in-package test seam.
- **Non-positive `SleepTime()` clamps** to `minSleep = 1s` with a Warn naming the task
  type, rather than stopping the loop — `SleepTime()` is config-derived and therefore
  transiently wrong.
- **Two goroutines per registered task.** The watchdog owns the sole `wg.Done()`, so a
  `Run` that ignores its context delays shutdown by at most `drainTimeout` instead of
  hanging `Manager.Wait()` forever. Double-release is structurally impossible; no
  `sync.Once`.
- **No package renames or relocations (OQ-4).** The `tasks` package directory stays
  wherever other files remain (buffs, maps, rankings, reactors, skills).
- **atlas-parcel and atlas-mts are out of scope (OQ-3).** They drive themselves with
  their own ticker/stopCh/WaitGroup and are a separate follow-up. They are the only
  expected survivors of the `) Run()` grep in Task 24.

## Dependencies and ordering

- **Task 1 gates everything.** Tasks 2–23 call `routine.Register`, which does not
  exist until Task 1 lands. Task 1 is otherwise purely additive — nothing else in the
  repo compiles differently after it.
- **Tasks 2–23 are mutually independent.** Each service is its own Go module and no
  service imports another's `tasks` package, so they are safely parallel.
- **Each service unit is atomic.** Within a module, deleting `tasks/task.go` before
  every `Run()` in that module is converted leaves the module uncompilable. Never
  split a service unit across two commits.
- **Task 24 lands last** and is the only completion evidence: 23 modules change, so a
  module-local build proves nothing about the repo.

## Task sizing — the deliberately large units

`plan-lint.sh` F4 warns above ~6 files. These exceed it on purpose, because the unit
is atomic (see above) and the edits are the same mechanical change repeated, which is
the shape that batches well inside one implementer budget:

| Task | Files | Why not split |
|---|---|---|
| Task 15 atlas-monsters | 14 | 8 impls registered from one 8-line `main.go` block, plus 19 test call sites in 4 files. Splitting produces an intermediate non-compiling module. |
| Task 13 atlas-maps | 6 | 4 impls in one package + `main.go` + the deletion. |
| Task 4 atlas-buffs | 6 | 3 impls in one package + 1 test + `main.go` + the deletion. |
| Task 20 atlas-rps | 6 | 1 impl + 2 test files + `main.go` + the deletion + `docs/domain.md`. |
| Task 22 atlas-summons | 7 | 2 impls + 3 test files + `main.go` + the deletion. |

Every other service unit is 3–5 files.

## Codemod evaluation (`docs/codemod-vs-agents.md`)

The transformation is templated and repeated 22 times, which triggers the rule, so it
was evaluated at plan time. **Verdict: agent fan-out, no rewriter.** The rule requires
at most one judgment step per site; this change has four a rewriter cannot decide:
which context a body should root, whether a `ctx` struct field became dead, whether the
constructor parameter may then be dropped, and what `ctx` each of the ~45 test call
sites should pass. An implementer may still use a scripted pre-pass for the two purely
syntactic parts (delete the 22 `task.go`; rewrite the `tasks.Register(l, X)(` prefix).

## Accepted consequences

- **Shutdown log noise.** A sweep in flight at SIGTERM now aborts with
  `context.Canceled` instead of running to completion, producing error logs during
  shutdown that did not appear before. This is the intent of FR-11; every affected
  sweep is idempotent and re-runs on the next tick after restart.
- **First-tick delay.** The four services on variant-1 `task.go` (buffs, maps,
  reactors, skills) lose their immediate first `Run`. `design.md` §7 confirms per-task
  that none of the eight affected sweeps depends on running before its first interval
  — they sweep registries or tables that are empty at boot.
- **Teardown races the drain.** `TeardownFunc`s fire concurrently with the drain, so a
  handle-closing teardown can close a DB/Redis handle mid-query. The query returns an
  error rather than crashing; today the same `Run` is abandoned mid-query at exit, so
  this is not a regression. Ordering teardown behind the drain would change
  `atlas-service`'s teardown contract and is out of scope.

## Notes on inventory drift

- `design.md` §5.5 mentions an `atlas-parcel/parcel/task.go` cross-reference to
  `tasks.Register`. A fresh grep at plan time (`grep -rn 'tasks\.Register\|tasks\.Task'
  --include='*.go' services`) returns **no** atlas-parcel hit — the only non-`main.go`
  hits are the maps ×2, doors ×2, character ×2, and rps ×1 doc comments, each assigned
  to its owning service task. Nothing extra to do there.
- `design.md` §5.4 says "~45 test call sites across 12 services"; the plan-time grep
  finds them in **10** services (buffs, character, doors, expressions, monsters,
  mounts, rankings, rps, summons, world). Task 24's `) Run()` grep is the authority if
  either count is stale at execution time.
- `services/atlas-channel/atlas.com/channel/session/task.go` satisfies the task shape
  but is **not registered anywhere** in `main.go` (only `channel3.NewHeartbeat` at
  :349 and `combo.NewDecayTick` at :363 are). Its signature is still converted in
  Task 5, because Task 24's acceptance grep requires zero `) Run()` outside
  atlas-parcel and atlas-mts.
