# Atlas-Invites Remediation Context

**Last Updated:** 2026-01-13

---

## Key Files

### Source Files to Modify

| File | Purpose | Changes Required |
|------|---------|------------------|
| `services/atlas-invites/atlas.com/invites/invite/rest.go` | REST model and Transform | Fix Transform to use accessor methods |
| `services/atlas-invites/atlas.com/invites/invite/registry.go` | In-memory storage | Update to use builder for model construction |
| `services/atlas-invites/atlas.com/invites/character/resource.go` | REST endpoint | Migrate to framework handler (P2) |
| `services/atlas-invites/atlas.com/invites/rest/handler.go` | Custom handler registration | Deprecate/remove after migration (P2) |
| `services/atlas-invites/atlas.com/invites/README.md` | Service documentation | Add architecture decision section |

### New Files to Create

| File | Purpose |
|------|---------|
| `services/atlas-invites/atlas.com/invites/invite/builder.go` | Fluent builder with validation |
| `services/atlas-invites/atlas.com/invites/invite/builder_test.go` | Builder validation tests |
| `services/atlas-invites/atlas.com/invites/invite/processor_test.go` | Processor operation tests |
| `services/atlas-invites/atlas.com/invites/invite/registry_test.go` | Registry storage and concurrency tests |
| `services/atlas-invites/atlas.com/invites/invite/rest_test.go` | Transform function tests |
| `services/atlas-invites/atlas.com/invites/invite/mock/producer.go` | Mock Kafka producer for testing |

### Reference Files (Patterns)

| File | Purpose |
|------|---------|
| `services/atlas-fame/atlas.com/fame/fame/builder.go` | Builder pattern example |
| `services/atlas-fame/atlas.com/fame/fame/builder_test.go` | Builder test patterns |
| `services/atlas-fame/atlas.com/fame/fame/processor_test.go` | Processor test patterns |
| `.claude/skills/backend-dev-guidelines/resources/testing-guide.md` | Testing conventions |

---

## Critical Code Sections

### Transform Function (ARCH-012 Fix)

**Current Code (`invite/rest.go:34-43`):**
```go
func Transform(m Model) (RestModel, error) {
    return RestModel{
        Id:           m.id,           // PROBLEM: Private field access
        Type:         m.inviteType,   // PROBLEM: Private field access
        ReferenceId:  m.referenceId,  // PROBLEM: Private field access
        OriginatorId: m.originatorId, // PROBLEM: Private field access
        TargetId:     m.targetId,     // PROBLEM: Private field access
        Age:          m.age,          // PROBLEM: Private field access
    }, nil
}
```

**Required Fix:**
```go
func Transform(m Model) (RestModel, error) {
    return RestModel{
        Id:           m.Id(),           // Use accessor
        Type:         m.Type(),         // Use accessor
        ReferenceId:  m.ReferenceId(),  // Use accessor
        OriginatorId: m.OriginatorId(), // Use accessor
        TargetId:     m.TargetId(),     // Use accessor
        Age:          m.Age(),          // Use accessor
    }, nil
}
```

### Model Construction in Registry (ARCH-003)

**Current Code (`invite/registry.go:51-60`):**
```go
m := Model{
    tenant:       t,
    id:           inviteId,
    inviteType:   inviteType,
    referenceId:  referenceId,
    originatorId: originatorId,
    targetId:     targetId,
    worldId:      worldId,
    age:          time.Now(),
}
```

**Will be Replaced by Builder:**
```go
m, err := NewBuilder().
    SetTenant(t).
    SetId(inviteId).
    SetInviteType(inviteType).
    SetReferenceId(referenceId).
    SetOriginatorId(originatorId).
    SetTargetId(targetId).
    SetWorldId(worldId).
    SetAge(time.Now()).
    Build()
if err != nil {
    // Handle validation error
}
```

---

## Model Structure

### Domain Model (`invite/model.go`)

```go
type Model struct {
    tenant       tenant.Model  // Multi-tenant isolation
    id           uint32        // Unique invite ID (generated)
    inviteType   string        // "BUDDY", "PARTY", "GUILD", etc.
    referenceId  uint32        // Reference to related entity
    originatorId uint32        // Character who sent invite
    targetId     uint32        // Character who receives invite
    worldId      byte          // Game world ID
    age          time.Time     // Creation timestamp
}

// Public accessors
func (m Model) Tenant() tenant.Model
func (m Model) Id() uint32
func (m Model) Type() string
func (m Model) ReferenceId() uint32
func (m Model) OriginatorId() uint32
func (m Model) TargetId() uint32
func (m Model) WorldId() byte
func (m Model) Age() time.Time
func (m Model) Expired(timeout time.Duration) bool
```

### REST Model (`invite/rest.go`)

```go
type RestModel struct {
    Id           uint32    `json:"-"`
    Type         string    `json:"type"`
    ReferenceId  uint32    `json:"referenceId"`
    OriginatorId uint32    `json:"originatorId"`
    TargetId     uint32    `json:"targetId"`
    Age          time.Time `json:"age"`
}

// JSON:API interface
func (r RestModel) GetName() string      // Returns "invites"
func (r RestModel) GetID() string        // Returns string ID
func (r *RestModel) SetID(strId string) error
```

---

## Processor Interface

```go
type Processor interface {
    // Query operations
    GetByCharacterId(characterId uint32) ([]Model, error)
    ByCharacterIdProvider(characterId uint32) model.Provider[[]Model]

    // Command operations with message buffer
    Create(mb *message.Buffer) func(referenceId uint32) func(worldId byte) func(inviteType string) func(originatorId uint32) func(targetId uint32) func(transactionId uuid.UUID) (Model, error)
    Accept(mb *message.Buffer) func(referenceId uint32) func(worldId byte) func(inviteType string) func(actorId uint32) func(transactionId uuid.UUID) (Model, error)
    Reject(mb *message.Buffer) func(originatorId uint32) func(worldId byte) func(inviteType string) func(actorId uint32) func(transactionId uuid.UUID) (Model, error)

    // AndEmit variants (emit immediately)
    CreateAndEmit(referenceId uint32, worldId byte, inviteType string, originatorId uint32, targetId uint32, transactionId uuid.UUID) (Model, error)
    AcceptAndEmit(referenceId uint32, worldId byte, inviteType string, actorId uint32, transactionId uuid.UUID) (Model, error)
    RejectAndEmit(originatorId uint32, worldId byte, inviteType string, actorId uint32, transactionId uuid.UUID) (Model, error)
}
```

---

## Registry Operations

```go
type Registry struct {
    lock           sync.Mutex
    tenantInviteId map[tenant.Model]uint32          // ID counter per tenant
    inviteReg      map[tenant.Model]map[uint32]map[string][]Model  // tenant -> targetId -> type -> invites
    tenantLock     map[tenant.Model]*sync.RWMutex  // Per-tenant locks
}

func GetRegistry() *Registry                           // Singleton accessor
func (r *Registry) Create(...) Model                   // Create and store invite
func (r *Registry) GetByOriginator(...) (Model, error) // Find by originator
func (r *Registry) GetByReference(...) (Model, error)  // Find by reference
func (r *Registry) GetForCharacter(...) ([]Model, error) // Get all for character
func (r *Registry) Delete(...) error                   // Remove invite
func (r *Registry) GetExpired(timeout) ([]Model, error) // Find expired
```

---

## Testing Patterns

### Test Setup Helpers (from atlas-fame)

```go
func setupTestLogger(t *testing.T) logrus.FieldLogger {
    t.Helper()
    l, _ := test.NewNullLogger()
    return l
}

func setupTestTenant(t *testing.T) tenant.Model {
    t.Helper()
    ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
    if err != nil {
        t.Fatalf("Failed to create tenant: %v", err)
    }
    return ten
}

func setupTestContext(t *testing.T, ten tenant.Model) context.Context {
    t.Helper()
    return tenant.WithContext(context.Background(), ten)
}
```

### Table-Driven Test Pattern

```go
func TestBuilderValidation(t *testing.T) {
    tests := []struct {
        name        string
        setup       func() *Builder
        expectError bool
        errorMsg    string
    }{
        {
            name: "valid invite",
            setup: func() *Builder {
                return NewBuilder().SetTenant(setupTestTenant(t)).SetId(1)...
            },
            expectError: false,
        },
        {
            name: "missing tenant",
            setup: func() *Builder {
                return NewBuilder().SetId(1)... // No tenant
            },
            expectError: true,
            errorMsg:    "tenant is required",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := tt.setup().Build()
            if tt.expectError {
                assert.Error(t, err)
                assert.Equal(t, tt.errorMsg, err.Error())
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

---

## Kafka Topics

| Topic | Purpose |
|-------|---------|
| `TOPIC_INVITE_COMMAND` | Incoming invite commands (create, accept, reject) |
| `TOPIC_INVITE_STATUS_EVENT` | Outgoing status events (created, accepted, rejected) |

---

## Dependencies Between Tasks

```
1.1 Test Infrastructure ──┐
                          │
1.2 Fix Transform ────────┼──> Phase 1 Complete
                          │
1.3 Processor Tests ──────┤
                          │
1.4 Registry Tests ───────┘

                          │
                          v

2.1 Builder Pattern ──────┬──> Phase 2 Complete
                          │
2.2 Document Architecture─┘

                          │
                          v

3.1 Framework Handlers ───┬──> Phase 3 Complete
                          │
3.2 Provider Documentation┘
```

---

## Decisions Log

| Decision | Rationale | Date |
|----------|-----------|------|
| Maintain in-memory architecture | Invites are ephemeral, persistence not required | 2026-01-13 |
| Skip entity.go/provider.go creation | Service intentionally doesn't use database | 2026-01-13 |
| Skip administrator.go creation | Write operations through processor are appropriate | 2026-01-13 |
| Prioritize tests over builder | Tests enable safe refactoring for all other changes | 2026-01-13 |

---

## Audit Reference

- **Audit File:** `docs/audits/atlas-invites/audit.md`
- **Audit JSON:** `docs/audits/atlas-invites/audit.json`
- **Overall Status:** needs-work
- **Confidence:** high
