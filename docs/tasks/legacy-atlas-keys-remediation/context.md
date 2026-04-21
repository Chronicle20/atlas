# Atlas-Keys Remediation - Context

**Last Updated:** 2026-01-13

---

## Key Files

### Audit Files
| File | Purpose |
|------|---------|
| `dev/audits/atlas-keys/audit.md` | Full audit report with findings |
| `dev/audits/atlas-keys/audit.json` | Machine-readable audit data |

### Service Files to Modify
| File | Lines | Changes Needed |
|------|-------|----------------|
| `services/atlas-keys/atlas.com/keys/key/model.go` | 21 | Add CharacterId() accessor |
| `services/atlas-keys/atlas.com/keys/key/entity.go` | 23 | Add Make() and ToEntity() functions |
| `services/atlas-keys/atlas.com/keys/key/processor.go` | 143 | Remove makeKey function (moved to entity.go) |
| `services/atlas-keys/atlas.com/keys/key/rest.go` | 35 | Optionally add TransformSlice() |

### New Files to Create
| File | Purpose |
|------|---------|
| `key/builder.go` | Fluent model builder with validation |
| `key/builder_test.go` | Builder validation tests |
| `key/processor_test.go` | Processor business logic tests |
| `key/mock/processor.go` | Mock implementation of Processor interface |
| `character/resource_test.go` | REST handler tests (optional) |

---

## Reference Implementations

### Builder Pattern
**File:** `services/atlas-expressions/atlas.com/expressions/expression/builder.go`

Key elements to replicate:
- `ModelBuilder` struct with private fields matching Model
- `NewModelBuilder(t tenant.Model)` constructor
- `CloneModelBuilder(m Model)` for cloning
- Fluent setters returning `*ModelBuilder`
- `Build() (Model, error)` with validation
- `MustBuild() Model` for trusted sources
- Accessor methods on builder

### Mock Pattern
**File:** `services/atlas-expressions/atlas.com/expressions/expression/mock/processor.go`

Key elements to replicate:
- Struct with `*Func` fields for each interface method
- Each method checks if function is set, calls it, or returns zero value
- Enables test-specific behavior injection

---

## Decisions Made

### Decision 1: Validation Rules for Builder
**Context:** What validation should the builder enforce?
**Decision:** Start with minimal validation:
- `characterId > 0` (required)
- No tenant validation (not stored in Model, only in entity)

**Rationale:** The Model doesn't contain tenant information (it's only on the entity), so we can't validate it. Keep validation simple and focused on actual Model fields.

### Decision 2: Entity Transformation Naming
**Context:** Should we use `Make` or `makeKey`?
**Decision:** Use `Make(entity) (Model, error)` for public API.

**Rationale:** Follows established pattern in other services. The function transforms an entity to a model, which is a common operation that should be public.

### Decision 3: ToEntity Implementation
**Context:** Should Model have a ToEntity method?
**Decision:** Yes, add `ToEntity(tenantId uuid.UUID) entity` method.

**Rationale:** Required for round-trip transformations. The tenant ID must be passed as parameter since Model doesn't store it.

### Decision 4: Mock Location
**Context:** Should mocks be in `key/mock/` or a top-level `mock/` package?
**Decision:** Use `key/mock/processor.go`.

**Rationale:** Follows pattern established in other services (e.g., atlas-expressions). Keeps mocks close to the interfaces they mock.

---

## Dependencies Between Changes

```
CharacterId accessor (independent)
         │
         v
Mock infrastructure ──────────────┐
         │                        │
         v                        v
Entity transformation ──> Processor tests
         │
         v
Builder pattern ───────────> Builder tests
```

**Critical Path:**
1. Entity transformation must happen before builder (builder may use Make internally)
2. Mock infrastructure must exist before processor tests
3. CharacterId accessor should be done first (other code may need it)

---

## Gotchas and Edge Cases

### 1. Model Has No Tenant Field
The `Model` struct doesn't include tenant information - it's only on the `entity`. This means:
- Builder cannot validate tenant
- `ToEntity()` must accept tenant ID as parameter
- This is intentional - Model represents domain data, entity represents storage

### 2. Default Keys Are Hardcoded
The default key mappings in `processor.go:12-14` are hardcoded arrays. These define:
- 40 default key bindings
- Types range from 4-6
- Actions range from 0-106

Consider these ranges when implementing builder validation.

### 3. Transaction ID Parameter Unused
Methods like `Reset`, `CreateDefault`, `Delete`, `ChangeKey` accept `transactionId` but don't use it. This is likely for future tracing. Don't remove these parameters.

### 4. Composite Primary Key
The entity uses a composite primary key: `(CharacterId, Key)`. This affects:
- How updates work (must match both fields)
- How deletes work (can delete by CharacterId alone)

---

## Testing Considerations

### Database Testing Options

**Option A: Mock Database (Recommended for unit tests)**
- Mock the GORM `*gorm.DB`
- Use mock processor for handler tests
- Fast, isolated, no external dependencies

**Option B: Integration Tests with Real DB**
- Use testcontainers for PostgreSQL
- Run actual queries
- Slower but more realistic
- Consider for critical paths only

### Test Data
Default key bindings provide good test data:
```go
defaultKey = []int32{18, 65, 2, 23, 3, 4, 5, 6, 16, 17, ...}
defaultType = []int8{4, 6, 4, 4, 4, 4, 4, 4, 4, 4, ...}
defaultAction = []int32{0, 106, 10, 1, 12, 13, 18, 24, 8, 5, ...}
```

---

## Validation After Remediation

Run these commands to verify remediation is complete:

```bash
# Run all tests
cd services/atlas-keys/atlas.com/keys
go test ./...

# Check test coverage
go test -cover ./...

# Verify no lint issues
golangci-lint run

# Re-run audit
# (if automated audit tooling exists)
```

### Expected Audit Results After Remediation
| Check | Before | After |
|-------|--------|-------|
| ARCH-002 | WARN | PASS |
| ARCH-003 | FAIL | PASS |
| ARCH-006 | FAIL | PASS |
| TEST-001 | FAIL | PASS |
| TEST-002 | FAIL | PASS |
