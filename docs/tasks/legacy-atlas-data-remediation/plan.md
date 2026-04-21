# Atlas-Data Service Remediation Plan

**Service Path:** `services/atlas-data/atlas.com/data`
**Audit Reference:** `dev/audits/atlas-data/audit.md`
**Last Updated:** 2026-01-13

---

## 1. Executive Summary

This plan addresses issues identified in the atlas-data service audit. The service is a specialized read-only data service that parses game XML files and serves configuration data via REST API. While the service is architecturally sound and functional, the audit identified several areas for improvement:

- **P2 Issues:** Error logging gaps, misplaced code organization
- **P3 Issues:** Missing test coverage, documentation needs

**Total Estimated Effort:** Medium (2-3 days of focused work)

**Key Outcomes:**
- Improved observability through consistent error logging
- Better code organization following project conventions
- Comprehensive handler test coverage
- Documented service-specific patterns

---

## 2. Current State Analysis

### Service Overview

The atlas-data service handles:
- XML game data file parsing (WZ format)
- Document storage with hybrid caching (in-memory + database)
- REST API endpoints for 15+ domain types
- Kafka command processing for data loading

### Issues Identified

| ID | Category | Status | Impact | Priority |
|----|----------|--------|--------|----------|
| REST-003 | Error Handling | WARN | Low | P2 |
| ARCH-003 | Code Organization | WARN | Low | P2 |
| TEST-001 | Test Coverage | WARN | Medium | P3 |
| ARCH-001 | Documentation | WARN | Low | P3 |

### Files Requiring Changes

**Error Logging (7 files):**
- `consumable/resource.go`
- `cash/resource.go`
- `commodity/resource.go`
- `etc/resource.go`
- `pet/resource.go`
- `setup/resource.go`
- `map/resource.go`

**Code Organization (1 file):**
- `map/resource.go` - Move models to `map/rest.go`

**New Test Files (15 packages):**
- All domain packages need `resource_test.go`

---

## 3. Proposed Future State

### Error Handling Pattern

All 500 status responses will follow this pattern:

```go
if err != nil {
    d.Logger().WithError(err).Errorf("Unable to retrieve %s data.", domainType)
    w.WriteHeader(http.StatusInternalServerError)
    return
}
```

### Code Organization

`map/rest.go` will contain all REST models:
- `RestModel` (existing)
- `DropPositionRestModel` (moved from resource.go)
- `PositionRestModel` (moved from resource.go)
- `FootholdRestModel` (moved from resource.go)

### Test Coverage

Each domain package will have `resource_test.go` with:
- Endpoint integration tests
- Error handling verification
- JSON:API compliance checks
- Tenant isolation tests (where applicable)

---

## 4. Implementation Phases

### Phase 1: Error Logging Remediation (P2)

**Objective:** Add consistent error logging to all 500 response paths

**Scope:** 7 resource.go files with missing error logging

**Tasks:**
1. Add error logging to `consumable/resource.go`
2. Add error logging to `cash/resource.go`
3. Add error logging to `commodity/resource.go`
4. Add error logging to `etc/resource.go`
5. Add error logging to `pet/resource.go`
6. Add error logging to `setup/resource.go`
7. Add error logging to `map/resource.go`

**Acceptance Criteria:**
- All `WriteHeader(http.StatusInternalServerError)` calls preceded by error logging
- Logging includes context (domain type, operation, relevant IDs)
- Consistent log level (Error for 500s, Debug for 404s)

**Effort:** S

---

### Phase 2: Code Organization Remediation (P2)

**Objective:** Move misplaced request models to appropriate files

**Scope:** `map/resource.go` and `map/rest.go`

**Tasks:**
1. Move `DropPositionRestModel` from `map/resource.go` to `map/rest.go`
2. Move `PositionRestModel` from `map/resource.go` to `map/rest.go`
3. Move `FootholdRestModel` from `map/resource.go` to `map/rest.go`
4. Add necessary imports to `map/rest.go`
5. Verify handlers still compile and function correctly

**Acceptance Criteria:**
- All JSON:API REST models in `rest.go`
- No model definitions in `resource.go`
- Existing functionality preserved
- All tests pass

**Effort:** S

---

### Phase 3: Handler Test Implementation (P3)

**Objective:** Add comprehensive handler integration tests

**Scope:** All 15 domain packages

**Tasks:**

#### 3.1 Test Infrastructure Setup
1. Create shared test utilities package
2. Implement mock ServerInformation
3. Create test database setup helpers
4. Implement tenant context request builder

#### 3.2 Core Domain Tests (High Traffic Endpoints)
5. Add `map/resource_test.go`
6. Add `monster/resource_test.go`
7. Add `npc/resource_test.go`
8. Add `skill/resource_test.go`
9. Add `equipment/resource_test.go`
10. Add `consumable/resource_test.go`

#### 3.3 Secondary Domain Tests
11. Add `cash/resource_test.go`
12. Add `commodity/resource_test.go`
13. Add `etc/resource_test.go`
14. Add `pet/resource_test.go`
15. Add `quest/resource_test.go`
16. Add `reactor/resource_test.go`
17. Add `setup/resource_test.go`
18. Add `characters/templates/resource_test.go`

#### 3.4 Data Upload Tests
19. Add `data/resource_test.go` for upload endpoint

**Test Coverage Requirements:**
- GET endpoints (single resource and collections)
- Query parameter handling
- Error responses (404, 500)
- JSON:API response structure validation
- Tenant header handling

**Acceptance Criteria:**
- Each domain has resource_test.go
- Tests cover success and error paths
- JSON:API compliance verified
- Tests run in CI pipeline

**Effort:** M-L

---

### Phase 4: Documentation (P3)

**Objective:** Document service-specific pattern deviations

**Scope:** Guidelines documentation

**Tasks:**
1. Document read-only data service pattern
2. Explain handler→storage direct access rationale
3. Document RestModel as domain model pattern
4. Document generic document entity pattern

**Acceptance Criteria:**
- Pattern deviations documented with rationale
- Future maintainers understand design choices
- Documentation in appropriate location (guidelines or service README)

**Effort:** S

---

## 5. Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Logging changes affect performance | Low | Low | Use appropriate log levels (Error/Debug) |
| Model moves break imports | Low | Medium | Verify all imports after move |
| Test setup complexity | Medium | Low | Reuse patterns from atlas-marriages |
| Test maintenance burden | Medium | Low | Focus on critical paths, not 100% coverage |

---

## 6. Success Metrics

| Metric | Current | Target |
|--------|---------|--------|
| 500 responses with logging | 0% | 100% |
| Request models in rest.go | 0/3 | 3/3 |
| Domain packages with handler tests | 0/15 | 15/15 |
| Documentation coverage | None | Complete |

---

## 7. Dependencies

### Internal Dependencies
- atlas-rest package (for test utilities)
- atlas-model package (for Provider patterns)
- atlas-tenant package (for context handling)

### External Dependencies
- testify (assertion library)
- httptest (standard library)
- sqlite3 (test database driver)

---

## 8. Resource Requirements

### Skills Needed
- Go HTTP handler testing
- JSON:API specification knowledge
- Atlas backend patterns familiarity

### Time Allocation
- Phase 1: 1-2 hours
- Phase 2: 1 hour
- Phase 3: 6-8 hours
- Phase 4: 1-2 hours

---

## 9. Implementation Order

Recommended execution order based on dependencies and priorities:

1. **Phase 1** - Error logging (no dependencies, quick wins)
2. **Phase 2** - Code organization (no dependencies, improves maintainability)
3. **Phase 3.1** - Test infrastructure (required before domain tests)
4. **Phase 3.2** - Core domain tests (highest value)
5. **Phase 3.3** - Secondary domain tests (complete coverage)
6. **Phase 3.4** - Data upload tests (specialized endpoint)
7. **Phase 4** - Documentation (captures decisions for future reference)

---

## 10. Verification Steps

After completing each phase:

1. **Phase 1:** Run service, trigger errors, verify logs appear
2. **Phase 2:** `go build ./...` succeeds, existing tests pass
3. **Phase 3:** `go test ./...` passes, coverage report reviewed
4. **Phase 4:** Documentation reviewed by team member

Final verification:
```bash
cd services/atlas-data
go build ./...
go test ./... -v
go test ./... -cover
```
