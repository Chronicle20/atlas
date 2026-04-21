# atlas-cashshop Service Remediation Plan

**Last Updated:** 2026-01-13
**Audit Source:** `docs/audits/atlas-cashshop/audit.md`
**Overall Status:** needs-work (76% pass rate)

---

## 1. Executive Summary

The `atlas-cashshop` service audit identified **4 non-blocking issues** across 17 architectural checks:

| Issue | Severity | Effort | Priority |
|-------|----------|--------|----------|
| Missing service README.md | Medium | M | P1 |
| Limited test coverage (2 files) | Medium | L | P1 |
| Transform functions access private fields | Low | S | P2 |
| Administrator/Provider naming inconsistency | Low | S | P3 |

**Goal:** Remediate all issues to achieve 100% compliance with Atlas backend architecture guidelines.

**Scope:** This plan covers documentation, code fixes, and comprehensive test coverage for the atlas-cashshop service.

---

## 2. Current State Analysis

### Service Overview

The `atlas-cashshop` service manages:
- **Wallet**: Account currency (credit, points, prepaid)
- **Wishlist**: Character wishlist items
- **Cash Shop Items**: Purchasable items with unique cash IDs
- **Inventory**: Account-level inventory with compartments (Explorer/Cygnus/Legend) and assets
- **External Integrations**: Character and commodity lookups via REST clients

### Issues by Category

#### 2.1 Documentation (ARCH-010: FAIL)
- No README.md exists at service root
- Missing API contract documentation
- Missing Kafka topic documentation

#### 2.2 Code Quality (ARCH-008, ARCH-016: WARN)
- Transform functions in `rest.go` files directly access private model fields
- Affected files: `wallet/rest.go`, `wishlist/rest.go`, `cashshop/item/rest.go`
- `cashshop/item/administrator.go` contains `createEntityProvider` with inconsistent naming

#### 2.3 Test Coverage (ARCH-012: WARN)
- Only 2 test files exist:
  - `cashshop/inventory/asset/reservation/cache_test.go`
  - `cashshop/inventory/rest_test.go`
- Missing processor tests for all domains
- Missing builder validation tests

---

## 3. Proposed Future State

After remediation:

| Check | Current | Target |
|-------|---------|--------|
| ARCH-008 (REST JSON:API) | warn | pass |
| ARCH-010 (Documentation) | fail | pass |
| ARCH-012 (Testing Coverage) | warn | pass |
| ARCH-016 (Admin/Provider Separation) | warn | pass |
| **Overall Pass Rate** | 76% | 100% |

---

## 4. Implementation Phases

### Phase 1: Documentation (P1)
**Goal:** Create comprehensive service documentation

#### Task 1.1: Create README.md
**Effort:** M | **Priority:** P1

Create `services/atlas-cashshop/atlas.com/cashshop/README.md` documenting:
- Service overview and purpose
- Domain model structure (wallet, wishlist, inventory, compartment, asset, item)
- REST API endpoints with request/response examples
- Kafka commands and events
- Environment variables
- Cross-service dependencies

**Acceptance Criteria:**
- [ ] README.md exists at service root
- [ ] All REST endpoints documented with HTTP methods and paths
- [ ] All Kafka topics documented with command/event types
- [ ] All environment variables documented with descriptions
- [ ] Cross-service dependencies listed (atlas-character, atlas-commodity)

**Reference:** Use `services/atlas-storage/atlas.com/storage/README.md` as template

---

### Phase 2: Code Fixes (P2/P3)
**Goal:** Resolve encapsulation and naming issues

#### Task 2.1: Fix wallet/rest.go Transform function
**Effort:** S | **Priority:** P2

**Location:** `services/atlas-cashshop/atlas.com/cashshop/wallet/rest.go:32-40`

**Current:**
```go
func Transform(m Model) (RestModel, error) {
    return RestModel{
        Id:        m.id,           // Direct field access
        AccountId: m.accountId,
        Credit:    m.credit,
        Points:    m.points,
        Prepaid:   m.prepaid,
    }, nil
}
```

**Target:**
```go
func Transform(m Model) (RestModel, error) {
    return RestModel{
        Id:        m.Id(),         // Use accessor
        AccountId: m.AccountId(),
        Credit:    m.Credit(),
        Points:    m.Points(),
        Prepaid:   m.Prepaid(),
    }, nil
}
```

**Acceptance Criteria:**
- [ ] Transform function uses accessor methods
- [ ] Service compiles without errors
- [ ] Existing tests pass

---

#### Task 2.2: Fix wishlist/rest.go Transform function
**Effort:** S | **Priority:** P2

**Location:** `services/atlas-cashshop/atlas.com/cashshop/wishlist/rest.go:30-36`

**Current:**
```go
func Transform(m Model) (RestModel, error) {
    return RestModel{
        Id:           m.id,
        CharacterId:  m.characterId,
        SerialNumber: m.serialNumber,
    }, nil
}
```

**Target:**
```go
func Transform(m Model) (RestModel, error) {
    return RestModel{
        Id:           m.Id(),
        CharacterId:  m.CharacterId(),
        SerialNumber: m.SerialNumber(),
    }, nil
}
```

**Acceptance Criteria:**
- [ ] Transform function uses accessor methods
- [ ] Service compiles without errors
- [ ] Existing tests pass

---

#### Task 2.3: Fix cashshop/item/rest.go Transform function
**Effort:** S | **Priority:** P2

**Location:** `services/atlas-cashshop/atlas.com/cashshop/cashshop/item/rest.go:31-39`

**Current:**
```go
func Transform(m Model) (RestModel, error) {
    return RestModel{
        Id:          m.id,
        CashId:      m.cashId,
        TemplateId:  m.templateId,
        Quantity:    m.quantity,
        Flag:        m.flag,
        PurchasedBy: m.purchasedBy,
    }, nil
}
```

**Target:**
```go
func Transform(m Model) (RestModel, error) {
    return RestModel{
        Id:          m.Id(),
        CashId:      m.CashId(),
        TemplateId:  m.TemplateId(),
        Quantity:    m.Quantity(),
        Flag:        m.Flag(),
        PurchasedBy: m.PurchasedBy(),
    }, nil
}
```

**Acceptance Criteria:**
- [ ] Transform function uses accessor methods
- [ ] Service compiles without errors
- [ ] Existing tests pass

---

#### Task 2.4: Rename createEntityProvider in item/administrator.go
**Effort:** S | **Priority:** P3

**Location:** `services/atlas-cashshop/atlas.com/cashshop/cashshop/item/administrator.go:25-51`

**Current:** `createEntityProvider` - uses "Provider" suffix but performs write operations

**Target:** `create` - follows administrator.go naming conventions

**Changes Required:**
1. Rename function from `createEntityProvider` to `create`
2. Update all callers in `processor.go`

**Acceptance Criteria:**
- [ ] Function renamed to `create`
- [ ] All callers updated
- [ ] Service compiles without errors
- [ ] Existing tests pass

---

### Phase 3: Test Coverage (P1)
**Goal:** Add comprehensive unit tests for all domains

#### Task 3.1: Add wallet/processor_test.go
**Effort:** M | **Priority:** P1

Test coverage for:
- `GetByAccountId` - retrieve wallet by account
- `Create` / `CreateAndEmit` - wallet creation with Kafka emission
- `Update` / `UpdateAndEmit` - wallet updates
- `Delete` / `DeleteAndEmit` - wallet deletion
- `Purchase` - currency deduction

**Acceptance Criteria:**
- [ ] Test file exists at `wallet/processor_test.go`
- [ ] All processor methods have test coverage
- [ ] Tests use mocks for database and Kafka
- [ ] Tests validate multi-tenancy isolation

---

#### Task 3.2: Add wallet/rest_test.go
**Effort:** S | **Priority:** P1

Test coverage for:
- `Transform` - Model to RestModel conversion
- `Extract` - RestModel to Model conversion
- `GetName`, `GetID`, `SetID` - JSON:API interface

**Acceptance Criteria:**
- [ ] Test file exists at `wallet/rest_test.go`
- [ ] Transform and Extract produce correct values
- [ ] JSON:API interface methods tested

---

#### Task 3.3: Add wishlist/processor_test.go
**Effort:** M | **Priority:** P1

Test coverage for:
- `GetByCharacterId` - retrieve wishlist items
- `Create` / `CreateAndEmit` - add to wishlist
- `Delete` / `DeleteAndEmit` - remove from wishlist

**Acceptance Criteria:**
- [ ] Test file exists at `wishlist/processor_test.go`
- [ ] All processor methods have test coverage
- [ ] Tests validate character ownership

---

#### Task 3.4: Add wishlist/rest_test.go
**Effort:** S | **Priority:** P1

Test coverage for:
- `Transform` - Model to RestModel conversion
- `Extract` - RestModel to Model conversion

**Acceptance Criteria:**
- [ ] Test file exists at `wishlist/rest_test.go`
- [ ] Transform and Extract tested

---

#### Task 3.5: Add cashshop/item/processor_test.go
**Effort:** M | **Priority:** P1

Test coverage for:
- `GetById` - retrieve item by ID
- `GetByCashId` - retrieve by unique cash ID
- `Create` / `CreateAndEmit` - item creation with unique cash ID generation
- `Delete` / `DeleteAndEmit` - item deletion

**Acceptance Criteria:**
- [ ] Test file exists at `cashshop/item/processor_test.go`
- [ ] Unique cash ID generation tested
- [ ] All processor methods covered

---

#### Task 3.6: Add cashshop/item/rest_test.go
**Effort:** S | **Priority:** P1

Test coverage for:
- `Transform` - Model to RestModel conversion
- `Extract` - RestModel to Model conversion

**Acceptance Criteria:**
- [ ] Test file exists at `cashshop/item/rest_test.go`
- [ ] Transform and Extract tested

---

#### Task 3.7: Add cashshop/inventory/compartment/processor_test.go
**Effort:** M | **Priority:** P1

Test coverage for:
- `GetById` - retrieve compartment
- `GetByInventoryId` - retrieve compartments for inventory
- `Create` / `CreateAndEmit` - compartment creation
- `AddAsset` - asset assignment
- Capacity validation

**Acceptance Criteria:**
- [ ] Test file exists at `cashshop/inventory/compartment/processor_test.go`
- [ ] All processor methods covered
- [ ] Capacity limits validated

---

#### Task 3.8: Add cashshop/inventory/compartment/rest_test.go
**Effort:** S | **Priority:** P1

Test coverage for compartment REST transformations.

**Acceptance Criteria:**
- [ ] Test file exists
- [ ] Transform and Extract tested

---

#### Task 3.9: Add cashshop/inventory/asset/processor_test.go
**Effort:** M | **Priority:** P1

Test coverage for:
- `GetById` - retrieve asset
- `GetByCompartmentId` - retrieve assets in compartment
- `Create` / `CreateAndEmit` - asset creation
- `Delete` / `DeleteAndEmit` - asset deletion

**Acceptance Criteria:**
- [ ] Test file exists at `cashshop/inventory/asset/processor_test.go`
- [ ] All processor methods covered
- [ ] Item-asset relationship validated

---

#### Task 3.10: Add cashshop/inventory/asset/rest_test.go
**Effort:** S | **Priority:** P1

Test coverage for asset REST transformations.

**Acceptance Criteria:**
- [ ] Test file exists
- [ ] Transform and Extract tested

---

#### Task 3.11: Add builder validation tests
**Effort:** M | **Priority:** P2

Add tests for builder patterns:
- `cashshop/item/model.go` Builder
- `cashshop/inventory/compartment/model.go` ModelBuilder
- `cashshop/inventory/model.go` ModelBuilder

**Acceptance Criteria:**
- [ ] Builder tests exist for item, compartment, inventory
- [ ] Required field validation tested
- [ ] Build() produces correct models

---

## 5. Risk Assessment and Mitigation

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Transform accessor changes break functionality | Medium | Low | Run existing tests, manual API verification |
| Administrator rename breaks callers | Medium | Low | Search all references before renaming |
| Test mocking complexity | Low | Medium | Use existing test patterns from other services |
| Documentation becomes outdated | Low | Medium | Keep README close to code, update during changes |

---

## 6. Success Metrics

| Metric | Current | Target |
|--------|---------|--------|
| Audit pass rate | 76% | 100% |
| Test file count | 2 | 12+ |
| Documentation coverage | 0% | 100% |
| Transform encapsulation violations | 3 | 0 |
| Naming inconsistencies | 1 | 0 |

---

## 7. Required Resources and Dependencies

### Dependencies
- Access to `services/atlas-cashshop` source code
- Go test framework
- Existing test patterns from `atlas-storage`, `atlas-account`

### External Service Knowledge
- `atlas-character` REST API (for client integration tests)
- `atlas-commodity` REST API (for client integration tests)
- Kafka topic configurations

---

## 8. File Summary

### Files to Create
| File | Purpose |
|------|---------|
| `README.md` | Service documentation |
| `wallet/processor_test.go` | Wallet processor tests |
| `wallet/rest_test.go` | Wallet REST tests |
| `wishlist/processor_test.go` | Wishlist processor tests |
| `wishlist/rest_test.go` | Wishlist REST tests |
| `cashshop/item/processor_test.go` | Item processor tests |
| `cashshop/item/rest_test.go` | Item REST tests |
| `cashshop/inventory/compartment/processor_test.go` | Compartment processor tests |
| `cashshop/inventory/compartment/rest_test.go` | Compartment REST tests |
| `cashshop/inventory/asset/processor_test.go` | Asset processor tests |
| `cashshop/inventory/asset/rest_test.go` | Asset REST tests |

### Files to Modify
| File | Change |
|------|--------|
| `wallet/rest.go` | Use accessor methods in Transform |
| `wishlist/rest.go` | Use accessor methods in Transform |
| `cashshop/item/rest.go` | Use accessor methods in Transform |
| `cashshop/item/administrator.go` | Rename createEntityProvider to create |
| `cashshop/item/processor.go` | Update caller of renamed function |
