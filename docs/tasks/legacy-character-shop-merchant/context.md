# Character Shop & Hired Merchant — Context

Last Updated: 2026-02-25

## Key Files

### Reference Templates (read-only — use as patterns)
| File | Purpose | Why Important |
|------|---------|--------------|
| `services/atlas-npc-shops/atlas.com/npc/main.go` | Service entrypoint | Standard initialization pattern (logger, redis, db, kafka, rest, teardown) |
| `services/atlas-npc-shops/atlas.com/npc/shops/model.go` | Domain model | Immutable model with private fields + accessors |
| `services/atlas-npc-shops/atlas.com/npc/shops/builder.go` | Model builder | Fluent builder pattern with validation |
| `services/atlas-npc-shops/atlas.com/npc/shops/entity.go` | GORM entity | Entity ↔ Model transforms, Migration function |
| `services/atlas-npc-shops/atlas.com/npc/shops/processor.go` | Business logic | Processor interface + implementation, transaction patterns |
| `services/atlas-npc-shops/atlas.com/npc/shops/provider.go` | DB queries | EntityProvider pattern with automatic tenant filtering |
| `services/atlas-npc-shops/atlas.com/npc/shops/registry.go` | Redis registry | TenantRegistry initialization and usage |
| `services/atlas-npc-shops/atlas.com/npc/shops/rest.go` | JSON:API model | RestModel, Transform, Extract, GetReferences |
| `services/atlas-npc-shops/atlas.com/npc/rest/handler.go` | HTTP handlers | Handler registration, route setup |
| `services/atlas-npc-shops/atlas.com/npc/kafka/message/shops/kafka.go` | Kafka messages | Command/Event type definitions |
| `services/atlas-npc-shops/atlas.com/npc/kafka/consumer/shops/consumer.go` | Kafka consumers | Handler registration pattern |
| `services/atlas-npc-shops/atlas.com/npc/kafka/producer/producer.go` | Kafka producers | Event emission via message.Buffer |

### Integration Points (read-only — services we interact with)
| File | Purpose | Integration |
|------|---------|-------------|
| `services/atlas-inventory/atlas.com/inventory/` | Item management | Kafka commands for item acquire/release |
| `services/atlas-character/atlas.com/character/` | Character state | Disconnect events, meso operations |
| `services/atlas-cashshop/atlas.com/cashshop/` | Cash items | Permit verification (514, 503) |
| `services/atlas-maps/atlas.com/maps/` | Map management | Placement validation, FM rooms |

### Shared Libraries (read-only — patterns we leverage)
| File | Purpose | Key API |
|------|---------|---------|
| `libs/atlas-redis/tenant_registry.go` | Multi-tenant Redis | `Get`, `Put`, `Remove`, `Update`, `GetAllValues` |
| `libs/atlas-redis/registry.go` | Generic Redis registry | Base registry pattern |
| `libs/atlas-database/tenant_scope.go` | Auto tenant filtering | GORM callbacks, automatic WHERE injection |
| `libs/atlas-kafka/` | Kafka infrastructure | Consumer/Producer, message.Buffer |
| `libs/atlas-tenant/` | Tenant context | `MustFromContext(ctx)`, `WithContext(ctx, t)` |
| `libs/atlas-model/` | Model providers | `Map`, `FixedProvider`, `SliceProvider` |
| `libs/atlas-rest/` | REST utilities | `ParseCharacterId`, `MarshalResponse`, `HandlerDependency` |

### New Files (to create)
| File | Purpose |
|------|---------|
| `services/atlas-merchant/atlas.com/merchant/main.go` | Service entrypoint |
| `services/atlas-merchant/atlas.com/merchant/go.mod` | Module dependencies |
| `services/atlas-merchant/atlas.com/merchant/shop/model.go` | Shop domain model |
| `services/atlas-merchant/atlas.com/merchant/shop/builder.go` | Shop model builder |
| `services/atlas-merchant/atlas.com/merchant/shop/entity.go` | Shop GORM entity |
| `services/atlas-merchant/atlas.com/merchant/shop/processor.go` | Shop business logic (state machine, purchases) |
| `services/atlas-merchant/atlas.com/merchant/shop/provider.go` | Shop DB providers |
| `services/atlas-merchant/atlas.com/merchant/shop/registry.go` | Active shop Redis registry |
| `services/atlas-merchant/atlas.com/merchant/shop/rest.go` | Shop JSON:API model |
| `services/atlas-merchant/atlas.com/merchant/listing/model.go` | Listing domain model |
| `services/atlas-merchant/atlas.com/merchant/listing/builder.go` | Listing model builder |
| `services/atlas-merchant/atlas.com/merchant/listing/entity.go` | Listing GORM entity |
| `services/atlas-merchant/atlas.com/merchant/listing/processor.go` | Listing business logic |
| `services/atlas-merchant/atlas.com/merchant/listing/provider.go` | Listing DB providers |
| `services/atlas-merchant/atlas.com/merchant/listing/rest.go` | Listing JSON:API model |
| `services/atlas-merchant/atlas.com/merchant/visitor/registry.go` | Visitor Redis registry |
| `services/atlas-merchant/atlas.com/merchant/message/model.go` | Chat message model |
| `services/atlas-merchant/atlas.com/merchant/message/entity.go` | Chat message entity |
| `services/atlas-merchant/atlas.com/merchant/frederick/model.go` | Frederick storage model |
| `services/atlas-merchant/atlas.com/merchant/frederick/entity.go` | Frederick storage entity |
| `services/atlas-merchant/atlas.com/merchant/frederick/provider.go` | Frederick DB providers |
| `services/atlas-merchant/atlas.com/merchant/frederick/reaper.go` | 100-day cleanup reaper |
| `services/atlas-merchant/atlas.com/merchant/placement/processor.go` | Placement validation |
| `services/atlas-merchant/atlas.com/merchant/kafka/message/merchant/kafka.go` | Command/Event types |
| `services/atlas-merchant/atlas.com/merchant/kafka/consumer/merchant/consumer.go` | Command handlers |
| `services/atlas-merchant/atlas.com/merchant/kafka/consumer/character/consumer.go` | Disconnect handler |
| `services/atlas-merchant/atlas.com/merchant/kafka/consumer/inventory/consumer.go` | Inventory confirmations |
| `services/atlas-merchant/atlas.com/merchant/kafka/producer/producer.go` | Event emission |
| `services/atlas-merchant/atlas.com/merchant/rest/handler.go` | HTTP handler setup |
| `services/atlas-merchant/atlas.com/merchant/logger/logger.go` | Logger setup |
| `services/atlas-merchant/atlas.com/merchant/tracing/tracing.go` | Tracing setup |
| `services/atlas-merchant/atlas.com/merchant/service/service.go` | Lifecycle management |
| `services/atlas-merchant/atlas-merchant.yml` | Docker Compose |
| `services/atlas-merchant/Dockerfile` | Production build |
| `services/atlas-merchant/Dockerfile.dev` | Development build |
| `services/atlas-merchant/Dockerfile.debug` | Debug build |

## Architecture Decisions

### Decision 1: Single Service for Both Shop Types
**Chose**: One `atlas-merchant` service with `ShopType` discriminator
**Rationale**: Character shops and hired merchants share 90%+ of domain logic (listings, visitors, purchases, bundles, state machine). Behavioral differences (online requirement, Frederick, expiration) are cleanly handled via strategy/conditional logic within the processor. Two services would duplicate the listing, visitor, purchase, and broadcast code.
**Trade-off**: Slightly more complex processor logic with shop-type conditionals. Acceptable because the shared logic vastly outweighs the differences.

### Decision 2: PostgreSQL as Primary Store (not Redis-only)
**Chose**: PostgreSQL for shop/listing persistence, Redis for ephemeral state
**Rationale**: Hired merchants must survive logout AND server restarts (up to 24 hours). This requires durable persistence. Redis TTLRegistry alone cannot guarantee data integrity for items with real monetary value (mesos). PostgreSQL with GORM entities follows the established Atlas pattern and gets automatic tenant filtering for free.
**Trade-off**: Slightly higher latency for listing reads vs pure Redis. Mitigated by Redis caching of active shop summaries for map display.

### Decision 3: Optimistic Locking for Concurrent Purchases
**Chose**: Version column on listings with `WHERE version = ?` on update
**Rationale**: Multiple buyers can attempt to purchase the last bundle simultaneously. First write wins; others get a version conflict and receive an item-unavailable event. This matches the saga orchestrator's proven pattern and avoids holding row locks during the full purchase validation chain.
**Trade-off**: Failed purchases require re-read. Acceptable because purchase conflicts are rare (most shops have multiple bundles available).

### Decision 4: Item Transfer via Acquire/Release Pattern
**Chose**: Release item from source (character inventory), acquire full snapshot into destination (merchant listing or buyer inventory)
**Rationale**: Items must exist in exactly one place at all times. The acquire/release pattern transfers ownership atomically. Full item snapshots (stats, scrolls, flags) are stored as JSONB in the listing — this is what visitors see and what gets transferred to buyers. Follows the established Atlas ownership transfer pattern.
**Trade-off**: JSONB snapshots increase listing row size. Acceptable because listings are bounded (max 16 per shop) and the snapshot eliminates runtime queries to item data services.

### Decision 5: Frederick as Internal Domain (not atlas-storage)
**Chose**: Frederick storage tables entirely within atlas-merchant (not atlas-storage)
**Rationale**: Frederick is the Hired Merchant-specific NPC for recovering unsold items/mesos after merchant closure. It is unrelated to general character storage (atlas-storage). On hired merchant close, unsold listing item snapshots move directly into `frederick_items` table — no external service call needed. Frederick retrieval releases items back to character inventory via Kafka.
**Trade-off**: Notification/reaper logic adds complexity within the merchant service. Acceptable because Frederick's lifecycle is tightly coupled to hired merchant closure and the data is already local.

### Decision 6: Bundle Model with Ordered Dynamic List
**Chose**: Listings as ordered PostgreSQL rows with display index, no fixed slot system
**Rationale**: The spec explicitly states listings are a dynamic ordered list with collapse on removal. Using an integer `display_order` column that gets recomputed on removal is simpler than managing fixed 16-slot arrays with gaps.
**Trade-off**: Reordering requires remove + re-add (matching spec requirement). Removal triggers an UPDATE of subsequent listings' display_order values.

## Dependencies

### Internal Libraries
- `github.com/Chronicle20/atlas-database` — PostgreSQL, GORM, tenant filtering
- `github.com/Chronicle20/atlas-redis` — TenantRegistry, IndexRegistry
- `github.com/Chronicle20/atlas-kafka` — Consumer/Producer infrastructure
- `github.com/Chronicle20/atlas-tenant` — Multi-tenant context
- `github.com/Chronicle20/atlas-model` — Model providers
- `github.com/Chronicle20/atlas-rest` — REST server, handlers, JSON:API
- `github.com/Chronicle20/atlas-constants` — Item/map constants

### External Libraries
- `gorm.io/gorm` — ORM
- `gorm.io/driver/postgres` — PostgreSQL driver
- `github.com/google/uuid` — UUID generation
- `github.com/sirupsen/logrus` — Logging
- `github.com/redis/go-redis/v9` — Redis client (via atlas-redis)

### Test Dependencies (go.mod only)
- `github.com/alicebob/miniredis/v2` — Redis testing
- `github.com/stretchr/testify` — Assertions

### External Services
| Service | Integration | Direction |
|---------|-------------|-----------|
| `atlas-inventory` | Item acquire/release for listing and purchase | Produce commands, consume confirmations |
| `atlas-character` | Disconnect events, meso operations | Consume events, produce commands |
| `atlas-cashshop` | Permit verification (514, 503) | REST query or Kafka query |
| `atlas-maps` | FM validation, proximity | REST query |

**Note**: Frederick is entirely internal to atlas-merchant — no atlas-storage integration.

### Infrastructure
- PostgreSQL database: `atlas_merchant` (shops, listings, messages, frederick_items, frederick_mesos)
- Redis: shared cluster (existing)
- Kafka: 4 new topics (merchant commands, status events, listing events, frederick notifications)

## Key Constraints

1. **Automatic tenant filtering**: All DB queries auto-filtered — no manual tenant WHERE clauses
2. **Pointer receiver MarshalJSON**: `tenant.Model` uses pointer receiver — always `json.Marshal(&t)`
3. **go.work resolution**: Never add `atlas-redis v0.0.0` to service go.mod
4. **At-least-once Kafka**: All handlers must be idempotent
5. **Message buffer**: Use `message.Buffer` for atomic batch Kafka sends within transactions
6. **String WHERE clauses**: Use `.Where("col = ?", val)` for GORM queries (avoid zero-value gotcha)
