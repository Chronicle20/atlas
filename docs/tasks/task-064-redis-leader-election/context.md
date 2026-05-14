# Context — Redis Leader Election Library

> Companion to `prd.md`, `design.md`, and `plan.md`. This file is the
> "everything an implementer needs to load before starting" cheat sheet.

## Goal in one paragraph

Build `libs/atlas-lock`, a new Go module that wraps
`github.com/bsm/redislock` to expose a single misuse-resistant
`LeaderElection.Run(ctx, fn)` primitive. Then integrate it into
`services/atlas-monsters/atlas.com/monsters/main.go` so the six
existing sweep tickers (`main.go:88-93`) only run on the elected
pod. A `MONSTER_LEADER_ELECTION_ENABLED` env var preserves the
old single-pod behavior for docker-compose and emergency rollback.

## Key files

| Role | Path | Purpose |
|---|---|---|
| Spec | `docs/tasks/task-064-redis-leader-election/prd.md` | Product requirements |
| Spec | `docs/tasks/task-064-redis-leader-election/design.md` | Architecture + tradeoffs (D-1..D-13) |
| Plan | `docs/tasks/task-064-redis-leader-election/plan.md` | Bite-sized TDD steps |
| New module | `libs/atlas-lock/leader.go` | `LeaderElection` type, `New`, `Run`, options |
| New module | `libs/atlas-lock/metrics.go` | `promauto` counters |
| New module | `libs/atlas-lock/doc.go` | Package doc with split-brain caveat |
| New module | `libs/atlas-lock/leader_test.go` | miniredis-based unit tests |
| New module | `libs/atlas-lock/README.md` | Correctness-boundary doc |
| Existing | `libs/atlas-redis/lock.go` | The OTHER `Lock` type — do NOT extend (see design §3.2) |
| Existing | `services/atlas-monsters/atlas.com/monsters/main.go` (lines 88-93) | The six `tasks.Register` call sites being gated |
| Existing | `services/atlas-monsters/atlas.com/monsters/tasks/task.go` | `tasks.Register` already obeys `<-ctx.Done()` — no changes needed |
| Modify | `services/atlas-monsters/atlas.com/monsters/go.mod` | Add `atlas-lock` direct require + `replace ../../../../libs/atlas-lock` |
| Modify | `go.work` | Add `./libs/atlas-lock` |
| Modify | `docs/TODO.md` | 14 follow-up adoption entries (PRD §7.3) |
| Reference | `libs/atlas-tenant/`, `libs/atlas-redis/` | Sibling-lib layout to mirror |
| Reference | `libs/atlas-redis/lock.go` (existing `Lock`) | Why NOT extend it: design §3.2 |

## Decisions to honor (from design §2)

| # | Decision |
|---|---|
| D-1 | New module `libs/atlas-lock` (do not extend `libs/atlas-redis`) |
| D-2 | Wrap `bsm/redislock` (do not reimplement NX-renew-release) |
| D-3 | Public API = `New(rc, name, opts...)` + `Run(ctx, fn)` + functional options. NO `Acquire`/`Release`/`Refresh` exposed. |
| D-4 | Lease key: `atlas:lock:<name>` — service-scoped; reject empty/whitespace |
| D-5 | atlas-monsters wiring: `fn(leaderCtx) { tasks.Register(l, leaderCtx)(...); <-leaderCtx.Done() }`. Existing `tasks.Register` ctx-cancel semantics handle teardown. |
| D-6 | Kill switch: `MONSTER_LEADER_ELECTION_ENABLED` env var (default `true`) |
| D-7 | Failover-during-fn: in-flight task `Run()` calls finish naturally; the next `<-time.After` tick is what's skipped |
| D-8 | Panic in `fn`: recovered, logged ERROR, lease explicitly released, outer loop continues |
| D-9 | Four `promauto` counters labeled by `name` (and `reason` where applicable). No `service` label. |
| D-10 | Single-Redis split-brain documented in `doc.go`, README, integration commit message, PR description. NO code mitigation. |
| D-11 | `RegistryAudit` is INSIDE the gate |
| D-12 | Metrics prefix is fixed `atlas_lock_*` (no option) |
| D-13 | Renewal-failure threshold inherits `bsm/redislock` policy; not exposed as a knob in v1 |

## Dependencies

`libs/atlas-lock/go.mod` requires:

```
github.com/Chronicle20/atlas/libs/atlas-lock
go 1.25.5

require (
    github.com/bsm/redislock v0.9.4   // pick latest tag compatible with go-redis/v9
    github.com/redis/go-redis/v9 v9.19.0
    github.com/sirupsen/logrus v1.9.4
    github.com/prometheus/client_golang v1.23.2
    github.com/alicebob/miniredis/v2 v2.37.0     // test
    github.com/stretchr/testify v1.11.1          // test
)
```

`bsm/redislock` is the canonical Go redis-lock library (MIT, ~500 LOC).
Its API (per its README at the time of writing):

```go
import "github.com/bsm/redislock"

locker := redislock.New(rc)                                  // takes *goredis.Client
lock, err := locker.Obtain(ctx, key, ttl, &redislock.Options{
    RetryStrategy: redislock.NoRetry(),                      // we own retry
})
// returns *redislock.Lock or err. err == redislock.ErrNotObtained when held.

err = lock.Refresh(ctx, ttl, nil)                            // PEXPIRE only if value matches
err = lock.Release(ctx)                                      // DEL only if value matches
```

If the upstream version drift requires API tweaks, the wrapper is the ONLY
place to update — the design pins us on Obtain/Refresh/Release exclusively.

## Patterns to follow

- **Sibling-lib layout:** copy structure from `libs/atlas-tenant/` or `libs/atlas-redis/`. Each lib is its own Go module with `go.mod`, optional `README.md`, package files at the lib root. No `internal/` layering for v1.
- **`replace` directive in consumer:** `services/atlas-monsters/atlas.com/monsters/go.mod` already has 16 `replace` directives for sibling libs — add one more, mirroring the others' relative path (`../../../../libs/atlas-lock`).
- **`go.work` entry:** add `./libs/atlas-lock` to the `use (...)` block, alphabetical with other libs.
- **Env-loader pattern (task-060):** atlas-monsters task-060 introduced the "parse → range-check → warn-and-default" loader pattern. Reuse it for `MONSTER_LEADER_TTL`, `MONSTER_LEADER_REFRESH`, `MONSTER_LEADER_BACKOFF`. Look at recent commits in `services/atlas-monsters/atlas.com/monsters/monster/` to find the helper.
- **Metrics:** atlas-monsters already exposes `/metrics` via `promhttp.Handler()` (`main.go:84`). Counters declared via `promauto.NewCounterVec` auto-register and appear at `/metrics` for free.
- **Logging:** `logrus.FieldLogger` is the project's logger interface. Use `WithField`/`WithError`. Default logger in `New` if `WithLogger` not provided: `logrus.New()`.

## Gotchas

- **Two `Lock` types in scope.** `libs/atlas-redis` has a `Lock` (short-lived NX/Del) used by `services/atlas-messengers`. We are NOT touching it. Our package name is `lock` (path `libs/atlas-lock`). Don't import both into the same file unless aliased — design §3.2 explains the divergence.
- **TTL/Refresh range invariants.** `WithTTL` accepts `[5s, 5m]`; `WithRefreshInterval` accepts `[1s, TTL/2]`; `WithBackoff` accepts `[1s, 1m]`; `WithGracePeriod` accepts `[1s, 30s]`. Out-of-range options return an error from `New(...)` — do NOT silently clamp. (The atlas-monsters env loader is the layer that warns-and-defaults.)
- **`Run` is single-shot per instance.** Documented in design §5: one `Run` per `LeaderElection`. Construct N instances if you need N concurrent roles.
- **`fn` MUST block until its `leaderCtx` is cancelled.** If `fn` returns early, `Run` releases the lease and re-enters the acquire loop — which probably is not what the caller wants. atlas-monsters' integration uses `fn(leaderCtx) { register(...); <-leaderCtx.Done() }` for exactly this reason.
- **Renewal classification.** `redislock.ErrNotObtained` from `Refresh` = lease was lost (cancel `leaderCtx`). Other errors = transient (increment counter, keep ticking). Do NOT cancel on transient errors; let bsm/redislock's own TTL drain handle it.
- **Acquire-failure classification.** `redislock.ErrNotObtained` from `Obtain` = held by another pod (`reason="held_by_other"`). Other errors = Redis problem (`reason="redis_error"`). Both retry on `Backoff`.
- **`Release` on shutdown is best-effort.** If Redis is gone, `Release` will fail; that's fine — the lease will TTL out. Don't return the error from `Run`; log at `Debug`.
- **miniredis time vs real time.** `miniredis.RunT(t)` uses its own internal clock for TTLs. `mr.FastForward(d)` advances it. Tests that exercise renewal extension can run in real time with short TTLs (`TTL=1s`, `RefreshInterval=200ms`); tests that exercise lease loss should use `mr.FastForward(TTL+1*time.Second)` to deterministically expire the lease.
- **`go test -race ./...` MUST pass** in both `libs/atlas-lock` and `services/atlas-monsters/atlas.com/monsters` (PRD §10 acceptance criterion). Pay attention to the renewer/fn goroutine coordination — use `context.WithCancel`, not flag-based signaling.
- **Worktree discipline.** This task lives at `.worktrees/task-064-redis-leader-election/` on branch `task-064-redis-leader-election`. Every shell command and file write uses absolute paths under that worktree. Never write task artifacts under main's `docs/tasks/`.

## Out of scope (do NOT do)

- Per-service rollout to atlas-buffs, atlas-pets, atlas-drops, atlas-maps, etc. — only `docs/TODO.md` entries, no code changes.
- Modifying `libs/atlas-redis/lock.go` or atlas-messengers' usage of it.
- Multi-Redis Redlock.
- Per-tenant leader election.
- Replacing sweep tasks with event-driven alternatives.
- Exposing `Acquire`/`Release`/`Refresh` on the public API.
- Adding a metrics-prefix option.

## Success markers

When you're done:
- `cd libs/atlas-lock && go test -race ./...` clean.
- `cd services/atlas-monsters/atlas.com/monsters && go test -race ./...` clean.
- `cd services/atlas-monsters/atlas.com/monsters && go build ./...` clean.
- All checkboxes in `prd.md` §10 acceptance criteria are satisfied.
- `docs/TODO.md` lists 14 follow-up adoption tasks linked to this one.
- The integration commit message and the eventual PR description acknowledge the single-Redis split-brain caveat with the phrase "Redlock is out of scope".
