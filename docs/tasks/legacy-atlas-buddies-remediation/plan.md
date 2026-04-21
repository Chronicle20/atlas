# Atlas Buddies Service Remediation Plan

**Service:** `atlas-buddies`
**Service Path:** `services/atlas-buddies/atlas.com/buddies`
**Audit Reference:** `docs/audits/atlas-buddies/audit.md`
**Last Updated:** 2026-01-13

---

## 1. Executive Summary

This plan addresses 4 non-blocking issues and 1 structural gap identified in the `atlas-buddies` service audit. The service is functional and follows core Atlas patterns, but requires minor corrections to achieve full compliance with backend development guidelines.

**Audit Status:** `needs-work`
**Blocking Issues:** 0
**Non-Blocking Issues:** 4
**Estimated Total Effort:** Small (S) - All fixes are straightforward code corrections

### Key Objectives
1. Fix JSON:API interface compliance (`GetName()` receiver type)
2. Remove dead code in REST handler
3. Enforce immutability pattern consistency
4. Optionally add builder pattern for domain models

---

## 2. Current State Analysis

### What's Working Well
- Model immutability with private fields and accessors
- Proper Entity/Model separation with `Make()` transformation
- Kafka producer/consumer patterns correctly implemented
- Layer separation (handlers -> processors -> administrators)
- Good test coverage for critical paths

### Issues Identified

| Issue ID | Severity | Location | Description |
|----------|----------|----------|-------------|
| REST-001 | Medium | `character/rest.go:14` | `GetName()` uses pointer receiver instead of value receiver |
| CODE-001 | Low | `list/resource.go:106-118` | Commented-out code in handler; returns 202 but does nothing |
| CODE-002 | Low | `list/resource.go:93` | Direct field access `bl.buddies` instead of `bl.Buddies()` |
| ARCH-004 | Low | `list/`, `buddy/` | Missing `builder.go` files for model construction |

---

## 3. Proposed Future State

After remediation, the service will:
- Have all JSON:API interface methods using correct receiver types
- Contain no dead/commented code in handlers
- Consistently use accessor methods for model field access
- (Optional) Have builder patterns for model construction with validation

---

## 4. Implementation Phases

### Phase 1: Critical Fixes (Priority P1)

Quick fixes that restore compliance with Atlas guidelines.

#### Task 1.1: Fix GetName() Receiver Type
**File:** `services/atlas-buddies/atlas.com/buddies/character/rest.go`
**Line:** 14
**Effort:** S

**Current Code:**
```go
func (r *RestModel) GetName() string {
    return "characters"
}
```

**Required Change:**
```go
func (r RestModel) GetName() string {
    return "characters"
}
```

**Acceptance Criteria:**
- [x] `GetName()` method uses value receiver
- [x] Service compiles without errors
- [x] Existing tests pass

**Status:** COMPLETED

---

#### Task 1.2: Clean Up Dead Handler Code
**File:** `services/atlas-buddies/atlas.com/buddies/list/resource.go`
**Lines:** 106-118
**Effort:** S

**Current Code:**
```go
func handleAddBuddyToBuddyList(d *rest.HandlerDependency, _ *rest.HandlerContext, i buddy.RestModel) http.HandlerFunc {
    return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            //err := producer.ProviderImpl(d.Logger())(d.Context())(EnvCommandTopic)(addBuddyCommandProvider(characterId, i.CharacterId, i.Group, i.CharacterName, i.ChannelId, i.Visible))
            //if err != nil {
            //  w.WriteHeader(http.StatusInternalServerError)
            //  return
            //}

            w.WriteHeader(http.StatusAccepted)
        }
    })
}
```

**Decision:** Remove the endpoint entirely (decided 2026-01-13)

**Changes Made:**
- Removed `handleAddBuddyToBuddyList` function
- Removed route registration in `InitResource` (line 34)
- Removed `AddBuddyToBuddyList` constant (line 24)

**Acceptance Criteria:**
- [x] No commented-out code remains in handler
- [x] Endpoint removed and no longer registered
- [x] Service compiles without errors
- [x] Existing tests pass

**Status:** COMPLETED

---

### Phase 2: Pattern Consistency (Priority P2)

Fixes that improve code consistency with established patterns.

#### Task 2.1: Fix Direct Field Access
**File:** `services/atlas-buddies/atlas.com/buddies/list/resource.go`
**Line:** 93
**Effort:** S

**Current Code:**
```go
res, err := model.SliceMap(buddy.Transform)(model.FixedProvider(bl.buddies))()()
```

**Required Change:**
```go
res, err := model.SliceMap(buddy.Transform)(model.FixedProvider(bl.Buddies()))()()
```

**Acceptance Criteria:**
- [x] Direct field access replaced with accessor method
- [x] Service compiles without errors
- [x] Existing tests pass

**Status:** COMPLETED

---

### Phase 3: Optional Enhancements (Priority P3)

Optional improvements that add validation and improve maintainability.

#### Task 3.1: Add Builder for list.Model
**File:** `services/atlas-buddies/atlas.com/buddies/list/builder.go`
**Effort:** M

**Description:** Create builder pattern for `list.Model` following the established pattern in `atlas-account/account/builder.go`.

**Implementation:**
```go
package list

import (
    "atlas-buddies/buddy"
    "errors"
    "github.com/google/uuid"
)

type Builder struct {
    tenantId    uuid.UUID
    id          uuid.UUID
    characterId uint32
    capacity    byte
    buddies     []buddy.Model
}

func NewBuilder(tenantId uuid.UUID, characterId uint32) *Builder {
    return &Builder{
        tenantId:    tenantId,
        characterId: characterId,
        capacity:    20, // default capacity
        buddies:     []buddy.Model{},
    }
}

func (b *Builder) SetId(id uuid.UUID) *Builder {
    b.id = id
    return b
}

func (b *Builder) SetCapacity(capacity byte) *Builder {
    b.capacity = capacity
    return b
}

func (b *Builder) SetBuddies(buddies []buddy.Model) *Builder {
    b.buddies = buddies
    return b
}

func (b *Builder) Build() (Model, error) {
    if b.tenantId == uuid.Nil {
        return Model{}, errors.New("tenantId is required")
    }
    if b.characterId == 0 {
        return Model{}, errors.New("characterId is required")
    }
    if b.capacity == 0 {
        return Model{}, errors.New("capacity must be greater than 0")
    }

    return Model{
        tenantId:    b.tenantId,
        id:          b.id,
        characterId: b.characterId,
        capacity:    b.capacity,
        buddies:     b.buddies,
    }, nil
}
```

**Acceptance Criteria:**
- [x] Builder struct created with fluent API methods
- [x] Build() validates required fields
- [x] Unit tests added for builder (7 tests)

**Status:** COMPLETED

---

#### Task 3.2: Add Builder for buddy.Model
**File:** `services/atlas-buddies/atlas.com/buddies/buddy/builder.go`
**Effort:** M

**Description:** Create builder pattern for `buddy.Model` for consistent model construction.

**Acceptance Criteria:**
- [x] Builder struct created with fluent API methods
- [x] Build() validates required fields (listId, characterId)
- [x] Unit tests added for builder (5 tests)

**Status:** COMPLETED

---

## 5. Risk Assessment and Mitigation

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Receiver type change breaks marshaling | Medium | Low | JSON:API spec requires value receivers for GetName; change is a correction |
| Removing endpoint breaks clients | Medium | Low | Endpoint currently does nothing; verify no clients depend on it |
| Field access change causes runtime error | Low | Very Low | Both `bl.buddies` and `bl.Buddies()` return same value |
| Builder introduction causes regression | Low | Low | Builder is additive; existing code paths continue to work |

---

## 6. Success Metrics

- [x] All 4 non-blocking issues resolved
- [x] Service compiles without errors
- [x] All existing tests pass
- [x] Builder patterns added for list.Model and buddy.Model
- [x] 12 new unit tests added for builders
- [ ] Re-audit shows status: `pass`

---

## 7. Required Resources and Dependencies

### Prerequisites
- Go 1.21+ development environment
- Access to atlas-buddies service source code
- Understanding of JSON:API specification

### External Dependencies
- None - all changes are internal to the service

### Review Requirements
- Code review before merge
- Re-run audit to verify compliance

---

## 8. Verification Steps

After implementing all changes:

1. **Build verification:**
   ```bash
   cd services/atlas-buddies/atlas.com/buddies
   go build ./...
   ```

2. **Test verification:**
   ```bash
   go test ./...
   ```

3. **Re-audit:**
   Run `/backend-audit atlas-buddies` to verify all issues resolved

---

## 9. Notes and Decisions

### Decision Log

| Decision | Rationale | Date |
|----------|-----------|------|
| Recommend removing dead endpoint | Handler does nothing; keeping it suggests functionality that doesn't exist | 2026-01-13 |
| Builder tasks marked optional | Current direct construction works; builder adds value only if validation becomes complex | 2026-01-13 |

### Open Questions

1. **Should the `handleAddBuddyToBuddyList` endpoint be implemented or removed?**
   - Requires product/design input on whether "add buddy via REST" is a planned feature
   - Current recommendation: Remove unless feature is planned

2. **Is the buddy package ever needed for independent operations?**
   - Current design always accesses buddies through their parent list
   - If independent buddy operations are needed, complete the package structure
