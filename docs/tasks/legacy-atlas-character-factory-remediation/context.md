# atlas-character-factory Remediation Context

**Last Updated:** 2026-01-13

---

## 1. Key Files

### 1.1 Files to Modify

| File | Path | Purpose | Changes Required |
|------|------|---------|------------------|
| processor.go | `services/atlas-character-factory/atlas.com/character-factory/factory/processor.go` | Business logic + caches | Extract caches, add Processor interface |
| character/processor.go | `services/atlas-character-factory/atlas.com/character-factory/character/processor.go` | Cross-service client | Remove commented code |
| character/requests.go | `services/atlas-character-factory/atlas.com/character-factory/character/requests.go` | REST client functions | Remove commented code |
| saga/model.go | `services/atlas-character-factory/atlas.com/character-factory/saga/model.go` | Saga types | Optional: split payloads |

### 1.2 Files to Create

| File | Path | Purpose |
|------|------|---------|
| cache.go | `services/atlas-character-factory/atlas.com/character-factory/factory/cache.go` | Singleton cache implementations |
| payloads.go | `services/atlas-character-factory/atlas.com/character-factory/saga/payloads.go` | Optional: payload type definitions |

### 1.3 Files NOT Modified (Deferred)

| File | Path | Reason |
|------|------|--------|
| resource.go | `services/atlas-character-factory/atlas.com/character-factory/factory/resource.go` | REST-003 deferred - requires atlas-rest fix first |

### 1.4 Reference Files

| File | Path | Purpose |
|------|------|---------|
| saga/processor.go | `services/atlas-character-factory/atlas.com/character-factory/saga/processor.go` | Example Processor interface pattern |
| audit.md | `dev/audits/atlas-character-factory/audit.md` | Full audit findings |
| audit.json | `dev/audits/atlas-character-factory/audit.json` | Structured audit data |

---

## 2. Key Decisions

### 2.1 REST-003 Deferred

**Decision:** Skip removal of custom error helpers (`writeErrorResponse`, `categorizeError`).

**Rationale:** The `atlas-rest` library's POST handler does not check HTTP status codes. If we remove the error response body:
- `ContentLength == 0` triggers success path in `libs/atlas-rest/requests/post.go:56`
- `atlas-login` would not receive an error
- Character creation failures would appear as successes to users

**Prerequisite:** Fix `libs/atlas-rest/requests/post.go` to check `r.StatusCode` like `get.go` does.

### 2.2 Processor Interface Pattern

**Decision:** Convert `Create` curried function to Processor interface method.

**Rationale:** Aligns with established pattern in `saga/processor.go` and improves testability.

**Reference Implementation (saga/processor.go):**
```go
type Processor interface {
    Create(s Saga) error
}

type ProcessorImpl struct {
    l   logrus.FieldLogger
    ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
    return &ProcessorImpl{l: l, ctx: ctx}
}
```

### 2.3 Cache Separation

**Decision:** Extract singleton caches to dedicated `factory/cache.go`.

**Rationale:** Follows file responsibility alignment (ARCH-002). Processor files should contain business logic only.

**What to Move:**
- `FollowUpSagaTemplate` struct (lines 20-27)
- `FollowUpSagaTemplateStore` struct and all methods (lines 28-100)
- `SagaCompletionTracker` struct (lines 121-130)
- `SagaCompletionTrackerStore` struct and all methods (lines 132-234)

---

## 3. Dependencies

### 3.1 Internal Dependencies

| From | To | Type |
|------|-----|------|
| factory/resource.go | factory/processor.go | Processor interface |
| factory/processor.go | factory/cache.go | Cache stores (after refactor) |
| factory/processor.go | saga/processor.go | Saga creation |
| kafka/consumer/* | factory/processor.go | Cache access functions |

### 3.2 Test Dependencies

| Test File | Dependencies |
|-----------|--------------|
| factory/processor_test.go | Create function, cache stores |
| factory/singleton_test.go | Cache store singleton patterns |

---

## 4. Code Snippets

### 4.1 Commented Code to Remove (character/processor.go)

```go
// Lines 9-23
//func byIdProvider(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) model.Provider[Model] {
//  ...
//}

// Lines 39-70
//func CreateItem(l logrus.FieldLogger) ...
//func EquipItem(l logrus.FieldLogger) ...
```

### 4.2 Commented Code to Remove (character/requests.go)

```go
// Lines 22-24
//func requestById(id uint32) requests.Request[RestModel] {
//  ...
//}

// Lines 50-67
//func requestCreateItem(...) ...
//func requestEquipItem(...) ...
//func requestEquipableItemBySlot(...) ...
//func requestItemBySlot(...) ...
```

---

## 5. Verification Commands

```bash
# Navigate to service directory
cd services/atlas-character-factory/atlas.com/character-factory

# Run all tests
go test ./...

# Check for compilation errors
go build ./...

# Verify no commented code remains (after cleanup)
grep -rn "^//" character/*.go | grep -v "^character/rest.go" | grep "func"
```

---

## 6. Related Audit Checks

| Check ID | Status | Relevance |
|----------|--------|-----------|
| REST-003 | FAIL | **DEFERRED** - requires atlas-rest fix |
| ARCH-002 | WARN | Direct target of Phase 1 |
| ARCH-005 | WARN | Direct target of Phase 2 |
| ARCH-001 | PASS | Layer separation already correct |
| ARCH-006 | PASS | Singleton pattern already correct |

---

## 7. Deferred Work: atlas-rest Fix

When addressing REST-003 in the future, the following changes are needed in `libs/atlas-rest/requests/post.go`:

```go
// Current (problematic):
if r.ContentLength == 0 {
    // Returns success!
    return result, nil
}

// Should be (like get.go):
if r.StatusCode == http.StatusOK || r.StatusCode == http.StatusAccepted || r.StatusCode == http.StatusCreated {
    if r.ContentLength == 0 {
        return result, nil
    }
    return processResponse[A](r)
}
if r.StatusCode == http.StatusBadRequest {
    return result, errors.New("bad request")
}
// etc.
```
