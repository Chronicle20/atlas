# Atlas Chalkboards Remediation - Context Document

**Last Updated:** 2026-01-13

---

## 1. Key Files

### Source Files (To Modify)

| File | Purpose | Modifications Needed |
|------|---------|---------------------|
| `chalkboard/registry.go` | In-memory chalkboard storage | Add tenant isolation via ChalkboardKey |
| `chalkboard/processor.go` | Business logic for chalkboards | Extract tenant, pass to registry |
| `chalkboard/resource.go` | REST endpoint registration | Migrate to server.RegisterHandler |
| `rest/handler.go` | Custom handler infrastructure | Delete RegisterHandler functions |
| `atlas-ingress.yml` | Kubernetes ingress routes | Add character endpoint route |

### Source Files (To Create)

| File | Purpose |
|------|---------|
| `chalkboard/registry_test.go` | Registry tests with tenant isolation verification |
| `chalkboard/processor_test.go` | Processor business logic tests |
| `chalkboard/rest_test.go` | REST transform and JSON:API tests |
| `character/registry_test.go` | Character registry tests |
| `character/processor_test.go` | Character processor tests |
| `chalkboard/builder.go` | Builder pattern (optional) |

### Reference Files

| File | Purpose |
|------|---------|
| `services/atlas-account/.../rest_test.go` | Test pattern examples |
| `services/atlas-account/.../registry_test.go` | Registry test examples |
| `character/processor.go:30` | Correct tenant extraction pattern |
| `character/model.go:10-15` | MapKey struct with tenant field |

---

## 2. Key Code Patterns

### 2.1 Tenant Isolation Pattern (From character package)

**Reference:** `character/model.go:10-15`
```go
type MapKey struct {
    Tenant    tenant.Model
    WorldId   world.Id
    ChannelId channel.Id
    MapId     _map.Id
}
```

**Reference:** `character/processor.go:26-31`
```go
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
    return &ProcessorImpl{
        l:   l,
        ctx: ctx,
        t:   tenant.MustFromContext(ctx),  // Extract tenant from context
    }
}
```

### 2.2 Current Chalkboard Registry (To Be Fixed)

**File:** `chalkboard/registry.go:5-8`
```go
type Registry struct {
    mutex             sync.RWMutex
    characterRegister map[uint32]string  // PROBLEM: No tenant isolation
}
```

### 2.3 Target Chalkboard Registry

```go
type ChalkboardKey struct {
    Tenant      tenant.Model
    CharacterId uint32
}

type Registry struct {
    mutex             sync.RWMutex
    characterRegister map[ChalkboardKey]string
}
```

### 2.4 Current Handler Registration (To Be Removed)

**File:** `rest/handler.go:63-74`
```go
func RegisterHandler(l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler GetHandler) http.HandlerFunc {
    return func(si jsonapi.ServerInformation) func(handlerName string, handler GetHandler) http.HandlerFunc {
        return func(handlerName string, handler GetHandler) http.HandlerFunc {
            return server.RetrieveSpan(l, handlerName, context.Background(), func(sl logrus.FieldLogger, sctx context.Context) http.HandlerFunc {
                fl := sl.WithFields(logrus.Fields{"originator": handlerName, "type": "rest_handler"})
                return server.ParseTenant(fl, sctx, func(tl logrus.FieldLogger, tctx context.Context) http.HandlerFunc {
                    return handler(&HandlerDependency{l: tl, ctx: tctx}, &HandlerContext{si: si})
                })
            })
        }
    }
}
```

This duplicates `server.RegisterHandler` from atlas-rest library.

### 2.5 Test Pattern (From atlas-account)

**File:** `services/atlas-account/.../rest_test.go`
```go
func TestTransform(t *testing.T) {
    // Arrange
    m, _ := NewBuilder(...).Build()

    // Act
    rm, err := Transform(m)

    // Assert
    if err != nil {
        t.Fatalf("Transform failed: %v", err)
    }
    if rm.Id != expected {
        t.Errorf("Id mismatch. Expected %v, got %v", expected, rm.Id)
    }
}
```

---

## 3. Dependencies

### External Packages

| Package | Import Path | Purpose |
|---------|-------------|---------|
| atlas-tenant | `github.com/Chronicle20/atlas-tenant` | Tenant context extraction |
| atlas-rest | `github.com/Chronicle20/atlas-rest/server` | Standard handler registration |
| atlas-constants | `github.com/Chronicle20/atlas-constants/*` | Field, world, channel, map types |
| logrus | `github.com/sirupsen/logrus` | Logging |
| api2go | `github.com/jtumidanski/api2go/jsonapi` | JSON:API serialization |
| mux | `github.com/gorilla/mux` | HTTP routing |

### Internal Dependencies

| Package | Purpose |
|---------|---------|
| `atlas-chalkboards/character` | Character location tracking |
| `atlas-chalkboards/rest` | Custom handler infrastructure (to be removed) |
| `atlas-chalkboards/kafka/producer` | Kafka message production |
| `atlas-chalkboards/kafka/message/chalkboard` | Chalkboard Kafka message types |

---

## 4. Key Decisions

### 4.1 Tenant Isolation Approach
**Decision:** Add tenant field to registry key struct
**Rationale:** Mirrors the character package pattern, maintains consistency
**Alternative Considered:** Separate registries per tenant (rejected - memory overhead, complexity)

### 4.2 Handler Migration Strategy
**Decision:** Direct replacement with library handlers
**Rationale:** Custom handlers are nearly identical to library, no semantic changes needed
**Risk:** Handler signature differences may require handler function updates

### 4.3 Test Coverage Priority
**Decision:** Registry and processor tests first, then REST tests
**Rationale:** Registry tests verify tenant isolation (security), processor tests verify business logic

### 4.4 Builder Pattern (Optional)
**Decision:** Low priority, implement if time permits
**Rationale:** Model is simple (2 fields), inline construction is acceptable

---

## 5. Audit Check Mappings

| Audit Check | Issue ID | Phase | Tasks |
|-------------|----------|-------|-------|
| ARCH-009 | NB-003 | Phase 1 | 1.1-1.4 |
| ARCH-011 | NB-002 | Phase 2 | 2.1-2.4 |
| ARCH-008 | NB-001 | Phase 3 | 3.1-3.3 |
| ARCH-012 | NB-004 | Phase 4 | 4.1 |
| ARCH-002 | NB-005 | Phase 5 | 5.1-5.3 |

---

## 6. Current Ingress Configuration

**File:** `atlas-ingress.yml:128-130`
```nginx
location ~ ^/api/worlds/[^/]+/channels/[^/]+/maps/[^/]+/chalkboards(/.*)?$ {
    proxy_pass http://atlas-chalkboards.atlas.svc.cluster.local:8080;
}
```

**Missing Route (To Add):**
```nginx
location ~ ^/api/chalkboards/[^/]+$ {
    proxy_pass http://atlas-chalkboards.atlas.svc.cluster.local:8080;
}
```

---

## 7. Service Endpoints

| Method | Endpoint | Handler | Ingress Status |
|--------|----------|---------|----------------|
| GET | `/chalkboards/{characterId}` | handleGetChalkboard | MISSING |
| GET | `/worlds/{worldId}/channels/{channelId}/maps/{mapId}/chalkboards` | handleGetChalkboardsInMap | Configured |

---

## 8. Kafka Topics

| Topic | Direction | Purpose |
|-------|-----------|---------|
| TOPIC_CHALKBOARD_COMMAND | Consumer | Receive chalkboard set/clear commands |
| TOPIC_CHARACTER_STATUS | Consumer | Track character map transitions |
| TOPIC_CHALKBOARD_STATUS | Producer | Emit chalkboard status events |
