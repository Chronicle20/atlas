# Merchant Audit Remediation — Context

- **Last Updated:** 2026-02-24

---

## Audit Source

- `dev/audits/atlas-merchant/audit.md` — Full audit findings
- `dev/audits/atlas-merchant/audit.json` — Machine-readable audit data

---

## Key Files to Modify

### Phase 1 — Blocking Issues

| File | Action | Audit Check |
|------|--------|-------------|
| `shop/provider.go` | Remove `create()`, `update()` | ARCH-001 |
| `shop/administrator.go` | **CREATE** — move `create()`, `update()` here | ARCH-001 |
| `listing/provider.go` | Remove 6 write functions | ARCH-001 |
| `listing/administrator.go` | **CREATE** — move 6 write functions here | ARCH-001 |
| `atlas-ingress.yml` | Add merchant route (alphabetically) | STRUCT-001 |

### Phase 2 — Subdomain Layering

| File | Action | Audit Check |
|------|--------|-------------|
| `frederick/model.go` | **CREATE** — ItemModel, MesoModel immutable types | STRUCT-004 |
| `frederick/administrator.go` | **CREATE** — storeItems, storeMesos, clearItems, clearMesos, createNotification, clearNotifications | ARCH-002 |
| `frederick/processor.go` | Refactor to delegate to administrator/provider; return models not entities | ARCH-002, STRUCT-004 |
| `frederick/provider.go` | Ensure only reads; processor returns models via Make functions | ARCH-002 |
| `message/model.go` | **CREATE** — immutable Message Model | STRUCT-005 |
| `message/administrator.go` | **CREATE** — create() write operation | ARCH-003 |
| `message/provider.go` | **CREATE** — getByShopId() read operation | ARCH-003 |
| `message/processor.go` | Refactor to delegate to administrator/provider | ARCH-003 |
| `kafka/consumer/merchant/consumer.go` | Update Frederick retrieval handler for new model types | STRUCT-004 |
| `shop/processor.go` | Update storeToFrederick for new model types | STRUCT-004 |

### Phase 3 — Kafka Event Correctness

| File | Action | Audit Check |
|------|--------|-------------|
| `shop/processor.go` | Change `ExitMaintenance` return type to include auto-close info | KAFKA-004 |
| `kafka/consumer/merchant/consumer.go` | Fix handleExitMaintenanceCommand event emission | KAFKA-004 |
| `kafka/consumer/character/consumer.go` | Add StatusEventShopClosed emission on logout | KAFKA-004 |

### Phase 4 — REST Convention Alignment

| File | Action | Audit Check |
|------|--------|-------------|
| `rest/handler.go` | Evaluate migration to standard server.RegisterHandler | ARCH-004 |
| `shop/resource.go` | Rename `InitResource` to `InitializeRoutes` | ARCH-005 |
| `main.go` | Update caller to `shop.InitializeRoutes` | ARCH-005 |
| `shop/rest.go` | Remove dead `Extract()` function | REST-004 |
| `shop/state.go` | **CREATE** — move ShopType, State, CloseReason from model.go | MODEL-004 |
| `shop/model.go` | Remove enum types/constants (moved to state.go) | MODEL-004 |

### Phase 5 — Kafka AndEmit Pattern

| File | Action | Audit Check |
|------|--------|-------------|
| `kafka/message/message.go` | **CREATE** — Buffer, Emit, EmitWithResult | KAFKA-003 |
| `shop/processor.go` | Add producer field; add AndEmit variants to interface | KAFKA-003 |
| `kafka/consumer/merchant/consumer.go` | Migrate handlers to use AndEmit | KAFKA-003 |

### Phase 6 — Documentation, Testing & Polish

| File | Action | Audit Check |
|------|--------|-------------|
| `shop/validation_test.go` | **CREATE** — validation tests | TEST-001 |
| `shop/processor_test.go` | Extend with state machine tests | TEST-001 |
| `listing/provider_test.go` | **CREATE** — provider tests | TEST-001 |
| `services/atlas-merchant/README.md` | Add REST/Kafka tables; fix broken doc links | STRUCT-002 |
| `.bruno/` | **CREATE** — Bruno collection with 4 endpoints | STRUCT-003 |
| `listing/exports.go` | Add rationale comment | ARCH-006 |

---

## Reference Implementations

### administrator.go Pattern
**Source:** `services/atlas-notes/atlas.com/notes/note/administrator.go`
```go
func createNote(db *gorm.DB, tenantId uuid.UUID, note Model) (Model, error) {
    entity := MakeEntity(tenantId, note)
    err := database.ExecuteTransaction(db, func(tx *gorm.DB) error {
        return tx.Create(&entity).Error
    })
    return Make(entity)
}
```

### server.RegisterHandler Pattern
**Source:** `services/atlas-buddies/atlas.com/buddies/list/resource.go`
```go
registerGet := rest.RegisterHandler(l)(si)
r.HandleFunc("", registerGet(GetBuddyList, handleGetBuddyList(db))).Methods(http.MethodGet)
```

### AndEmit Pattern
**Source:** `services/atlas-notes/atlas.com/notes/note/processor.go`
```go
func (p *ProcessorImpl) CreateAndEmit(...) (Model, error) {
    return message.EmitWithResult[Model, byte](p.producer)(p.Create)(args...)
}
func (p *ProcessorImpl) Create(mb *message.Buffer) func(...) (Model, error) {
    // business logic
    mb.Put(topic, eventProvider)
    return model, nil
}
```

### InitializeRoutes Pattern
**Source:** `services/atlas-notes/atlas.com/notes/note/resource.go`
```go
func InitializeRoutes(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
    return func(db *gorm.DB) server.RouteInitializer {
        return func(router *mux.Router, l logrus.FieldLogger) { ... }
    }
}
```

---

## Key Decisions

| # | Decision | Rationale |
|---|----------|-----------|
| 1 | Same-package move for administrator.go (no signature changes) | Minimizes blast radius; callers in same package don't need import changes |
| 2 | Frederick gets full model layer | Returns entities to external packages (consumer), needs proper encapsulation |
| 3 | Message model layer is minimal | Small package, simple CRUD, but guideline compliance requires it |
| 4 | ExitMaintenance returns result struct | Cleanest way to communicate auto-close to consumer without breaking existing pattern |
| 5 | Accept listing exports pattern | Intra-service cross-package access; listing is subdomain of shop aggregate |
| 6 | server.RegisterHandler migration is evaluate-first | May not be compatible; will accept deviation if needed |
| 7 | AndEmit only for high-value operations | PurchaseBundle (4 messages), OpenShop, CloseShop — not all 13 handlers need it |

---

## Dependencies Between Phases

```
Phase 1 ─→ Phase 2  (admin pattern established before subdomain layering)
Phase 1 ─→ Phase 3  (builds must be clean)
Phase 1 ─→ Phase 4  (builds must be clean)
Phase 3 ─→ Phase 5  (event correctness before wrapping in AndEmit)
All    ─→ Phase 6  (tests cover final code state)
```

---

## Build/Test Commands

```bash
# From service directory:
cd services/atlas-merchant/atlas.com/merchant

# Build verification:
go build

# Test execution:
go test ./... -count=1

# Test with race detection:
go test ./... -race -count=1

# Test with coverage:
go test ./... -cover -count=1
```
