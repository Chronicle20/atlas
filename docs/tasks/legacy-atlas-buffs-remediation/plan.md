# atlas-buffs Service Remediation Plan

**Last Updated:** 2026-01-13
**Service:** `services/atlas-buffs`
**Audit Reference:** `docs/audits/atlas-buffs/audit.md`
**Overall Audit Status:** `needs-work` (56% pass rate)

---

## Executive Summary

The `atlas-buffs` service audit identified **1 blocking issue** (zero test coverage) and **7 non-blocking issues** requiring remediation. The service is an intentional architectural variant using in-memory storage for ephemeral buff state, which is appropriate for its use case. However, the lack of test coverage represents a significant quality risk, particularly for the concurrent registry operations.

**Key Remediation Goals:**
1. Establish comprehensive test coverage (blocking)
2. Implement message buffer pattern for atomic Kafka emissions
3. Fix model immutability violations
4. Align REST models with JSON:API conventions
5. Add input validation to factory functions
6. Clean up naming inconsistencies

---

## Current State Analysis

### Audit Summary

| Metric | Value |
|--------|-------|
| Total Checks | 16 |
| Passing | 9 |
| Warnings | 6 |
| Failures | 1 |
| Pass Rate | 56% |

### Issues by Priority

| Priority | Issue | Effort | Check ID |
|----------|-------|--------|----------|
| P0 | Missing test coverage | M | ARCH-012 |
| P1 | No message buffer pattern | M | ARCH-015 |
| P2 | Character model exposes mutable map | S | ARCH-002 |
| P2 | stat.RestModel missing JSON:API interface | S | ARCH-008 |
| P2 | No input validation in NewBuff | S | ARCH-003 |
| P3 | Expiration task struct named 'Respawn' | S | - |
| P3 | Document in-memory architecture decision | S | ARCH-014 |

### Architectural Context

The service intentionally deviates from standard Atlas patterns:
- **No database persistence** - Uses in-memory registry (appropriate for ephemeral buff state)
- **No entity.go** - No GORM entities needed
- **No administrator/provider separation** - Registry handles both read/write (acceptable for in-memory services)

These deviations are documented as intentional and acceptable.

---

## Proposed Future State

After remediation:
- **100% test coverage** on critical paths (registry, processor, transforms)
- **Atomic Kafka emissions** via message buffer pattern
- **Immutable models** with defensive copies
- **Consistent REST patterns** across all models
- **Validated inputs** preventing invalid state
- **Clear naming** that reflects component purpose

---

## Implementation Phases

### Phase 1: Test Infrastructure (P0 - Blocking)
**Goal:** Establish test coverage for critical components

This phase addresses the only blocking issue. Tests must cover:
- Registry thread safety and concurrent access
- Processor business logic (Apply, Cancel, ExpireBuffs)
- REST model transforms
- Buff expiration logic

### Phase 2: Message Buffer Pattern (P1)
**Goal:** Implement atomic Kafka emissions

Current state emits Kafka messages directly without atomicity guarantees. The message buffer pattern ensures all-or-nothing emission, particularly important for `ExpireBuffs()` which emits multiple messages.

### Phase 3: Model Improvements (P2)
**Goal:** Fix immutability violations and add validation

Three related improvements:
1. Character model returns defensive copy of buffs map
2. stat.RestModel implements JSON:API interface
3. NewBuff factory validates inputs

### Phase 4: Cleanup (P3)
**Goal:** Address naming and documentation issues

Low-priority cleanup tasks that improve code clarity.

---

## Detailed Tasks

### Phase 1: Test Infrastructure

#### 1.1 Create registry_test.go
**Effort:** M | **Files:** `character/registry_test.go` (new)

Test concurrent registry operations:
- Apply buff from multiple goroutines
- Cancel buff while applying
- GetExpired while applying
- Tenant isolation verification
- Singleton initialization

**Acceptance Criteria:**
- [ ] Tests pass with `-race` flag
- [ ] Coverage >80% on registry.go
- [ ] Tests verify tenant isolation

#### 1.2 Create processor_test.go
**Effort:** M | **Files:** `character/processor_test.go` (new)

Test processor business logic:
- GetById returns correct model
- Apply creates buff in registry
- Cancel removes buff from registry
- ExpireBuffs processes all tenants

**Acceptance Criteria:**
- [ ] All processor methods have test coverage
- [ ] Tests use mock/stub for Kafka producer
- [ ] Edge cases covered (not found, empty state)

#### 1.3 Create buff/model_test.go
**Effort:** S | **Files:** `buff/model_test.go` (new)

Test buff model:
- NewBuff creates valid buff
- Expired() returns correct value based on time
- Accessors return expected values

**Acceptance Criteria:**
- [ ] Time-based expiration tested
- [ ] All accessors verified

#### 1.4 Create rest transform tests
**Effort:** S | **Files:** `buff/rest_test.go`, `buff/stat/rest_test.go` (new)

Test REST model transforms:
- Transform functions produce valid JSON:API models
- ID generation and retrieval

**Acceptance Criteria:**
- [ ] Transform functions tested
- [ ] JSON:API interface methods verified

---

### Phase 2: Message Buffer Pattern

#### 2.1 Implement message buffer in processor
**Effort:** M | **Files:** `character/processor.go`

Refactor processor to use message buffer pattern:
- Create `ApplyAndEmit` pattern that buffers message before registry operation
- Create `CancelAndEmit` pattern
- Ensure atomic emit in `ExpireBuffs`

**Current Code (processor.go:38-41):**
```go
func (p *ProcessorImpl) Apply(...) error {
    b := GetRegistry().Apply(...)
    _ = producer.ProviderImpl(p.l)(p.ctx)(...)(appliedStatusEventProvider(...))
    return nil
}
```

**Target Pattern:**
```go
func (p *ProcessorImpl) Apply(...) error {
    messages := message.NewBuffer()
    b := GetRegistry().Apply(...)
    messages.Add(appliedStatusEventProvider(...))
    return messages.Emit(p.l, p.ctx, character2.EnvEventStatusTopic)
}
```

**Acceptance Criteria:**
- [ ] Apply uses message buffer
- [ ] Cancel uses message buffer
- [ ] ExpireBuffs collects all messages before emitting
- [ ] Kafka errors are logged (not suppressed)

#### 2.2 Create message buffer utility
**Effort:** S | **Files:** `kafka/message/buffer.go` (new)

Create message buffer type if not already in shared library.

**Acceptance Criteria:**
- [ ] Buffer can accumulate messages
- [ ] Emit sends all messages atomically
- [ ] Errors are properly propagated

---

### Phase 3: Model Improvements

#### 3.1 Fix character model mutability
**Effort:** S | **Files:** `character/model.go`

Return defensive copy of buffs map.

**Current Code (model.go:15-17):**
```go
func (m Model) Buffs() map[int32]buff.Model {
    return m.buffs
}
```

**Target Code:**
```go
func (m Model) Buffs() map[int32]buff.Model {
    result := make(map[int32]buff.Model, len(m.buffs))
    for k, v := range m.buffs {
        result[k] = v
    }
    return result
}
```

**Acceptance Criteria:**
- [ ] Buffs() returns copy
- [ ] External mutations don't affect internal state
- [ ] Existing tests pass

#### 3.2 Add JSON:API interface to stat.RestModel
**Effort:** S | **Files:** `buff/stat/rest.go`

Add GetName, GetID, SetID methods for consistency.

**Acceptance Criteria:**
- [ ] GetName() returns "stats" (or appropriate name)
- [ ] GetID()/SetID() implemented
- [ ] Consistent with other REST models

#### 3.3 Add NewBuff input validation
**Effort:** S | **Files:** `buff/model.go`

Validate inputs in factory function.

**Current Code (model.go:42-51):**
```go
func NewBuff(sourceId int32, duration int32, changes []stat.Model) Model {
    return Model{...}
}
```

**Target Code:**
```go
func NewBuff(sourceId int32, duration int32, changes []stat.Model) (Model, error) {
    if duration <= 0 {
        return Model{}, errors.New("duration must be positive")
    }
    if len(changes) == 0 {
        return Model{}, errors.New("changes cannot be empty")
    }
    return Model{...}, nil
}
```

**Note:** This changes the function signature - callers must be updated.

**Acceptance Criteria:**
- [ ] Validates duration > 0
- [ ] Validates changes not empty
- [ ] Returns error for invalid input
- [ ] Callers updated to handle error

---

### Phase 4: Cleanup

#### 4.1 Rename Respawn struct to Expiration
**Effort:** S | **Files:** `tasks/expiration.go`

**Current Code:**
```go
type Respawn struct {
    l        logrus.FieldLogger
    interval int
}
```

**Target Code:**
```go
type Expiration struct {
    l        logrus.FieldLogger
    interval int
}
```

**Acceptance Criteria:**
- [ ] Struct renamed to Expiration
- [ ] All references updated
- [ ] Tests pass

#### 4.2 Document in-memory architecture in README
**Effort:** S | **Files:** `README.md`

Add section explaining why service uses in-memory storage.

**Content to add:**
```markdown
## Architecture Notes

This service intentionally uses in-memory storage rather than database
persistence. This is appropriate because buff state is:
- Ephemeral (seconds to minutes lifetime)
- Derived from commands (source of truth is the commanding service)
- Safe to lose on restart (game state will re-apply buffs)
```

**Acceptance Criteria:**
- [ ] README explains in-memory design decision
- [ ] Documents that data loss on restart is acceptable

---

## Risk Assessment

### High Risk
| Risk | Mitigation |
|------|------------|
| Registry race conditions in tests | Use `-race` flag in all test runs |
| Breaking change to NewBuff signature | Update all callers in same PR |

### Medium Risk
| Risk | Mitigation |
|------|------------|
| Message buffer changes emission timing | Test with integration tests before deploy |
| Defensive copy impacts performance | Benchmark before/after |

### Low Risk
| Risk | Mitigation |
|------|------------|
| Struct rename breaks external references | Verify no external consumers |

---

## Success Metrics

| Metric | Current | Target |
|--------|---------|--------|
| Test Coverage | 0% | >80% |
| Audit Pass Rate | 56% | >90% |
| Blocking Issues | 1 | 0 |
| Warnings | 6 | 2 (intentional deviations only) |

---

## Dependencies

### Internal
- `kafka/message` package for buffer implementation (may need to create)
- Atlas test utilities for mocking

### External
- None

---

## Notes

### Intentional Deviations (Do Not Fix)
These warnings are documented as acceptable architectural decisions:
- **ARCH-005** (Provider Pattern): Registry combines read/write - acceptable for in-memory services
- **ARCH-016** (Administrator/Provider Separation): Same as above

### Additional Observations from Audit
1. **Error suppression** - Processor ignores Kafka errors with `_ = producer.ProviderImpl(...)`. This should be addressed in Phase 2 by using message buffer with proper error handling.
2. **Expiration goroutines** - `ExpireBuffs()` spawns goroutines per tenant which could cause many concurrent Kafka emissions. Consider batching in Phase 2.
