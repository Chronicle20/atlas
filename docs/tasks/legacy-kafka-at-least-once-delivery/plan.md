# Kafka At-Least-Once Delivery

Last Updated: 2026-02-19

## Executive Summary

All 48 Atlas Kafka consumers use `ReadMessage()` from `segmentio/kafka-go`, which auto-commits the offset **before** the message handler runs. If a consumer crashes mid-processing, that message is permanently lost (at-most-once delivery). This plan switches to `FetchMessage()` + explicit `CommitMessages()` after successful handler execution, giving at-least-once delivery. The change is concentrated in `libs/atlas-kafka` — no individual service code needs modification.

## Current State Analysis

### The Problem (Confirmed via Source)

`segmentio/kafka-go` v0.4.50 `Reader.ReadMessage()` is literally:
```go
func (r *Reader) ReadMessage(ctx context.Context) (Message, error) {
    m, err := r.FetchMessage(ctx)
    if err != nil {
        return Message{}, fmt.Errorf("fetching message: %w", err)
    }
    if r.useConsumerGroup() {
        if err := r.CommitMessages(ctx, m); err != nil {
            return Message{}, fmt.Errorf("committing message: %w", err)
        }
    }
    return m, nil
}
```

The offset is committed the instant the message is fetched. The consumer loop in `manager.go:170` then dispatches the message to handlers in goroutines. If any handler crashes or the process dies, the message is gone.

### Current Architecture

**Library**: `libs/atlas-kafka/consumer/manager.go`
- `KafkaReader` interface exposes only `ReadMessage()` + `Close()`
- Consumer loop: retry `ReadMessage()` up to 10 times → spawn goroutine per message → spawn goroutine per handler
- Handlers are fire-and-forget: the consumer reads the next message immediately without waiting for handlers to complete
- 48 services, ~166 consumer files, ~531 handler registrations

**Handler Types**:
- `PersistentConfig` — stays registered, processes every message (vast majority)
- `OneTimeConfig` — auto-removes after first match (3 usages: atlas-consumables, atlas-character-factory)

**Consumer Group Patterns**:
- Standard: static group ID (e.g., `"Account Service"`) — partition-based load balancing
- Broadcast: per-instance group ID (atlas-channel, atlas-login) — every instance gets every message

### Key Constraint: Fire-and-Forget Handler Dispatch

The current architecture dispatches handlers in separate goroutines and immediately reads the next message. Switching to at-least-once requires the consumer loop to **wait for handlers to complete** before committing. This is the fundamental architectural change.

## Proposed Future State

### Design: Sequential Processing with Post-Handler Commit

```
loop:
    msg := FetchMessage()           // does NOT commit
    ctx := parseHeaders(msg)
    runAllHandlers(ctx, msg)        // BLOCKS until all handlers complete
    CommitMessages(msg)             // commit only after success
```

**Key changes**:
1. `KafkaReader` interface gains `FetchMessage()` and `CommitMessages()`
2. Consumer loop processes one message at a time (per partition) — handlers must complete before the next message is read
3. Handler errors are surfaced and prevent commit (the message will be redelivered)
4. `handlerWg` moves from "track for shutdown" to "gate for commit"

### Why Sequential (Not Batched)

- Simplicity: one message in, process, commit, repeat
- No reorder risk: handlers see messages in partition order
- Throughput is sufficient: each consumer reads from one topic, handlers are typically fast (Redis/HTTP calls)
- Batch optimization can be added later if profiling shows it's needed

### OneTimeConfig Compatibility

OneTimeConfig handlers (3 usages) register a handler that auto-removes after one match. Under at-least-once delivery, a redelivered message could re-trigger a one-time handler that already fired. However:
- `atlas-consumables` uses OneTimeConfig for request-reply correlation with validators that match on specific transaction IDs — a redelivered message would not match because the handler is already gone
- `atlas-character-factory` re-registers itself after each match
- These patterns are safe under at-least-once delivery without changes

## Implementation Phases

### Phase 1: Extend KafkaReader Interface (libs/atlas-kafka)

**Scope**: `libs/atlas-kafka/consumer/manager.go`

Expand the `KafkaReader` and `MessageReader` interfaces to support `FetchMessage()` + `CommitMessages()`. The `kafka.Reader` from segmentio already implements both methods, so the default `ReaderProducer` (which calls `kafka.NewReader()`) works without changes.

#### Tasks

1. **Add FetchMessage and CommitMessages to interfaces** [S]
   - Add `FetchMessage(ctx context.Context) (kafka.Message, error)` to `MessageReader`
   - Add `CommitMessages(ctx context.Context, msgs ...kafka.Message) error` to `KafkaReader`
   - `kafka.Reader` already satisfies both — no adapter needed
   - Acceptance: interfaces compile, `kafka.NewReader()` satisfies them

2. **Replace ReadMessage with FetchMessage in consumer loop** [M]
   - In `start()`, change `c.reader.ReadMessage(readerCtx)` to `c.reader.FetchMessage(readerCtx)`
   - Acceptance: messages are fetched but NOT auto-committed

3. **Make handler dispatch synchronous** [M]
   - Remove the outer goroutine that wraps handler execution (line 192)
   - Run all handlers for a message, wait for all to complete
   - Collect handler errors — if any handler returns an error, do NOT commit
   - After all handlers succeed, call `c.reader.CommitMessages(readerCtx, msg)`
   - Acceptance: message is only committed after all handlers have completed successfully

4. **Add commit error handling** [S]
   - If `CommitMessages()` fails, log the error and continue (the message will be redelivered on next fetch, which is safe under at-least-once)
   - Acceptance: commit failures are logged, consumer continues

5. **Handle handler panics** [S]
   - Wrap handler execution in recover() to prevent a panicking handler from killing the consumer loop
   - A panic should be treated like an error — do not commit the message
   - Acceptance: panicking handler does not crash consumer, message is redelivered

### Phase 2: Update Tests (libs/atlas-kafka)

**Scope**: `libs/atlas-kafka/consumer/manager_test.go`

#### Tasks

6. **Update MockReader to implement new interface** [M]
   - Add `FetchMessage()` and `CommitMessages()` to `MockReader` and `ChannelMockReader`
   - Track committed messages in a slice for test assertions
   - Remove `ReadMessage()` from mock (or keep as wrapper for backward compat if external mocks exist)
   - Acceptance: all existing tests pass with updated mocks

7. **Add test: message committed only after handler completes** [M]
   - Verify that `CommitMessages()` is called after handler execution, not before
   - Use a slow handler and check commit timing
   - Acceptance: test passes, demonstrates at-least-once guarantee

8. **Add test: handler error prevents commit** [M]
   - Register a handler that returns an error
   - Verify `CommitMessages()` is NOT called for that message
   - Acceptance: test passes, uncommitted message would be redelivered

9. **Add test: handler panic prevents commit** [S]
   - Register a handler that panics
   - Verify consumer continues and message is not committed
   - Acceptance: test passes, consumer survives panic

10. **Add test: multiple handlers all complete before commit** [M]
    - Register multiple handlers for the same topic
    - Verify commit happens only after ALL handlers complete
    - Acceptance: test passes

11. **Verify existing tests still pass** [S]
    - `TestGracefulShutdown`, `TestSpanPropagation`, `TestTenantPropagation`
    - Acceptance: `go test ./... -count=1` passes in `libs/atlas-kafka`

### Phase 3: Build Validation (All Services)

**Scope**: All 48 services

#### Tasks

12. **Build all services** [M]
    - Since `KafkaReader` interface changed, all services using atlas-kafka must build
    - Any service with a custom `KafkaReader` mock in tests must be updated
    - Run `go build` across all services in the workspace
    - Acceptance: `go build ./...` succeeds from workspace root

13. **Fix any broken service tests** [S-L depending on count]
    - Services with custom `MockReader` implementations in tests need FetchMessage/CommitMessages added
    - Acceptance: `go test ./...` passes across workspace

### Phase 4: Idempotency Audit (Documentation Only)

Switching to at-least-once means handlers may see the same message twice. Most handlers are naturally idempotent (upsert semantics, Redis SET operations) or tolerate duplicates. This phase documents which handlers need attention.

#### Tasks

14. **Audit handler idempotency across services** [L]
    - Categorize each service's handlers as:
      - **Naturally idempotent**: SET/upsert operations, cache writes, state machine transitions
      - **Tolerates duplicates**: operations that are safe to repeat (e.g., re-emitting an event that downstream consumers handle idempotently)
      - **Needs attention**: operations that are NOT idempotent (e.g., incrementing counters, appending to lists, creating non-deduplicated records)
    - Document findings in this plan
    - Acceptance: all ~531 handler registrations categorized

15. **Add idempotency guards where needed** [M-XL depending on audit]
    - For handlers identified as non-idempotent, add guards (e.g., check-before-write, transaction IDs, deduplication)
    - This is a follow-up task driven by the audit results
    - Acceptance: all handlers are safe under at-least-once delivery

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Throughput regression from sequential processing | Medium | Low | Profile before/after. Current handlers are fast (Redis/HTTP). Batch optimization available if needed |
| Handler that blocks indefinitely stalls consumer | Low | High | Add per-handler timeout (configurable, default 30s). Log and skip on timeout |
| Duplicate processing causes side effects | Medium | Medium | Phase 4 audit identifies non-idempotent handlers. Most are upserts/cache writes |
| Breaking change for services with custom mocks | Low | Low | Interface addition is backward-compatible if mocks embed `kafka.Reader`. Search-and-fix in Phase 3 |
| OneTimeConfig handlers fire twice on redeliver | Low | Low | Validators match on transaction IDs — stale handlers are already removed |

## Success Metrics

1. **Zero message loss**: under consumer restart/crash, no messages are permanently skipped
2. **All tests pass**: lib and service tests pass with new consumer loop
3. **No throughput regression**: message processing latency stays within 2x of current (measured per-consumer)
4. **Idempotency documented**: every handler categorized for duplicate-message safety

## Required Resources and Dependencies

- **segmentio/kafka-go v0.4.50**: already in use, `FetchMessage()` and `CommitMessages()` are available
- **No new dependencies**: this is a refactor of existing code
- **No infrastructure changes**: Kafka brokers, topics, consumer groups unchanged
- **No service code changes**: all changes in `libs/atlas-kafka` (Phase 1-2), services only need rebuild (Phase 3)

## Timeline Estimates

| Phase | Effort | Notes |
|-------|--------|-------|
| Phase 1: Interface + Consumer Loop | M | Core change, ~100 lines modified |
| Phase 2: Tests | M | 5-6 new tests, mock updates |
| Phase 3: Build Validation | S-M | Mechanical mock updates if needed |
| Phase 4: Idempotency Audit | L | 48 services, ~531 handlers to review |
