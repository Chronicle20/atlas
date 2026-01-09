# atlas-storage

Cross-character, account-level storage microservice for the Atlas platform. Manages shared storage that characters on the same account (within a world) can deposit and withdraw items from.

## Overview

The storage service provides:

- **Account-scoped storage**: Each storage is uniquely identified by `(worldId, accountId)` with a default capacity of 4 slots
- **Asset management**: Supports deposits and withdrawals of items with slot tracking
- **Mesos management**: Store and manage mesos in storage with SET, ADD, and SUBTRACT operations
- **Stackable items**: Local storage of quantity/owner/flag data for consumables, setup, and etc items
- **Reference data**: External lookups for equipables (atlas-equipables) and pets (atlas-pets)
- **Merge and sort**: Intelligent merging of stackable items respecting slotMax limits
- **Saga orchestration**: Integrated with atlas-saga-orchestrator for transactional operations

## Architecture

### Domain Model

```
Storage
├── id (UUID)
├── worldId (byte)
├── accountId (uint32)
├── capacity (uint32, default: 4)
├── mesos (uint32)
└── assets []Asset

Asset
├── id (uint32, auto-increment)
├── storageId (UUID)
├── slot (int16)
├── templateId (uint32)
├── expiration (time.Time)
├── referenceId (uint32)
├── referenceType (string)
└── referenceData (generic)

Stackable (local storage for consumable/setup/etc)
├── assetId (uint32)
├── quantity (uint32)
├── ownerId (uint32)
└── flag (uint16)
```

### Reference Types

| Type | Description | Reference Data Source |
|------|-------------|----------------------|
| `equipable` | Regular equipment | atlas-equipables service |
| `cash_equipable` | Cash shop equipment | atlas-equipables service |
| `consumable` | Use items | Local (stackable table) |
| `setup` | Chair/setup items | Local (stackable table) |
| `etc` | Etc items | Local (stackable table) |
| `cash` | Cash shop items | None |
| `pet` | Pets | atlas-pets service |

## API

### REST Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/storage/accounts/{accountId}?worldId={worldId}` | Get storage metadata (capacity, mesos) |
| POST | `/api/storage/accounts/{accountId}?worldId={worldId}` | Create storage (default capacity: 4, mesos: 0) |
| GET | `/api/storage/accounts/{accountId}/assets?worldId={worldId}` | Get all assets in storage |
| GET | `/api/storage/accounts/{accountId}/assets/{assetId}?worldId={worldId}` | Get single asset by ID |

### Kafka Commands

**Topic**: `COMMAND_TOPIC_STORAGE`

| Command | Description |
|---------|-------------|
| `DEPOSIT` | Deposit item into storage |
| `WITHDRAW` | Withdraw item from storage |
| `UPDATE_MESOS` | Update storage mesos |
| `DEPOSIT_ROLLBACK` | Rollback a deposit (saga compensation) |
| `ARRANGE` | Merge and sort stackable items in storage |

### Kafka Events

**Topic**: `EVENT_TOPIC_STORAGE_STATUS`

| Event | Description |
|-------|-------------|
| `DEPOSITED` | Item was deposited |
| `WITHDRAWN` | Item was withdrawn |
| `MESOS_UPDATED` | Mesos amount changed |
| `ARRANGED` | Items were merged and sorted |
| `ERROR` | Operation failed |

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_HOST` | PostgreSQL host | - |
| `DATABASE_PORT` | PostgreSQL port | 5432 |
| `DATABASE_NAME` | Database name | - |
| `DATABASE_USER` | Database user | - |
| `DATABASE_PASS` | Database password | - |
| `BOOTSTRAP_SERVERS` | Kafka bootstrap servers | - |
| `COMMAND_TOPIC_STORAGE` | Command topic name | - |
| `EVENT_TOPIC_STORAGE_STATUS` | Event topic name | - |
| `DATA_URL` | atlas-data service URL | - |
| `EQUIPABLES_URL` | atlas-equipables service URL | - |
| `PETS_URL` | atlas-pets service URL | - |

## Saga Integration

The storage service integrates with atlas-saga-orchestrator through three actions:

### deposit_to_storage

Deposits an item from character inventory into storage.

```json
{
  "characterId": 12345,
  "accountId": 1001,
  "worldId": 0,
  "slot": 1,
  "templateId": 1302000,
  "referenceId": 100,
  "referenceType": "equipable",
  "expiration": "2025-12-31T23:59:59Z",
  "quantity": 1,
  "ownerId": 0,
  "flag": 0
}
```

### withdraw_from_storage

Withdraws an item from storage to character inventory.

```json
{
  "characterId": 12345,
  "accountId": 1001,
  "worldId": 0,
  "assetId": 500,
  "targetSlot": 1,
  "quantity": 10
}
```

### update_storage_mesos

Updates mesos in storage.

```json
{
  "characterId": 12345,
  "accountId": 1001,
  "worldId": 0,
  "mesos": 10000,
  "operation": "ADD"
}
```

## Merge and Sort

The `MergeAndSort` operation combines stackable items with matching characteristics:

### Merge Rules

1. Only items with the **same ownerId AND same flag** can be merged
2. **Rechargeable consumables** (throwing stars, arrows) cannot be merged
3. Items are merged up to their **slotMax** limit (from atlas-data)
4. After merging, all items are **sorted by templateId**

### slotMax Lookup

| Item Type | Service | Endpoint |
|-----------|---------|----------|
| Consumable | atlas-data | `/data/consumables/{id}` |
| Setup | atlas-data | `/data/setups/{id}` |
| Etc | atlas-data | `/data/etcs/{id}` |

## Development

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o atlas-storage
```

### Database Migrations

Migrations are handled automatically via GORM AutoMigrate on startup.

## Dependencies

- **atlas-data**: Item data lookups (slotMax, rechargeable flag)
- **atlas-equipables**: Equipable item reference data
- **atlas-pets**: Pet reference data
- **atlas-saga-orchestrator**: Transaction coordination
