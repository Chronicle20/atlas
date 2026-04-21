# atlas-character-factory Remediation Plan

**Last Updated:** 2026-01-13

---

## 1. Executive Summary

This plan addresses the non-blocking issues identified in the `atlas-character-factory` service audit. The service is a specialized microservice for character creation saga orchestration that differs from standard CRUD services (no database, no GORM, no entities/providers).

**Overall Audit Status:** NEEDS-WORK (no blocking issues)

**Scope:** 4 remediation objectives across 2 phases
- 4 P2 issues (code organization and pattern alignment)

**Estimated Total Effort:** Small to Medium

---

## 2. Current State Analysis

### 2.1 Issues Summary

| ID | Issue | Priority | Impact | Effort | Status |
|----|-------|----------|--------|--------|--------|
| REST-003 | Custom `writeErrorResponse()` helper violates anti-patterns | P1 | MEDIUM | S | **SKIPPED** |
| ARCH-002 | Singleton caches mixed in processor.go | P2 | LOW | S | In Scope |
| ARCH-005 | `factory/Create()` uses curried function instead of Processor interface | P2 | MEDIUM | M | In Scope |
| COMMENTED | Commented code in character package | P2 | LOW | S | In Scope |
| SAGA-SPLIT | saga/model.go is 454 lines with many payload types | P2 | LOW | S | In Scope |

### 2.2 REST-003 Exclusion Rationale

**REST-003 is excluded from this remediation** due to a breaking change risk:

The `atlas-rest` library's POST request handler (`libs/atlas-rest/requests/post.go`) does not check HTTP status codes. It only checks:
- If `ContentLength == 0` → returns success with empty result
- Otherwise → attempts to unmarshal response body as JSON:API

If we remove the error response body and only use `w.WriteHeader(statusCode)`:
- The response would have `ContentLength == 0`
- The `atlas-rest` library would return `(result, nil)` — **treating the error as success**
- `atlas-login` would not receive an error and would not show the user an error message

**Prerequisite:** Fix `atlas-rest` POST handler to check HTTP status codes (like GET handler does) before addressing REST-003.

### 2.3 Key Files Affected

| File | Lines | Issues |
|------|-------|--------|
| `factory/processor.go` | 520 | ARCH-002 (singleton caches), ARCH-005 (Create function) |
| `character/processor.go` | 71 | COMMENTED (lines 9-23, 39-70) |
| `character/requests.go` | 68 | COMMENTED (lines 22-67) |
| `saga/model.go` | 454 | SAGA-SPLIT (many payload types) |

### 2.4 Architectural Context

This service is intentionally different from standard Atlas microservices:
- **No Database Layer** - No GORM, entities, or persistence
- **No Provider Pattern** - Data comes from other services via REST
- **Saga Initiator** - Acts as a saga initiator, not a saga participant
- **Event-Driven** - Heavy use of Kafka for coordination

---

## 3. Proposed Future State

### 3.1 Target Structure

After remediation:

```
factory/
├── cache.go           # NEW: FollowUpSagaTemplateStore, SagaCompletionTrackerStore
├── processor.go       # MODIFIED: Processor interface + business logic only
├── resource.go        # UNCHANGED (REST-003 skipped)
└── rest.go            # UNCHANGED

saga/
├── model.go           # MODIFIED: Core types only (optional)
├── payloads.go        # NEW: All payload type definitions (optional)
├── builder.go         # UNCHANGED
└── processor.go       # UNCHANGED

character/
├── model.go           # UNCHANGED
├── processor.go       # MODIFIED: Commented code removed
├── requests.go        # MODIFIED: Commented code removed
└── rest.go            # UNCHANGED
```

### 3.2 Pattern Alignment

**Factory Processor Interface:**
```go
// factory/processor.go
type Processor interface {
    Create(ctx context.Context, input RestModel) (string, error)
}

type ProcessorImpl struct {
    l logrus.FieldLogger
}

func NewProcessor(l logrus.FieldLogger) Processor {
    return &ProcessorImpl{l: l}
}

func (p *ProcessorImpl) Create(ctx context.Context, input RestModel) (string, error) {
    // Business logic here
}
```

---

## 4. Implementation Phases

### Phase 1: Code Organization (P2)

**Objective:** Improve file organization by separating concerns.

**Tasks:**
1. Create `factory/cache.go` with singleton cache types:
   - Move `FollowUpSagaTemplate` struct
   - Move `FollowUpSagaTemplateStore` struct and methods
   - Move `SagaCompletionTracker` struct
   - Move `SagaCompletionTrackerStore` struct and methods
   - Move exported convenience functions (`GetFollowUpSagaTemplate`, `RemoveFollowUpSagaTemplate`, etc.)

2. Remove commented code from character package:
   - Delete lines 9-23 in `character/processor.go` (byIdProvider, GetById)
   - Delete lines 39-70 in `character/processor.go` (CreateItem, EquipItem)
   - Delete lines 22-24 in `character/requests.go` (requestById)
   - Delete lines 50-67 in `character/requests.go` (requestCreateItem, requestEquipItem, requestEquipableItemBySlot, requestItemBySlot)

3. Create `saga/payloads.go` (optional, consider scope):
   - Move all `*Payload` type definitions from `saga/model.go`
   - Keep core types (`Saga`, `Step`, `Type`, `Status`, `Action`) in `saga/model.go`

**Acceptance Criteria:**
- `factory/cache.go` contains all singleton cache implementations
- `factory/processor.go` contains only business logic (reduced from 520 lines)
- No commented code remains in `character/` package
- All existing tests pass
- Code compiles without errors

### Phase 2: Pattern Alignment (P2)

**Objective:** Refactor factory to use Processor interface pattern.

**Tasks:**
1. Define `Processor` interface in `factory/processor.go`:
   ```go
   type Processor interface {
       Create(ctx context.Context, input RestModel) (string, error)
   }
   ```

2. Create `ProcessorImpl` struct:
   ```go
   type ProcessorImpl struct {
       l logrus.FieldLogger
   }
   ```

3. Add constructor function:
   ```go
   func NewProcessor(l logrus.FieldLogger) Processor {
       return &ProcessorImpl{l: l}
   }
   ```

4. Convert `Create` from curried function to method:
   - Change from: `func Create(l logrus.FieldLogger) func(ctx context.Context) func(input RestModel) (string, error)`
   - Change to: `func (p *ProcessorImpl) Create(ctx context.Context, input RestModel) (string, error)`

5. Update `factory/resource.go` to use the new Processor:
   - Inject processor via handler dependency or create inline

6. Update all tests to use the new interface

**Acceptance Criteria:**
- `factory/processor.go` defines `Processor` interface
- `ProcessorImpl` implements `Processor` interface
- Handler uses processor via interface (testable/mockable)
- Pattern matches `saga/processor.go` structure
- All existing tests pass or are updated

---

## 5. Risk Assessment and Mitigation

### 5.1 Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Test breakage during refactor | Medium | Low | Run tests after each file change |
| Import cycle after file split | Low | Medium | Plan package boundaries carefully |

### 5.2 Mitigation Strategies

1. **Incremental Changes:** Each phase should be a separate commit
2. **Test-First Verification:** Run `go test ./...` after each significant change

---

## 6. Success Metrics

| Metric | Target |
|--------|--------|
| All tests passing | 100% |
| No commented code in character/ | 0 lines |
| Processor interface defined | Yes |
| factory/processor.go line count | < 350 lines |

---

## 7. Required Resources and Dependencies

### 7.1 Dependencies

- No external dependencies required
- All changes are internal refactoring

### 7.2 Files to Modify/Create

**Create:**
- `factory/cache.go`
- `saga/payloads.go` (optional)

**Modify:**
- `factory/processor.go`
- `character/processor.go`
- `character/requests.go`
- `saga/model.go` (optional)

**Test Files (may need updates):**
- `factory/processor_test.go`
- `factory/singleton_test.go`

---

## 8. Notes and Considerations

### 8.1 Architectural Justification

This service is intentionally different from standard CRUD services. The remediation maintains this architecture while improving code organization and pattern consistency.

### 8.2 Optional Items

The saga/model.go split (SAGA-SPLIT) is marked as "consider" rather than required. At 454 lines, it's large but manageable. Evaluate based on time constraints.

### 8.3 Deferred Items

**REST-003** is deferred until the `atlas-rest` library is updated to properly check HTTP status codes for POST requests. A separate task should be created to:
1. Update `libs/atlas-rest/requests/post.go` to check `r.StatusCode`
2. Then revisit REST-003 in `atlas-character-factory`

### 8.4 Future Considerations

The audit noted that `FollowUpSagaTemplateStore` and `SagaCompletionTrackerStore` have no TTL or cleanup mechanism. This is not addressed in this remediation but should be considered for future work.
