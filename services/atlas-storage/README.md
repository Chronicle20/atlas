# atlas-storage

## Responsibility

The atlas-storage service manages account-level storage (warehouse) for items and mesos. It provides persistent storage that is shared across all characters on an account within a world. The service handles deposit, withdrawal, and arrangement of items, maintains in-memory projections for active storage sessions, and participates in item transfer sagas with the inventory system.

## External Dependencies

- **PostgreSQL**: Persistent storage for storages, assets, and stackable data
- **Kafka**: Command consumption and event production for storage operations
- **atlas-data**: Item template data for consumables, setup items, and etc items (slotMax, mergeability)
- **atlas-equipables**: Equipment data decoration for stored equipable items
- **atlas-pets**: Pet data decoration for stored pet items

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| `REST_PORT` | HTTP server port |
| `DATABASE_*` | PostgreSQL connection configuration |
| `COMMAND_TOPIC_STORAGE` | Kafka topic for storage commands |
| `COMMAND_TOPIC_STORAGE_SHOW` | Kafka topic for show/close storage commands |
| `COMMAND_TOPIC_STORAGE_COMPARTMENT` | Kafka topic for compartment transfer commands |
| `EVENT_TOPIC_STORAGE_STATUS` | Kafka topic for storage status events |
| `EVENT_TOPIC_STORAGE_COMPARTMENT_STATUS` | Kafka topic for compartment status events |
| `EVENT_TOPIC_CHARACTER_STATUS` | Kafka topic for character status events (consumed) |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
