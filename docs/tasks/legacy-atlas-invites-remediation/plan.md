# Atlas-Invites Service Remediation Plan

**Last Updated:** 2026-01-13

---

## 1. Executive Summary

This plan addresses issues identified in the `atlas-invites` service audit (`docs/audits/atlas-invites/audit.md`). The service is an in-memory invite management system that handles buddy, party, and guild invitations via Kafka commands and REST queries. While functionally coherent, the audit identified several structural deviations from the standard DDD/persistence patterns.

### Key Issues Summary

| Priority | Issue | Impact | Effort |
|----------|-------|--------|--------|
| P0 | No test coverage (ARCH-013) | High | L |
| P0 | Transform uses private fields (ARCH-012) | Medium | S |
| P1 | Missing builder pattern (ARCH-003) | Medium | M |
| P1 | Undocumented in-memory architecture (ARCH-002) | Medium | S |
| P2 | Custom handler registration (ARCH-011) | Low | S |
| P2 | No provider/administrator separation (ARCH-005/016) | Medium | M |

### Success Criteria

1. Test coverage for all processor operations (Create, Accept, Reject)
2. Test coverage for registry concurrent access patterns
3. Transform function uses accessor methods exclusively
4. Builder pattern with validation for model construction
5. Architectural decision for in-memory storage documented

---

## 2. Current State Analysis

### Architecture Overview

The `atlas-invites` service uses an intentional **in-memory architecture** for ephemeral invite data:

```
atlas-invites/atlas.com/invites/
├── invite/
│   ├── model.go          # Immutable domain model (PASS)
│   ├── processor.go      # Business logic with AndEmit pattern (PASS)
│   ├── registry.go       # In-memory singleton storage (WARN - no builder)
│   ├── rest.go           # REST model (FAIL - private field access)
│   ├── producer.go       # Kafka event providers (PASS)
│   └── task.go           # Timeout cleanup task
├── character/
│   └── resource.go       # REST endpoint (uses custom handler)
├── rest/
│   └── handler.go        # Custom handler registration (WARN - duplicates framework)
└── kafka/
    ├── consumer/         # Kafka consumers (PASS)
    └── producer/         # Kafka producers (PASS)
```

### What's Working Well

- Immutable domain model with private fields and public accessors
- Processor layer with proper `AndEmit` pattern for Kafka events
- Multi-tenancy context extraction and data isolation
- Thread-safe registry with proper mutex usage
- Kafka producer/consumer patterns follow guidelines
- Comprehensive service documentation in README

### What Needs Improvement

1. **Zero test coverage** - Critical gap that must be addressed first
2. **Encapsulation violation** - `rest.go:Transform` accesses private model fields
3. **No builder pattern** - Model construction happens inline without validation
4. **Custom handler code** - Duplicates functionality from `atlas-rest/server`

---

## 3. Proposed Future State

### Target Architecture

```
atlas-invites/atlas.com/invites/
├── invite/
│   ├── model.go          # (unchanged) Immutable domain model
│   ├── builder.go        # (NEW) Fluent builder with validation
│   ├── processor.go      # (unchanged) Business logic
│   ├── registry.go       # (refactored) Uses builder for model construction
│   ├── rest.go           # (fixed) Transform uses accessor methods
│   ├── producer.go       # (unchanged) Kafka event providers
│   ├── task.go           # (unchanged) Timeout cleanup
│   ├── builder_test.go   # (NEW) Builder validation tests
│   ├── processor_test.go # (NEW) Processor operation tests
│   └── registry_test.go  # (NEW) Registry concurrency tests
├── character/
│   └── resource.go       # (migrated) Uses server.RegisterHandler
├── rest/
│   └── handler.go        # (deprecated) Custom handlers removed
└── README.md             # (updated) Architectural decision documented
```

### Design Decisions

1. **Maintain in-memory architecture** - This is intentional for ephemeral invite data
2. **No entity.go needed** - Service doesn't use database persistence
3. **No provider.go needed** - Registry serves as the data access layer for in-memory storage
4. **No administrator.go needed** - Write operations through processor are appropriate

---

## 4. Implementation Phases

### Phase 1: P0 Blocking Issues (Required)

These issues block other development work and must be completed first.

#### Section 1.1: Test Infrastructure Setup

**Objective:** Establish test infrastructure and patterns for the service.

| # | Task | Effort | Acceptance Criteria |
|---|------|--------|---------------------|
| 1.1.1 | Create test helper functions for logger, tenant, and context setup | S | Helper functions match patterns in `atlas-fame/fame/processor_test.go` |
| 1.1.2 | Create mock directory structure (`invite/mock/`) | S | Directory exists with mock files |

#### Section 1.2: Fix Transform Encapsulation (ARCH-012)

**Objective:** Update Transform function to use accessor methods.

| # | Task | Effort | Acceptance Criteria |
|---|------|--------|---------------------|
| 1.2.1 | Update `invite/rest.go:Transform` to use accessor methods | S | Transform uses `m.Id()`, `m.Type()`, `m.ReferenceId()`, etc. instead of `m.id`, `m.inviteType`, `m.referenceId` |
| 1.2.2 | Add Transform unit test to verify output matches model data | S | Test passes, verifying all fields are correctly mapped |

**Code Change:**
```go
// Before (line 34-43)
return RestModel{
    Id:           m.id,           // Private field access
    Type:         m.inviteType,   // Private field access
    ...
}

// After
return RestModel{
    Id:           m.Id(),         // Accessor method
    Type:         m.Type(),       // Accessor method
    ...
}
```

#### Section 1.3: Processor Tests (ARCH-013)

**Objective:** Implement processor tests for Create, Accept, and Reject operations.

| # | Task | Effort | Acceptance Criteria |
|---|------|--------|---------------------|
| 1.3.1 | Create `invite/processor_test.go` with test setup | M | File exists with proper imports and test helpers |
| 1.3.2 | Test `NewProcessor` initialization and tenant extraction | S | Tests verify processor initializes correctly and panics without tenant |
| 1.3.3 | Test `GetByCharacterId` with empty and populated registry | S | Tests pass for empty results and results with data |
| 1.3.4 | Test `Create` operation with message buffer mock | M | Test verifies model creation and event emission |
| 1.3.5 | Test `CreateAndEmit` integration | S | Test verifies full flow with producer |
| 1.3.6 | Test `Accept` operation - locates and deletes invite | M | Test verifies invite lookup, deletion, and event emission |
| 1.3.7 | Test `Accept` error path - invite not found | S | Test verifies proper error handling |
| 1.3.8 | Test `Reject` operation - locates and deletes invite | M | Test verifies invite lookup, deletion, and event emission |
| 1.3.9 | Test `Reject` error path - invite not found | S | Test verifies proper error handling |

#### Section 1.4: Registry Tests (ARCH-013)

**Objective:** Implement registry tests for storage operations and concurrency.

| # | Task | Effort | Acceptance Criteria |
|---|------|--------|---------------------|
| 1.4.1 | Create `invite/registry_test.go` with test setup | S | File exists with tenant setup helpers |
| 1.4.2 | Test `GetRegistry` singleton behavior | S | Test verifies same instance returned |
| 1.4.3 | Test `Create` stores invite correctly | S | Test verifies invite retrievable after creation |
| 1.4.4 | Test `Create` returns existing invite for duplicate referenceId | S | Test verifies idempotent behavior |
| 1.4.5 | Test `GetByOriginator` retrieval | S | Test verifies correct invite returned |
| 1.4.6 | Test `GetByReference` retrieval | S | Test verifies correct invite returned |
| 1.4.7 | Test `GetForCharacter` returns all invite types | S | Test verifies aggregation across types |
| 1.4.8 | Test `Delete` removes invite | S | Test verifies invite no longer retrievable |
| 1.4.9 | Test `GetExpired` filters by timeout | M | Test verifies only expired invites returned |
| 1.4.10 | Test concurrent Create operations | M | Test with goroutines verifies thread safety |
| 1.4.11 | Test tenant isolation | S | Test verifies tenants cannot see each other's invites |

---

### Phase 2: P1 Important Issues

These issues improve code quality and maintainability.

#### Section 2.1: Builder Pattern (ARCH-003)

**Objective:** Implement builder pattern with validation for model construction.

| # | Task | Effort | Acceptance Criteria |
|---|------|--------|---------------------|
| 2.1.1 | Create `invite/builder.go` with fluent builder | M | Builder struct with Set* methods returning *Builder |
| 2.1.2 | Implement `Build()` with validation | M | Returns error for invalid tenant, zero IDs, empty type |
| 2.1.3 | Create `invite/builder_test.go` | M | Table-driven tests for all validation rules |
| 2.1.4 | Update `registry.go:Create` to use builder | S | Model constructed via builder, not inline |
| 2.1.5 | Run full test suite to verify integration | S | `go test ./... -count=1` passes |

**Builder Validation Rules:**
- `tenant` must not be zero value
- `id` must be > 0 (generated by registry)
- `inviteType` must not be empty
- `originatorId` must be > 0
- `targetId` must be > 0
- `worldId` validation (if applicable)

#### Section 2.2: Document Architecture Decision (ARCH-002)

**Objective:** Document the intentional in-memory architecture in README.

| # | Task | Effort | Acceptance Criteria |
|---|------|--------|---------------------|
| 2.2.1 | Add "Architecture Decision" section to README.md | S | Section explains why in-memory storage is used |
| 2.2.2 | Document that invites are ephemeral and don't require persistence | S | Clear explanation of design rationale |
| 2.2.3 | Note implications for service restart (invites lost) | S | Operational considerations documented |

---

### Phase 3: P2 Nice-to-Have Improvements

These issues are lower priority but improve consistency with framework patterns.

#### Section 3.1: Migrate to Framework Handler Registration (ARCH-011)

**Objective:** Remove custom handler registration in favor of framework handlers.

| # | Task | Effort | Acceptance Criteria |
|---|------|--------|---------------------|
| 3.1.1 | Update `character/resource.go` to use `server.RegisterHandler` | M | Handler registered using framework function |
| 3.1.2 | Verify handler dependency structure compatibility | S | Dependencies properly passed to handler |
| 3.1.3 | Remove or deprecate `rest/handler.go` custom registration | S | Custom functions no longer used |
| 3.1.4 | Run full test suite | S | All tests pass after migration |

**Note:** This requires careful analysis of whether the custom `HandlerDependency` and `HandlerContext` types can be replaced with framework equivalents.

#### Section 3.2: Provider Pattern (Optional - ARCH-005)

**Objective:** Consider provider pattern if persistence is ever added.

| # | Task | Effort | Acceptance Criteria |
|---|------|--------|---------------------|
| 3.2.1 | Document current data access approach | S | README explains registry as data layer |
| 3.2.2 | Create migration guide for future persistence | M | Document steps if DB persistence needed |

---

## 5. Risk Assessment and Mitigation

### Risk 1: Registry Test Isolation

**Risk:** Tests may interfere with each other due to singleton registry.
**Mitigation:** Reset registry state between tests or use subtests with cleanup.
**Impact:** Medium
**Likelihood:** High

### Risk 2: Breaking Existing Functionality

**Risk:** Builder validation may reject currently accepted inputs.
**Mitigation:** Analyze existing Kafka consumers to understand valid input ranges.
**Impact:** High
**Likelihood:** Low

### Risk 3: Handler Migration Compatibility

**Risk:** Custom handler types may not map cleanly to framework types.
**Mitigation:** Evaluate framework handler interface before starting migration.
**Impact:** Medium
**Likelihood:** Medium

---

## 6. Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Test Coverage | >80% for processor, registry | `go test -cover ./invite/...` |
| All Tests Pass | 100% | `go test ./... -count=1` returns exit code 0 |
| Transform Encapsulation | No private field access | Code review / static analysis |
| Builder Validation | All invalid inputs rejected | Test suite includes edge cases |
| Documentation | Architecture documented | README.md contains decision section |

---

## 7. Dependencies

### Internal Dependencies

- `github.com/Chronicle20/atlas-tenant` - Tenant model and context
- `github.com/Chronicle20/atlas-model` - Model provider pattern
- `github.com/Chronicle20/atlas-rest/server` - REST handler framework
- `github.com/sirupsen/logrus` - Logging

### Test Dependencies

- `github.com/stretchr/testify/assert` - Assertion helpers
- `github.com/sirupsen/logrus/hooks/test` - Null logger for tests
- `github.com/google/uuid` - UUID generation for test tenants

---

## 8. Execution Order

```
Phase 1 (P0 - Blocking)
├── 1.1 Test Infrastructure Setup
├── 1.2 Fix Transform Encapsulation (ARCH-012)
├── 1.3 Processor Tests (ARCH-013)
└── 1.4 Registry Tests (ARCH-013)

Phase 2 (P1 - Important)
├── 2.1 Builder Pattern (ARCH-003)
│   └── Depends on: Phase 1 complete (tests exist to verify changes)
└── 2.2 Document Architecture Decision (ARCH-002)
    └── Independent - can be done anytime

Phase 3 (P2 - Nice-to-Have)
├── 3.1 Migrate to Framework Handlers (ARCH-011)
│   └── Depends on: Phase 1 complete (tests verify migration)
└── 3.2 Provider Pattern Documentation (ARCH-005)
    └── Independent - can be done anytime
```

---

## 9. Notes and Ambiguities

1. **Registry Singleton Pattern:** The singleton `GetRegistry()` pattern may complicate testing. Consider adding a `ResetRegistry()` function for test isolation, or using dependency injection for testability.

2. **Missing Transaction ID in REST Model:** The audit notes that `RestModel` doesn't include `transactionId`. Verify if this is intentional - transactions are typically internal concerns not exposed via REST.

3. **Package Import Aliases:** Files use `invite2` and `invite3` aliases indicating naming conflicts. This is cosmetic but could be improved.

4. **Concurrent Test Considerations:** When writing registry tests with goroutines, ensure proper synchronization and use `-race` flag during test runs.
