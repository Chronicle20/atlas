# atlas-cashshop Remediation - Context

**Last Updated:** 2026-01-13

---

## Key Files

### Audit Source
- `dev/audits/atlas-cashshop/audit.md` - Full audit report
- `dev/audits/atlas-cashshop/audit.json` - Machine-readable audit data

### Service Root
- `services/atlas-cashshop/atlas.com/cashshop/`

### Files Requiring Modification

#### Transform Function Fixes (P2)
| File | Lines | Issue |
|------|-------|-------|
| `wallet/rest.go` | 32-40 | Uses `m.id`, `m.accountId`, etc. instead of accessors |
| `wishlist/rest.go` | 30-36 | Uses `m.id`, `m.characterId`, `m.serialNumber` directly |
| `cashshop/item/rest.go` | 31-39 | Uses `m.id`, `m.cashId`, etc. directly |

#### Administrator Rename (P3)
| File | Lines | Issue |
|------|-------|-------|
| `cashshop/item/administrator.go` | 25-51 | `createEntityProvider` → `create` |
| `cashshop/item/processor.go` | TBD | Update caller of renamed function |

### Reference Templates

#### README Template
- `services/atlas-storage/atlas.com/storage/README.md` - Use as documentation template

#### Test Patterns
- `cashshop/inventory/asset/reservation/cache_test.go` - Existing cache tests
- `cashshop/inventory/rest_test.go` - Existing REST transform tests

---

## Domain Structure

```
services/atlas-cashshop/atlas.com/cashshop/
├── wallet/                    # Account currency domain
│   ├── administrator.go       # Write: createEntity, updateEntity, deleteEntity
│   ├── entity.go             # GORM entity, Migration, Make
│   ├── model.go              # Immutable model with accessors
│   ├── processor.go          # Business logic orchestration
│   ├── provider.go           # Read: byAccountIdEntityProvider
│   ├── resource.go           # REST handler registration
│   └── rest.go               # JSON:API RestModel, Transform, Extract
│
├── wishlist/                  # Character wishlist domain
│   ├── administrator.go
│   ├── entity.go
│   ├── model.go
│   ├── processor.go
│   ├── provider.go
│   ├── resource.go
│   └── rest.go               # NEEDS FIX: Transform uses direct field access
│
├── cashshop/                  # Cash shop core domain
│   ├── processor.go          # Top-level orchestration
│   │
│   ├── item/                  # Cash items subdomain
│   │   ├── administrator.go  # NEEDS FIX: createEntityProvider naming
│   │   ├── entity.go
│   │   ├── model.go          # Builder pattern
│   │   ├── processor.go
│   │   ├── provider.go
│   │   ├── resource.go
│   │   └── rest.go           # NEEDS FIX: Transform uses direct field access
│   │
│   ├── inventory/             # Inventory aggregate
│   │   ├── model.go          # Virtual model (no entity)
│   │   ├── processor.go
│   │   ├── resource.go
│   │   ├── rest.go
│   │   ├── rest_test.go      # EXISTING TEST
│   │   │
│   │   ├── compartment/       # Compartment subdomain
│   │   │   ├── administrator.go
│   │   │   ├── entity.go
│   │   │   ├── model.go      # ModelBuilder
│   │   │   ├── processor.go
│   │   │   ├── provider.go
│   │   │   ├── resource.go
│   │   │   └── rest.go
│   │   │
│   │   └── asset/             # Asset subdomain
│   │       ├── administrator.go
│   │       ├── entity.go
│   │       ├── model.go
│   │       ├── processor.go
│   │       ├── provider.go
│   │       ├── resource.go
│   │       ├── rest.go
│   │       └── reservation/
│   │           ├── cache.go
│   │           └── cache_test.go  # EXISTING TEST
│   │
│   └── commodity/             # External REST client
│       ├── model.go
│       ├── processor.go
│       ├── requests.go
│       └── rest.go
│
├── character/                 # External REST client
│   ├── model.go
│   ├── processor.go
│   ├── requests.go
│   ├── rest.go
│   ├── inventory/
│   ├── compartment/
│   └── equipment/
│
├── kafka/
│   ├── consumer/              # account, character, item, wallet, cashshop
│   ├── message/               # Message buffer
│   └── producer/              # wallet, wishlist, item, asset, compartment
│
├── rest/                      # Custom handler abstraction
├── database/
├── logger/
├── service/
├── tracing/
├── retry/
└── main.go
```

---

## Model Accessors (Required for Transform fixes)

### wallet/model.go
```go
func (m Model) Id() uuid.UUID        // Line ~13
func (m Model) AccountId() uint32    // Line ~17
func (m Model) Credit() uint32       // Line ~21
func (m Model) Points() uint32       // Line ~25
func (m Model) Prepaid() uint32      // Line ~29
```

### wishlist/model.go
```go
func (m Model) Id() uuid.UUID           // Line ~11
func (m Model) CharacterId() uint32     // Line ~15
func (m Model) SerialNumber() uint32    // Line ~19
```

### cashshop/item/model.go
```go
func (m Model) Id() uint32          // Line ~15
func (m Model) CashId() int64       // Line ~19
func (m Model) TemplateId() uint32  // Line ~23
func (m Model) Quantity() uint32    // Line ~27
func (m Model) Flag() uint16        // Line ~31
func (m Model) PurchasedBy() uint32 // Line ~35
```

---

## Dependencies Between Tasks

```
Phase 1 (Documentation)
└── Task 1.1: Create README.md
    └── No dependencies

Phase 2 (Code Fixes)
├── Task 2.1: Fix wallet/rest.go      → No dependencies
├── Task 2.2: Fix wishlist/rest.go    → No dependencies
├── Task 2.3: Fix item/rest.go        → No dependencies
└── Task 2.4: Rename administrator.go → No dependencies

Phase 3 (Tests)
├── Task 3.1: wallet/processor_test.go  → After Task 2.1
├── Task 3.2: wallet/rest_test.go       → After Task 2.1
├── Task 3.3: wishlist/processor_test.go → After Task 2.2
├── Task 3.4: wishlist/rest_test.go     → After Task 2.2
├── Task 3.5: item/processor_test.go    → After Task 2.3, 2.4
├── Task 3.6: item/rest_test.go         → After Task 2.3
├── Task 3.7: compartment/processor_test.go → No dependencies
├── Task 3.8: compartment/rest_test.go  → No dependencies
├── Task 3.9: asset/processor_test.go   → No dependencies
├── Task 3.10: asset/rest_test.go       → No dependencies
└── Task 3.11: Builder tests            → No dependencies
```

---

## Decisions Made

| Decision | Rationale |
|----------|-----------|
| Use accessor methods in Transform | Maintains encapsulation, follows other services' patterns |
| Rename `createEntityProvider` to `create` | Matches naming convention in other administrator.go files |
| Create test files per domain | Follows Go convention, matches audit recommendations |
| Use atlas-storage README as template | Most comprehensive existing documentation |

---

## Kafka Topics (for README documentation)

### Commands
- `COMMAND_TOPIC_CASHSHOP` - Cash shop operations
- `COMMAND_TOPIC_WALLET` - Wallet operations
- `COMMAND_TOPIC_WISHLIST` - Wishlist operations

### Events
- `EVENT_TOPIC_WALLET_STATUS` - Wallet status events
- `EVENT_TOPIC_WISHLIST_STATUS` - Wishlist status events
- `EVENT_TOPIC_CASHSHOP_ITEM_STATUS` - Item status events
- `EVENT_TOPIC_CASHSHOP_ASSET_STATUS` - Asset status events
- `EVENT_TOPIC_CASHSHOP_COMPARTMENT_STATUS` - Compartment status events

### Consumers Listen To
- `EVENT_TOPIC_ACCOUNT_STATUS` - Account creation triggers wallet creation
- `EVENT_TOPIC_CHARACTER_STATUS` - Character events
- `EVENT_TOPIC_CASHSHOP_ITEM_STATUS` - Item lifecycle events

---

## REST Endpoints (for README documentation)

### Wallet
- `GET /api/accounts/{accountId}/wallet` - Get account wallet
- `PATCH /api/accounts/{accountId}/wallet` - Update wallet

### Wishlist
- `GET /api/characters/{characterId}/cash-shop/wishlist` - Get wishlist
- `POST /api/characters/{characterId}/cash-shop/wishlist` - Add to wishlist
- `DELETE /api/characters/{characterId}/cash-shop/wishlist/{itemId}` - Remove from wishlist

### Inventory
- `GET /api/accounts/{accountId}/cash-shop/inventory` - Get full inventory
- `GET /api/accounts/{accountId}/cash-shop/inventory/compartments` - Get compartments
- `GET /api/accounts/{accountId}/cash-shop/inventory/compartments/{compartmentId}` - Get compartment
- `GET /api/accounts/{accountId}/cash-shop/inventory/assets` - Get assets
- `GET /api/accounts/{accountId}/cash-shop/inventory/assets/{assetId}` - Get asset

### Cash Shop Items (Reference Data)
- `GET /api/cash-shop/items` - List available items
- `GET /api/cash-shop/items/{itemId}` - Get item details
