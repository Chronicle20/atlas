# Atlas-Expressions Service Remediation Plan

**Service Path:** `services/atlas-expressions/atlas.com/expressions`
**Audit Reference:** `docs/audits/atlas-expressions/audit.md`
**Last Updated:** 2026-01-13

---

## Executive Summary

The atlas-expressions service is a Kafka-only, in-memory microservice for managing ephemeral character expressions (facial animations with 5-second TTL). The audit identified **0 blocking issues** and **5 non-blocking issues** with an overall status of `needs-work`.

This remediation plan addresses:
1. **Missing test coverage** (P0 - High impact)
2. **Excessively deep currying in processor** (P1 - Medium impact)
3. **Message buffer pattern not properly utilized** (P2 - Low impact)
4. **Optional: Builder pattern for Model** (P2 - Low impact)

The service intentionally deviates from standard patterns (no database, no REST, custom Registry singleton) due to its ephemeral data nature. These deviations are justified and documented.

---

## Current State Analysis

### Architecture Overview

The service uses an in-memory Registry singleton instead of the standard provider/administrator/entity pattern because:
- Expressions are ephemeral (5-second TTL)
- Data is recoverable (resets on character map exit)
- Performance is critical (O(1) memory access)
- No REST interface needed (Kafka-only)

### Issue Summary

| Issue ID | Description | Severity | Effort |
|----------|-------------|----------|--------|
| NB-001 | Missing test coverage | High | M |
| NB-002 | Excessively deep currying in `Change` method | Medium | S |
| NB-003 | Registry combines read/write (documented justification) | Low | S |
| NB-004 | Not using `message.Emit` pattern | Low | S |
| NB-005 | Missing Builder pattern | Low | S |

### Files Requiring Changes

| File | Changes Required |
|------|------------------|
| `expression/model.go` | Add builder (optional) |
| `expression/processor.go` | Flatten currying, use message.Emit |
| `expression/registry.go` | Add ResetForTesting method |
| `expression/model_test.go` | New file - model tests |
| `expression/processor_test.go` | New file - processor tests |
| `expression/registry_test.go` | New file - registry tests |
| `expression/mock/processor.go` | New file - processor mock |
| `expression/task_test.go` | New file - task tests |

---

## Proposed Future State

### Phase 1: Test Infrastructure Setup (P0)

Establish test infrastructure to enable comprehensive testing:

1. **Add `ResetForTesting()` method to Registry** - Allows test isolation
2. **Create mock processor** - Enables testing components that depend on Processor
3. **Set up test helpers** - Common tenant/model creation utilities

### Phase 2: Comprehensive Test Coverage (P0)

Add test files covering all domain logic:

1. **Model tests** - Verify immutability and getter methods
2. **Registry tests** - CRUD operations, tenant isolation, concurrency
3. **Processor tests** - Business logic, buffer operations
4. **Task tests** - Expression revert functionality

### Phase 3: Processor Refactoring (P1)

Flatten the excessive currying and adopt proper message emission:

1. **Flatten `Change` signature** - Match `ChangeAndEmit` flat parameter style
2. **Use `message.Emit` pattern** - Proper atomic message emission

### Phase 4: Optional Improvements (P2)

Low-priority improvements for consistency:

1. **Add Builder pattern** - Optional, for guideline consistency
2. **Document Registry architecture** - Update README with justification

---

## Implementation Phases

### Phase 1: Test Infrastructure Setup

**Objective:** Create foundation for comprehensive testing

#### Task 1.1: Add ResetForTesting to Registry
- **File:** `expression/registry.go`
- **Description:** Add method to reset registry state between tests
- **Pattern Reference:** `services/atlas-buffs/atlas.com/buffs/character/registry.go:171-176`
- **Acceptance Criteria:**
  - Method resets all internal maps
  - Method is thread-safe
  - Only intended for test use (documented)

#### Task 1.2: Create Processor Mock
- **File:** `expression/mock/processor.go` (new)
- **Description:** Create mock implementation of Processor interface
- **Pattern Reference:** `services/atlas-drops/atlas.com/drops/drop/mock/processor.go`
- **Acceptance Criteria:**
  - Implements full Processor interface
  - Each method has configurable func field
  - Default implementations return zero values

#### Task 1.3: Create Test Helpers
- **File:** `expression/test_helpers_test.go` (new)
- **Description:** Common utilities for creating test tenants and models
- **Acceptance Criteria:**
  - `setupTestTenant(t *testing.T)` helper function
  - Test context creation with tenant

---

### Phase 2: Comprehensive Test Coverage

**Objective:** Achieve comprehensive test coverage for all domain logic

#### Task 2.1: Model Tests
- **File:** `expression/model_test.go` (new)
- **Pattern Reference:** `services/atlas-buffs/atlas.com/buffs/buff/model_test.go`
- **Test Cases:**
  - All getter methods return correct values
  - Model fields are properly encapsulated
  - Expiration time calculation
- **Acceptance Criteria:**
  - 100% coverage of Model methods
  - Tests verify immutability (no setters)

#### Task 2.2: Registry Tests
- **File:** `expression/registry_test.go` (new)
- **Pattern Reference:** `services/atlas-buffs/atlas.com/buffs/character/registry_test.go`
- **Test Cases:**
  - `add()` creates expression correctly
  - `popExpired()` returns and removes expired expressions
  - `clear()` removes expression for character
  - Tenant isolation (separate tenants don't see each other's data)
  - Concurrent access safety
  - GetRegistry singleton behavior
- **Acceptance Criteria:**
  - All registry operations covered
  - Concurrency tests with multiple goroutines
  - Tenant isolation verified

#### Task 2.3: Processor Tests
- **File:** `expression/processor_test.go` (new)
- **Test Cases:**
  - `NewProcessor` extracts tenant from context
  - `Change` adds expression to registry and buffers message
  - `ChangeAndEmit` integrates buffer and emission
  - `Clear` removes expression from registry
  - `ClearAndEmit` integrates clearing with buffer
  - Error handling for missing tenant in context
- **Acceptance Criteria:**
  - All Processor interface methods tested
  - Buffer operations verified
  - Integration tests for AndEmit variants

#### Task 2.4: Task Tests
- **File:** `expression/task_test.go` (new)
- **Test Cases:**
  - `NewRevertTask` initializes correctly
  - `Run` processes expired expressions
  - `SleepTime` returns configured interval
- **Acceptance Criteria:**
  - Task lifecycle tested
  - Expiration processing verified

---

### Phase 3: Processor Refactoring

**Objective:** Flatten excessive currying and use proper message patterns

#### Task 3.1: Flatten Change Method Signature
- **File:** `expression/processor.go`
- **Current Signature:**
  ```go
  Change(mb *message.Buffer) func(transactionId uuid.UUID) func(characterId uint32) func(worldId world.Id) func(channelId channel.Id) func(mapId _map.Id) func(expression uint32) (Model, error)
  ```
- **Target Signature:**
  ```go
  Change(mb *message.Buffer, transactionId uuid.UUID, characterId uint32, worldId world.Id, channelId channel.Id, mapId _map.Id, expression uint32) (Model, error)
  ```
- **Acceptance Criteria:**
  - Signature flattened to single function call
  - All call sites updated
  - Tests pass

#### Task 3.2: Flatten Clear Method Signature
- **File:** `expression/processor.go`
- **Current Signature:**
  ```go
  Clear(mb *message.Buffer) func(transactionId uuid.UUID) func(characterId uint32) (Model, error)
  ```
- **Target Signature:**
  ```go
  Clear(mb *message.Buffer, transactionId uuid.UUID, characterId uint32) (Model, error)
  ```
- **Acceptance Criteria:**
  - Signature flattened
  - All call sites updated
  - Tests pass

#### Task 3.3: Adopt message.Emit Pattern
- **File:** `expression/processor.go`
- **Description:** Refactor `ChangeAndEmit` and `ClearAndEmit` to use `message.Emit()` or `message.EmitWithResult()` functions
- **Pattern Reference:** `kafka/message/message.go:32-47` (Emit function)
- **Current Implementation:**
  ```go
  mb := message.NewBuffer()
  model, err := p.Change(mb)(...)
  for t := range mb.GetAll() {
      err = producer.ProviderImpl(p.l)(p.ctx)(t)(...)
  }
  ```
- **Target Implementation:**
  ```go
  return message.EmitWithResult[Model, ChangeInput](
      producer.ProviderImpl(p.l)(p.ctx),
  )(func(mb *message.Buffer) func(input ChangeInput) (Model, error) {
      return func(input ChangeInput) (Model, error) {
          return p.Change(mb, input.TransactionId, ...)
      }
  })(input)
  ```
- **Acceptance Criteria:**
  - Uses message.Emit or EmitWithResult pattern
  - Atomic emission behavior preserved
  - Tests pass

#### Task 3.4: Update Processor Interface
- **File:** `expression/processor.go`
- **Description:** Update Processor interface to match new flat signatures
- **Acceptance Criteria:**
  - Interface updated
  - Mock updated to match
  - All implementations aligned

---

### Phase 4: Optional Improvements

**Objective:** Additional consistency improvements (low priority)

#### Task 4.1: Add Builder Pattern (Optional)
- **File:** `expression/builder.go` (new)
- **Description:** Add builder for expression.Model for guideline consistency
- **Acceptance Criteria:**
  - Builder with fluent API
  - `Build()` returns immutable Model
  - Registry updated to use builder

#### Task 4.2: Document Registry Architecture
- **File:** `services/atlas-expressions/README.md`
- **Description:** Add section documenting the intentional deviation from provider/administrator pattern
- **Acceptance Criteria:**
  - Justification for in-memory architecture documented
  - Registry singleton pattern explained

---

## Risk Assessment and Mitigation

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Refactoring breaks existing consumers | High | Low | Kafka interface unchanged; internal refactoring only |
| Registry race conditions during testing | Medium | Medium | Add ResetForTesting with proper locking |
| Test isolation failures | Medium | Low | Use ResetForTesting between tests |
| Message emission order changes | Low | Low | Atomic emission via message.Emit preserves order |

---

## Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Test coverage | >80% | `go test -cover` |
| All tests pass | 100% | CI pipeline |
| Processor currying depth | 1 level | Code review |
| message.Emit pattern used | Yes | Code review |

---

## Dependencies

### Internal Dependencies
- `kafka/message/message.go` - Already has Emit/EmitWithResult functions
- `kafka/producer/producer.go` - Producer provider pattern

### External Dependencies
- `github.com/stretchr/testify` - Test assertions (already in use elsewhere)
- `github.com/Chronicle20/atlas-tenant` - Tenant creation for tests

---

## Effort Summary

| Phase | Effort | Priority |
|-------|--------|----------|
| Phase 1: Test Infrastructure | S | P0 |
| Phase 2: Test Coverage | M | P0 |
| Phase 3: Processor Refactoring | S | P1 |
| Phase 4: Optional Improvements | S | P2 |
| **Total** | **M** | - |

---

## Implementation Notes

### Pattern References

1. **Registry with ResetForTesting:**
   - `services/atlas-buffs/atlas.com/buffs/character/registry.go:171-176`

2. **Processor Mock:**
   - `services/atlas-drops/atlas.com/drops/drop/mock/processor.go`

3. **Model Tests:**
   - `services/atlas-buffs/atlas.com/buffs/buff/model_test.go`

4. **Registry Tests:**
   - `services/atlas-buffs/atlas.com/buffs/character/registry_test.go`

5. **message.Emit Pattern:**
   - `services/atlas-expressions/atlas.com/expressions/kafka/message/message.go:32-65`

### Architectural Decisions

1. **No entity.go:** Justified - no database persistence
2. **No provider/administrator:** Justified - Registry singleton appropriate for in-memory
3. **No REST endpoints:** Justified - Kafka-only service by design
4. **No builder.go:** Optional addition for consistency
