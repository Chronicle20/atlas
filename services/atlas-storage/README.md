# atlas-storage

## Responsibility

The atlas-storage service manages account-level storage (warehouse) for items and mesos. It provides persistent storage that is shared across all characters on an account within a world. The service handles deposit, withdrawal, and arrangement of items, maintains in-memory projections for active storage sessions, and participates in item transfer sagas with the inventory system. All item types (equipment, consumables, setup, etc, cash, pets) are represented as a single unified asset model with all fields stored inline.

## External Dependencies

- **PostgreSQL**: Persistent storage for storages and assets
- **Kafka**: Command consumption and event production for storage operations
- **atlas-data**: Item template data for consumables, setup items, and etc items (slotMax, rechargeable flag)

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| `REST_PORT` | HTTP server port |
| `DATABASE_*` | PostgreSQL connection configuration |
| `BOOTSTRAP_SERVERS` | Kafka bootstrap servers |
| `SERVICE_MODE` | `MIXED` (default) enables Kafka consumers; `READ_ONLY` runs REST only |
| `COMMAND_TOPIC_STORAGE` | Kafka topic for storage commands |
| `COMMAND_TOPIC_STORAGE_COMPARTMENT` | Kafka topic for compartment transfer commands |
| `EVENT_TOPIC_STORAGE_STATUS` | Kafka topic for storage status events |
| `EVENT_TOPIC_STORAGE_COMPARTMENT_STATUS` | Kafka topic for compartment status events |
| `EVENT_TOPIC_CHARACTER_STATUS` | Kafka topic for character status events (consumed) |
| `EVENT_TOPIC_ACCOUNT_STATUS` | Kafka topic for account status events (consumed) |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
