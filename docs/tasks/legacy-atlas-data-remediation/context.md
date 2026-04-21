# Atlas-Data Remediation Context

**Last Updated:** 2026-01-13

---

## Key Files

### Source Files Requiring Modification

| File | Location | Changes Needed |
|------|----------|----------------|
| consumable/resource.go | `services/atlas-data/atlas.com/data/consumable/resource.go` | Add error logging (lines 45-48) |
| cash/resource.go | `services/atlas-data/atlas.com/data/cash/resource.go` | Add error logging (lines 31-34) |
| commodity/resource.go | `services/atlas-data/atlas.com/data/commodity/resource.go` | Add error logging (lines 30-34) |
| etc/resource.go | `services/atlas-data/atlas.com/data/etc/resource.go` | Add error logging (lines 31-34) |
| pet/resource.go | `services/atlas-data/atlas.com/data/pet/resource.go` | Add error logging (lines 31-34) |
| setup/resource.go | `services/atlas-data/atlas.com/data/setup/resource.go` | Add error logging (lines 31-34) |
| map/resource.go | `services/atlas-data/atlas.com/data/map/resource.go` | Add error logging (lines 305-312), move models |
| map/rest.go | `services/atlas-data/atlas.com/data/map/rest.go` | Receive moved models |

### Test Files to Create

| File | Location |
|------|----------|
| map/resource_test.go | `services/atlas-data/atlas.com/data/map/resource_test.go` |
| monster/resource_test.go | `services/atlas-data/atlas.com/data/monster/resource_test.go` |
| npc/resource_test.go | `services/atlas-data/atlas.com/data/npc/resource_test.go` |
| skill/resource_test.go | `services/atlas-data/atlas.com/data/skill/resource_test.go` |
| equipment/resource_test.go | `services/atlas-data/atlas.com/data/equipment/resource_test.go` |
| consumable/resource_test.go | `services/atlas-data/atlas.com/data/consumable/resource_test.go` |
| cash/resource_test.go | `services/atlas-data/atlas.com/data/cash/resource_test.go` |
| commodity/resource_test.go | `services/atlas-data/atlas.com/data/commodity/resource_test.go` |
| etc/resource_test.go | `services/atlas-data/atlas.com/data/etc/resource_test.go` |
| pet/resource_test.go | `services/atlas-data/atlas.com/data/pet/resource_test.go` |
| quest/resource_test.go | `services/atlas-data/atlas.com/data/quest/resource_test.go` |
| reactor/resource_test.go | `services/atlas-data/atlas.com/data/reactor/resource_test.go` |
| setup/resource_test.go | `services/atlas-data/atlas.com/data/setup/resource_test.go` |
| characters/templates/resource_test.go | `services/atlas-data/atlas.com/data/characters/templates/resource_test.go` |
| data/resource_test.go | `services/atlas-data/atlas.com/data/data/resource_test.go` |

### Reference Files

| File | Purpose |
|------|---------|
| `docs/audits/atlas-data/audit.md` | Source audit document |
| `docs/audits/atlas-data/audit.json` | Structured audit data |
| `services/atlas-data/atlas.com/data/rest/handler.go` | Shared handler utilities |
| `services/atlas-data/atlas.com/data/document/storage.go` | Document storage pattern |
| `services/atlas-marriages/atlas.com/marriages/marriage/resource_test.go` | Test pattern reference |

---

## Key Decisions

### D1: Error Logging Strategy

**Decision:** Use `Errorf` for 500 errors, `Debugf` for 404 errors

**Rationale:**
- 500 errors indicate unexpected failures requiring investigation
- 404 errors are expected during normal operation (data not found)
- Consistent with existing patterns in the service

**Pattern:**
```go
// For 500 errors
d.Logger().WithError(err).Errorf("Unable to retrieve %s.", resourceType)

// For 404 errors (already in use)
d.Logger().WithError(err).Debugf("Unable to locate %s %d.", resourceType, id)
```

### D2: Model Relocation Scope

**Decision:** Move only input/request models from `resource.go` to `rest.go`

**Rationale:**
- `rest.go` should contain all JSON:API model definitions
- `resource.go` should contain only route registration and handlers
- Other domains already follow this pattern

**Models to Move:**
- `DropPositionRestModel`
- `PositionRestModel`
- `FootholdRestModel`

### D3: Test Scope

**Decision:** Focus handler tests on HTTP layer, not business logic

**Rationale:**
- XML parsing already tested in `reader_test.go`
- REST serialization tested in `rest_test.go`
- Handler tests verify HTTP routing, status codes, and response structure

**Test Categories:**
1. Success paths (200 OK)
2. Not found paths (404)
3. Error paths (500)
4. JSON:API structure compliance

### D4: Documentation Location

**Decision:** Document patterns in service README or guidelines supplement

**Rationale:**
- Service-specific deviations belong with the service
- Guidelines supplements capture approved exceptions
- Future maintainers check both locations

---

## Dependencies

### Package Dependencies

```
atlas-data dependencies:
├── atlas.com/rest          # REST utilities, JSON:API helpers
├── atlas.com/model         # Provider pattern
├── atlas.com/tenant        # Multi-tenancy context
├── atlas.com/database      # Transaction helpers
├── atlas.com/kafka         # Producer/Consumer
├── github.com/sirupsen/logrus
├── github.com/gorilla/mux
└── gorm.io/gorm
```

### Test Dependencies

```
test dependencies:
├── github.com/stretchr/testify/assert
├── github.com/stretchr/testify/require
├── net/http/httptest
└── gorm.io/driver/sqlite   # In-memory test DB
```

---

## Code Patterns

### Current Error Handling (Incomplete)

```go
// Current pattern - missing error logging
rp := document.NewStorage[uint32, RestModel](d.Logger(), db, GetModelRegistry(), "TYPE").AllProvider(d.Context())
ms, err := rp()
if err != nil {
    w.WriteHeader(http.StatusInternalServerError)
    return
}
```

### Target Error Handling (Complete)

```go
// Target pattern - with error logging
rp := document.NewStorage[uint32, RestModel](d.Logger(), db, GetModelRegistry(), "TYPE").AllProvider(d.Context())
ms, err := rp()
if err != nil {
    d.Logger().WithError(err).Errorf("Unable to retrieve TYPE data.")
    w.WriteHeader(http.StatusInternalServerError)
    return
}
```

### Handler Test Pattern

```go
func TestGetResource_Success(t *testing.T) {
    // Setup
    db := setupTestDB(t)
    router := setupRouter(db)
    server := httptest.NewServer(router)
    defer server.Close()

    // Seed test data
    seedTestData(t, db)

    // Execute
    req := createRequest(t, "GET", server.URL+"/resources/1", tenantID)
    resp, err := http.DefaultClient.Do(req)
    require.NoError(t, err)
    defer resp.Body.Close()

    // Assert
    assert.Equal(t, http.StatusOK, resp.StatusCode)

    var result map[string]interface{}
    err = json.NewDecoder(resp.Body).Decode(&result)
    require.NoError(t, err)

    data := result["data"].(map[string]interface{})
    assert.Equal(t, "resources", data["type"])
}
```

---

## Risk Mitigations

### Model Move Risks

**Risk:** Import cycles after moving models
**Mitigation:** Models have no dependencies on resource.go, safe to move

**Risk:** Handler compilation failures
**Mitigation:** Models remain in same package, no import changes needed in resource.go

### Test Implementation Risks

**Risk:** Complex test setup
**Mitigation:** Start with simplest domain (cash), establish pattern, replicate

**Risk:** Flaky tests due to shared state
**Mitigation:** Use isolated test databases, clean state between tests

---

## Open Questions

### Resolved

1. **Q:** Should we rename processor.go to loader.go?
   **A:** Deferred - low priority, document current pattern instead

2. **Q:** Should storebank filter use FilteredProvider?
   **A:** Deferred - current implementation works, not blocking

### Pending

1. Should handler tests use real document storage or mocks?
   - Recommendation: Real storage with in-memory SQLite for integration coverage

2. Should we add performance benchmarks?
   - Recommendation: Not in scope for this remediation, address separately

---

## Related Documentation

- `.claude/skills/backend-dev-guidelines/` - Backend development standards
- `docs/audits/atlas-data/audit.md` - Full audit report
- `docs/audits/atlas-data/audit.json` - Structured audit findings
