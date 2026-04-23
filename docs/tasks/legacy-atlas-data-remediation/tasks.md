# Atlas-Data Remediation Tasks

**Last Updated:** 2026-01-13

---

## Phase 1: Error Logging Remediation [P2] - COMPLETE

- [x] **1.1** Add error logging to `consumable/resource.go` (lines 45-48)
  - Acceptance: Error logged with context before 500 status
  - Effort: S

- [x] **1.2** Add error logging to `cash/resource.go` (lines 31-34)
  - Acceptance: Error logged with context before 500 status
  - Effort: S

- [x] **1.3** Add error logging to `commodity/resource.go` (lines 30-34)
  - Acceptance: Error logged with context before 500 status
  - Effort: S

- [x] **1.4** Add error logging to `etc/resource.go` (lines 31-34)
  - Acceptance: Error logged with context before 500 status
  - Effort: S

- [x] **1.5** Add error logging to `pet/resource.go` (lines 31-34)
  - Acceptance: Error logged with context before 500 status
  - Effort: S

- [x] **1.6** Add error logging to `setup/resource.go` (lines 31-34)
  - Acceptance: Error logged with context before 500 status
  - Effort: S

- [x] **1.7** Add error logging to `map/resource.go` (lines 305-306, 311-312)
  - Acceptance: Error logged with context before 500 status
  - Effort: S

- [x] **1.8** Verify all changes compile successfully
  - Acceptance: `go build ./...` succeeds
  - Effort: S

---

## Phase 2: Code Organization Remediation [P2] - COMPLETE

- [x] **2.1** Move `DropPositionRestModel` from `map/resource.go` to `map/rest.go`
  - Acceptance: Model definition and methods in rest.go
  - Effort: S

- [x] **2.2** Move `PositionRestModel` from `map/resource.go` to `map/rest.go`
  - Acceptance: Model definition and methods in rest.go
  - Effort: S

- [x] **2.3** Move `FootholdRestModel` from `map/resource.go` to `map/rest.go`
  - Acceptance: Model definition and methods in rest.go
  - Effort: S
  - Note: `point` package import already existed in rest.go

- [x] **2.4** Verify handlers compile and function correctly
  - Acceptance: `go build ./...` succeeds, all tests pass
  - Effort: S

---

## Phase 3: Handler Test Implementation [P3]

### 3.1 Test Infrastructure

- [ ] **3.1.1** Create test utilities package or shared test helpers
  - Acceptance: Reusable test setup functions available
  - Effort: M

- [ ] **3.1.2** Implement mock ServerInformation for tests
  - Acceptance: GetVersion, GetURI, GetPrefix, GetBaseURL implemented
  - Effort: S

- [ ] **3.1.3** Create test database setup helpers
  - Acceptance: In-memory SQLite setup function works
  - Effort: S

- [ ] **3.1.4** Implement tenant context request builder
  - Acceptance: Helper creates requests with proper tenant headers
  - Effort: S

### 3.2 Core Domain Tests

- [ ] **3.2.1** Add `map/resource_test.go`
  - Acceptance: Tests for GET /maps, GET /maps/{id}, POST endpoints
  - Effort: M

- [ ] **3.2.2** Add `monster/resource_test.go`
  - Acceptance: Tests for GET /monsters, GET /monsters/{id}
  - Effort: S

- [ ] **3.2.3** Add `npc/resource_test.go`
  - Acceptance: Tests for GET /npcs, GET /npcs/{id}, filter endpoints
  - Effort: S

- [ ] **3.2.4** Add `skill/resource_test.go`
  - Acceptance: Tests for GET /skills, GET /skills/{id}
  - Effort: S

- [ ] **3.2.5** Add `equipment/resource_test.go`
  - Acceptance: Tests for GET /equipment, GET /equipment/{id}
  - Effort: S

- [x] **3.2.6** Add `consumable/resource_test.go`
  - Acceptance: Tests for GET /consumables, GET /consumables/{id}
  - Effort: S
  - Note: Implemented with filter tests, tenant isolation, error handling, and JSON:API compliance

### 3.3 Secondary Domain Tests

- [ ] **3.3.1** Add `cash/resource_test.go`
  - Acceptance: Tests for GET /cash, GET /cash/{id}
  - Effort: S

- [ ] **3.3.2** Add `commodity/resource_test.go`
  - Acceptance: Tests for GET /commodities, GET /commodities/{id}
  - Effort: S

- [ ] **3.3.3** Add `etc/resource_test.go`
  - Acceptance: Tests for GET /etc, GET /etc/{id}
  - Effort: S

- [ ] **3.3.4** Add `pet/resource_test.go`
  - Acceptance: Tests for GET /pets, GET /pets/{id}
  - Effort: S

- [ ] **3.3.5** Add `quest/resource_test.go`
  - Acceptance: Tests for GET /quests, GET /quests/{id}
  - Effort: S

- [ ] **3.3.6** Add `reactor/resource_test.go`
  - Acceptance: Tests for GET /reactors, GET /reactors/{id}
  - Effort: S

- [ ] **3.3.7** Add `setup/resource_test.go`
  - Acceptance: Tests for GET /setup, GET /setup/{id}
  - Effort: S

- [ ] **3.3.8** Add `characters/templates/resource_test.go`
  - Acceptance: Tests for character template endpoints
  - Effort: S

### 3.4 Data Upload Tests

- [ ] **3.4.1** Add `data/resource_test.go`
  - Acceptance: Tests for upload endpoint
  - Effort: M

### 3.5 Test Verification

- [ ] **3.5.1** Run full test suite
  - Acceptance: `go test ./...` passes
  - Effort: S

- [ ] **3.5.2** Generate coverage report
  - Acceptance: Coverage report shows handler coverage
  - Effort: S

---

## Phase 4: Documentation [P3]

- [ ] **4.1** Document read-only data service pattern
  - Acceptance: Pattern explained with rationale
  - Effort: S

- [ ] **4.2** Document handler→storage direct access pattern
  - Acceptance: Exception documented with justification
  - Effort: S

- [ ] **4.3** Document RestModel as domain model pattern
  - Acceptance: Design choice explained
  - Effort: S

- [ ] **4.4** Document generic document entity pattern
  - Acceptance: Storage pattern explained
  - Effort: S

---

## Final Verification

- [ ] **V1** Full build verification
  - Acceptance: `go build ./...` succeeds
  - Effort: S

- [ ] **V2** Full test verification
  - Acceptance: `go test ./...` passes
  - Effort: S

- [ ] **V3** Coverage verification
  - Acceptance: Handler coverage meets target
  - Effort: S

---

## Summary

| Phase | Tasks | Effort | Priority | Status |
|-------|-------|--------|----------|--------|
| Phase 1 | 8 | S | P2 | COMPLETE |
| Phase 2 | 4 | S | P2 | COMPLETE |
| Phase 3 | 21 | M-L | P3 | Pending |
| Phase 4 | 4 | S | P3 | Pending |
| Verification | 3 | S | - | Pending |
| **Total** | **40** | **M-L** | - | - |

---

## Progress Tracking

**Phase 1:** 8/8 complete
**Phase 2:** 4/4 complete
**Phase 3:** 1/21 complete (consumable/resource_test.go done - includes test utilities pattern)
**Phase 4:** 0/4 complete
**Verification:** 0/3 complete

**Overall:** 13/40 complete (33%)
