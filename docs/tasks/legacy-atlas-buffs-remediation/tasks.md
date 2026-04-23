# atlas-buffs Remediation Tasks

**Last Updated:** 2026-01-13
**Plan Reference:** `plan.md`
**Context Reference:** `context.md`

---

## Progress Summary

| Phase | Status | Tasks | Completed |
|-------|--------|-------|-----------|
| Phase 1: Test Infrastructure | Complete | 4 | 4 |
| Phase 2: Message Buffer Pattern | Complete | 2 | 2 |
| Phase 3: Model Improvements | Complete | 3 | 3 |
| Phase 4: Cleanup | Complete | 2 | 2 |
| **Total** | **Complete** | **11** | **11** |

---

## Phase 1: Test Infrastructure (P0 - BLOCKING)

### 1.1 Create registry_test.go
- [x] Create `character/registry_test.go`
- [x] Add `ResetForTesting()` method to registry
- [x] Test `Apply()` with concurrent goroutines
- [x] Test `Cancel()` during concurrent `Apply()`
- [x] Test `GetExpired()` with concurrent modifications
- [x] Test tenant isolation (operations on tenant A don't affect tenant B)
- [x] Test singleton initialization
- [x] Verify all tests pass with `-race` flag
- [x] Achieve >80% coverage on `registry.go`

### 1.2 Create processor_test.go
- [x] Create `character/processor_test.go`
- [x] Test `GetById()` returns correct model
- [x] Test `GetById()` returns error when not found
- [x] Test `Apply()` creates buff in registry
- [x] Test `Cancel()` removes buff from registry
- [x] Test `Cancel()` with non-existent buff (should not error)
- [x] Test `ExpireBuffs()` processes all tenants
- [x] Mock or stub Kafka producer for isolation
- [x] Test tenant context extraction

### 1.3 Create buff/model_test.go
- [x] Create `buff/model_test.go`
- [x] Test `NewBuff()` creates buff with correct fields
- [x] Test `Expired()` returns false immediately after creation
- [x] Test validation rejects invalid duration
- [x] Test all accessor methods return expected values
- [x] Test UUID is generated for each buff

### 1.4 Create REST transform tests
- [x] Create `buff/rest_test.go`
- [x] Test `Transform()` produces valid RestModel
- [x] Test `GetName()` returns expected type
- [x] Test `GetID()` returns buff UUID
- [x] Test `SetID()` updates ID correctly
- [x] Create `buff/stat/rest_test.go`
- [x] Test `Transform()` for stat model

---

## Phase 2: Message Buffer Pattern (P1)

### 2.1 Create message buffer utility
- [x] Check if message buffer exists in shared kafka library
- [x] Create `kafka/message/buffer.go`
- [x] Implement `Buffer` struct to hold messages
- [x] Implement `Put()` method to accumulate messages
- [x] Implement `Emit()` method to send all messages
- [x] Return aggregate error if any emit fails

### 2.2 Implement message buffer in processor
- [x] Refactor `Apply()` to use message buffer
- [x] Refactor `Cancel()` to use message buffer
- [x] Refactor `ExpireBuffs()` to use message buffer
- [x] Replace `_ = producer.ProviderImpl(...)` with error handling
- [x] Log errors instead of suppressing them

---

## Phase 3: Model Improvements (P2)

### 3.1 Fix character model mutability
- [x] Modify `Buffs()` in `character/model.go`
- [x] Return defensive copy instead of direct reference
- [x] Verify existing tests still pass

### 3.2 Add JSON:API interface to stat.RestModel
- [x] Add `GetName()` method returning "stats"
- [x] Add `GetID()` method
- [x] Add `SetID()` method
- [x] Ensure consistent with other REST models in service
- [x] Add tests for new interface methods

### 3.3 Add NewBuff input validation
- [x] Change signature to `NewBuff(...) (Model, error)`
- [x] Add validation: `duration > 0`
- [x] Add validation: `len(changes) > 0`
- [x] Update caller in `character/registry.go`
- [x] Update tests to use new signature
- [x] Add tests for validation errors

---

## Phase 4: Cleanup (P3)

### 4.1 Rename Respawn struct to Expiration
- [x] Rename `Respawn` to `Expiration` in `tasks/expiration.go`
- [x] Update `NewExpiration()` return type
- [x] Verify no other references exist

### 4.2 Document in-memory architecture in README
- [x] Add "Architecture Notes" section to `README.md`
- [x] Explain why in-memory storage is appropriate
- [x] Document that data loss on restart is acceptable
- [x] Mention that buff state is derived, not source of truth

---

## Verification Checklist

### After Phase 1
- [x] All tests pass with `go test ./... -race`
- [x] No race conditions detected
- [x] No test pollution between parallel runs

### After Phase 2
- [x] Kafka emissions use buffer pattern
- [x] Errors are properly returned (not suppressed)
- [x] ExpireBuffs collects all messages before emitting

### After Phase 3
- [x] Models return defensive copies
- [x] Invalid buff construction is rejected
- [x] REST models implement JSON:API interface

### After Phase 4
- [x] Code naming reflects actual purpose (Expiration not Respawn)
- [x] Architecture decisions are documented

### Final Verification
- [x] All tests pass with race detector
- [x] Build succeeds
- [x] No blocking issues remain

---

## Implementation Summary

### Files Created
- `character/registry_test.go` - Registry concurrent access tests
- `character/processor_test.go` - Processor business logic tests
- `buff/model_test.go` - Buff model tests with validation
- `buff/rest_test.go` - REST transform tests
- `buff/stat/rest_test.go` - Stat REST transform tests
- `kafka/message/buffer.go` - Message buffer utility

### Files Modified
- `character/registry.go` - Fixed race conditions, added `getOrCreateTenantMaps()` helper
- `character/processor.go` - Implemented message buffer pattern
- `character/model.go` - Returns defensive copy of buffs map
- `buff/model.go` - Added input validation with error return
- `buff/stat/rest.go` - Added JSON:API interface methods
- `tasks/expiration.go` - Renamed Respawn to Expiration
- `README.md` - Added Architecture Notes section

### Key Improvements
1. **Fixed race conditions** in registry by introducing `getOrCreateTenantMaps()` helper
2. **Added comprehensive test coverage** for registry, processor, and models
3. **Implemented message buffer pattern** for atomic Kafka emissions
4. **Added input validation** to prevent invalid buff construction
5. **Improved immutability** by returning defensive copies
6. **Added missing JSON:API interface** to stat.RestModel
7. **Documented architecture decision** for in-memory storage
