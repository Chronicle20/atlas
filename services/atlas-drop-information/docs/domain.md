# Domain

## Monster Drop

### Responsibility

Represents an item that can drop from a specific monster.

### Core Models

**Model** (`monster/drop/model.go`)
- `tenantId` - UUID identifying the tenant
- `id` - unique drop identifier
- `monsterId` - identifier of the monster that drops the item
- `itemId` - identifier of the dropped item
- `minimumQuantity` - minimum quantity dropped
- `maximumQuantity` - maximum quantity dropped
- `questId` - associated quest identifier (0 if none)
- `chance` - drop chance value

**JSONModel / DropJSON** (`monster/drop/subdomain.go`)
- `JSONModel.Drops` - list of `DropJSON` entries decoded from a catalog file's `attributes.drops`
- `DropJSON` - `itemId`, `minimumQuantity`, `maximumQuantity`, `questId` (omitempty), `chance`

### Invariants

- All fields are immutable after construction
- Model is constructed via Builder pattern
- Builder rejects a nil `tenantId`

### Processors

**Processor** (`monster/drop/processor.go`)
- `GetAll` - retrieves every monster drop for the tenant, unpaged (used by `continent.Processor.GetAll`'s aggregation)
- `GetForMonster` - retrieves a page of drops for a specific monster
- `GetForItem` - retrieves a page of drops for a specific item
- `Count` - returns the total row count for the tenant

**Subdomain** (`monster/drop/subdomain.go`)
- `Build` - constructs `Model` instances from a catalog entity ID (`monster-{monsterId}.json`) and decoded `JSONModel`
- `BulkCreate` - persists built models via `BulkCreateMonsterDrop`
- `DeleteAllForTenant` - removes all monster drops for the tenant
- `Count` - reports current row count for seed status reporting

---

## Continent Drop

### Responsibility

Represents an item that can drop globally across a continent or all continents.

### Core Models

**Model** (`continent/drop/model.go`)
- `tenantId` - UUID identifying the tenant
- `id` - unique drop identifier
- `continentId` - identifier of the continent (-1 for global drops)
- `itemId` - identifier of the dropped item
- `minimumQuantity` - minimum quantity dropped
- `maximumQuantity` - maximum quantity dropped
- `questId` - associated quest identifier (0 if none)
- `chance` - drop chance value

**JSONModel / DropJSON** (`continent/drop/subdomain.go`)
- `JSONModel.Drops` - list of `DropJSON` entries decoded from a catalog file's `attributes.drops`
- `DropJSON` - `itemId`, `minimumQuantity`, `maximumQuantity`, `questId` (omitempty), `chance`

### Invariants

- All fields are immutable after construction
- Model is constructed via Builder pattern
- Builder rejects a nil `tenantId`
- `continentId` of -1 indicates a global drop applying to all continents

### Processors

**Processor** (`continent/drop/processor.go`)
- `GetAll` - retrieves all continent drops for the current tenant
- `Count` - returns the total row count for the tenant

**Subdomain** (`continent/drop/subdomain.go`)
- `Build` - constructs `Model` instances from a catalog entity ID (`continent-{continentId}.json`, signed) and decoded `JSONModel`
- `BulkCreate` - persists built models via `BulkCreateContinentDrop`
- `DeleteAllForTenant` - removes all continent drops for the tenant
- `Count` - reports current row count for seed status reporting

---

## Continent

### Responsibility

Aggregates continent drops grouped by continent identifier.

### Core Models

**Model** (`continent/model.go`)
- `id` - continent identifier
- `drops` - slice of continent drop models

### Processors

**Processor** (`continent/processor.go`)
- `GetAll` - retrieves all continent drops grouped by continent ID

---

## Reactor Drop

### Responsibility

Represents an item that can drop from a specific reactor.

### Core Models

**Model** (`reactor/drop/model.go`)
- `tenantId` - UUID identifying the tenant
- `id` - unique drop identifier
- `reactorId` - identifier of the reactor that drops the item
- `itemId` - identifier of the dropped item
- `questId` - associated quest identifier (0 if none)
- `chance` - drop chance value

**JSONModel / DropJSON** (`reactor/drop/subdomain.go`)
- `JSONModel.Drops` - list of `DropJSON` entries decoded from a catalog file's included `drops` resources
- `DropJSON` - `itemId`, `questId` (omitempty), `chance`

### Invariants

- All fields are immutable after construction
- Model is constructed via Builder pattern
- Builder rejects a nil `tenantId`

### Processors

**Processor** (`reactor/drop/processor.go`)
- `GetAll` - retrieves all reactor drops for the current tenant
- `GetForReactor` - retrieves all drops for a specific reactor
- `Count` - returns the total row count for the tenant

**Subdomain** (`reactor/drop/subdomain.go`)
- `Build` - constructs `Model` instances from a catalog entity ID (`reactor-{reactorId}.json`) and decoded `JSONModel`
- `BulkCreate` - persists built models via `BulkCreateReactorDrop`
- `DeleteAllForTenant` - removes all reactor drops for the tenant
- `Count` - reports current row count for seed status reporting
