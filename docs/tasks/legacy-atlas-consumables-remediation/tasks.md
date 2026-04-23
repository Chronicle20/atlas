# Atlas Consumables Remediation - Task Checklist

**Last Updated:** 2026-01-13

---

## Phase 1: Code Compliance Fixes (P1)

### Section 1.1: Fix JSON:API Interface Compliance
- [x] **1.1.1** Fix GetName() receiver in character/rest.go
  - Change `func (r *RestModel) GetName()` to `func (r RestModel) GetName()`
  - File: `services/atlas-consumables/atlas.com/consumables/character/rest.go:43`
- [x] **1.1.2** Verify all other rest.go files comply
  - Check: `data/consumable/rest.go` (already correct)
  - Check: `data/equipable/rest.go` (correct)
  - Check: `data/map/rest.go` (correct)
  - Check: `compartment/rest.go` (correct)
  - Check: `inventory/rest.go` (correct)
  - Check: `pet/rest.go` (correct)
  - Check: `portal/rest.go` (correct)
  - Check: `monster/drop/position/rest.go` (correct)
  - Check: `cash/rest.go` (correct)

---

## Phase 2: Core Test Infrastructure (P0)

### Section 2.1: Test Helpers and Mocking Strategy
- [x] **2.1.1** Create test helper utilities
  - Created logger factory function (`testLogger()`)
  - Created context with tenant helper (`testContext()`)
  - Created in test files directly (map/character/*_test.go)
- [x] **2.1.2** Design mock strategy for REST clients
  - Tested pure functions directly without mocking
  - Deferred complex mocking for ApplyItemEffects to future iteration

---

## Phase 3: Registry Tests (P1)

### Section 3.1: map/character/registry_test.go
- [x] **3.1.1** Test AddCharacter and GetMap
  - Add character, verify GetMap returns correct MapKey
- [x] **3.1.2** Test RemoveCharacter
  - Add then remove character, verify GetMap returns not found
- [x] **3.1.3** Test GetMap not found case
  - Query non-existent character, verify returns false
- [x] **3.1.4** Test concurrent access
  - Use goroutines to add/remove/get simultaneously
  - Tested with 100 goroutines x 100 operations each
- [x] **3.1.5** Test singleton pattern
  - Verified getRegistry() returns same instance

### Section 3.2: map/character/processor_test.go
- [x] **3.2.1** Test Enter operation
  - Call Enter, verify character registered to map
- [x] **3.2.2** Test Exit operation
  - Call Exit, verify character removed
- [x] **3.2.3** Test TransitionMap
  - Call TransitionMap, verify location updated
- [x] **3.2.4** Test TransitionChannel
  - Call TransitionChannel, verify channel updated
- [x] **3.2.5** Test GetMap operation
  - Verify returns correct map model
- [x] **3.2.6** Test GetMap not found
  - Verify error for non-existent character

---

## Phase 4: Business Logic Tests (P0)

### Section 4.1: Effect Application Tests
- [ ] **4.1.1** Test ApplyItemEffects with HP recovery
  - Create consumable with SpecTypeHP
  - Verify character HP changes
  - *Deferred: Requires REST client mocking*
- [ ] **4.1.2** Test ApplyItemEffects with MP recovery
  - *Deferred: Requires REST client mocking*
- [ ] **4.1.3** Test ApplyItemEffects with HP percentage recovery
  - *Deferred: Requires REST client mocking*
- [ ] **4.1.4** Test ApplyItemEffects with stat buffs
  - *Deferred: Requires REST client mocking*
- [ ] **4.1.5** Test ApplyItemEffects with multiple effects
  - *Deferred: Requires REST client mocking*

### Section 4.2: Scroll Validation Tests
- [ ] **4.2.1** Test ValidateScrollUse with available slots
  - *Deferred: Requires full processor context*
- [ ] **4.2.2** Test ValidateScrollUse with no slots
  - *Deferred: Requires full processor context*
- [ ] **4.2.3** Test ValidateScrollUse clean slate scroll
  - *Deferred: Requires equipable data processor mock*
- [ ] **4.2.4** Test ValidateScrollUse spike scroll
  - Indirectly tested via IsNotSlotConsumingScroll
- [ ] **4.2.5** Test ValidateScrollUse cold protection
  - Indirectly tested via IsNotSlotConsumingScroll

### Section 4.3: Chaos Scroll Logic Tests
- [x] **4.3.1** Test rollStatAdjustment distribution
  - Call multiple times, verify range -5 to +5
  - Verified probability distribution (~18.38% for 0)
- [x] **4.3.2** Test generateChaosChanges
  - Verify only non-zero stats get changes
  - Verify correct number of changes generated
  - Verify error on mismatched lengths
- [x] **4.3.3** Test applyChaos
  - Verify all 14 stat types processed when non-zero
  - Verify partial stats generate correct count
- [x] **4.3.4** Test HP/MP multiplier in chaos
  - Verify HP/MP adjustments are multiplied by 10
  - Verified results are in range [-50, 50]

### Section 4.4: Helper Function Tests
- [x] **4.4.1** Test IsNotSlotConsumingScroll
  - Spike scroll returns true
  - Cold protection scroll returns true
  - Regular scroll returns false

---

## Phase 5: Optional Enhancements (P2)

### Section 5.1: Builder Pattern Consideration
- [ ] **5.1.1** Analyze equipment/model.go usage
  - Search for Set() method calls
  - Document modification patterns
  - Decide if builder needed
- [ ] **5.1.2** Add builder if needed
  - Implement builder with Clone() pattern
  - Only if modification flows identified

---

## Summary

| Phase | Section | Total Tasks | Completed |
|-------|---------|-------------|-----------|
| 1 | 1.1 | 2 | 2 |
| 2 | 2.1 | 2 | 2 |
| 3 | 3.1 | 5 | 5 |
| 3 | 3.2 | 6 | 6 |
| 4 | 4.1 | 5 | 0 |
| 4 | 4.2 | 5 | 0 |
| 4 | 4.3 | 4 | 4 |
| 4 | 4.4 | 1 | 1 |
| 5 | 5.1 | 2 | 0 |
| **Total** | | **32** | **20** |

---

## Test Coverage Results

```
atlas-consumables/map/character    100.0% coverage (12 tests)
atlas-consumables/consumable       8.5% coverage (11 tests)
```

### Tests Created

**map/character/registry_test.go** (6 tests)
- TestRegistry_AddCharacter_And_GetMap
- TestRegistry_RemoveCharacter
- TestRegistry_GetMap_NotFound
- TestRegistry_AddCharacter_OverwritesExisting
- TestRegistry_ConcurrentAccess
- TestRegistry_Singleton

**map/character/processor_test.go** (6 tests)
- TestProcessor_Enter
- TestProcessor_Exit
- TestProcessor_GetMap
- TestProcessor_GetMap_NotFound
- TestProcessor_TransitionMap
- TestProcessor_TransitionChannel

**consumable/processor_test.go** (11 tests)
- TestIsNotSlotConsumingScroll_SpikeScroll
- TestIsNotSlotConsumingScroll_ColdProtectionScroll
- TestIsNotSlotConsumingScroll_RegularScroll
- TestRollStatAdjustment_ReturnsValidRange
- TestRollStatAdjustment_ZeroIsMostCommon
- TestGenerateChaosChanges_SkipsZeroStats
- TestGenerateChaosChanges_GeneratesForNonZeroStats
- TestGenerateChaosChanges_MismatchedLengths
- TestApplyChaos_AllStats
- TestApplyChaos_PartialStats
- TestApplyChaos_HPMPMultiplier

---

## Quick Reference Commands

```bash
# Run tests with coverage
cd services/atlas-consumables/atlas.com/consumables
go test ./... -cover

# Run tests with race detection
go test ./... -race

# Run specific package tests
go test ./map/character/... -v
go test ./consumable/... -v

# Check for pointer receiver violations
grep -r "func (r \*RestModel) GetName" .
```
