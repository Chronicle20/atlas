# Saga Orchestrator Durability — Implementation Plan

Last Updated: 2026-02-19

## Executive Summary

Replace the in-memory saga store (`InMemoryCache`) in `atlas-saga-orchestrator` with a PostgreSQL-backed persistence layer. This eliminates the critical data loss risk on service restart, fixes the read-modify-write race condition, and adds timeout/reaper infrastructure for stuck sagas. The implementation preserves the existing `Cache` interface contract, requiring zero changes to the 14 Kafka consumers, the REST API, the compensator, or the 60+ action handlers.

## Current State Analysis

### Architecture
- **Store**: `saga/cache.go` — singleton `InMemoryCache` with `sync.RWMutex`, keyed `map[uuid.UUID]map[uuid.UUID]Saga` (tenantId → transactionId → Saga)
- **Interface**: `Cache` with 4 methods: `GetAll`, `GetById`, `Put`, `Remove`
- **Processor**: `saga/processor.go` — all mutation flows through `GetById` → mutate → `Put` (non-atomic across calls)
- **Model**: `saga/model.go` — immutable `Saga` + `Step[any]` with custom JSON marshaling and 70+ action-typed payloads
- **Consumers**: 14 Kafka consumer packages, all call `processor.StepCompleted(txnId, success)`
- **No database**: go.mod has zero database drivers; service is purely in-memory + Kafka + REST

### Problems
1. **Durability**: Service restart loses all in-flight sagas with no recovery
2. **Race condition**: Concurrent Kafka consumers can interleave `GetById`/`Put` for the same saga — the `RWMutex` protects individual operations but not the read-modify-write sequence
3. **Memory leak**: No TTL, timeout, or reaper — a saga waiting for a Kafka response that never arrives will persist forever
4. **No idempotency**: Duplicate Kafka events can advance a saga twice

## Proposed Future State

### Database Schema
```sql
CREATE TABLE sagas (
    transaction_id UUID PRIMARY KEY,
    tenant_id      UUID NOT NULL,
    saga_type      VARCHAR(64) NOT NULL,
    initiated_by   VARCHAR(255) NOT NULL,
    status         VARCHAR(16) NOT NULL DEFAULT 'active',  -- active, completed, failed, compensating
    saga_data      JSONB NOT NULL,                         -- full Saga JSON (steps + payloads)
    version        INTEGER NOT NULL DEFAULT 1,             -- optimistic locking
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    timeout_at     TIMESTAMPTZ,                            -- per-saga deadline

    CONSTRAINT sagas_status_check CHECK (status IN ('active', 'completed', 'failed', 'compensating'))
);

CREATE INDEX idx_sagas_tenant_id ON sagas(tenant_id);
CREATE INDEX idx_sagas_status ON sagas(status) WHERE status = 'active';
CREATE INDEX idx_sagas_timeout ON sagas(timeout_at) WHERE status = 'active' AND timeout_at IS NOT NULL;
CREATE INDEX idx_sagas_created_at ON sagas(created_at) WHERE status = 'active';
```

### Key Design Decisions

1. **JSONB for saga data**: The `Saga` struct already has complete JSON marshaling/unmarshaling. Storing the entire saga (including all steps and their typed payloads) as a single JSONB column avoids a complex normalized schema for 70+ payload types. PostgreSQL JSONB supports indexing if needed later.

2. **Optimistic locking via `version` column**: Each `Put` increments `version` and includes `WHERE version = ?` in the UPDATE. If two concurrent consumers race, one gets a version conflict and retries (the saga's state will reflect the other consumer's update on re-read). This replaces the broken `sync.RWMutex` approach.

3. **`SELECT ... FOR UPDATE` for `AtomicUpdateSaga`**: The existing `AtomicUpdateSaga` method (used for step expansion) will use a real database transaction with row-level locking.

4. **Keep the `Cache` interface**: `PostgresStore` implements `Cache` — all 14 consumers, the REST layer, and the compensator remain untouched.

5. **Saga lifecycle status in a top-level column**: Enables efficient queries for the reaper (`WHERE status = 'active' AND timeout_at < NOW()`) without parsing JSONB.

## Implementation Phases

### Phase 1: PostgreSQL Store (Core)
Replace `InMemoryCache` with `PostgresStore` implementing the same `Cache` interface.

### Phase 2: Optimistic Locking
Add version-based conflict detection to prevent concurrent step completions from corrupting state.

### Phase 3: Startup Recovery
On service start, load all active sagas and re-drive them through `Step()` to resume processing.

### Phase 4: Timeout & Reaper
Add configurable timeouts and a background goroutine that compensates abandoned sagas.

### Phase 5: Idempotent Step Completion
Prevent duplicate Kafka events from advancing a saga more than once per step.

---

## Detailed Tasks

### Phase 1: PostgreSQL Store

**1.1 Add database dependencies to go.mod** [S]
- Add `gorm.io/gorm`, `gorm.io/driver/postgres` to the service go.mod
- Add `github.com/lib/pq` if needed for raw driver
- Run `go mod tidy`
- **Acceptance**: `go build` succeeds with new dependencies

**1.2 Create GORM entity** [M]
- New file: `saga/entity.go`
- Define `entity` struct with GORM tags mapping to the `sagas` table
- Include `TransactionId`, `TenantId`, `SagaType`, `InitiatedBy`, `Status`, `SagaData` (JSONB), `Version`, `CreatedAt`, `UpdatedAt`, `TimeoutAt`
- Add `TableName()` method returning `"sagas"`
- **Acceptance**: Entity compiles, tags match schema

**1.3 Create database migration / auto-migrate** [S]
- In `main.go`, add GORM `db.AutoMigrate(&entity{})` during startup
- Create indexes via GORM tags or raw SQL migration
- **Acceptance**: Service starts and creates the `sagas` table with correct schema

**1.4 Implement `PostgresStore`** [L]
- New file: `saga/store.go`
- Implement `Cache` interface:
  - `GetAll(tenantId)` → `SELECT * FROM sagas WHERE tenant_id = ? AND status = 'active'`
  - `GetById(tenantId, transactionId)` → `SELECT * FROM sagas WHERE transaction_id = ? AND tenant_id = ?`
  - `Put(tenantId, saga)` → UPSERT: INSERT on conflict UPDATE, serialize saga via `json.Marshal`
  - `Remove(tenantId, transactionId)` → `UPDATE sagas SET status = 'completed' WHERE ...` (soft delete, not hard delete — preserves audit trail)
- Deserialize `saga_data` back to `Saga` via `json.Unmarshal` on reads
- **Acceptance**: All 4 Cache methods work with PostgreSQL; existing unit tests pass after swapping implementation

**1.5 Wire `PostgresStore` into service startup** [S]
- In `main.go`, initialize GORM connection using env vars (`DATABASE_URL` or `DB_HOST`/`DB_PORT`/`DB_USER`/`DB_PASSWORD`/`DB_NAME`)
- Replace `GetCache()` singleton with `PostgresStore` instance
- Update `GetCache()` to return the PostgreSQL-backed implementation
- **Acceptance**: Service starts, connects to PostgreSQL, saga CRUD works end-to-end

**1.6 Update Docker/deployment configuration** [S]
- Add PostgreSQL connection env vars to `atlas-saga-orchestrator.yml`
- Add PostgreSQL service dependency to Docker Compose / deployment
- **Acceptance**: Service deploys with PostgreSQL connectivity

### Phase 2: Optimistic Locking

**2.1 Add version tracking to `Put`** [M]
- On `Put`: if saga exists, `UPDATE ... SET version = version + 1 WHERE version = ?`
- If `RowsAffected == 0`, return a `VersionConflictError`
- On INSERT (new saga), set `version = 1`
- **Acceptance**: Concurrent `Put` calls for the same saga — one succeeds, one gets conflict error

**2.2 Add retry logic to processor** [M]
- In `StepCompletedWithResult`, `MarkEarliestPendingStep`, `MarkFurthestCompletedStepFailed`: wrap the `GetById` → mutate → `Put` sequence in a retry loop (max 3 attempts)
- On `VersionConflictError`: re-read saga from DB and re-apply the mutation
- **Acceptance**: Under concurrent load, no saga state corruption; all conflicts resolved via retry

**2.3 Convert `AtomicUpdateSaga` to real DB transaction** [M]
- Replace the current `GetById` → modify → `Put` pattern with `SELECT ... FOR UPDATE` inside a GORM transaction
- The `updateFunc` runs within the transaction; commit writes the updated saga
- **Acceptance**: Step expansion (TransferToStorage, etc.) is fully atomic

### Phase 3: Startup Recovery

**3.1 Load active sagas on startup** [M]
- After DB connection and consumer registration, query `SELECT * FROM sagas WHERE status IN ('active', 'compensating')`
- For each saga, call `processor.Step(saga.TransactionId())` to re-drive the state machine
- Add a startup flag/env var to control recovery behavior (enabled by default)
- **Acceptance**: Service restart resumes all in-flight sagas; saga that was mid-step re-executes the current step's handler

**3.2 Handle startup recovery edge cases** [M]
- Idempotent handlers: handlers that send Kafka commands are already fire-and-forget — re-sending a command is safe (downstream services should handle duplicates)
- Log clearly which sagas are being recovered and their current step
- **Acceptance**: Recovery doesn't cause duplicate side effects or panic on any saga type

### Phase 4: Timeout & Reaper

**4.1 Add configurable saga timeout** [S]
- Add `SAGA_DEFAULT_TIMEOUT` env var (default: 5 minutes)
- On `Put` (new saga), set `timeout_at = NOW() + timeout_duration`
- Optionally allow per-saga-type timeouts via a config map
- **Acceptance**: New sagas have a `timeout_at` column populated

**4.2 Implement stale saga reaper** [L]
- Background goroutine started in `main.go`, runs every 30 seconds (configurable via `SAGA_REAPER_INTERVAL`)
- Query: `SELECT * FROM sagas WHERE status = 'active' AND timeout_at < NOW() FOR UPDATE SKIP LOCKED`
- For each timed-out saga:
  - Mark the current pending step as `Failed`
  - Set saga status to `compensating`
  - Call `processor.Step(txnId)` to trigger compensation
- Respect teardown manager for graceful shutdown
- **Acceptance**: Stuck sagas are automatically compensated after timeout; reaper respects shutdown signals

**4.3 Add saga timeout monitoring** [S]
- Log warnings when a saga approaches timeout (e.g., 80% of timeout elapsed)
- Add structured log fields for monitoring/alerting
- **Acceptance**: Near-timeout sagas produce warning logs

### Phase 5: Idempotent Step Completion

**5.1 Guard against duplicate step completions** [M]
- In `StepCompletedWithResult`: after `GetById`, check if the earliest pending step is actually still `Pending`
- If no pending step exists (already completed/failed by a prior event), log and return nil (no-op)
- The optimistic locking from Phase 2 already prevents the race at the DB level, but this adds an explicit application-level check
- **Acceptance**: Sending the same Kafka event twice doesn't advance the saga past where it should be

---

## Risk Assessment & Mitigation

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| JSONB serialization mismatch (70+ payload types) | High | Medium | Existing `MarshalJSON`/`UnmarshalJSON` already handles all types; add round-trip tests for each saga type |
| Optimistic locking retry loops under high contention | Medium | Low | Sagas are keyed by unique transaction ID; contention only occurs when multiple steps complete simultaneously for the same saga (rare due to sequential step execution) |
| Startup recovery re-sends Kafka commands | Medium | Medium | Downstream services already handle duplicate commands (create-if-not-exists pattern); add idempotency keys if needed |
| PostgreSQL becomes SPOF | High | Low | Service already depends on Kafka as SPOF; PostgreSQL with streaming replication is standard; can add connection retry with backoff |
| Performance regression from DB round-trips | Low | Low | Sagas are low-throughput (order of ~10-100/sec, not thousands); each saga operation is a single indexed row lookup |

## Success Metrics

1. **Zero saga data loss**: Service restart with active sagas → all resume and complete
2. **No race conditions**: Concurrent Kafka events for the same saga resolve cleanly via optimistic locking
3. **Stale saga cleanup**: Sagas exceeding timeout are compensated within 1 reaper cycle (30s)
4. **No regression**: All existing saga types (inventory_transaction, quest_reward, trade_transaction, character_creation, storage_operation, character_respawn, gachapon_transaction) pass end-to-end
5. **Latency**: P99 saga step processing time < 10ms overhead from DB operations

## Required Resources & Dependencies

- **PostgreSQL instance**: Existing infrastructure or new container in Docker Compose
- **Go dependencies**: `gorm.io/gorm`, `gorm.io/driver/postgres`
- **No external service changes**: All Kafka consumers, REST endpoints, and downstream services remain unchanged
- **Testing**: `github.com/DATA-DOG/go-sqlmock` or similar for unit tests; existing integration test patterns

## Timeline Estimates

| Phase | Effort | Dependencies |
|-------|--------|-------------|
| Phase 1: PostgreSQL Store | L | None |
| Phase 2: Optimistic Locking | M | Phase 1 |
| Phase 3: Startup Recovery | M | Phase 1 |
| Phase 4: Timeout & Reaper | L | Phase 1, Phase 2 |
| Phase 5: Idempotent Completion | S | Phase 2 |
