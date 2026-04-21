# Saga Orchestrator Durability — Context

Last Updated: 2026-02-19

## Key Files

### Core (must modify)
| File | Purpose | Changes Needed |
|------|---------|---------------|
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/cache.go` | In-memory cache (124 lines) | Replace `InMemoryCache` with `PostgresStore`; keep `Cache` interface |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/processor.go` | Saga state machine (1228 lines) | Add retry logic around `GetById`→`Put` sequences; convert `AtomicUpdateSaga` to DB transaction |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/main.go` | Service entrypoint (105 lines) | Add GORM init, auto-migrate, startup recovery, reaper goroutine |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/go.mod` | Dependencies | Add GORM + postgres driver |
| `services/atlas-saga-orchestrator/atlas-saga-orchestrator.yml` | Docker Compose config | Add PostgreSQL env vars |

### New Files
| File | Purpose |
|------|---------|
| `saga/entity.go` | GORM entity mapping to `sagas` table |
| `saga/store.go` | `PostgresStore` implementing `Cache` interface |
| `saga/store_test.go` | Unit tests for PostgresStore |

### Reference (read-only, do not modify)
| File | Purpose | Why Important |
|------|---------|--------------|
| `saga/model.go` (1816 lines) | Saga + Step[any] domain model | Has `MarshalJSON`/`UnmarshalJSON` for JSONB serialization |
| `saga/compensator.go` | Compensation logic | Called by reaper for timed-out sagas |
| `saga/handler.go` | 60+ action handlers | These fire Kafka commands; must be idempotent for recovery |
| `saga/resource.go` | REST API (GET/POST sagas) | Uses `Processor` — no changes needed |
| `kafka/consumer/*/consumer.go` (14 packages) | Kafka event consumers | All call `processor.StepCompleted` — no changes needed |

## Architecture Decisions

### Decision 1: JSONB vs Normalized Tables
**Chose**: Single `saga_data JSONB` column
**Rationale**: 70+ payload types with different structures. Normalizing would require a `saga_steps` table with a polymorphic `payload` column anyway. The `Saga` struct already has complete JSON serialization. JSONB preserves the existing data model exactly. PostgreSQL JSONB supports indexing on specific fields if needed later.
**Trade-off**: Cannot query individual step fields via SQL without JSONB operators. Acceptable because all access patterns are by `transaction_id`.

### Decision 2: Optimistic Locking vs SELECT FOR UPDATE
**Chose**: Optimistic locking (version column) for normal operations, `SELECT FOR UPDATE` only for `AtomicUpdateSaga`
**Rationale**: Most saga operations are low-contention (one consumer at a time per saga). Optimistic locking avoids holding DB locks during Kafka command execution. `AtomicUpdateSaga` (step expansion) is the only case that needs true transactional atomicity.

### Decision 3: Soft Delete vs Hard Delete
**Chose**: Soft delete — `Remove()` sets `status = 'completed'` instead of `DELETE`
**Rationale**: Preserves audit trail of completed sagas. Enables future analytics (saga duration, failure rates). Active saga queries use `WHERE status = 'active'` index.

### Decision 4: Keep `Cache` Interface
**Chose**: `PostgresStore` implements existing `Cache` interface
**Rationale**: Zero changes to consumers, REST API, handlers, compensator. Minimizes blast radius. The interface is simple (4 methods) and maps cleanly to SQL operations.

### Decision 5: Recovery Strategy
**Chose**: Re-drive via `Step()` on startup
**Rationale**: The saga state machine is already designed to be re-entrant — `Step()` reads current state and decides next action. Handlers send Kafka commands (fire-and-forget), so re-sending is safe. Downstream services handle duplicate commands via create-if-not-exists patterns.

## Dependencies

### Internal
- `github.com/Chronicle20/atlas-tenant` — tenant context extraction
- `github.com/Chronicle20/atlas-kafka` — consumer/producer infrastructure (unchanged)
- `github.com/Chronicle20/atlas-model` — model.Provider pattern (unchanged)

### External (new)
- `gorm.io/gorm` — ORM for PostgreSQL
- `gorm.io/driver/postgres` — GORM PostgreSQL driver
- PostgreSQL 14+ instance

### Test Dependencies
- `github.com/stretchr/testify` — already in go.mod
- `github.com/DATA-DOG/go-sqlmock` — for mocking GORM DB in unit tests

## Environment Variables (New)

| Variable | Description | Default |
|----------|-------------|---------|
| `DB_HOST` | PostgreSQL host | `localhost` |
| `DB_PORT` | PostgreSQL port | `5432` |
| `DB_USER` | PostgreSQL user | `postgres` |
| `DB_PASSWORD` | PostgreSQL password | (required) |
| `DB_NAME` | Database name | `saga_orchestrator` |
| `SAGA_DEFAULT_TIMEOUT` | Saga timeout duration | `5m` |
| `SAGA_REAPER_INTERVAL` | Reaper check interval | `30s` |
| `SAGA_RECOVERY_ENABLED` | Enable startup recovery | `true` |

## Saga Types & Payload Complexity

The 7 saga types and their approximate step counts:

| Saga Type | Typical Steps | Notes |
|-----------|--------------|-------|
| `inventory_transaction` | 2-6 | Award/destroy items, equip/unequip |
| `quest_reward` | 3-8 | Multiple items + experience + meso |
| `trade_transaction` | 4-10 | Transfer items between two characters |
| `character_creation` | 5-8 | Create character + initial equipment |
| `storage_operation` | 2-4 | Expand into release/accept sub-steps |
| `character_respawn` | 2-3 | Warp + HP restore |
| `gachapon_transaction` | 3-6 | Deduct meso + award random items |

All saga types use the same `Saga` JSON structure — the difference is in the `Step.action` field and its typed payload.

## Concurrency Model (Current vs Proposed)

### Current (Broken)
```
Consumer A: GetById(txn) → returns Saga{step1:completed, step2:pending}
Consumer B: GetById(txn) → returns Saga{step1:completed, step2:pending}  // same state!
Consumer A: MarkStep2Completed → Put(Saga{step1:completed, step2:completed})
Consumer B: MarkStep2Failed → Put(Saga{step1:completed, step2:failed})  // overwrites A!
```

### Proposed (Optimistic Locking)
```
Consumer A: GetById(txn) → Saga{..., version:3}
Consumer B: GetById(txn) → Saga{..., version:3}
Consumer A: Put(Saga{..., version:3}) → UPDATE WHERE version=3 → success, version=4
Consumer B: Put(Saga{..., version:3}) → UPDATE WHERE version=3 → 0 rows → VersionConflictError
Consumer B: retry → GetById(txn) → Saga{..., version:4} → re-apply mutation → Put → success
```
