# atlas-character-factory Remediation Tasks

**Last Updated:** 2026-01-13

---

## Deferred: REST-003 (Anti-Pattern Fix)

**Status:** SKIPPED - Requires `atlas-rest` library fix first

The `writeErrorResponse()` and `categorizeError()` functions in `factory/resource.go` cannot be removed until `libs/atlas-rest/requests/post.go` is updated to check HTTP status codes. See context document for details.

---

## Phase 1: Code Organization (P2) - Effort: S

### Task 1.1: Create factory/cache.go
- [ ] Create new file `factory/cache.go`
- [ ] Add package declaration and imports
- [ ] Move `FollowUpSagaTemplate` struct (lines 20-27 of processor.go)
- [ ] Move `FollowUpSagaTemplateStore` struct and methods (lines 28-100)
- [ ] Move singleton variables `templateStoreInstance`, `templateStoreOnce` (lines 34-38)
- [ ] Move `GetFollowUpSagaTemplateStore()` function
- [ ] Move `storeFollowUpSagaTemplate()` function
- [ ] Move `GetFollowUpSagaTemplate()` function
- [ ] Move `RemoveFollowUpSagaTemplate()` function
- [ ] Move `SagaCompletionTracker` struct (lines 121-130)
- [ ] Move `SagaCompletionTrackerStore` struct and methods (lines 132-234)
- [ ] Move singleton variables `sagaTrackerStoreInstance`, `sagaTrackerStoreOnce`
- [ ] Move `GetSagaCompletionTrackerStore()` function
- [ ] Move `StoreFollowUpSagaTracking()` function
- [ ] Move `MarkSagaCompleted()` function
- [ ] Verify processor.go still compiles (imports from cache.go)
- [ ] Run `go test ./...` to verify tests pass

**Acceptance Criteria:**
- `factory/cache.go` contains all cache types and functions
- `factory/processor.go` contains only business logic
- All tests pass

### Task 1.2: Remove Commented Code from character/processor.go
- [ ] Delete commented `byIdProvider` function (lines 9-15)
- [ ] Delete commented `GetById` function (lines 17-23)
- [ ] Delete commented `CreateItem` function (lines 39-49)
- [ ] Delete commented `EquipItem` function (lines 51-70)
- [ ] Remove any unused imports
- [ ] Run `go build ./...` to verify compilation

**Acceptance Criteria:**
- No commented function definitions remain
- File compiles successfully

### Task 1.3: Remove Commented Code from character/requests.go
- [ ] Delete commented `requestById` function (lines 22-24)
- [ ] Delete commented `requestCreateItem` function (lines 50-54)
- [ ] Delete commented `requestEquipItem` function (lines 56-59)
- [ ] Delete commented `requestEquipableItemBySlot` function (lines 61-63)
- [ ] Delete commented `requestItemBySlot` function (lines 65-67)
- [ ] Remove any unused imports
- [ ] Remove unused constants (if any)
- [ ] Run `go build ./...` to verify compilation

**Acceptance Criteria:**
- No commented function definitions remain
- File compiles successfully

### Task 1.4: (Optional) Split saga/model.go Payloads
- [ ] Evaluate if split is worth the effort (454 lines)
- [ ] If proceeding:
  - [ ] Create `saga/payloads.go`
  - [ ] Move all `*Payload` types to payloads.go
  - [ ] Keep `Saga`, `Step`, `Type`, `Status`, `Action`, `ExperienceDistributions` in model.go
  - [ ] Move `UnmarshalJSON` to payloads.go (depends on payload types)
  - [ ] Verify imports work correctly
  - [ ] Run tests

**Acceptance Criteria:**
- If done: `saga/model.go` < 200 lines, `saga/payloads.go` contains payload types
- If skipped: Document reason in task notes

---

## Phase 2: Pattern Alignment (P2) - Effort: M

### Task 2.1: Define Processor Interface
- [ ] Add `Processor` interface in `factory/processor.go`:
  ```go
  type Processor interface {
      Create(ctx context.Context, input RestModel) (string, error)
  }
  ```
- [ ] Verify interface is exported (capital P)

### Task 2.2: Create ProcessorImpl
- [ ] Add `ProcessorImpl` struct:
  ```go
  type ProcessorImpl struct {
      l logrus.FieldLogger
  }
  ```
- [ ] Add constructor:
  ```go
  func NewProcessor(l logrus.FieldLogger) Processor {
      return &ProcessorImpl{l: l}
  }
  ```

### Task 2.3: Convert Create Function to Method
- [ ] Change `Create` from curried function to method:
  - FROM: `func Create(l logrus.FieldLogger) func(ctx context.Context) func(input RestModel) (string, error)`
  - TO: `func (p *ProcessorImpl) Create(ctx context.Context, input RestModel) (string, error)`
- [ ] Update function body to use `p.l` instead of `l` parameter
- [ ] Ensure all internal logic remains unchanged
- [ ] Keep helper functions (`validName`, `validGender`, etc.) as package-level functions
- [ ] Keep saga builder functions as package-level functions

### Task 2.4: Update Resource Handler
- [ ] Modify `handleCreateCharacter` to use Processor interface:
  ```go
  func handleCreateCharacter(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
      return func(w http.ResponseWriter, r *http.Request) {
          processor := NewProcessor(d.Logger())
          transactionId, err := processor.Create(d.Context(), input)
          // ... rest of handler
      }
  }
  ```
- [ ] Alternative: Pass processor via dependency injection if pattern exists

### Task 2.5: Update Tests
- [ ] Review `factory/processor_test.go` for `Create` function calls
- [ ] Update test calls to use new interface:
  ```go
  processor := NewProcessor(logger)
  result, err := processor.Create(ctx, input)
  ```
- [ ] Verify all tests pass
- [ ] Consider adding interface mock for isolation testing

### Task 2.6: Final Verification
- [ ] Run `go build ./...`
- [ ] Run `go test ./...`
- [ ] Verify line count of `factory/processor.go` is reduced
- [ ] Review structure matches `saga/processor.go` pattern

**Acceptance Criteria:**
- `Processor` interface is defined and exported
- `ProcessorImpl` implements `Processor`
- Handler uses processor via interface
- All tests pass
- Pattern consistent with `saga/processor.go`

---

## Verification Checklist

After all phases complete:

- [ ] `go build ./...` passes
- [ ] `go test ./...` passes (all tests)
- [ ] No commented code in `character/` package
- [ ] `factory/cache.go` exists with cache implementations
- [ ] `factory/processor.go` has `Processor` interface
- [ ] Pattern matches guidelines

---

## Notes

- Run tests frequently during refactoring
- Commit after each phase for easy rollback
- If any task fails, stop and investigate before continuing
- REST-003 is deferred - do not modify `factory/resource.go` error handling
