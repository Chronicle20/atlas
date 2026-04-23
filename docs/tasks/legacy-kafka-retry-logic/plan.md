# Kafka Retry Logic: Exponential Backoff with Jitter

**Last Updated: 2026-02-19**
**Priority: Low**
**Branch: TBD**

---

## Executive Summary

The shared Kafka retry logic (`libs/atlas-kafka/retry/retry.go`) uses a fixed 1-second sleep between attempts with no exponential backoff or jitter. This creates thundering herd risk during broker failures and provides suboptimal recovery characteristics. Additionally, ~25 services contain copy-pasted local retry packages (used for DB connection retries) that duplicate this same flawed pattern. The atlas-marriages service already has a well-tested `ExecuteWithRetry` implementation with exponential backoff, context awareness, and error classification that can serve as a reference.

This plan upgrades the shared retry library, consolidates duplicate retry packages across services, and configures appropriate retry strategies for different failure domains.

---

## Current State Analysis

### Shared Library (`libs/atlas-kafka/retry/retry.go`)

```go
func Try(fn RepeatableFunc, retries int) error {
    attempt := 1
    for {
        cont, err := fn(attempt)
        if !cont || err == nil {
            return err
        }
        attempt++
        if attempt > retries {
            return errors.New("max retry reached")  // original error lost
        }
        time.Sleep(1 * time.Second)  // fixed, no backoff, no jitter
    }
}
```

### Problems

| Problem | Impact |
|---------|--------|
| Fixed 1s sleep | Thundering herd on broker recovery; unnecessary delay for fast failures |
| No jitter | All consumers/producers retry at identical intervals |
| Original error swallowed | `errors.New("max retry reached")` loses root cause |
| No error classification | Non-retryable errors (auth failures, bad data) waste retry budget |
| ~25 copy-pasted retry packages | Maintenance burden; fixes don't propagate |
| `RepeatableFuncWithResponse` | Dead code in ~15 service retry packages |
| REST defaults to 1 attempt | No retry for HTTP calls; no service overrides this |

### Existing Patterns in Codebase

| Location | Strategy | Notes |
|----------|----------|-------|
| `libs/atlas-kafka/retry` | Fixed 1s | Shared library (consumer + producer) |
| `libs/atlas-rest/retry` | Fixed 1s | HTTP client retry |
| `services/atlas-maps/retry` | Linear backoff | `attempt * 1s` |
| `services/atlas-marriages/retry` | Exponential + jitter | Full implementation with config builder |
| `services/atlas-saga-orchestrator` | Fixed 10ms | Optimistic lock conflicts only |

---

## Proposed Future State

### Shared Retry Library (`libs/atlas-kafka/retry`)

A single, configurable retry function with:

1. **Exponential backoff** with configurable base delay and multiplier
2. **Full jitter** (randomize between 0 and calculated delay) to decorrelate retries
3. **Maximum delay cap** to bound worst-case wait times
4. **Error wrapping** that preserves the original error
5. **Context awareness** for clean shutdown during retry waits
6. **Backward-compatible `Try()` signature** so existing callers still compile

### Default Configurations by Domain

| Domain | Max Retries | Initial Delay | Max Delay | Backoff Factor |
|--------|-------------|---------------|-----------|----------------|
| Kafka consumer (fetch) | 10 | 100ms | 10s | 2.0 |
| Kafka producer (write) | 10 | 100ms | 10s | 2.0 |
| DB connection (startup) | 10 | 500ms | 30s | 2.0 |
| REST HTTP requests | 3 | 200ms | 5s | 2.0 |

### Service Consolidation

All ~25 service-local retry packages will be removed. Services will import the shared library directly for DB connection retries, or use the pattern already established in the Kafka consumer/producer code paths.

---

## Implementation Phases

### Phase 1: Upgrade Shared Library

Enhance `libs/atlas-kafka/retry/retry.go` with a new `RetryConfig`-based API while keeping the legacy `Try()` function as a backward-compatible wrapper.

### Phase 2: Upgrade Kafka Consumer & Producer

Update `libs/atlas-kafka/consumer/manager.go` and `libs/atlas-kafka/producer/producer.go` to use the new backoff-aware retry.

### Phase 3: Upgrade REST Client

Update `libs/atlas-rest/retry/retry.go` to use the same pattern. Configure reasonable defaults for HTTP retries.

### Phase 4: Consolidate Service Retry Packages

Remove copy-pasted `retry/retry.go` from all ~25 services. Replace DB connection retry calls with the shared library.

### Phase 5: Clean Up

Remove dead code (`RepeatableFuncWithResponse`), remove the atlas-marriages local retry (now superseded by shared library), update atlas-maps to use shared library.

---

## Detailed Tasks

### Phase 1: Upgrade Shared Library

**1.1 Add RetryConfig and ExecuteWithRetry to `libs/atlas-kafka/retry`**
- Effort: M
- Add `RetryConfig` struct with fields: `MaxRetries`, `InitialDelay`, `MaxDelay`, `BackoffFactor`
- Add builder methods: `WithMaxRetries()`, `WithInitialDelay()`, `WithMaxDelay()`, `WithBackoffFactor()`
- Add `DefaultConfig()` returning sensible defaults (3 retries, 100ms initial, 10s max, 2.0 factor)
- Implement `ExecuteWithRetry(ctx, config, operation)` with:
  - Exponential backoff: `delay = initialDelay * backoffFactor^(attempt-1)`
  - Full jitter: `actualDelay = rand(0, calculatedDelay)`
  - Max delay cap
  - Context-aware sleep (select on ctx.Done() and time.After)
  - Error wrapping: `fmt.Errorf("after %d attempts, last error: %w", attempts, lastErr)`
- Acceptance: Function computes correct delays; jitter randomizes; context cancellation interrupts sleep; original error preserved via `%w`

**1.2 Make legacy `Try()` a wrapper around `ExecuteWithRetry`**
- Effort: S
- `Try()` wraps the callback into the new API shape and calls `ExecuteWithRetry` with `context.Background()` and a config matching current behavior (fixed 1s delay, no jitter) to preserve exact backward compatibility during transition
- Acceptance: Existing `Try()` tests still pass with no changes

**1.3 Add comprehensive tests**
- Effort: M
- Test exponential delay calculation (mock time or measure bounds)
- Test jitter stays within [0, calculatedDelay]
- Test max delay cap
- Test context cancellation during retry
- Test error wrapping preserves original error
- Test zero-retry config (execute once, no retry)
- Acceptance: All tests pass; coverage for all backoff/jitter/context paths

### Phase 2: Upgrade Kafka Consumer & Producer

**2.1 Update consumer fetch retry**
- Effort: S
- File: `libs/atlas-kafka/consumer/manager.go`
- Replace `retry.Try(readerFunc, 10)` with `retry.ExecuteWithRetry(ctx, config, readerFunc)`
- Use consumer config: 10 retries, 100ms initial, 10s max, 2.0 factor
- Pass the consumer's context so shutdown cancels retry waits
- Acceptance: Consumer retries with backoff; clean shutdown interrupts retry sleep

**2.2 Update producer write retry**
- Effort: S
- File: `libs/atlas-kafka/producer/producer.go`
- Replace `retry.Try(tryMessage(...), 10)` with `retry.ExecuteWithRetry(ctx, config, ...)`
- Use producer config: 10 retries, 100ms initial, 10s max, 2.0 factor
- Pass a proper context (not `context.Background()`) so shutdown cancels retry waits
- Acceptance: Producer retries with backoff; context cancellation stops retries

**2.3 Update tests**
- Effort: S
- Update any existing consumer/producer tests that mock or assert retry behavior
- Acceptance: All `libs/atlas-kafka` tests pass

### Phase 3: Upgrade REST Client

**3.1 Replace `libs/atlas-rest/retry` with shared library import**
- Effort: S
- Option A: Make `libs/atlas-rest` depend on `libs/atlas-kafka/retry` (creates coupling)
- Option B: Extract retry into its own `libs/atlas-retry` module (cleaner, but more work)
- Option C: Copy the upgraded retry logic into `libs/atlas-rest/retry` (simple, minor duplication)
- **Recommended: Option B** — extract to `libs/atlas-retry` so both kafka and rest can import it
- Acceptance: REST retry uses exponential backoff with jitter

**3.2 Configure REST default retry count**
- Effort: S
- Change default from 1 to 3 for GET requests (idempotent)
- Keep default at 1 for POST/DELETE (non-idempotent) unless explicitly configured
- Acceptance: GET requests retry 3 times with backoff; POST/DELETE still single-attempt by default

### Phase 4: Consolidate Service Retry Packages

**4.1 Identify all service-local retry packages**
- Effort: S
- Services with local retry: atlas-account, atlas-character, atlas-inventory, atlas-buddies, atlas-ban, atlas-tenants, atlas-data, atlas-configurations, atlas-guilds, atlas-families, atlas-fame, atlas-keys, atlas-notes, atlas-npc-shops, atlas-npc-conversations, atlas-portal-actions, atlas-reactor-actions, atlas-storage, atlas-quest, atlas-cashshop, atlas-map-actions, atlas-party-quests, atlas-gachapons, atlas-drop-information, atlas-skills, atlas-pets, atlas-saga-orchestrator
- Acceptance: Complete list verified against codebase

**4.2 Replace DB connection retry in each service**
- Effort: L (bulk, ~25 services)
- In each service's `database/connection.go`:
  - Remove import of local `retry` package
  - Import shared `retry` from `libs/atlas-retry` (or `libs/atlas-kafka/retry`)
  - Use DB startup config: 10 retries, 500ms initial, 30s max, 2.0 factor
- Acceptance: Each service builds and tests pass; DB connection retry uses backoff

**4.3 Delete local retry packages**
- Effort: S
- Remove `retry/retry.go` and `retry/retry_test.go` from each service
- Remove `RepeatableFuncWithResponse` dead code
- Acceptance: No service-local retry packages remain (except atlas-marriages Phase 5, atlas-maps Phase 5)

### Phase 5: Clean Up

**5.1 Remove atlas-marriages local retry**
- Effort: S
- Replace `ExecuteWithRetry` calls in `scheduler/proposal_expiry.go` and `scheduler/ceremony_timeout.go` with shared library
- Delete local retry package
- Acceptance: atlas-marriages builds and tests pass using shared retry

**5.2 Remove atlas-maps local retry**
- Effort: S
- Replace linear backoff usage with shared library exponential backoff
- Delete local retry package
- Acceptance: atlas-maps builds and tests pass using shared retry

**5.3 Verify no remaining local retry packages**
- Effort: S
- `grep -r "retry.Try" services/` confirms all imports point to shared library
- Acceptance: Zero service-local retry packages in codebase

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Behavioral change in retry timing breaks assumptions | Low | Medium | Phase 1 keeps `Try()` backward-compatible; phases 2-4 are explicit opt-in |
| Jitter makes tests non-deterministic | Medium | Low | Inject rand source or test delay bounds rather than exact values |
| Extracting `libs/atlas-retry` breaks go.work | Low | Low | Add to `go.work` and verify all services resolve |
| Bulk changes to 25 services introduce bugs | Low | Medium | Each service gets independent build+test verification |
| Longer backoff delays slow down startup | Low | Low | DB startup config uses 500ms initial which is already faster than current 1s fixed for first few attempts |

---

## Success Metrics

1. **Zero copy-pasted retry packages** — all services use shared library
2. **Exponential backoff with jitter** active on all Kafka consumer/producer paths
3. **Original errors preserved** — no more "max retry reached" without root cause
4. **All services build and pass tests** after migration
5. **Context-aware retries** — shutdown signals interrupt retry waits within one sleep cycle

---

## Required Resources and Dependencies

- No external dependencies; all changes are internal
- atlas-marriages `ExecuteWithRetry` serves as proven reference implementation
- If Option B (extract `libs/atlas-retry`): need to create new Go module and update `go.work`

---

## Timeline Estimates

| Phase | Effort | Dependencies |
|-------|--------|-------------|
| Phase 1: Upgrade Shared Library | M | None |
| Phase 2: Kafka Consumer & Producer | S | Phase 1 |
| Phase 3: REST Client | M | Phase 1 |
| Phase 4: Consolidate Services | L | Phase 1 |
| Phase 5: Clean Up | S | Phases 2, 3, 4 |

Phases 2, 3, and 4 can proceed in parallel after Phase 1 completes.
