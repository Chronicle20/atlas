# atlas-configurations Service Remediation Plan

**Last Updated:** 2026-01-13
**Service:** `services/atlas-configurations`
**Audit Reference:** `docs/audits/atlas-configurations/audit.md`
**Overall Audit Status:** `needs-work` (62% pass rate)

---

## Executive Summary

The `atlas-configurations` service audit identified **1 blocking issue** (zero test coverage for domain packages) and **3 non-blocking issues** requiring remediation. The service is a REST-only configuration management service that manages templates, tenants, and service configurations for the Atlas platform.

**Key Finding:** This service intentionally deviates from multi-tenancy patterns because it *manages* tenant configurations rather than operating within a tenant context. This is an acceptable architectural decision that should be documented.

**Key Remediation Goals:**
1. Establish comprehensive test coverage for domain packages (blocking)
2. Document intentional multi-tenancy deviation
3. Consolidate administrator/processor write operations
4. Add immutable domain models if business logic grows (deferred)

---

## Current State Analysis

### Audit Summary

| Metric | Value |
|--------|-------|
| Total Checks | 13 |
| Passing | 8 |
| Warnings | 4 |
| Failures | 2 |
| Pass Rate | 62% |

### Issues by Priority

| Priority | Issue | Effort | Check ID |
|----------|-------|--------|----------|
| P0 | Missing test coverage for domain packages | L | TEST-001 |
| P1 | Document multi-tenancy deviation | S | ARCH-008 |
| P2 | Administrator/Processor overlap - unused functions | S | ARCH-003 |
| P2 | No immutable domain models | M | ARCH-001/002 |

### Architectural Context

The service has intentional deviations from standard Atlas patterns:
- **No multi-tenancy context** - This service manages tenant configurations for other services; it doesn't operate within a tenant scope
- **RestModel as domain model** - For a simple CRUD configuration service with JSON blob storage, the RestModel serves adequately as both API transport and internal representation
- **Services package is read-only** - No administrator.go needed as there are no write operations

These deviations are reasonable for a configuration management service.

### Service Structure

```
services/atlas-configurations/
├── atlas.com/configurations/
│   ├── main.go                    # Service entry point
│   ├── database/                  # Database utilities
│   ├── logger/                    # Logging setup
│   ├── rest/                      # REST handler utilities
│   ├── retry/                     # Retry utilities
│   ├── seeder/                    # Seed data import (has tests)
│   ├── service/                   # Service lifecycle
│   ├── services/                  # Services domain (read-only)
│   ├── templates/                 # Templates domain (CRUD)
│   ├── tenants/                   # Tenants domain (CRUD)
│   └── tracing/                   # Distributed tracing
```

---

## Proposed Future State

After remediation:
- **Comprehensive test coverage** on all domain packages (templates, tenants, services)
- **Documented architectural decisions** explaining multi-tenancy deviation
- **Consistent code organization** with administrator.go used for write operations
- **Clean codebase** with no unused functions

---

## Implementation Phases

### Phase 1: Test Infrastructure (P0 - Blocking)
**Goal:** Establish test coverage for all domain packages

This phase addresses the only blocking issue. The seeder package already has tests; this phase adds tests for:
- Templates package (processor methods, REST transforms)
- Tenants package (processor methods, REST transforms, history tracking)
- Services package (processor methods, REST transforms)

### Phase 2: Documentation (P1)
**Goal:** Document intentional architectural deviations

Add clear documentation explaining why this service doesn't use multi-tenancy context patterns.

### Phase 3: Code Cleanup (P2)
**Goal:** Consolidate write operations and remove unused code

Either:
- Option A: Remove administrator.go files and keep write operations in processor.go (simpler, current pattern)
- Option B: Refactor processor to delegate to administrator.go (aligns with guidelines)

Recommendation: Option A - This service's simplicity doesn't benefit from the separation.

### Phase 4: Model Improvements (P2 - Deferred)
**Goal:** Add immutable domain models if validation/business logic is added

Currently deferred as the service is a simple CRUD service with JSON blob storage. This phase should be implemented if:
- Validation logic is added to templates or tenants
- Business rules need to be enforced on configuration data
- Multiple transformation layers are introduced

---

## Detailed Tasks

### Phase 1: Test Infrastructure

#### 1.1 Create templates/processor_test.go
**Effort:** M | **Files:** `templates/processor_test.go` (new)

Test processor methods:
- `GetAll()` returns all templates
- `GetById()` returns specific template
- `GetByRegionAndVersion()` returns matching template
- `Create()` persists new template and returns UUID
- `UpdateById()` modifies existing template
- `DeleteById()` removes template

**Acceptance Criteria:**
- [ ] All processor methods have test coverage
- [ ] Tests use in-memory SQLite database
- [ ] CRUD operations verified end-to-end
- [ ] Edge cases covered (not found, duplicate region/version)

#### 1.2 Create templates/rest_test.go
**Effort:** S | **Files:** `templates/rest_test.go` (new)

Test REST model:
- `GetName()` returns "templates"
- `GetID()` returns template ID
- `SetID()` updates ID correctly
- JSON marshaling/unmarshaling works correctly

**Acceptance Criteria:**
- [ ] JSON:API interface methods verified
- [ ] Nested structures serialize correctly

#### 1.3 Create tenants/processor_test.go
**Effort:** M | **Files:** `tenants/processor_test.go` (new)

Test processor methods:
- `GetAll()` returns all tenants
- `GetById()` returns specific tenant
- `GetByRegionAndVersion()` returns matching tenant
- `Create()` persists new tenant (with optional ID)
- `UpdateById()` modifies existing tenant
- `DeleteById()` removes tenant

**Acceptance Criteria:**
- [ ] All processor methods have test coverage
- [ ] Tests use in-memory SQLite database
- [ ] ID can be provided or auto-generated
- [ ] History tracking verified (if applicable)

#### 1.4 Create tenants/rest_test.go
**Effort:** S | **Files:** `tenants/rest_test.go` (new)

Test REST model:
- `GetName()` returns "tenants"
- `GetID()` returns tenant ID
- `SetID()` updates ID correctly
- Nested configuration structures serialize correctly

**Acceptance Criteria:**
- [ ] JSON:API interface methods verified
- [ ] Nested structures serialize correctly

#### 1.5 Create services/processor_test.go
**Effort:** S | **Files:** `services/processor_test.go` (new)

Test processor methods (read-only):
- Provider methods return correct data
- Entity to RestModel transformation works

**Acceptance Criteria:**
- [ ] Read operations verified
- [ ] Transformation logic tested

#### 1.6 Create services/rest_test.go
**Effort:** S | **Files:** `services/service/rest_test.go`, `services/task/rest_test.go` (new)

Test REST models:
- JSON:API interface methods
- Nested structure serialization

**Acceptance Criteria:**
- [ ] All REST models have interface tests

---

### Phase 2: Documentation

#### 2.1 Add Architecture Notes to README
**Effort:** S | **Files:** `README.md`

Add section explaining architectural decisions:

```markdown
## Architecture Notes

### Multi-Tenancy
This service intentionally does not use the standard `tenant.MustFromContext(ctx)`
pattern. Unlike other Atlas services that operate within a tenant context, this
service *manages* tenant configurations. The "tenant" here is the configuration
being managed, not the request context.

### Domain Model Layer
This service uses RestModel directly as the domain representation. This is
acceptable for a simple CRUD configuration service with JSON blob storage
(`json.RawMessage`). If validation or business logic is added, consider
introducing model.go/builder.go for proper domain separation.

### Services Package
The services domain is read-only (no Create/Update/Delete operations), so it
intentionally lacks an administrator.go file.
```

**Acceptance Criteria:**
- [ ] Multi-tenancy deviation documented
- [ ] Domain model decision documented
- [ ] Services package read-only nature documented

#### 2.2 Add inline code comments
**Effort:** S | **Files:** `templates/processor.go`, `tenants/processor.go`, `services/processor.go`

Add comments at the top of processor files explaining why multi-tenancy context is not used.

**Acceptance Criteria:**
- [ ] Each processor file has explanatory comment
- [ ] Comments reference README for details

---

### Phase 3: Code Cleanup

#### 3.1 Remove unused administrator.go functions
**Effort:** S | **Files:** `templates/administrator.go`, `tenants/administrator.go`

Option A (Recommended): Keep write operations in processor.go, remove unused functions from administrator.go.

Currently:
- `templates/administrator.go` has `create()`, `update()`, `delete()` functions
- `templates/processor.go` has its own Create logic, but calls `update()` and `delete()` from administrator.go
- This creates confusion about where write operations should live

Resolution:
- Keep processor.go as the source of truth for all operations
- Either remove administrator.go entirely OR have it contain only reusable transaction functions called by processor

**Acceptance Criteria:**
- [ ] No unused functions in administrator.go
- [ ] Clear ownership of write operations
- [ ] All tests pass after cleanup

---

### Phase 4: Model Improvements (Deferred)

#### 4.1 Add model.go and builder.go for templates (DEFERRED)
**Effort:** M | **Files:** `templates/model.go`, `templates/builder.go` (new)

Only implement if validation or business logic is added.

**Trigger conditions for implementation:**
- Validation rules added to template data
- Business logic beyond simple CRUD
- Multiple consumers of template data with different transformation needs

#### 4.2 Add model.go and builder.go for tenants (DEFERRED)
**Effort:** M | **Files:** `tenants/model.go`, `tenants/builder.go` (new)

Same trigger conditions as templates.

---

## Risk Assessment

### High Risk
| Risk | Mitigation |
|------|------------|
| Tests interfering with production data | Use in-memory SQLite for tests |
| Breaking existing seed functionality | Ensure seeder tests still pass |

### Medium Risk
| Risk | Mitigation |
|------|------------|
| Removing code that is actually used | Search for all usages before deletion |
| Documentation becoming stale | Include in PR review checklist |

### Low Risk
| Risk | Mitigation |
|------|------------|
| Model layer deferred too long | Add to tech debt tracking |

---

## Success Metrics

| Metric | Current | Target |
|--------|---------|--------|
| Test Coverage (domain packages) | 0% | >70% |
| Audit Pass Rate | 62% | >85% |
| Blocking Issues | 1 | 0 |
| Warnings | 4 | 2 (documented intentional deviations) |

---

## Dependencies

### Internal
- Atlas test utilities for database mocking
- Shared test patterns from other services

### External
- `github.com/DATA-DOG/go-sqlmock` or similar for database testing
- Or: In-memory SQLite (`gorm.io/driver/sqlite`)

---

## Notes

### Intentional Deviations (Document, Do Not Fix)
These findings are architectural decisions that should be documented rather than changed:
- **ARCH-008** (Multi-Tenancy Context): Service manages tenants, doesn't operate within tenant context
- **ARCH-001/002** (Immutable Models): RestModel is sufficient for simple CRUD with JSON blobs

### Test Strategy Recommendations
1. **Use in-memory SQLite** for integration tests rather than mocking GORM
2. **Table-driven tests** for CRUD operations with various inputs
3. **Test the Make() function** that transforms Entity to RestModel
4. **Verify JSON serialization** of nested configuration structures

### Existing Test Pattern
The `seeder/seeder_test.go` provides a good example of the testing style to follow:
- Uses testdata directory for fixtures
- Table-driven tests for variations
- Clear setup/teardown patterns
