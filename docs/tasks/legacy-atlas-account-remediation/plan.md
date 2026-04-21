# Atlas-Account Service Remediation Plan

**Last Updated:** 2026-01-13
**Status:** COMPLETE
**Audit Reference:** `/dev/audits/atlas-account/audit.md`
**Service Path:** `services/atlas-account`
**Overall Audit Status:** `needs-work` (11 pass, 4 warn, 1 fail)

---

## Executive Summary

The `atlas-account` service audit identified one blocking issue and several non-blocking issues that need remediation to achieve full compliance with Atlas backend architecture guidelines. This plan provides a structured approach to address all identified issues, prioritized by impact and risk.

**Key Issues to Address:**
1. **BLOCKING:** Missing `builder.go` with fluent API and invariant validation (ARCH-003)
2. **SECURITY:** Password logging in debug output (processor.go:135)
3. **MEDIUM:** Limited processor test coverage (ARCH-012)
4. **MEDIUM:** REST Transform function accesses private fields directly (ARCH-008)
5. **LOW:** Provider pattern slight variant from standard (ARCH-005)
6. **LOW:** Custom REST handler abstraction (ARCH-008)
7. **LOW:** State constants inline in model.go (no state.go)

---

## Current State Analysis

### Passing Checks (11/15)
- ARCH-001: Layer Separation
- ARCH-002: Model Immutability
- ARCH-004: Processor Pattern
- ARCH-006: Producer Pattern
- ARCH-007: Multi-Tenancy Context
- ARCH-009: Ingress Configuration
- ARCH-010: Documentation
- ARCH-011: Kafka Consumer Pattern
- ARCH-013: Singleton Cache Pattern
- ARCH-014: Entity Pattern
- ARCH-015: Message Buffer Pattern

### Warning Checks (4/15)
- ARCH-005: Provider Pattern (minor variant)
- ARCH-008: REST JSON:API Pattern (Transform uses private fields)
- ARCH-012: Testing Coverage (1 of ~15 processor methods tested)
- Security: Password logged in debug output

### Failing Checks (1/15)
- ARCH-003: Builder Pattern (missing builder.go)

---

## Proposed Future State

After remediation, the service will:
1. Have a compliant `builder.go` with fluent API and invariant validation
2. Achieve comprehensive processor test coverage (all methods tested)
3. Use model accessors consistently in Transform functions
4. Have state constants extracted to dedicated `state.go`
5. Remove all sensitive data from log output
6. Document the provider pattern variant as acceptable

---

## Implementation Phases

### Phase 1: Critical Security Fix (P0)
**Objective:** Remove password logging from debug output

| Task | Description | Effort | Acceptance Criteria |
|------|-------------|--------|---------------------|
| 1.1 | Remove password from Create log | S | processor.go:135 no longer logs password |
| 1.2 | Audit all log statements | S | No sensitive data (passwords, PINs, PICs) in any log statement |
| 1.3 | Verify fix | S | `grep -r "password\|Password" *.go` shows no log output |

### Phase 2: Blocking Issue - Builder Pattern (P1)
**Objective:** Create builder.go with fluent API and validation

| Task | Description | Effort | Acceptance Criteria |
|------|-------------|--------|---------------------|
| 2.1 | Create builder.go skeleton | S | File created with Builder struct |
| 2.2 | Implement NewBuilder | S | Constructor with required fields (tenantId, name) |
| 2.3 | Add fluent setter methods | M | SetPassword, SetPin, SetPic, SetGender, SetBanned, SetTOS |
| 2.4 | Implement Build() with validation | M | Validates: name not empty, tenantId not nil |
| 2.5 | Refactor administrator.go Make() | S | Make() uses builder internally |
| 2.6 | Add builder unit tests | M | Tests for valid builds and invariant violations |

**Reference Implementation:** `services/atlas-character/atlas.com/character/character/builder.go`

### Phase 3: REST Pattern Compliance (P2)
**Objective:** Fix Transform function and evaluate handler migration

| Task | Description | Effort | Acceptance Criteria |
|------|-------------|--------|---------------------|
| 3.1 | Refactor Transform() to use accessors | S | Uses m.Id(), m.Name(), etc. instead of m.id, m.name |
| 3.2 | Refactor Extract() to use builder | S | Uses builder pattern for model construction |
| 3.3 | Evaluate shared RegisterHandler migration | S | Decision documented in context.md |
| 3.4 | Add missing Gender accessor if needed | S | model.go has Gender() method |

### Phase 4: State Constants Extraction (P2)
**Objective:** Extract state constants to dedicated state.go

| Task | Description | Effort | Acceptance Criteria |
|------|-------------|--------|---------------------|
| 4.1 | Create state.go | S | File with State type and constants |
| 4.2 | Move State type from model.go | S | model.go imports from state.go |
| 4.3 | Move state constants | S | StateNotLoggedIn, StateLoggedIn, StateTransition in state.go |
| 4.4 | Add state helper functions | S | IsLoggedIn(), IsTransition() helper methods |

### Phase 5: Comprehensive Test Coverage (P1)
**Objective:** Add table-driven tests for all processor methods

| Task | Description | Effort | Acceptance Criteria |
|------|-------------|--------|---------------------|
| 5.1 | Create test fixtures | M | setupTestDatabase, sample tenant, mock producer |
| 5.2 | Test Create and CreateAndEmit | S | Happy path + error cases |
| 5.3 | Test GetOrCreate | M | Existing account, auto-register on/off |
| 5.4 | Test Update | M | PIN, PIC, TOS, Gender updates |
| 5.5 | Test Login/Logout | L | State transitions, session handling |
| 5.6 | Test AttemptLogin/AttemptLoginAndEmit | L | All error paths (banned, already logged in, wrong password, too many attempts) |
| 5.7 | Test ProgressState/ProgressStateAndEmit | L | All state transitions |
| 5.8 | Test GetById, GetByName, GetByTenant | M | Found, not found cases |
| 5.9 | Test LoggedInTenantProvider | S | Filter logged-in accounts |
| 5.10 | Add integration test for Teardown | M | Verifies cleanup behavior |

**Test Pattern:** Table-driven tests following `testing-guide.md`

### Phase 6: Provider Pattern Documentation (P3)
**Objective:** Document acceptable provider variant

| Task | Description | Effort | Acceptance Criteria |
|------|-------------|--------|---------------------|
| 6.1 | Document variant in README | S | Explains database.EntityProvider pattern |
| 6.2 | Consider standardization | S | Decision on future alignment |

---

## Risk Assessment and Mitigation

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Builder refactor breaks Make() | Medium | High | Incremental refactor, maintain backward compat temporarily |
| Test coverage takes longer than estimated | Medium | Low | Prioritize critical paths first |
| Transform refactor causes serialization issues | Low | Medium | Verify JSON output unchanged via integration tests |
| State extraction causes import cycles | Low | High | Keep state.go minimal, test compilation after changes |

---

## Success Metrics

| Metric | Current | Target |
|--------|---------|--------|
| Audit Pass Rate | 11/15 (73%) | 15/15 (100%) |
| Processor Test Coverage | 1/15 methods | 15/15 methods |
| Security Issues | 1 (password logging) | 0 |
| Builder Pattern | Missing | Implemented with validation |

---

## Required Resources and Dependencies

### Files to Modify
- `account/processor.go` - Remove password logging
- `account/model.go` - Extract state type (or create state.go)
- `account/administrator.go` - Refactor Make() to use builder
- `account/rest.go` - Use accessor methods in Transform/Extract
- `account/processor_test.go` - Expand test coverage

### Files to Create
- `account/builder.go` - Fluent builder with validation
- `account/state.go` - State constants and helpers

### External Dependencies
- None (all changes internal to service)

### Reference Materials
- `.claude/skills/backend-dev-guidelines/` - Architecture guidelines
- `services/atlas-character/atlas.com/character/character/builder.go` - Builder reference

---

## Execution Order

1. **Phase 1** (P0) - Security fix first - no dependencies
2. **Phase 2** (P1) - Builder pattern - enables Phase 3
3. **Phase 3** (P2) - REST compliance - depends on Phase 2
4. **Phase 4** (P2) - State extraction - parallel with Phase 3
5. **Phase 5** (P1) - Test coverage - can start after Phase 2
6. **Phase 6** (P3) - Documentation - last priority

---

## Notes and Decisions

### Handler Migration Decision
The custom `rest/handler.go` works correctly and follows the same patterns as the shared library. Migration to `server.RegisterHandler` from atlas-rest would be a larger refactor with limited benefit. **Recommendation:** Document as acceptable variant, defer migration to future roadmap.

### Provider Pattern Variant
The current `database.EntityProvider[T]` return type is functionally equivalent to the documented pattern. The difference is stylistic. **Recommendation:** Document as acceptable variant in README.

### Registry Component
The `registry.go` singleton cache pattern is well-implemented and domain-specific. No changes required.
