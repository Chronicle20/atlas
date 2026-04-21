# Saga Orchestrator Durability — Task Checklist

Last Updated: 2026-02-19

## Phase 1: PostgreSQL Store (Core)

- [x] **1.1** Add `gorm.io/gorm` and `gorm.io/driver/postgres` to go.mod
- [x] **1.2** Create `saga/entity.go` — GORM entity with `sagas` table mapping
- [x] **1.3** Add GORM auto-migrate in `main.go` startup
- [x] **1.4** Create `saga/store.go` — `PostgresStore` implementing `Cache` interface
  - [x] `GetAll(tenantId)` — query active sagas by tenant
  - [x] `GetById(tenantId, transactionId)` — single saga lookup
  - [x] `Put(tenantId, saga)` — upsert with JSON serialization
  - [x] `Remove(tenantId, transactionId)` — soft delete (set status=completed)
- [x] **1.5** Wire `PostgresStore` into `main.go`, replace `GetCache()` singleton
- [x] **1.6** Update `atlas-saga-orchestrator.yml` with PostgreSQL env vars
- [x] **1.7** Tests pass: `go test ./... -count=1`
- [x] **1.8** Build succeeds: `go build`

## Phase 2: Optimistic Locking

- [x] **2.1** Add `version` field to entity, `PUT` increments version with `WHERE version = ?`
- [x] **2.2** Define `VersionConflictError` type
- [x] **2.3** Optimistic locking built into `Put` — conflict detected via `RowsAffected == 0`
- [x] **2.4** Version tracking via in-memory map (read version on `GetById`, use in `Put`)
- [x] **2.5** Tests pass: `go test ./... -count=1`
- [x] **2.6** Build succeeds: `go build`

## Phase 3: Startup Recovery

- [x] **3.1** Query active/compensating sagas on startup via `GetAllActive()`
- [x] **3.2** Re-drive each recovered saga through `processor.Step()`
- [x] **3.3** Add `SAGA_RECOVERY_ENABLED` env var
- [x] **3.4** Add structured logging for recovered sagas
- [x] **3.5** Reconstruct full tenant context (region, version) from entity for recovery
- [x] **3.6** Tests pass: `go test ./... -count=1`
- [x] **3.7** Build succeeds: `go build`

## Phase 4: Timeout & Reaper

- [x] **4.1** Add `timeout_at` to entity, set on saga creation
- [x] **4.2** Add `SAGA_DEFAULT_TIMEOUT` env var (default 5m)
- [x] **4.3** Implement reaper goroutine in `main.go`
  - [x] Query timed-out active sagas with `FOR UPDATE SKIP LOCKED`
  - [x] Mark current step as Failed
  - [x] Trigger compensation via `processor.Step()`
- [x] **4.4** Add `SAGA_REAPER_INTERVAL` env var (default 30s)
- [x] **4.5** Integrate reaper with teardown manager for graceful shutdown
- [x] **4.6** Tests pass: `go test ./... -count=1`
- [x] **4.7** Build succeeds: `go build`

## Phase 5: Idempotent Step Completion

- [x] **5.1** In `StepCompletedWithResult`, check saga has pending steps before mutation
- [x] **5.2** Log and return nil for duplicate completions (no-op)
- [x] **5.3** Tests pass: `go test ./... -count=1`
- [x] **5.4** Build succeeds: `go build`

## Post-Implementation

- [ ] End-to-end test: create saga via REST, verify persisted in PostgreSQL
- [ ] End-to-end test: kill service mid-saga, restart, verify saga resumes
- [ ] End-to-end test: verify timed-out saga triggers compensation
- [x] Update `docs/architectural-improvements.md` to mark saga durability as resolved
- [x] Update `MEMORY.md` with saga orchestrator migration status
