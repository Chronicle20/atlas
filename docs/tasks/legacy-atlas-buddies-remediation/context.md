# Atlas Buddies Remediation - Context Document

**Last Updated:** 2026-01-13

---

## 1. Key Files

### Primary Files to Modify

| File | Purpose | Issues |
|------|---------|--------|
| `services/atlas-buddies/atlas.com/buddies/character/rest.go` | JSON:API RestModel for character cross-service client | REST-001: Wrong receiver type on GetName() |
| `services/atlas-buddies/atlas.com/buddies/list/resource.go` | REST route handlers for buddy list operations | CODE-001: Dead code; CODE-002: Direct field access |

### Files for Reference (Patterns)

| File | Purpose |
|------|---------|
| `services/atlas-account/atlas.com/account/account/builder.go` | Reference implementation of builder pattern |
| `services/atlas-buddies/atlas.com/buddies/list/model.go` | Current model structure (has Buddies() accessor) |
| `services/atlas-buddies/atlas.com/buddies/buddy/model.go` | Buddy sub-domain model |

### Audit Artifacts

| File | Purpose |
|------|---------|
| `docs/audits/atlas-buddies/audit.md` | Human-readable audit report |
| `docs/audits/atlas-buddies/audit.json` | Machine-readable audit data |

---

## 2. Code Patterns

### JSON:API Interface Methods - Correct Pattern

All JSON:API interface methods should use **value receivers** except for methods that modify the receiver:

```go
// CORRECT - value receiver for read-only methods
func (r RestModel) GetName() string { return "resources" }
func (r RestModel) GetID() string { return strconv.Itoa(int(r.Id)) }
func (r RestModel) GetReferences() []jsonapi.Reference { ... }

// CORRECT - pointer receiver for methods that modify
func (r *RestModel) SetID(idStr string) error { r.Id = ...; return nil }
func (r *RestModel) SetToOneReferenceID(name, ID string) error { ... }
```

### Model Immutability Pattern

Models use private fields with public accessor methods:

```go
type Model struct {
    tenantId    uuid.UUID  // private
    characterId uint32     // private
    buddies     []buddy.Model // private
}

func (m Model) Buddies() []buddy.Model {
    return m.buddies  // accessor method
}

// INCORRECT - direct field access from outside package context
// bl.buddies

// CORRECT - use accessor
// bl.Buddies()
```

### Builder Pattern

```go
type Builder struct {
    field1 Type1
    field2 Type2
}

func NewBuilder(required1 Type1) *Builder {
    return &Builder{
        field1: required1,
        field2: defaultValue,
    }
}

func (b *Builder) SetField2(v Type2) *Builder {
    b.field2 = v
    return b
}

func (b *Builder) Build() (Model, error) {
    // validation
    if b.field1 == zero {
        return Model{}, errors.New("field1 is required")
    }
    return Model{field1: b.field1, field2: b.field2}, nil
}
```

---

## 3. Service Architecture

```
atlas-buddies/
├── buddy/           # Sub-domain: buddy model
│   ├── model.go     # Immutable model
│   ├── entity.go    # GORM entity with Make()
│   └── rest.go      # JSON:API RestModel
├── character/       # Cross-service REST client
│   ├── model.go     # Domain model for external data
│   ├── rest.go      # JSON:API RestModel [HAS ISSUE]
│   ├── requests.go  # HTTP client functions
│   └── processor.go # REST call processor
├── list/            # Primary domain: buddy lists
│   ├── model.go     # Immutable model
│   ├── entity.go    # GORM entity with Make()
│   ├── provider.go  # Read operations
│   ├── administrator.go # Write operations
│   ├── processor.go # Business logic
│   ├── resource.go  # REST handlers [HAS ISSUES]
│   └── rest.go      # JSON:API RestModel
├── invite/          # Invitation processing (Kafka-only)
├── kafka/           # Kafka infrastructure
│   ├── consumer/    # Event consumers
│   ├── message/     # Message definitions
│   └── producer/    # Message producers
└── [infrastructure packages]
```

---

## 4. Dependencies

### Internal Dependencies
- `buddy` package is used by `list` package
- `character` package provides cross-service data
- `invite` package handles invitation flows via Kafka

### External Service Dependencies
- `atlas-characters` - Character information lookups
- External invite service via Kafka

### Library Dependencies
- `github.com/jtumidanski/api2go/jsonapi` - JSON:API implementation
- `github.com/Chronicle20/atlas-model/model` - Model utilities
- `github.com/Chronicle20/atlas-rest/server` - REST server utilities
- `gorm.io/gorm` - ORM for database operations

---

## 5. Test Coverage

| Package | Test Files | Coverage Focus |
|---------|------------|----------------|
| `list` | `processor_test.go`, `administrator_test.go` | Capacity operations, integration tests |
| `kafka/consumer/list` | `consumer_test.go` | Handler type guards |

Tests use SQLite in-memory databases for integration testing.

---

## 6. Related Documentation

- **Backend Dev Guidelines:** `.claude/skills/backend-dev-guidelines/SKILL.md`
- **Audit Template:** `.claude/skills/backend-audit/`
- **JSON:API Spec:** https://jsonapi.org/

---

## 7. Decision Context

### Why Remove Dead Endpoint vs Implement?

The `handleAddBuddyToBuddyList` endpoint:
- Currently returns 202 Accepted but performs no action
- Has commented-out Kafka producer code
- References undefined `addBuddyCommandProvider`
- References undefined `EnvCommandTopic`

**Recommendation:** Remove unless there's a specific product requirement for REST-initiated buddy adds. The invite flow via Kafka is the intended mechanism.

### Why Builder Pattern is Optional

The builder pattern provides:
- Validation at construction time
- Fluent API for complex object construction
- Documentation of required vs optional fields

However, the current models are simple enough that direct construction works fine. Builder becomes valuable when:
- Models have many fields with complex defaults
- Validation logic is needed at construction
- Multiple construction paths exist

---

## 8. Quick Reference Commands

```bash
# Build service
cd services/atlas-buddies/atlas.com/buddies && go build ./...

# Run tests
cd services/atlas-buddies/atlas.com/buddies && go test ./...

# Run specific test
go test -run TestUpdateCapacity ./list/...

# Check for compilation errors
go vet ./...
```
