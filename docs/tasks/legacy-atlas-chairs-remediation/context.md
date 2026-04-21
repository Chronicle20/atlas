# Atlas-Chairs Remediation - Context Document

**Last Updated:** 2026-01-13

---

## 1. Key Files

### Service Files (to be modified/tested)
| File | Purpose | Modification Type |
|------|---------|-------------------|
| `services/atlas-chairs/atlas.com/chairs/chair/processor.go` | Chair business logic | Add tests |
| `services/atlas-chairs/atlas.com/chairs/chair/registry.go` | In-memory chair storage | Add tests |
| `services/atlas-chairs/atlas.com/chairs/character/processor.go` | Character tracking | Add tests |
| `services/atlas-chairs/atlas.com/chairs/character/registry.go` | Character map storage | Add tests |
| `services/atlas-chairs/atlas.com/chairs/rest/handler.go` | REST handler registration | Potential migration |

### Infrastructure Files
| File | Purpose | Modification Type |
|------|---------|-------------------|
| `atlas-ingress.yml:132-134` | Nginx ingress config | Add route |

### Reference Files (patterns to follow)
| File | Purpose |
|------|---------|
| `services/atlas-account/atlas.com/account/account/processor_test.go` | Test patterns |
| `services/atlas-account/atlas.com/account/account/rest_test.go` | REST test patterns |
| `dev/audits/atlas-chairs/audit.md` | Full audit findings |
| `dev/audits/atlas-chairs/audit.json` | Machine-readable audit data |

---

## 2. Key Decisions

### Decision 1: In-Memory Registry Pattern
**Status:** Keep as-is
**Rationale:** The service intentionally uses in-memory registries instead of database persistence. Chair state is transient and appropriately volatile. The registry pattern acts as both storage and provider, which is acceptable for this use case.

### Decision 2: Builder Pattern for Chair Model
**Status:** Deferred (low priority)
**Rationale:** The Chair model only has 2 fields (`id` and `chairType`). Builder pattern adds overhead without clear benefit. Will reconsider if validation requirements expand.

### Decision 3: Handler Migration Approach
**Status:** Investigate during implementation
**Rationale:** Local `rest/handler.go` may have specific functionality not in shared library. Need to analyze before committing to migration. Path parsers (`ParseCharacterId`, etc.) may need to remain local.

---

## 3. Domain Model Summary

### Chair Model (`chair/model.go`)
```go
type Model struct {
    id        uint32  // Chair ID (fixed chair index or portable item ID)
    chairType string  // "fixed" or "portable"
}
// Accessors: Id(), Type()
```

### Character MapKey (`character/model.go`)
```go
type MapKey struct {
    Tenant    tenant.Model
    WorldId   world.Id
    ChannelId channel.Id
    MapId     _map.Id
}
```

---

## 4. API Endpoints

### Endpoint 1: Get Chair by Character
```
GET /api/chairs/{characterId}
Response: JSON:API single resource
```
**Ingress Status:** MISSING (needs to be added)

### Endpoint 2: Get Chairs in Map
```
GET /api/worlds/{worldId}/channels/{channelId}/maps/{mapId}/chairs
Response: JSON:API resource collection
```
**Ingress Status:** Working

---

## 5. Kafka Topics

### Consumed Topics
- Character lifecycle events (enter/exit map, channel transitions)

### Produced Topics
- `chair.status` - Chair status events (USED, CANCELLED, ERROR)

---

## 6. Dependencies

### External Services
| Service | Purpose | Access Pattern |
|---------|---------|----------------|
| DATA service | Map metadata (seat counts) | REST call via `data/map/processor.go` |

### Shared Libraries
| Library | Used For |
|---------|----------|
| `github.com/Chronicle20/atlas-tenant` | Multi-tenancy context |
| `github.com/Chronicle20/atlas-rest/server` | HTTP server patterns |
| `github.com/Chronicle20/atlas-model/model` | Provider patterns |
| `github.com/Chronicle20/atlas-constants/*` | World/Channel/Map/Field types |

---

## 7. Test Setup Patterns

### Creating Test Context
```go
import (
    "context"
    "github.com/Chronicle20/atlas-tenant"
    "github.com/sirupsen/logrus/hooks/test"
    "github.com/google/uuid"
)

func sampleTenant() tenant.Model {
    return tenant.New(uuid.New(), "GMS", 83, 1)
}

func setupTest(t *testing.T) (logrus.FieldLogger, context.Context) {
    l, _ := test.NewNullLogger()
    st := sampleTenant()
    tctx := tenant.WithContext(context.Background(), st)
    return l, tctx
}
```

### Creating Test Field
```go
import "github.com/Chronicle20/atlas-constants/field"

f := field.NewBuilder(worldId, channelId, mapId).Build()
```

---

## 8. Blocking Issues Summary

| Issue | File | Line | Required Action |
|-------|------|------|-----------------|
| INFRA-001 | `atlas-ingress.yml` | ~134 | Add location block for `/api/chairs` |

---

## 9. Non-Blocking Issues Summary

| Issue | File(s) | Required Action |
|-------|---------|-----------------|
| TEST-001 | `chair/`, `character/` | Add `*_test.go` files |
| STRUCT-003 | `rest/handler.go` | Investigate migration to shared pattern |
| STRUCT-001 | `chair/model.go` | Consider builder (optional) |

---

## 10. Out of Scope

1. **TODO at `chair/processor.go:73`:** Portable chair item ownership validation is pre-existing technical debt and not part of this remediation.

2. **Character registry duplication:** The character registry intentionally duplicates state from other services. This is a design decision to enable the "chairs in map" query pattern.

3. **Database persistence:** Adding database persistence is not planned. The in-memory design is intentional.
