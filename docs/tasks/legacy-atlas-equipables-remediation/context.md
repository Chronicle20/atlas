# Context Document - atlas-equipables Remediation

**Last Updated:** 2026-01-13

---

## 1. Key Files

### Service Code
| File | Purpose | Relevance |
|------|---------|-----------|
| `services/atlas-equipables/README.md` | API documentation | Tasks 1.1, 1.2 - Needs path corrections |
| `services/atlas-equipables/atlas.com/equipables/equipable/model.go` | Domain model + builder | Tasks 2.2, 4.2 - Builder separation |
| `services/atlas-equipables/atlas.com/equipables/equipable/processor.go` | Business logic | Task 2.1 - Primary test target |
| `services/atlas-equipables/atlas.com/equipables/equipable/rest.go` | REST transformation | Tasks 2.3, 4.1 - Tests + accessor refactor |
| `services/atlas-equipables/atlas.com/equipables/equipable/resource.go` | HTTP handlers | Task 3.2 - Error mapping |
| `services/atlas-equipables/atlas.com/equipables/equipable/entity.go` | GORM entity | Reference for migrations in tests |
| `services/atlas-equipables/atlas.com/equipables/equipable/administrator.go` | Write operations | Reference for understanding data flow |
| `services/atlas-equipables/atlas.com/equipables/equipable/provider.go` | Read operations | Reference for query patterns |

### Reference Files (Testing Patterns)
| File | Purpose |
|------|---------|
| `services/atlas-character/atlas.com/character/character/processor_test.go` | Processor test pattern |
| `services/atlas-character/atlas.com/character/character/rest_test.go` | REST test pattern |
| `services/atlas-families/atlas.com/family/family/builder_test.go` | Builder test pattern |
| `services/atlas-marriages/atlas.com/marriages/marriage/processor_test.go` | Additional processor patterns |

### Audit Files
| File | Purpose |
|------|---------|
| `docs/audits/atlas-equipables/audit.md` | Detailed audit findings |
| `docs/audits/atlas-equipables/audit.json` | Machine-readable findings |

---

## 2. Key Decisions

### D1: Test Database Strategy
**Decision:** Use in-memory SQLite for unit tests
**Rationale:** Matches existing codebase patterns (atlas-character, atlas-marriages). Fast, isolated, no external dependencies.
**Implementation:**
```go
db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
```

### D2: Mock External Data Service
**Decision:** Create mock for `data/equipable.Processor` to isolate unit tests
**Rationale:** `CreateRandom` depends on external service call (`p.edp.GetById`). Mocking allows testing without network calls.
**Implementation:** Inject mock processor via constructor or interface.

### D3: Error Type Approach
**Decision:** Use sentinel errors with `errors.Is()` comparison
**Rationale:** Simple, Go-idiomatic, matches existing patterns. No need for complex error hierarchies.
**Alternative Rejected:** Custom error types with methods - overkill for current requirements.

### D4: Transform Refactor Scope
**Decision:** Only change direct field access to accessors, no other modifications
**Rationale:** Minimize risk, tests validate no behavioral changes.

### D5: Builder Separation Scope
**Decision:** Move builder code to new file, no structural changes
**Rationale:** Pure code organization, tests ensure no regression.

---

## 3. Dependencies

### Go Module Dependencies (Existing)
```go
import (
    "gorm.io/gorm"
    "gorm.io/driver/sqlite"  // May need to add
    "github.com/Chronicle20/atlas-tenant"
    "github.com/Chronicle20/atlas-model/model"
    "github.com/sirupsen/logrus"
    "github.com/sirupsen/logrus/hooks/test"  // For null logger
    "github.com/google/uuid"
)
```

### Service Dependencies
- `atlas-data` service: Provides equipable template data via REST
- `kafka`: Message buffer and producer for AndEmit pattern

### Test Dependencies to Add
If not already in go.mod:
```
gorm.io/driver/sqlite
github.com/sirupsen/logrus/hooks/test
```

---

## 4. API Mapping

### Actual API (from resource.go:20-25)
| Method | Path | Handler |
|--------|------|---------|
| POST | `/api/equipables?random={random}` | `handleCreateRandomEquipment` |
| POST | `/api/equipables` | `handleCreateEquipment` |
| GET | `/api/equipables/{equipmentId}` | `handleGetEquipment` |
| PATCH | `/api/equipables/{equipmentId}` | `handleUpdateEquipment` |
| DELETE | `/api/equipables/{equipmentId}` | `handleDeleteEquipment` |

### README (Current - Incorrect)
| Method | Path |
|--------|------|
| POST | `/api/ess/equipment` |
| POST | `/api/ess/equipment?random=true` |
| GET | `/api/ess/equipment/{equipmentId}` |
| DELETE | `/api/ess/equipment/{equipmentId}` |
| (Missing) | PATCH endpoint |

---

## 5. Model Field Inventory

### Model Fields (28 total)
Used for test completeness validation:

```go
id             uint32
itemId         uint32
strength       uint16
dexterity      uint16
intelligence   uint16
luck           uint16
hp             uint16
mp             uint16
weaponAttack   uint16
magicAttack    uint16
weaponDefense  uint16
magicDefense   uint16
accuracy       uint16
avoidability   uint16
hands          uint16
speed          uint16
jump           uint16
slots          uint16
ownerName      string
locked         bool
spikes         bool
karmaUsed      bool
cold           bool
canBeTraded    bool
levelType      byte
level          byte
experience     uint32
hammersApplied uint32
expiration     time.Time
```

---

## 6. Error Scenarios

### Handler Error Mapping Guide
| Scenario | Current Status | Target Status |
|----------|----------------|---------------|
| Equipable not found (GET) | 404 | 404 |
| Equipable not found (UPDATE) | 404 | 404 |
| Equipable not found (DELETE) | 404 | 404 |
| Internal DB error (GET) | 404 | 500 |
| Internal DB error (UPDATE) | 404 | 500 |
| Internal DB error (DELETE) | 404 | 500 |
| Template not found (CREATE) | 500 | 500 (or 400) |
| Invalid input (CREATE) | 400 | 400 |

---

## 7. Testing Checklist

### Processor Tests
- [ ] Create with explicit stats
- [ ] Create with template stats (zero stats triggers template fetch)
- [ ] Create random (uses external data processor)
- [ ] Get by ID (found)
- [ ] Get by ID (not found)
- [ ] Update single field
- [ ] Update multiple fields
- [ ] Update preserves unchanged fields
- [ ] Delete (found)
- [ ] Delete (not found)

### Builder Tests
- [ ] NewBuilder sets ID
- [ ] All Set* methods return builder
- [ ] Build creates model with all values
- [ ] Clone copies all fields
- [ ] Add* methods handle overflow
- [ ] Add* methods handle underflow

### REST Tests
- [ ] Transform maps all 28 fields
- [ ] Extract maps all 28 fields
- [ ] GetName returns "equipables"
- [ ] GetID formats uint32 as string
- [ ] SetID parses string to uint32

---

## 8. Migration for Tests

The test database needs migration. Reference from `entity.go`:

```go
func Migration(db *gorm.DB) error {
    return db.AutoMigrate(&entity{})
}
```

Entity structure:
```go
type entity struct {
    TenantId       uuid.UUID `gorm:"not null"`
    Id             uint32    `gorm:"primaryKey;autoIncrement;not null"`
    ItemId         uint32    `gorm:"not null"`
    Strength       uint16    `gorm:"not null;default:0"`
    Dexterity      uint16    `gorm:"not null;default:0"`
    // ... remaining fields
}
```

---

## 9. Notes on External Dependencies

### Data Service Dependency
`processor.go:36` initializes external data processor:
```go
edp: equipable.NewProcessor(l, ctx),
```

This is used in:
- `Create` (line 71) when stats are all zero
- `CreateRandom` (line 100) always

**Testing Approach:** Either:
1. Mock the external processor
2. Use a fake HTTP server
3. Refactor processor to accept interface

Recommend Option 1 (mocking) for minimal changes.

### Kafka Dependency
Tests can use `message.NewBuffer()` which doesn't require Kafka connection. Messages are buffered but not sent in unit tests.
