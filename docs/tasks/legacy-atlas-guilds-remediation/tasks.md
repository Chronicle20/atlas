# Atlas-Guilds Remediation - Tasks

**Last Updated:** 2026-01-13 (Revision 4 - FINAL)

---

## Phase 1: Fix Model.Builder() Immutability (P1 - Medium Impact) - COMPLETED

### Task 1.1: Fix guild/builder.go Model.Builder() [S] - DONE
- [x] Read `guild/builder.go` to find Model.Builder() method
- [x] Create local value copies of all model fields before builder assignment
- [x] Verify tests pass: `go test ./guild/... -v`
- [x] Verify build succeeds: `go build ./guild/...`

### Task 1.2: Fix guild/member/builder.go Model.Builder() [S] - DONE
- [x] Read `guild/member/builder.go` to find Model.Builder() method
- [x] Create local value copies of all model fields before builder assignment
- [x] Verify tests pass: `go test ./guild/member/... -v`

### Task 1.3: Fix guild/title/builder.go Model.Builder() [S] - DONE
- [x] Read `guild/title/builder.go` to find Model.Builder() method
- [x] Create local value copies of all model fields before builder assignment
- [x] Verify tests pass: `go test ./guild/title/... -v`

### Task 1.4: Fix thread/builder.go Model.Builder() [S] - DONE
- [x] Read `thread/builder.go` to find Model.Builder() method
- [x] Create local value copies of all model fields before builder assignment
- [x] Verify tests pass: `go test ./thread/... -v`

### Task 1.5: Fix thread/reply/builder.go Model.Builder() [S] - DONE
- [x] Read `thread/reply/builder.go` to find Model.Builder() method
- [x] Create local value copies of all model fields before builder assignment
- [x] Verify tests pass: `go test ./thread/reply/... -v`

### Task 1.6: Add Builder Immutability Tests [S] - DONE
- [x] Add `TestModelBuilder_DoesNotMutateOriginal` to `guild/builder_test.go`
- [x] Add `TestModelBuilder_DoesNotMutateOriginal` to `guild/member/builder_test.go`
- [x] Add `TestModelBuilder_DoesNotMutateOriginal` to `guild/title/builder_test.go`
- [x] Add `TestModelBuilder_DoesNotMutateOriginal` to `thread/builder_test.go`
- [x] Add `TestModelBuilder_DoesNotMutateOriginal` to `thread/reply/builder_test.go`
- [x] Verify all tests pass: `go test ./... -v`

---

## Phase 2: Add Provider Files (P2 - Low Priority) - COMPLETED

### Task 2.1: Create guild/member/provider.go [S] - DONE
- [x] Create `guild/member/provider.go`
- [x] Add `getByGuildId(tenantId uuid.UUID, guildId uint32)` function
- [x] Add `getById(tenantId uuid.UUID, guildId uint32, characterId uint32)` function
- [x] Verify build succeeds: `go build ./guild/member/...`

### Task 2.2: Create guild/title/provider.go [S] - DONE
- [x] Create `guild/title/provider.go`
- [x] Add `getByGuildId(tenantId uuid.UUID, guildId uint32)` function
- [x] Verify build succeeds: `go build ./guild/title/...`

### Task 2.3: Create thread/reply/provider.go [S] - DONE
- [x] Create `thread/reply/provider.go`
- [x] Add `getByThreadId(tenantId uuid.UUID, threadId uint32)` function
- [x] Verify build succeeds: `go build ./thread/reply/...`

---

## Phase 3: Structural Cleanup (P2 - Low Priority) - COMPLETED

### Task 3.1: Handle empty administrator file [S] - DONE
- [x] Verify `guild/character/administrator.go` exists and is empty
- [x] Delete the empty file (already deleted in prior work)
- [x] Verify build succeeds: `go build ./...`
- [x] Verify tests pass: `go test ./... -count=1`

### Task 3.2: Add JSON:API methods to member.RestModel (Optional) [S] - DEFERRED
- [x] **Analysis:** member.RestModel is embedded-only in guild.RestModel
- [x] Mark as DEFERRED - not needed for embedded-only use

### Task 3.3: Add JSON:API methods to title.RestModel (Optional) [S] - DEFERRED
- [x] **Analysis:** title.RestModel is embedded-only in guild.RestModel
- [x] Mark as DEFERRED - not needed for embedded-only use

### Task 3.4: Add JSON:API methods to reply.RestModel (Optional) [S] - DEFERRED
- [x] **Analysis:** reply.RestModel is embedded-only in thread.RestModel
- [x] Mark as DEFERRED - not needed for embedded-only use

---

## Final Verification - COMPLETED

### Pre-Completion Checklist
- [x] Run `go build ./...` - no errors
- [x] Run `go test ./... -count=1` - all tests pass
- [x] Run `go vet ./...` - no new warnings (pre-existing warning in service/teardown.go)
- [x] Verify Model.Builder() fix applied to all 5 builder files
- [x] Verify immutability tests added to all 5 builder test files
- [x] Update audit.md with new status

---

## Progress Summary

| Phase | Total Tasks | Completed | Status |
|-------|-------------|-----------|--------|
| Phase 1: Builder Immutability | 6 | 6 | **COMPLETED** |
| Phase 2: Provider Files | 3 | 3 | **COMPLETED** |
| Phase 3: Structural Cleanup | 4 | 4 | **COMPLETED** |
| **Total** | **13** | **13** | **COMPLETED** |

---

## Previously Completed Work (from prior remediation)

The following work was completed in the previous remediation effort:

### Test Infrastructure (Completed)
- [x] `character/mock/processor.go` - Mock created
- [x] `party/mock/processor.go` - Mock created

### Processor Tests (Completed)
- [x] `guild/processor_test.go` - 16 tests
- [x] `guild/member/processor_test.go` - 7 tests
- [x] `guild/title/processor_test.go` - 6 tests
- [x] `thread/processor_test.go` - 10 tests
- [x] `thread/reply/processor_test.go` - 6 tests

---

## Notes

### Model.Builder() Fix Pattern

**Before (problematic):**
```go
func (m Model) Builder() *Builder {
    return &Builder{
        tenantId: &m.tenantId,  // Direct pointer - allows mutation
    }
}
```

**After (correct):**
```go
func (m Model) Builder() *Builder {
    tenantId := m.tenantId     // Create local copy
    return &Builder{
        tenantId: &tenantId,   // Pointer to copy - original unaffected
    }
}
```

### Immutability Test Pattern

```go
func TestModelBuilder_DoesNotMutateOriginal(t *testing.T) {
    original, _ := NewBuilder(...).Build()
    originalValue := original.SomeField()

    // Modify through builder
    builder := original.Builder()
    builder.SetSomeField("new value")
    _, _ = builder.Build()

    // Original should be unchanged
    assert.Equal(t, originalValue, original.SomeField())
}
```
