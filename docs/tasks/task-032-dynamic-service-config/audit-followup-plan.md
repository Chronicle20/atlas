# Plan Audit Follow-up — task-032-dynamic-service-config

**Plan Path:** docs/tasks/task-032-dynamic-service-config/plan.md
**Audit Date:** 2026-05-27
**Branch:** task-032-dynamic-service-config (current branch label in worktree)
**Base Branch:** main
**Scope:** Two follow-up commits landed after the original implementation/audit:
- `e5fbabadc` — block Get* on PublishSnapshot instead of fatal-ing
- `0b33ee5ee` — give projection subscriber a per-process consumer group

This audit asks: do these fixes regress any PRD acceptance criterion or any FR-CHN-*/FR-LGN-* requirement?

## Executive Summary

The two fixes are surgical, properly scoped, and do **not** regress any acceptance criterion. The caught-up gate, Evict hooks, drain semantics, and listener apply loop are all decoupled from both `configuration.PublishSnapshot` and the consumer-group identity, so the rewrites cannot accidentally flip the gate, suppress Evict, or skip handler deregister. All tests pass with `-race`. `go vet` is clean modulo two **disclosed** pre-existing warnings.

One concrete behavioral asymmetry surfaced that is worth noting (but is not a regression introduced by the fixes): **atlas-channel's main.go never calls `configuration.PublishSnapshot`**, so the new `readyCh` in atlas-channel's `configuration/registry.go` is effectively dead — any caller of `GetServiceConfig`/`GetTenantConfig` from atlas-channel will block the full 60s `readyTimeout` and return `ErrNotReady`. Today no production atlas-channel code path actually invokes these getters (the channel session timeout task in `session/task.go` is defined but is not wired into `main.go`), so this dead code is latent, not active. The fix commit itself documents this. atlas-login correctly calls `PublishSnapshot` from `main.go:130` after `WaitCaughtUp`.

## Per-Concern Findings

### 1. FR-CHN-4 / FR-CHN-5 — caught-up gate semantics

**Status:** No regression.

The per-process projection group ID does **not** affect the gate's correctness. The gate (`configuration/projection/caughtup.go`) compares two pieces of state:
- `snapshots` — per-topic end offsets fetched at boot via `consumer.ReadEndOffsets` (`configuration/projection/subscriber.go:54-65`).
- `consumed` — per-(topic,partition) highest observed offset from each delivered message (`subscriber.go:92,124`).

Both inputs are independent of consumer-group identity:
- `ReadEndOffsets` issues a broker-side `ReadOffsets` call. The high-water mark it returns is a property of the topic/partition, not of any consumer group.
- `Observe` records the offset of messages the handler actually receives.

So the new per-pod group ID just guarantees the broker hands the pod every record from `FirstOffset` (the pod sees the full compacted log). It can't fool the gate into a premature flip — the gate's `evaluateLocked` (`caughtup.go:91-122`) requires `consumed[partition] >= end-1` for every partition whose `end > 0`. A trivially-empty topic (`end == 0`) is the *only* way the gate flips with no observed messages, and that is correct: an empty topic has no state to project.

The fix-commit message accurately diagnoses the previous bug: shared group ID + committed offset at end-of-topic caused kafka-go's `SetStartOffset(FirstOffset)` to be ignored on restart, so `Observe` never fired and the gate never flipped. The per-process group is the right primitive.

**Compaction note** (concern 5): log compaction by design preserves the latest value per key; the projection state is rebuilt from that, which is exactly the snapshot atlas-channel needs. The PRD assumes log-compacted topics (FR-CFG-1, Sec 5.3) — this fix relies on the same assumption.

### 2. FR-CHN-18 / FR-CHN-19 — tenant Evict hooks

**Status:** No regression.

The Evict chain is independent of `configuration.PublishSnapshot`. Flow:
- `configuration/projection/loop.go:79-97` — apply loop calls `Registry.Drain(op.Key)` directly when the projection diff yields an `OpDrain`.
- `listener/registry.go:175-245` — `Drain` runs the four-phase teardown. After phase 4 it decrements the tenant ref count; if `r.refs[tenant] <= 0`, it calls `fireEvictors(r.l, key.TenantId)` (line 241-243).
- `listener/evict.go` — `fireEvictorsForTenant` iterates registered evictors. Evictors are registered from `main.go:274` (channel) and `main.go` (login).

`PublishSnapshot` is not in this path. The rewrite of `PublishSnapshot` (now grabs the lock, copies maps, releases, then signals `readyCh` outside the lock) keeps the same observable behavior — it still atomically replaces the legacy package-level vars — and `listener.added` (`listener/registry.go:137`) is emitted by `Registry.Add` independently.

### 3. FR-LGN-1 — atlas-login projection mirrors atlas-channel

**Status:** Both files stayed in lockstep on the rewrite. Pre-existing asymmetry remains.

The new error-return + `readyCh` blocking pattern is identical in:
- `services/atlas-channel/atlas.com/channel/configuration/registry.go`
- `services/atlas-login/atlas.com/login/configuration/registry.go`

Both define `ErrNotReady`, `ErrTenantNotConfigured`, `readyCh`, `readyOnce`, the 60s `readyTimeout`, and the same `waitReady()` helper. `PublishSnapshot` is byte-equivalent (modulo the comment noting which service's consumer set is affected).

Pre-existing asymmetry (not introduced by this commit):
- atlas-channel registry has `GetTenantConfigs()` (plural, no gate) for callers that want the whole map; atlas-login registry doesn't expose that. Confirmed `grep` shows no atlas-login caller needs it.
- **atlas-channel main.go does not call `PublishSnapshot`**, while atlas-login main.go does at line 130. Consequence: the `readyCh` in atlas-channel never closes during normal boot; any `Get*` call would hit the 60s timeout. The only in-tree caller — `services/atlas-channel/atlas.com/channel/session/task.go:24` — is not wired into atlas-channel main, so this is dead code today. The fix commit explicitly acknowledges this.

This is a latent bug surface, not a regression introduced by these fixes. If/when atlas-channel adds a session timeout task, main.go must also bridge `state.Snapshot()` → `configuration.PublishSnapshot` after `WaitCaughtUp` (the same shape as atlas-login `main.go:128-132`).

### 4. Sec 8.3 Observability — silent-failure window during boot

**Status:** Not silent. Callers already log on error. No new metric/log added, but no regression either.

The only production callers of `Get*` after the rewrite are:
- `services/atlas-login/atlas.com/login/kafka/consumer/account/session/consumer.go:88-92` — `GetTenantConfig`; on error, `l.WithError(err).Errorf("Unable to find server configuration.")` and the handler returns the err (the kafka manager logs handler errors).
- `services/atlas-login/atlas.com/login/socket/handler/accept_tos.go:45-49` — same pattern.
- `services/atlas-login/atlas.com/login/main.go:192` — boot-time `GetServiceConfig`; this runs after `PublishSnapshot` so it never blocks.

Each error site emits a `WithError(err).Errorf(...)` line, so during the boot window an operator does see `Unable to find server configuration` with the wrapped `ErrNotReady`/`ErrTenantNotConfigured` underneath. That is sufficient observability for triage, but the message text doesn't make the boot-window race itself obvious. Suggested follow-up (not blocking): include the sentinel error class in the log (e.g., `errors.Is(err, configuration.ErrNotReady)` → specific log) so operators don't have to grep the wrapped error text.

The previous behavior (`log.Fatalf`) was, of course, very loud — and the new behavior is correctly not-loud-and-not-crashing. The PRD's reliability requirement (Sec 8.5: "atlas-channel/atlas-login remain not-ready until catch-up succeeds. No crash-loop") is now actually upheld for the boot-race case.

### 5. Topic-compaction assumption

**Status:** Consistent with PRD design intent.

Per FR-CFG-1 and Sec 5.3, both topics are log-compacted (`cleanup.policy=compact`) with `delete.retention.ms >= 7 days`. Log compaction guarantees the latest record per key is retained indefinitely; only intermediate values for the same key are eligible for deletion. The projection (`configuration/projection/apply.go` + `state.go`) is keyed by service-id / tenant-id, so it only needs the latest per-key record to rebuild correct state. Per-process group IDs replaying from `FirstOffset` on every restart get exactly that: the compacted view of state.

Tombstones (FR-SCH-3) are handled in `subscriber.go:93-102, 125-138`. Compaction retains tombstones for `delete.retention.ms`, which is long enough that a freshly-booted pod observes any pending removal.

No regression. The design intent and the new behavior agree.

### 6. & 7. Build, vet, and tests

**`go build ./...`** — clean for both modules.

**`go vet ./...`**:
- `services/atlas-login/atlas.com/login` — reports the two disclosed pre-existing warnings:
  - `libs/atlas-rest/server/server.go:187:13: WaitGroup.Add called from inside new goroutine`
  - `socket/init.go:39:11: WaitGroup.Add called from inside new goroutine`
- `services/atlas-channel/atlas.com/channel` — reports only the disclosed `libs/atlas-rest/server/server.go:187` warning.

Both match the disclosed pre-existing-warning list. No new warnings introduced by the follow-up commits.

**`go test -race -count=1 ./...`**:

| Module | Result | Configuration/projection test results |
|---|---|---|
| `services/atlas-login/atlas.com/login` | PASS (all packages `ok`; no `FAIL` lines) | `atlas-login/configuration` 1.118s, `atlas-login/configuration/projection` 1.026s |
| `services/atlas-channel/atlas.com/channel` | PASS (all packages `ok`; no `FAIL` lines) | `atlas-channel/configuration` 1.119s, `atlas-channel/configuration/projection` 1.030s |

The new `TestGetServiceConfig_BlocksUntilPublishSnapshot` test in each module is in the green set — it asserts the precise behavior the fix was meant to deliver.

## Acceptance-Criteria Impact

| PRD Acceptance Criterion | Affected? | Status |
|---|---|---|
| atlas-channel boots with atlas-configurations unreachable and reaches ready once topic is caught up | Strengthened | Per-pod group is what actually makes this work on restart; gate logic unchanged. |
| /readyz returns not-ready before catch-up, ready after | No regression | Gate inputs (`ReadEndOffsets` + `Observe`) unaffected by group-id change. |
| Adding a tenant brings up new listener without restart | No regression | Apply loop and `Registry.Add` unaffected. |
| Removing a tenant triggers four-phase drain | No regression | Drain path unaffected; Evict fires on ref-count zero. |
| SIGTERM drains all listeners | No regression | `DrainAll` unaffected. |
| `Evict(t)` hook called when last listener drains | No regression | Verified — Evict chain decoupled from `PublishSnapshot`. |
| `go test -race ./...` passes | Holds | Both modules clean. |
| `go vet ./...` passes | Holds (modulo disclosed pre-existing) | No new warnings. |

## Risk Register

| # | Risk | Severity | Notes |
|---|---|---|---|
| 1 | atlas-channel's new `readyCh` never closes (main.go missing `PublishSnapshot` bridge) | Latent / low today | No production caller of `Get*` exists in atlas-channel. Becomes live if `session.NewTimeout` (or any other `Get*` caller) is wired into `main.go` without also wiring `PublishSnapshot`. Recommend a TODO/comment in atlas-channel `main.go` near the listener-bootstrap to mirror atlas-login's bridge. |
| 2 | Boot-window `GetTenantConfig` errors log generic "Unable to find server configuration" rather than naming the not-ready/race | Low | Functionality is correct; observability is adequate (`l.WithError(err).Errorf(...)` includes `ErrNotReady` in the wrapped error). Could be tightened by branching on `errors.Is(err, configuration.ErrNotReady)`. |
| 3 | Per-pod consumer groups create one new offset-storage group on the broker per pod start; broker may accumulate empty groups | Operational, low | kafka-go's default group-coordinator behavior eventually GCs idle groups (broker `offsets.retention.minutes`, default 7 days). Worth surfacing to operations if pod-restart cadence is high. |

## Overall Assessment

- **Plan Adherence:** FULL (the follow-ups address legitimate operational defects exposed by deploy verification; they do not violate plan tasks).
- **Recommendation:** READY_TO_MERGE.

## Action Items (non-blocking)

1. Optionally add a brief comment in atlas-channel `main.go` immediately after `WaitCaughtUp` succeeds noting that if any code in this service ever calls `configuration.GetServiceConfig`/`GetTenantConfig`, main.go must also call `configuration.PublishSnapshot(state.Snapshot())` here — mirroring atlas-login `main.go:128-132`. Today no caller exists, but the gap is easy to forget.
2. Optionally enhance the boot-window log sites (`kafka/consumer/account/session/consumer.go:90` and `socket/handler/accept_tos.go:47`) to special-case `errors.Is(err, configuration.ErrNotReady)` with a clearer "boot window, retry on next event" message.
3. Track per-pod projection consumer-group accumulation as an ops observation; confirm broker-side group GC is enabled at default retention.
