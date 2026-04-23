# Kafka At-Least-Once Delivery — Context

Last Updated: 2026-02-19

## Key Files

### Library (all changes here)

| File | Purpose |
|------|---------|
| `libs/atlas-kafka/consumer/manager.go` | Consumer loop, `KafkaReader`/`MessageReader` interfaces, `Consumer.start()` |
| `libs/atlas-kafka/consumer/config.go` | `Config` struct, `NewConfig()`, decorators |
| `libs/atlas-kafka/consumer/header.go` | `HeaderParser` type, `TenantHeaderParser`, `SpanHeaderParser` |
| `libs/atlas-kafka/consumer/manager_test.go` | `MockReader`, `ChannelMockReader`, existing tests |
| `libs/atlas-kafka/handler/handler.go` | `Handler` type: `func(logrus.FieldLogger, context.Context, kafka.Message) (bool, error)` |
| `libs/atlas-kafka/message/handler.go` | `PersistentConfig`, `OneTimeConfig`, `AdaptHandler` |
| `libs/atlas-kafka/retry/retry.go` | `Try()` — retry loop with 1s sleep |
| `libs/atlas-kafka/go.mod` | Module `github.com/Chronicle20/atlas-kafka`, depends on `segmentio/kafka-go v0.4.50` |

### Upstream (read-only reference)

| File | Purpose |
|------|---------|
| `segmentio/kafka-go@v0.4.50/reader.go:792` | `ReadMessage()` = `FetchMessage()` + `CommitMessages()` |
| `segmentio/kafka-go@v0.4.50/reader.go:815` | `FetchMessage()` — fetches without committing |
| `segmentio/kafka-go@v0.4.50/reader.go:878` | `CommitMessages()` — explicit commit |

### Service Consumers (rebuild required, no code changes)

All 48 services follow the same pattern:
```
services/<name>/atlas.com/<module>/kafka/consumer/<domain>/consumer.go
```
Each has `InitConsumers()` (registers reader) and `InitHandlers()` (registers handlers).

### OneTimeConfig Usages (special attention)

| File | Line | Pattern |
|------|------|---------|
| `services/atlas-consumables/atlas.com/consumables/consumable/processor.go` | 184, 454 | Request-reply with transaction ID validator |
| `services/atlas-character-factory/atlas.com/character-factory/kafka/consumer/character/consumer.go` | 84 | Self-re-registering OneTimeConfig |

## Key Decisions

### 1. Sequential vs Concurrent Message Processing

**Decision**: Sequential (process one message fully before fetching the next)

**Rationale**:
- The current concurrent model (fire-and-forget goroutines) is incompatible with at-least-once delivery because you cannot know when to commit
- Sequential processing is simpler, correct, and sufficient for current throughput needs
- Each consumer reads from one topic with one partition typically — concurrency within a single partition adds complexity without benefit
- If throughput becomes an issue, batch commit optimization can be added later

### 2. Handler Error Semantics

**Decision**: Any handler error prevents commit (message will be redelivered)

**Rationale**:
- At-least-once means "process at least once successfully"
- If a handler fails, the message should be retried
- Handlers that return errors are already rare (most log and continue)
- A handler that consistently fails will block the consumer — this is detectable and preferable to silent message loss

### 3. Interface Change Strategy

**Decision**: Extend `KafkaReader` interface additively

**Rationale**:
- Adding `FetchMessage()` and `CommitMessages()` to the interface
- `kafka.Reader` already implements both — default `ReaderProducer` needs no changes
- Service test mocks that implement `KafkaReader` will need updating, but this is mechanical

### 4. ReadMessage Removal

**Decision**: Remove `ReadMessage` from interface entirely

**Rationale**:
- No code should use `ReadMessage()` after this change — it defeats the purpose
- Keeping it in the interface invites accidental use
- Clean break is better than deprecation

## Dependencies

### Internal
- No dependency on other active work items
- Can proceed independently of Redis registry migration

### External
- `segmentio/kafka-go v0.4.50` — already in use, provides `FetchMessage()` and `CommitMessages()`

## Consumer Group Behavior Notes

### Standard Groups (46 services)
- Static group ID (e.g., `"Account Service"`)
- Kafka assigns partitions across instances
- At-least-once delivery works naturally: uncommitted offset → redelivery to same consumer group

### Broadcast Groups (2 services: atlas-channel, atlas-login)
- Per-instance group ID (template with UUID)
- Every instance processes every message
- At-least-once delivery still applies per-instance: each instance's offset is tracked independently

## Testing Strategy

### Mock Updates Required
- `MockReader` in `manager_test.go` needs `FetchMessage()` and `CommitMessages()`
- `ChannelMockReader` in `manager_test.go` needs `FetchMessage()` and `CommitMessages()`
- Search all services for custom `KafkaReader` mocks (unlikely but possible)

### Key Test Scenarios
1. Happy path: message fetched → handlers complete → commit called
2. Handler error: message fetched → handler fails → NO commit → message available for refetch
3. Handler panic: consumer survives, message not committed
4. Multiple handlers: all must complete before commit
5. Graceful shutdown: in-progress handler completes, pending message committed
6. OneTimeConfig: handler removed after match, commit still happens
