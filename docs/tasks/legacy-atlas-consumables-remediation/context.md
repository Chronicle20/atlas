# Atlas Consumables Remediation - Context

**Last Updated:** 2026-01-13

---

## Key Files

### Files Requiring Modification

| File | Purpose | Change Required |
|------|---------|-----------------|
| `services/atlas-consumables/atlas.com/consumables/character/rest.go:43` | JSON:API interface | Change `func (r *RestModel) GetName()` to value receiver |
| `services/atlas-consumables/atlas.com/consumables/equipment/model.go` | Equipment slot model | Consider adding builder (low priority) |

### Files Requiring Tests (New Files)

| File to Create | Tests For |
|----------------|-----------|
| `map/character/registry_test.go` | Registry singleton operations |
| `map/character/processor_test.go` | Character map tracking |
| `consumable/processor_test.go` | Core business logic |

### Reference Files for Test Patterns

| File | Pattern |
|------|---------|
| `services/atlas-configurations/.../templates/processor_test.go` | Processor test structure |
| `services/atlas-configurations/.../templates/rest_test.go` | REST model test patterns |

---

## Architecture Context

### Service Type
- **Kafka-only microservice** - No REST endpoints, no database
- **Event-driven** - Consumes Kafka messages, produces events
- **Stateless** - Only in-memory registry for character locations

### Package Structure
```
consumables/
├── main.go                    # Service entry point
├── consumable/
│   └── processor.go           # Core business logic (712 lines) - NEEDS TESTS
├── map/character/
│   ├── registry.go            # Singleton character registry - NEEDS TESTS
│   └── processor.go           # Registry operations - NEEDS TESTS
├── character/
│   └── rest.go                # JSON:API model - NEEDS FIX (GetName receiver)
├── equipment/
│   └── model.go               # Equipment slots - CONSIDER BUILDER
└── ... (other packages)
```

### Cross-Service Dependencies
The consumable processor calls these external services via REST:
- `character.NewProcessor()` - Character data retrieval
- `inventory.NewProcessor()` - Inventory operations
- `compartment.NewProcessor()` - Compartment reservation/consumption
- `consumable3.NewProcessor()` (data/consumable) - Consumable data lookup
- `equipable2.NewProcessor()` (data/equipable) - Equipment data lookup
- `_map3.NewProcessor()` (data/map) - Map data lookup

---

## Key Decisions

### Decision 1: REST-001 Fix Approach
**Decision:** Direct code change
**Rationale:** Simple one-line change from pointer to value receiver
**File:** `character/rest.go:43`
**Change:**
```go
// From:
func (r *RestModel) GetName() string {
// To:
func (r RestModel) GetName() string {
```

### Decision 2: Test Mocking Strategy
**Decision:** Test pure business logic first, defer REST client mocking
**Rationale:**
- `ApplyItemEffects()` is a package-level function that can be tested directly
- `ValidateScrollUse()` and scroll mechanics can be tested with model inputs
- Registry tests don't require external dependencies

### Decision 3: Registry Test Isolation
**Decision:** Accept singleton state sharing between tests
**Rationale:**
- Each test should clean up after itself
- Tests should not depend on registry being empty
- Alternative (creating test-specific registry) would require code changes

### Decision 4: Equipment Builder (Deferred)
**Decision:** Analyze before implementing
**Rationale:**
- Current `equipment/model.go` is simple (34 lines)
- Uses `Set()` method for mutations
- Builder only needed if complex modification flows exist

---

## Dependencies

### Go Module Dependencies
```go
import (
    "testing"
    "context"
    "github.com/sirupsen/logrus"
    "github.com/google/uuid"
    "github.com/Chronicle20/atlas-tenant"
    "github.com/Chronicle20/atlas-constants/character"
    "github.com/Chronicle20/atlas-constants/inventory"
    "github.com/Chronicle20/atlas-constants/item"
    "github.com/Chronicle20/atlas-constants/map"
)
```

### Test Data Requirements
- Consumable item specs (HP, MP, stat buffs, duration)
- Equipment reference data (slots, stats)
- Character model with HP/MP values
- Map model with world/channel/map IDs

---

## Audit Reference

### Checks Summary

| Check ID | Status | Impact |
|----------|--------|--------|
| ARCH-001 | pass | low |
| ARCH-002 | pass | low |
| ARCH-003 | pass | low |
| ARCH-004 | pass | low |
| MODEL-001 | pass | low |
| MODEL-002 | warn | medium |
| REST-001 | **fail** | medium |
| REST-002 | pass | low |
| PROC-001 | pass | low |
| CACHE-001 | pass | low |
| TEST-001 | **fail** | high |
| LAYER-001 | pass | low |

### Blocking Issues
1. TEST-001: No test coverage exists for the service

### Non-Blocking Issues
1. REST-001: GetName() uses pointer receiver in character/rest.go
2. MODEL-002: equipment/model.go may need builder for modification flows

---

## Code Snippets

### Current GetName() Implementation (INCORRECT)
```go
// character/rest.go:43
func (r *RestModel) GetName() string {
    return "characters"
}
```

### Required GetName() Implementation (CORRECT)
```go
// character/rest.go:43
func (r RestModel) GetName() string {
    return "characters"
}
```

### Registry Singleton Pattern (Reference)
```go
// map/character/registry.go:15-21
func getRegistry() *Registry {
    once.Do(func() {
        registry = &Registry{}
        registry.characterRegister = make(map[uint32]MapKey)
    })
    return registry
}
```

### ApplyItemEffects Function Signature (Test Target)
```go
// consumable/processor.go:69
func ApplyItemEffects(l logrus.FieldLogger, ctx context.Context, c character.Model, m _map2.Model, ci consumable3.Model, characterId uint32, itemId item2.Id)
```

### ValidateScrollUse Method Signature (Test Target)
```go
// consumable/processor.go:452
func (p *Processor) ValidateScrollUse(c character.Model, scrollItem asset.Model[any], equipItem asset.Model[asset.EquipableReferenceData]) bool
```
