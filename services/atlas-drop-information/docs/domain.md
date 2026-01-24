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

### Invariants

- All fields are immutable after construction
- Model is constructed via Builder pattern

### Processors

**Processor** (`monster/drop/processor.go`)
- `GetAll` - retrieves all monster drops for the current tenant
- `GetForMonster` - retrieves all drops for a specific monster

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

### Invariants

- All fields are immutable after construction
- Model is constructed via Builder pattern
- `continentId` of -1 indicates a global drop applying to all continents

### Processors

**Processor** (`continent/drop/processor.go`)
- `GetAll` - retrieves all continent drops for the current tenant

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

### Invariants

- All fields are immutable after construction
- Model is constructed via Builder pattern

### Processors

**Processor** (`reactor/drop/processor.go`)
- `GetAll` - retrieves all reactor drops for the current tenant
- `GetForReactor` - retrieves all drops for a specific reactor

---

## Seed

### Responsibility

Handles seeding of drop data from JSON files into the database.

### Core Models

**SeedResult** (`seed/seed.go`)
- `DeletedCount` - number of records deleted
- `CreatedCount` - number of records created
- `FailedCount` - number of records that failed to create
- `Errors` - slice of error messages

**CombinedSeedResult** (`seed/seed.go`)
- `MonsterDrops` - seed result for monster drops
- `ContinentDrops` - seed result for continent drops
- `ReactorDrops` - seed result for reactor drops

### Processors

**Processor** (`seed/processor.go`)
- `Seed` - executes full seed operation for monster, continent, and reactor drops
