# Redis Leader Election Library — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-05-08
---

## 1. Overview

Atlas services use in-process timer-driven sweep tasks (registered via `tasks.Register`) to perform periodic work: expiring statuses, decaying aggro, recovering HP, sending notifications, cleaning up orphan rows, etc. These tasks were written assuming a single pod per service. They iterate global state stored in Redis or Postgres and emit Kafka events as a side effect. Running any of these services with `replicas > 1` causes every event to be emitted N times — duplicate aggro flips, duplicate drop ticks, duplicate hunger decrements — because each pod independently scans the same global state and independently emits.

The fix is **leader election**: at any moment exactly one pod runs the sweep work, the others stand by and continue serving HTTP/Kafka. The industry-standard answer in a Kubernetes-native deployment is `coordination.k8s.io/v1` Leases, but Atlas is required to support docker-compose deployments as well, so the platform-agnostic answer applies: a Redis-backed distributed lock. Atlas already mandates Redis as a runtime dependency for every service in scope, so this leverages existing infrastructure with no new operational surface.

This task introduces a small shared library exposing a `LeaderElection` primitive built on top of `bsm/redislock` (battle-tested NX-acquire + Lua-renew + Lua-release wrapper around `go-redis`). The library hides the renewal loop from callers — the public API is `LeaderElection.Run(ctx, fn)`, where `fn` is invoked only while the lease is held and is interrupted via context cancellation when the lease is lost. The first consumer is `atlas-monsters`: the six per-pod sweep tickers in `main.go:88-93` (`RegistryAudit`, `StatusExpirationTask`, `DropTimerTask`, `MonsterAggroDecayTask`, `MonsterSkillPickerSweepTask`, `MonsterRecoveryTask`) get gated behind a single shared lease so duplicate emission is impossible regardless of replica count. Subsequent tasks will roll the same pattern into the other services enumerated in §7.

The well-known correctness limitation of single-Redis distributed locks is documented in §8.4: during Redis primary→replica failover, brief split-brain is possible. For Atlas's sweep workloads (idempotent-tolerant Kafka events whose downstream consumers must already handle at-least-once delivery) this is the right correctness/complexity trade-off; multi-Redis Redlock is explicitly out of scope.

## 2. Goals

Primary goals:

- Provide a deployment-agnostic (k8s and docker-compose both supported) primitive that any Atlas service can adopt to make a workload run on exactly one elected pod, with automatic failover within a bounded window.
- Eliminate duplicate Kafka emission from atlas-monsters' six sweep tasks when running with `replicas > 1`. After this task, scaling atlas-monsters' Deployment is safe with respect to sweep duplication.
- Hide renewal-loop complexity from callers. The public API must be misuse-resistant — there should be no way for a caller to forget to renew or forget to release on shutdown.
- Honest documentation of the single-Redis split-brain caveat, so future operators and integrators understand the correctness boundary they are accepting.
- Expose enough observability (counters for acquired/lost/renew-failed events) that operators can detect failover transitions and degraded states from existing dashboards.
- Catalogue every other Atlas service whose sweep tasks have the same multi-pod hazard, so follow-up adoption work is scoped before this task ships.

Non-goals:

- **Multi-Redis Redlock** (the formal spec across N independent Redis nodes). Atlas runs a single Redis instance per environment; the additional safety isn't worth the operational complexity for sweep workloads.
- **Per-tenant leader election.** Sweeps already iterate all tenants from one pod; electing a leader per tenant adds complexity for no benefit on this workload class.
- **Replacing sweep tasks with event-driven alternatives** (Redis keyspace notifications, Kafka deadlines, scheduled job queues). Worth doing eventually for some tasks; out of scope for this PRD.
- **Rolling out the lease to every service in §7 in this task.** Only `atlas-monsters` is the first consumer. Each other service gets its own follow-up task tracked in `docs/TODO.md`.
- **A general-purpose distributed lock for arbitrary critical sections.** The library exposes leader-election semantics specifically (long-lived, auto-renewing leases backing a `Run(ctx, fn)` pattern). Short-lived locks for transactional critical sections can come later if a use case appears; not designed-in here.
- **Replacing or modifying existing Redis-based atomic primitives** in atlas-monsters' `monster/registry.go` (`WATCH` + Lua scripts) or `libs/atlas-redis.TenantRegistry`. Those are unrelated patterns.

## 3. User Stories

- As an operator scaling atlas-monsters' Deployment from 1 to 3 replicas to handle peak load, I need confidence that sweep-driven Kafka events (status expirations, aggro decay, drop ticks, HP/MP recovery, skill repick, registry audit) are emitted exactly once per logical event, not three times.
- As an SRE diagnosing a flapping leader during a Redis failover, I need counters and log lines that make it clear which pod held the lease, which pod acquired it, and how long the gap was.
- As a developer rolling the same pattern into atlas-buffs, atlas-pets, atlas-drops, etc., I want a one-line wrapper at task-registration time, not a 100-line copy-paste of acquire/renew/release code per service.
- As a developer reading the library code six months from now, I should be unable to use it incorrectly: forgetting to renew should be impossible, forgetting to release on shutdown should be impossible, accidentally running the inner function on a pod that doesn't hold the lease should be impossible.
- As an operator running atlas-monsters in a single-pod docker-compose deployment, the library must add no extra latency, no extra Redis connections beyond what's already wired, and no startup hard dependency that would break my deployment if Redis is briefly unavailable. (Best-effort acquire with backoff, not panic.)

## 4. Functional Requirements

### 4.1 New library: `libs/atlas-lock`

- New Go module at `libs/atlas-lock/`, mirroring the layout of `libs/atlas-redis/`, `libs/atlas-tenant/`, etc.
- Module name: `github.com/Chronicle20/atlas/libs/atlas-lock`.
- The library wraps `github.com/bsm/redislock` rather than rolling NX/Lua semantics by hand. `bsm/redislock` is MIT-licensed, ~500 LOC, tracks `go-redis` versions, and implements the exact NX-acquire, Lua-renew, Lua-release contract we need. Wrapping (not re-exporting) lets us keep its types out of our public API so we can swap implementation later without breaking consumers.
- The dependency on `bsm/redislock` is a direct, versioned require in `libs/atlas-lock/go.mod`. No `replace` directives.

### 4.2 Public API surface

The library exposes leader-election semantics via a `Run`-style API that owns the renewal loop. Final exact shape decided at design time, but the surface MUST satisfy:

```go
package lock

// LeaderElection runs work on exactly one elected pod for a named lease.
type LeaderElection struct { /* opaque */ }

// New constructs a LeaderElection bound to a single shared *goredis.Client and
// a service-scoped lease key. The same name must be used by every pod that
// wants to compete for the same role.
func New(rc *goredis.Client, name string, opts ...Option) *LeaderElection

// Run blocks until ctx is cancelled. While the lease is held by this pod,
// fn is invoked. If the lease is lost (renewal fails, Redis blip, etc.),
// the inner ctx passed to fn is cancelled and fn must return promptly. The
// outer Run loop will keep trying to re-acquire indefinitely. Run returns
// only when the outer ctx is cancelled.
func (le *LeaderElection) Run(ctx context.Context, fn func(ctx context.Context)) error

// Options (functional-options pattern):
//   WithTTL(d time.Duration)            // lease TTL; default 30s, range [5s, 5m]
//   WithRefreshInterval(d time.Duration)// renewal cadence; default TTL/3
//   WithBackoff(d time.Duration)        // backoff between failed acquires; default 5s
//   WithLogger(l logrus.FieldLogger)    // observability
//   WithMetricsLabel(label string)      // distinguishes counters when one pod runs multiple LEs
```

- `Run` is the only mechanism by which caller code is invoked. There is intentionally no `Acquire` / `Release` / `Refresh` exposed to callers — those primitives belong to the library and the renewal loop.
- The `fn` argument receives a `ctx` that is a child of the outer `Run` ctx. When the lease is lost (or when `fn` is otherwise asked to stop), the child ctx is cancelled. `fn` is responsible for returning promptly when its ctx is cancelled. If `fn` blocks past a configurable grace window (default 5 s) after ctx-cancel, the library logs a warning and proceeds anyway — `Run` does not block on a runaway `fn`.
- Re-acquire logic is inside `Run`. After `fn` returns (whether due to lease loss or its own decision), `Run` waits a brief backoff, attempts to re-acquire, and re-invokes `fn` if successful.
- The library is thread-safe; many goroutines can hold separate `LeaderElection` instances; one `LeaderElection` instance must not have `Run` called more than once concurrently (documented).

### 4.3 Lease key shape

- Service-scoped, NOT tenant-scoped. Sweep tasks operate across all tenants from one pod, so they're a single logical singleton, not per-tenant.
- Key format: `atlas:lock:<name>` where `<name>` is the caller-provided string.
- Examples (atlas-monsters): `atlas:lock:monsters-sweep`. Future services will use names like `buffs-sweep`, `drops-sweep`, etc.
- The library does NOT wrap `<name>` in any additional namespace beyond `atlas:lock:`. Callers are responsible for choosing names that are unique across services and roles.
- The library MUST reject empty or whitespace-only names at construction time with a clear error.

### 4.4 atlas-monsters integration (first consumer)

- `services/atlas-monsters/atlas.com/monsters/main.go:88-93` currently registers six tasks via `tasks.Register(l, tdm.Context())(...)`. After this task: a single `LeaderElection` is constructed for `name="monsters-sweep"` using the existing `rc *goredis.Client` (`main.go:48`), and the six tasks are registered inside its `Run` callback.
- The exact wiring (whether each task gets its own goroutine inside the leader callback, whether they share a single ticker, etc.) is a design decision. Constraint: the public behavior of each task — its interval, its `Run()` semantics — must be unchanged. Only the gating changes.
- When the leader role is lost mid-flight (Redis blip), in-flight `Run()` calls of individual tasks must finish naturally; the next-tick scheduling stops until the lease is re-acquired. The grace-window behavior in §4.2 applies.
- When this pod was never the leader (another pod holds the lease), the six tasks never run. HTTP, Kafka consumers, and Kafka producers continue normally. (Confirmed: HTTP handlers and Kafka consumers in atlas-monsters are stateless and pod-local correctness — they do not depend on being the leader.)
- The kill-switch behavior: an env var `MONSTER_LEADER_ELECTION_ENABLED` (default `true`) bypasses the leader gate and registers the six tasks as before. Required so the existing single-pod docker-compose deployments continue to work even if Redis briefly hiccups during pod startup; in single-pod mode there's no harm in skipping the lease.

### 4.5 Configuration (atlas-monsters consumer)

| Env var | Purpose | Default | Min | Max |
|---|---|---|---|---|
| `MONSTER_LEADER_ELECTION_ENABLED` | Master toggle for the leader gate | `true` | n/a | n/a |
| `MONSTER_LEADER_TTL` | Lease TTL, Go duration | `30s` | `5s` | `5m` |
| `MONSTER_LEADER_REFRESH` | Renewal cadence | `10s` (= TTL/3) | `1s` | TTL/2 |
| `MONSTER_LEADER_BACKOFF` | Wait between failed acquire attempts | `5s` | `1s` | `1m` |

Out-of-range values fall back to defaults with a warning log (matching atlas-monsters' existing pattern from task-060). Library-level `WithXxx` options accept the same range constraints; the consumer's env loader is responsible for applying defaults before calling `lock.New`.

### 4.6 Multi-tenancy

- The library has no tenant model. Lease keys are service-scoped (§4.3).
- Inside the `fn` callback, sweep tasks continue to iterate global state across all tenants exactly as they do today. Tenant context is constructed per-iteration by each task using `tenant.WithContext(...)`, unchanged from current behavior.
- This task does NOT introduce any new tenant boundary, header, or scoping rule.

### 4.7 Error handling and degraded modes

- **Redis unavailable on startup:** `Run` blocks in its acquire loop, retrying on the configured backoff interval. The pod stays up, serves HTTP, consumes Kafka. The first time Redis becomes reachable, the pod attempts acquisition.
- **Redis unavailable mid-run:** The renewal loop fails; the inner ctx is cancelled; `fn` returns. `Run` enters its acquire-loop until Redis returns.
- **`fn` panics:** The panic is recovered by the library, logged as ERROR, and the lease is explicitly released. `Run` continues into the acquire loop. (The library protects the renewal goroutine from being killed by an uncaught panic in `fn`; the panic does NOT escape `Run`.)
- **Two pods both think they're leader (split-brain after failover):** This is the documented single-Redis correctness boundary. See §8.4. No mitigation in this task — operators must accept that downstream consumers tolerate at-least-once delivery (which they already must for Kafka).

### 4.8 Observability

The library exposes Prometheus counters via `promauto.NewCounterVec` (matches atlas-monsters task-060 pattern):

- `atlas_lock_acquired_total{name}` — incremented when a pod transitions from non-leader to leader for `name`.
- `atlas_lock_lost_total{name, reason}` — incremented when a pod transitions from leader to non-leader. `reason` ∈ {`renew_failed`, `context_cancelled`, `released`, `panic`}.
- `atlas_lock_renew_failed_total{name}` — incremented every time a single renewal attempt fails (not aggregated to lost-leader; one lost-leader transition may follow many renewal failures).
- `atlas_lock_acquire_failed_total{name, reason}` — incremented every time `Acquire` fails. `reason` ∈ {`held_by_other`, `redis_error`}. Useful for detecting "is anyone the leader right now?" gaps.

Cardinality bound: ≤ 1 series per `name` per pod per counter. Atlas has on the order of a dozen services adopting the pattern; cardinality is trivially bounded.

The library logs at `Info` for state transitions (acquire, lose-via-cancel, lose-via-renew-fail, release-on-shutdown) and at `Debug` for individual renewal attempts. No log on every successful renewal — that would be a noise flood at the chosen renewal cadence.

### 4.9 Testing

- Library `libs/atlas-lock` ships with unit tests covering: acquire-then-release, two competitors with only one acquiring, renewal extends the lease past TTL, lease-loss via miniredis disconnect cancels the inner ctx, panic in `fn` is recovered, `Run` exits when outer ctx cancels and explicitly releases the lease, configuration defaults and validation. Use `github.com/alicebob/miniredis/v2` (already pinned in atlas-monsters' go.mod) — no real Redis needed.
- atlas-monsters integration tests: stand up two `LeaderElection` instances against the same miniredis with the same name, verify only one runs the inner callback at a time, verify the second takes over within `TTL + backoff` after the first releases.
- Existing atlas-monsters tests must continue to pass without modification. The kill-switch (`MONSTER_LEADER_ELECTION_ENABLED=false` or unconfigured-test default) means no test contexts gain a Redis dependency they didn't previously have.
- `go test -race ./...` clean in both `libs/atlas-lock` and `services/atlas-monsters/atlas.com/monsters`.

## 5. API Surface

No HTTP, JSON:API, or Kafka surface changes.

The new library `libs/atlas-lock` is purely internal Go API (§4.2).

The consumer (`services/atlas-monsters`) is internal Go API only — no external callers see a difference because the gating is behind `tasks.Register`, which is not exposed.

## 6. Data Model

No persistent data changes. No database migrations. No new Kafka topics.

Redis state added by the library:
- One key per active lease: `atlas:lock:<name>`. Value: a per-pod random token written by `bsm/redislock`. TTL: configured (default 30 s). Total Redis-side memory cost: a handful of bytes per active lease per service. Trivial against the existing Redis allocation.

In-memory state inside the consumer (`atlas-monsters`):
- One `*lock.LeaderElection` instance held in `main.go` for the lifetime of the process.

## 7. Service Impact

### 7.1 Library: `libs/atlas-lock` (new)

- New module with own `go.mod`, `go.sum`, `README.md`.
- Direct dependency on `github.com/bsm/redislock`, `github.com/redis/go-redis/v9`, `github.com/sirupsen/logrus`, `github.com/prometheus/client_golang`, `github.com/alicebob/miniredis/v2` (test only).

### 7.2 `services/atlas-monsters`

- New direct dependency on `libs/atlas-lock` in `go.mod` (with `replace` directive to `../../../../libs/atlas-lock` mirroring the existing sibling-lib pattern).
- `main.go:88-93`: the six `tasks.Register(...)` calls move inside a `LeaderElection.Run(...)` callback, gated by `MONSTER_LEADER_ELECTION_ENABLED`.
- Otherwise unchanged. `monster/aggro_task.go`, `monster/picker_task.go`, `monster/recovery_task.go`, `monster/status_task.go`, `monster/drop_timer_task.go`, `monster/task.go` (RegistryAudit) are untouched — their `Run()` methods are wrapped, not modified.

### 7.3 Other Atlas services with the same multi-pod hazard

The following services register sweep tasks in their `main.go` files that iterate global state and emit Kafka or perform DB writes. Each will need its own follow-up task to adopt `libs/atlas-lock` before its Deployment can safely scale beyond one replica. **`docs/TODO.md` will be updated as part of this task to track them**.

| Service | File | Tasks | Notes |
|---|---|---|---|
| atlas-buffs | `services/atlas-buffs/atlas.com/buffs/main.go:63-64` | `NewExpiration`, `NewPoisonTick` | Iterates global buff state; emits Kafka. |
| atlas-ban | `services/atlas-ban/atlas.com/ban/main.go:79-80` | `NewExpiredBanCleanup`, `NewHistoryPurge` | DB cleanup; idempotent but wasteful at N pods. |
| atlas-drops | `services/atlas-drops/atlas.com/drops/main.go:92` | `NewExpirationTask` | Iterates drops, emits Kafka — duplicate emission per pod. |
| atlas-pets | `services/atlas-pets/atlas.com/pets/main.go:89` | `NewHungerTask` | DB iteration plus emission. |
| atlas-skills | `services/atlas-skills/atlas.com/skills/main.go:77` | `NewExpirationTask` | DB-backed, iterates global skill state. |
| atlas-reactors | `services/atlas-reactors/atlas.com/reactors/main.go:68` | `NewCooldownCleanup` | DB cleanup. |
| atlas-maps | `services/atlas-maps/atlas.com/maps/main.go:105-107` | `NewRespawn`, `NewWeather`, `NewMistTick` | Iterates global map state and emits Kafka — significant duplicate-event hazard. |
| atlas-merchant | `services/atlas-merchant/atlas.com/merchant/main.go:79-81` | `NewExpirationTask`, `NewCleanupTask`, `NewNotificationTask` | DB cleanup + notification emission. |
| atlas-guilds | `services/atlas-guilds/atlas.com/guilds/main.go:99` | `NewTransitionTimeout` | DB-backed transition cleanup. |
| atlas-account | `services/atlas-account/atlas.com/account/main.go:76` | `NewTransitionTimeout` | DB-backed. |
| atlas-world | `services/atlas-world/atlas.com/world/main.go:90` | `NewExpiration` | Channel-state expiration. |
| atlas-invites | `services/atlas-invites/atlas.com/invites/main.go:80` | `NewInviteTimeout` | Iterates pending invites. |
| atlas-expressions | `services/atlas-expressions/atlas.com/expressions/main.go:49` | `NewRevertTask` | Iterates expressions. |
| atlas-character | `services/atlas-character/atlas.com/character/main.go:102` | `NewTimeout` | Session timeout — needs review of whether work is per-pod or global. |
| atlas-login | `services/atlas-login/atlas.com/login/main.go:125` | `NewTimeout` | Sessions are pod-owned in atlas-login; this task is likely legitimate per-pod work and may NOT need the leader gate. Review before adding. |
| atlas-channel | `services/atlas-channel/atlas.com/channel/main.go:380` | `NewHeartbeat` | Heartbeat is per-pod state by design; should NOT use the leader gate. Listed for completeness. |

The two italicised services (atlas-login, atlas-channel) are listed for explicit review-and-decline, not for adoption. The remaining 14 are the candidate set for follow-up tasks.

## 8. Non-Functional Requirements

### 8.1 Performance

- `LeaderElection.Run` adds one Redis `SET NX PX` on each acquire attempt and one Redis `EVAL` (Lua renewal script) every refresh interval. Steady-state Redis load: ~6 commands per minute per service per pod (TTL/3 ≈ 10 s renewal + ~1 acquire-retry per minute on the standby pods). Negligible against the existing Redis traffic.
- Failover window: bounded by `TTL + backoff`. With defaults (30 s + 5 s), worst case 35 s of zero work after a leader pod dies. Tunable via env vars.
- Library overhead per `fn` invocation: zero — `fn` is called directly without an additional goroutine hop. The renewal loop runs in a separate library-managed goroutine that does not block `fn`.

### 8.2 Correctness

- Single-Redis split-brain is documented in §8.4. Outside of failover windows, the lease guarantees mutual exclusion modulo clock-skew assumptions inherent to TTL-based locks.
- `fn` MUST be idempotent enough to tolerate a brief overlap during a failover transition. Atlas sweep workloads emit Kafka events that downstream consumers must already handle as at-least-once (a Kafka invariant), so this is consistent with existing requirements.
- The library MUST recover from a panic inside `fn` without escaping `Run` and without leaking the lease (it must explicitly Release rather than waiting for TTL).
- `go test -race ./...` clean in both modules.

### 8.3 Observability

- Counters from §4.8 are present and labeled.
- One operator-facing dashboard panel can answer "is there a leader for `monsters-sweep` right now?" via `rate(atlas_lock_acquire_failed_total{name="monsters-sweep", reason="held_by_other"}[1m]) > 0` (someone is failing to acquire because someone else holds it = there is a leader).
- Logs at state transitions are sufficient to reconstruct a failover event after the fact.

### 8.4 Single-Redis split-brain caveat (must be documented in code and PR description)

When Redis fails over from primary to replica:

1. The lease key is replicated asynchronously. The replica may not have the latest lease state at the moment of promotion.
2. The promoted replica may show no lease, allowing a new pod to acquire it. Meanwhile, the original leader pod's renewal succeeds against the old primary up to the moment of disconnection.
3. For a brief window (typically 1–5 s in a healthy Redis Sentinel setup), two pods can each believe they hold the lease.

This is the well-known limitation of single-instance Redis distributed locks, addressed only by the formal Redlock protocol across N independent Redis nodes (out of scope per §2 non-goals). The library's documentation, the atlas-monsters integration commit message, and the consumer-side README MUST acknowledge this. Atlas's sweep workloads emit Kafka events whose downstream consumers must already handle at-least-once delivery (a Kafka semantic, not a leader-election semantic), so the brief overlap is operationally tolerable.

For workloads where this is unacceptable (e.g., financial transactions, exclusive resource claims with no idempotency at the consumer), this library MUST NOT be used. The PR description MUST flag this explicitly so future adopters don't blindly copy the pattern.

### 8.5 Security & multi-tenancy

- No new secrets, no new external network egress.
- No tenant boundary impact — the library is service-scoped (§4.6).

### 8.6 Operability

- Env-var-controlled (`MONSTER_LEADER_ELECTION_ENABLED=false`) bypass for any pod: gates the leader check entirely and registers the tasks unconditionally. Useful for emergency rollback or for single-pod docker-compose deployments where the lease is overhead.
- A pod that loses the leader role does NOT need to be restarted to compete again — the `Run` outer loop handles re-acquisition automatically.
- Operators monitor the `atlas_lock_*` counters and the lease key via `redis-cli GET atlas:lock:<name>` or `redis-cli TTL atlas:lock:<name>`.
- Graceful shutdown: when the outer ctx is cancelled (process termination), `Run` performs an explicit `Release` on its way out, freeing the lease for the next pod immediately rather than waiting for TTL expiry.

## 9. Open Questions

- **Library boundary** — standalone `libs/atlas-lock` (chosen in §4.1) vs. extending `libs/atlas-redis` with a `Lock`/`LeaderElection` type next to `TenantRegistry`. The "audit existing libs before a new one" rule pushes toward the latter; the "single-responsibility" instinct pushes toward the former. Calling this DECIDED for this PRD as standalone, but design phase may revisit.
- **API ergonomics for multiple roles per pod** — if a single pod ever needs to compete for multiple lease names (e.g., atlas-monsters runs `monsters-sweep` and someday `monsters-aggro-decay` separately), the current API requires N `LeaderElection` instances and N `Run` goroutines. Acceptable for now; revisit if N grows.
- **Renewal-failure semantics** — when one renewal attempt fails but the next succeeds, do we count the lease as "still held" without telling `fn`? Current proposal: yes (transient blips are fine). Open: should there be a "max consecutive renewal failures before lease loss" knob? `bsm/redislock` already handles this internally; we may inherit its policy or override.
- **Should `RegistryAudit` (atlas-monsters task at `monster/task.go`) be inside or outside the leader gate?** It walks the registry checking for orphans and logs anomalies; it does not emit Kafka. Inside the gate is safer (no duplicated work). Outside the gate gives every pod local visibility for diagnostics. Provisional answer: inside the gate.
- **Should the library expose a metrics-prefix option** so consumers can prefix counters with `atlas_<service>_lock_*` instead of the bare `atlas_lock_*`? Cardinality is fine either way; the question is dashboard authoring ergonomics. Provisional answer: no; bare `atlas_lock_*` with `name` label.

## 10. Acceptance Criteria

A reviewer can mark this task done when ALL of the following are true:

- [ ] New module `libs/atlas-lock` exists with a `LeaderElection` type, `New`, `Run`, and the option set in §4.2. Unit tests cover the §4.9 list. `go build ./...` and `go test -race ./...` clean.
- [ ] `services/atlas-monsters/atlas.com/monsters/main.go` constructs a `LeaderElection` for `name="monsters-sweep"` using the existing `*goredis.Client` and registers the six sweep tasks inside its `Run` callback.
- [ ] Setting `MONSTER_LEADER_ELECTION_ENABLED=false` causes `main.go` to register the six tasks unconditionally (kill-switch verified by integration test).
- [ ] All four observability counters in §4.8 are emitted and labeled.
- [ ] Manual: stand up two atlas-monsters pods against a single miniredis or local Redis; verify via `atlas_monsters_data_cache_misses_total` (or any sweep-emitted Kafka topic) that exactly one pod is performing sweep work at any given moment, and that killing the leader pod causes the standby to take over within `TTL + backoff` seconds.
- [ ] `docs/TODO.md` updated with one entry per service in §7.3 (excluding atlas-login and atlas-channel which are review-and-decline). Each entry links back to this task as the prerequisite.
- [ ] PRD updated to status `Implemented` and `Date implemented` on merge.
- [ ] PR description and library README clearly document the single-Redis split-brain caveat from §8.4.
- [ ] `go test -race ./...` clean in `libs/atlas-lock` and `services/atlas-monsters/atlas.com/monsters`.
