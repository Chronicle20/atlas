
---
title: Testing Conventions
description: Testing patterns and practices for Atlas Golang microservices.
---

# Testing Conventions


## Focus Areas

1. **Builders** — Validate invariants.
2. **Processors** — Test pure and `AndEmit` forms separately.
3. **Providers** — Validate retrieval and error paths.
4. **REST** — Verify status mapping and JSON:API output.

## Guidelines
- Prefer table-driven tests.
- Mock Kafka producers and DB providers.
- Verify tenant + span propagation.

## Example
```go
func TestBuilderValidation(t *testing.T) {
  _, err := NewBuilder().SetId(0).Build()
  require.Error(t, err)
}
```

---

## Interface Change Workflow

**CRITICAL:** When modifying any interface, follow this checklist to prevent compilation failures and ensure mock synchronization.

### Checklist for Interface Changes

When adding, modifying, or removing methods from an interface:

- [ ] **Update the interface definition** in the primary file (e.g., `processor.go`)
- [ ] **Update ALL mock implementations** in `mock/` directories
  - Add corresponding function fields to mock struct
  - Implement the new methods with proper nil-check and default behavior
  - Follow existing mock patterns (return nil error if function not set)
- [ ] **Run full test suite** in the current service: `go test ./... -count=1`
- [ ] **Search for other usages** across services that may depend on this interface
- [ ] **Update integration tests** that may depend on the interface behavior
- [ ] **Document the change** if it affects service contracts or behavior

### Mock Implementation Pattern

When adding a method to an interface, the mock must include:

1. **Function field** in the mock struct:
```go
type ProcessorMock struct {
    AwardFameAndEmitFunc func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount int16) error
    AwardFameFunc        func(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount int16) error
}
```

2. **Method implementation** with nil-check:
```go
func (m *ProcessorMock) AwardFameAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount int16) error {
    if m.AwardFameAndEmitFunc != nil {
        return m.AwardFameAndEmitFunc(transactionId, worldId, characterId, channelId, amount)
    }
    return nil
}

func (m *ProcessorMock) AwardFame(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount int16) error {
    if m.AwardFameFunc != nil {
        return m.AwardFameFunc(mb)
    }
    return func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount int16) error {
        return nil
    }
}
```

### Finding Mock Files

Common locations for mocks that must be updated:
- `{package}/mock/processor.go` - Service-specific processor mocks
- `{package}/mock/provider.go` - Provider interface mocks
- Test files using inline mock implementations

**Tip:** Use `grep -r "type.*Mock struct" .` to find all mock implementations in a service.

---

## Test Execution Standards

### When to Run Tests

Run the **full test suite** before committing in these situations:

1. **Interface modifications** - Any change to a Processor, Provider, or Administrator interface
2. **Shared package changes** - Modifications to code used across multiple services
3. **Business logic changes** - Any processor or administrator function modification
4. **Breaking changes** - Changes that could affect callers or consumers
5. **Before creating a PR** - Always run full suite before pushing

### Running Tests

**Full test suite (no cache):**
```bash
go test ./... -count=1
```

**With race detection:**
```bash
go test ./... -race -count=1
```

**With coverage:**
```bash
go test ./... -cover -count=1
```

**Specific package:**
```bash
go test ./character/... -v -count=1
```

### Test Failure Protocol

When tests fail:

1. **Do not ignore** - Failing tests indicate broken contracts
2. **Understand the failure** - Read the error message completely
3. **Fix the root cause** - Don't just update tests to pass
4. **Update mocks if needed** - Ensure mocks implement all interface methods
5. **Re-run full suite** - Verify the fix didn't break other tests
6. **Check dependent services** - Some changes may affect other microservices

---

## Mock Maintenance

### Why Mocks Must Stay in Sync

Mocks exist to enable isolated unit testing. When an interface changes but mocks don't:
- **Compilation fails** - Go compiler catches missing methods
- **Tests become unreliable** - Incomplete mocks lead to false positives
- **Integration breaks** - Real implementations diverge from tested behavior

### Mock Synchronization Rules

1. **One-to-one correspondence** - Every interface method must have a mock implementation
2. **Signature matching** - Mock method signatures must exactly match the interface
3. **Default behavior** - Mocks should have sensible defaults (usually nil/empty/no-op)
4. **Testability** - Mock fields should allow tests to inject custom behavior

### Verification

After updating mocks, verify with:

```bash
# Ensure mocks compile
go test ./*/mock/... -count=1

# Run tests that use mocks
go test ./... -run TestProcessor -count=1
```

### Common Mock Update Errors

**Missing method entirely:**
```
cannot use &ProcessorMock{} as Processor value in variable declaration:
*ProcessorMock does not implement Processor (missing method AwardFame)
```
**Fix:** Add the method to the mock struct and implement it.

**Wrong signature:**
```
method AwardFame has wrong signature
```
**Fix:** Match the exact parameter and return types from the interface.

---

## Pre-Commit Checklist

Before committing changes, especially to core business logic:

- [ ] Run `go test ./... -count=1` and verify all tests pass
- [ ] Run `go build` to ensure no compilation errors
- [ ] If you modified an interface, verify all mocks are updated
- [ ] If you added new business logic, ensure corresponding tests exist
- [ ] Review changed files for accidental debug code or commented-out logic
- [ ] Ensure no secrets, credentials, or sensitive data in code
- [ ] Check that tenant context is properly propagated in new code paths

### Recommended Git Hooks

Consider adding a pre-commit hook to automatically run tests:

```bash
# .git/hooks/pre-commit
#!/bin/bash
cd services/atlas-{service-name}/atlas.com/{service-name}
go test ./... -count=1
if [ $? -ne 0 ]; then
    echo "Tests failed. Commit aborted."
    exit 1
fi
```

---

## Common Testing Pitfalls

### 1. Skipping the Full Test Suite
**Problem:** Only running tests in the modified package.
**Solution:** Always run `go test ./... -count=1` to catch cross-package issues.

### 2. Using Cached Test Results
**Problem:** Tests pass due to cache, not because code is correct.
**Solution:** Use `-count=1` flag to disable test caching.

### 3. Forgetting Mock Updates
**Problem:** Interface changed but mock/processor.go not updated.
**Solution:** Follow the Interface Change Workflow checklist above.

### 4. Incomplete Mock Implementations
**Problem:** Mock has the method but doesn't properly handle all return types.
**Solution:** Copy patterns from existing mock methods, especially for curried functions.

### 5. Not Testing Error Paths
**Problem:** Only testing happy paths, errors go untested.
**Solution:** Write table-driven tests with both success and failure cases.

### 6. Ignoring Race Conditions
**Problem:** Tests pass normally but fail with `-race` flag.
**Solution:** Periodically run tests with `-race` to catch concurrency issues.

---

## AI Coding Assistant Guidance

When using AI tools to modify code:

1. **Always request test execution** after interface changes
2. **Ask AI to update mocks** when interface methods are added/modified
3. **Request full test suite runs** (`go test ./... -count=1`) not just single packages
4. **Verify AI updated ALL mock locations** - search for mock files yourself
5. **Don't accept "tests will pass"** - require actual test execution output
