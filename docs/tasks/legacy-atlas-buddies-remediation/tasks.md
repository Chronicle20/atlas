# Atlas Buddies Remediation - Task Checklist

**Last Updated:** 2026-01-13

---

## Phase 1: Critical Fixes (P1)

### Task 1.1: Fix GetName() Receiver Type
- [x] Open `services/atlas-buddies/atlas.com/buddies/character/rest.go`
- [x] Line 14: Change `func (r *RestModel) GetName()` to `func (r RestModel) GetName()`
- [x] Verify compilation: `go build ./...`
- [x] Run tests: `go test ./...`

**Issue Reference:** REST-001
**Effort:** S
**Status:** COMPLETED

---

### Task 1.2: Clean Up Dead Handler Code
- [x] **Decision:** Remove endpoint (decided 2026-01-13)
- [x] Open `services/atlas-buddies/atlas.com/buddies/list/resource.go`
- [x] Remove `AddBuddyToBuddyList` constant (line 24)
- [x] Remove route registration for POST `/buddies` (line 34)
- [x] Remove `handleAddBuddyToBuddyList` function (lines 106-118)
- [x] Verify compilation: `go build ./...`
- [x] Run tests: `go test ./...`

**Issue Reference:** CODE-001
**Effort:** S

---

## Phase 2: Pattern Consistency (P2)

### Task 2.1: Fix Direct Field Access
- [x] Open `services/atlas-buddies/atlas.com/buddies/list/resource.go`
- [x] Line 93: Change `bl.buddies` to `bl.Buddies()`
- [x] Verify compilation: `go build ./...`
- [x] Run tests: `go test ./...`

**Issue Reference:** CODE-002
**Effort:** S
**Status:** COMPLETED

---

## Phase 3: Optional Enhancements (P3)

### Task 3.1: Add Builder for list.Model
- [x] Create `services/atlas-buddies/atlas.com/buddies/list/builder.go`
- [x] Implement `Builder` struct
- [x] Implement `NewBuilder(tenantId, characterId)` constructor
- [x] Implement fluent setter methods: `SetId()`, `SetCapacity()`, `SetBuddies()`
- [x] Implement `Build()` with validation
- [x] Add unit tests in `list/builder_test.go`
- [x] Verify compilation and tests pass

**Issue Reference:** ARCH-004
**Effort:** M
**Status:** COMPLETED

---

### Task 3.2: Add Builder for buddy.Model
- [x] Create `services/atlas-buddies/atlas.com/buddies/buddy/builder.go`
- [x] Implement `Builder` struct
- [x] Implement `NewBuilder(listId, characterId)` constructor
- [x] Implement fluent setter methods: `SetGroup()`, `SetCharacterName()`, `SetChannelId()`, `SetInShop()`, `SetPending()`
- [x] Implement `Build()` with validation
- [x] Add unit tests in `buddy/builder_test.go`
- [x] Verify compilation and tests pass

**Issue Reference:** ARCH-004 (buddy sub-domain)
**Effort:** M
**Status:** COMPLETED

---

## Verification

### Final Verification Checklist
- [x] All Phase 1 tasks completed
- [x] All Phase 2 tasks completed
- [x] Service builds successfully: `go build ./...`
- [x] All tests pass: `go test ./...`
- [ ] No linting errors: `go vet ./...`
- [ ] Re-run audit: `/backend-audit atlas-buddies`
- [ ] Audit status shows `pass`

---

## Progress Summary

| Phase | Total Tasks | Completed | Status |
|-------|-------------|-----------|--------|
| Phase 1 (P1) | 2 | 2 | COMPLETE |
| Phase 2 (P2) | 1 | 1 | COMPLETE |
| Phase 3 (P3) | 2 | 2 | COMPLETE |
| **Total** | **5** | **5** | **All Tasks Complete** |

---

## Notes

- Phase 1 and Phase 2 tasks are required for audit compliance
- Phase 3 tasks are optional enhancements
- Task 1.2 requires a decision on endpoint removal vs implementation
- All tasks are independent and can be completed in any order within their phase
