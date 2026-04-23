# Atlas Consumables Service Remediation Plan

**Last Updated:** 2026-01-13

---

## Executive Summary

This plan addresses issues identified in the audit of the `atlas-consumables` service. The service is a **Kafka-only microservice** handling consumable item effects, scroll enhancement, and various item consumption mechanics. It has **no database** and **no REST endpoints** - relying entirely on Kafka messaging and REST client calls to other services.

**Audit Status:** `needs-work`
**Confidence:** `high`

### Key Issues to Address
| Priority | Issue | Impact | Effort |
|----------|-------|--------|--------|
| P0 | TEST-001: No test coverage | High | L |
| P1 | REST-001: GetName() pointer receiver | Medium | S |
| P2 | MODEL-002: equipment/model.go builder consideration | Low | S |

---

## Current State Analysis

### Service Architecture
- **Type:** Event-driven Kafka consumer/producer
- **Database:** None (stateless)
- **REST Endpoints:** None exposed
- **Cross-service Communication:** REST clients to character, inventory, compartment, data, monster, pet, portal services

### Key Components Requiring Tests
1. **consumable/processor.go** - Core business logic (~712 lines)
   - `ApplyItemEffects()` - Stat buff and HP/MP recovery
   - `RequestItemConsume()` - Item consumption orchestration
   - `ConsumeStandard()` - Standard consumable handling
   - `ConsumeTownScroll()` - Teleportation item handling
   - `ConsumePetFood()` - Pet fullness management
   - `ConsumeCashPetFood()` - Cash pet food handling
   - `ConsumeSummoningSack()` - Monster spawning
   - `RequestScroll()` - Scroll enhancement orchestration
   - `ConsumeScroll()` - Scroll success/failure mechanics
   - `ValidateScrollUse()` - Scroll validation logic

2. **map/character/registry.go** - In-memory character location tracking
   - Thread-safe singleton pattern with sync.Once
   - Add/Remove/Get operations

3. **character/rest.go** - JSON:API interface compliance issue

---

## Proposed Future State

### Test Coverage Goals
- Unit tests for all processor business logic
- Registry singleton thread-safety tests
- Scroll mechanics edge case coverage
- JSON:API interface compliance verification

### Code Compliance
- All `GetName()` methods use value receivers
- Consistent builder pattern where modification flows exist

---

## Implementation Phases

### Phase 1: Code Compliance Fixes (P1)
Quick fixes to bring code into compliance with backend guidelines.

**Section 1.1: Fix JSON:API Interface Compliance**

| # | Task | Acceptance Criteria | Effort | Dependencies |
|---|------|---------------------|--------|--------------|
| 1.1.1 | Fix GetName() receiver in character/rest.go | `func (r RestModel) GetName()` uses value receiver | S | None |
| 1.1.2 | Verify all other rest.go files comply | All GetName() methods use value receivers | S | 1.1.1 |

### Phase 2: Core Test Infrastructure (P0)
Establish test patterns and infrastructure for the service.

**Section 2.1: Test Helpers and Mocking Strategy**

| # | Task | Acceptance Criteria | Effort | Dependencies |
|---|------|---------------------|--------|--------------|
| 2.1.1 | Create test helper utilities | Logger factory, context creation, mock tenant setup | S | None |
| 2.1.2 | Design mock strategy for REST clients | Interface abstractions for character, inventory, consumable data processors | M | None |

### Phase 3: Registry Tests (P1)
Test the in-memory character registry singleton.

**Section 3.1: map/character/registry_test.go**

| # | Task | Acceptance Criteria | Effort | Dependencies |
|---|------|---------------------|--------|--------------|
| 3.1.1 | Test AddCharacter and GetMap | Adding character returns correct MapKey | S | 2.1.1 |
| 3.1.2 | Test RemoveCharacter | Removed character returns not found | S | 3.1.1 |
| 3.1.3 | Test GetMap not found case | Non-existent character returns false | S | 3.1.1 |
| 3.1.4 | Test concurrent access | Multiple goroutines can safely add/remove/get | M | 3.1.1 |

**Section 3.2: map/character/processor_test.go**

| # | Task | Acceptance Criteria | Effort | Dependencies |
|---|------|---------------------|--------|--------------|
| 3.2.1 | Test Enter operation | Character is registered to correct map | S | 3.1.1 |
| 3.2.2 | Test Exit operation | Character is removed from registry | S | 3.2.1 |
| 3.2.3 | Test TransitionMap | Character location updated correctly | S | 3.2.1 |
| 3.2.4 | Test TransitionChannel | Character channel updated correctly | S | 3.2.1 |

### Phase 4: Business Logic Tests (P0)
Core consumable processing tests.

**Section 4.1: consumable/processor_test.go - Effect Application**

| # | Task | Acceptance Criteria | Effort | Dependencies |
|---|------|---------------------|--------|--------------|
| 4.1.1 | Test ApplyItemEffects with HP recovery | HP increases by spec value | M | 2.1.2 |
| 4.1.2 | Test ApplyItemEffects with MP recovery | MP increases by spec value | S | 4.1.1 |
| 4.1.3 | Test ApplyItemEffects with HP percentage recovery | HP increases by percentage of MaxHP | S | 4.1.1 |
| 4.1.4 | Test ApplyItemEffects with stat buffs | Buffs applied with correct duration | M | 4.1.1 |
| 4.1.5 | Test ApplyItemEffects with multiple effects | All effects applied correctly | M | 4.1.4 |

**Section 4.2: consumable/processor_test.go - Scroll Validation**

| # | Task | Acceptance Criteria | Effort | Dependencies |
|---|------|---------------------|--------|--------------|
| 4.2.1 | Test ValidateScrollUse with available slots | Returns true when slots > 0 | S | 2.1.2 |
| 4.2.2 | Test ValidateScrollUse with no slots | Returns false when slots = 0 | S | 4.2.1 |
| 4.2.3 | Test ValidateScrollUse clean slate scroll | Validates level < original slots | M | 4.2.1 |
| 4.2.4 | Test ValidateScrollUse spike scroll | Skips slot validation | S | 4.2.1 |
| 4.2.5 | Test ValidateScrollUse cold protection | Skips slot validation | S | 4.2.1 |

**Section 4.3: consumable/processor_test.go - Chaos Scroll Logic**

| # | Task | Acceptance Criteria | Effort | Dependencies |
|---|------|---------------------|--------|--------------|
| 4.3.1 | Test rollStatAdjustment distribution | Returns values -5 to +5 | S | None |
| 4.3.2 | Test generateChaosChanges | Generates changes for non-zero stats only | M | 4.3.1 |
| 4.3.3 | Test applyChaos | All stat types processed correctly | M | 4.3.2 |
| 4.3.4 | Test HP/MP multiplier in chaos | HP/MP adjustments multiplied by 10 | S | 4.3.2 |

**Section 4.4: consumable/processor_test.go - Helper Functions**

| # | Task | Acceptance Criteria | Effort | Dependencies |
|---|------|---------------------|--------|--------------|
| 4.4.1 | Test IsNotSlotConsumingScroll | Identifies spike and cold protection scrolls | S | None |

### Phase 5: Optional Enhancements (P2)

**Section 5.1: Builder Pattern Consideration**

| # | Task | Acceptance Criteria | Effort | Dependencies |
|---|------|---------------------|--------|--------------|
| 5.1.1 | Analyze equipment/model.go usage | Document if modification flows exist | S | None |
| 5.1.2 | Add builder if needed | Builder with Clone() if modifications occur | S | 5.1.1 |

---

## Risk Assessment and Mitigation

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Mocking complexity for REST clients | High | Medium | Create interface abstractions incrementally |
| Registry tests affecting singleton state | Medium | High | Reset registry state between tests or use test-specific instances |
| Test isolation for Kafka consumers | Medium | Medium | Focus on unit testing business logic, defer integration tests |

---

## Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Test coverage for processor.go | > 70% | `go test -cover` |
| All GetName() methods compliant | 100% | Code review |
| Registry tests pass concurrently | Pass | `go test -race` |
| Zero test failures | 0 | CI pipeline |

---

## Required Resources and Dependencies

### Development Dependencies
- Go test framework (standard library)
- `github.com/sirupsen/logrus` for test logging
- `github.com/google/uuid` for transaction IDs
- `github.com/Chronicle20/atlas-tenant` for tenant context mocking

### External Service Dependencies (Mocked)
- character service
- inventory service
- compartment service
- data service (consumable data)
- data service (equipable data)

---

## Technical Notes

1. **No database by design** - This service is intentionally stateless. Tests should not attempt to introduce persistence.

2. **Registry singleton** - The `sync.Once` pattern means tests may share state. Consider either:
   - Testing with a fresh test registry
   - Resetting registry between tests
   - Accepting shared state and designing tests accordingly

3. **Random number generation in scroll mechanics** - Tests for scroll success/failure should either:
   - Mock the random source
   - Test boundary conditions
   - Use statistical assertions over multiple runs

4. **Kafka consumer tests deferred** - Integration testing of Kafka handlers is P2 priority. Focus P0 effort on pure business logic.

5. **ApplyItemEffects is a package-level function** - This is acceptable as it's pure business logic that doesn't require state. Tests can call it directly.
